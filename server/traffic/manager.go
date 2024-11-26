package traffic

import (
	"sync"
	"time"
)

type TrafficManager struct {
	mu          sync.Mutex
	trafficData map[string]int64
	threshold   int64
	dbTicker    *time.Ticker
	stopChan    chan struct{}
}

func NewTrafficManager(threshold int64) *TrafficManager {
	manager := &TrafficManager{
		trafficData: make(map[string]int64),
		threshold:   threshold,
		dbTicker:    time.NewTicker(5 * time.Second),
		stopChan:    make(chan struct{}),
	}
	go manager.run()
	return manager
}

func (tm *TrafficManager) AddTraffic(domain string, bytes int64) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.trafficData[domain] += bytes
}

func (tm *TrafficManager) run() {
	for {
		select {
		case <-tm.dbTicker.C:
			tm.saveAllToDatabase()
		case <-tm.stopChan:
			tm.dbTicker.Stop()
			return
		}
	}
}

func (tm *TrafficManager) saveAllToDatabase() {
	tm.mu.Lock()
	dataToSave := make(map[string]int64)
	for domain, bytes := range tm.trafficData {
		if bytes > 0 {
			dataToSave[domain] = bytes
			tm.trafficData[domain] = 0
		}
	}
	tm.mu.Unlock()

	for domain, traffic := range dataToSave {
		tm.updateDatabase(domain, traffic)
	}
}

func (tm *TrafficManager) Close() {
	close(tm.stopChan)

	tm.saveAllToDatabase()
}
