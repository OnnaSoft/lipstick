package traffic

import (
	"log"
	"sync"
	"time"

	"github.com/OnnaSoft/lipstick/server/db"
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
		dbTicker:    time.NewTicker(5 * time.Minute),
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

func (tm *TrafficManager) updateDatabase(domain string, traffic int64) {
	connection, err := db.GetConnection()
	if err != nil {
		log.Printf("Error connecting to database: %v", err)
		return
	}

	today := time.Now().Truncate(24 * time.Hour)
	var dailyConsumption db.DailyConsumption
	err = connection.FirstOrCreate(
		&dailyConsumption,
		db.DailyConsumption{Domain: domain, Date: today},
	).Error
	if err != nil {
		log.Printf("Error querying DailyConsumption: %v", err)
		return
	}

	err = connection.Model(&dailyConsumption).Update("BytesUsed", dailyConsumption.BytesUsed+traffic).Error
	if err != nil {
		log.Printf("Error updating DailyConsumption: %v", err)
	}
}

func (tm *TrafficManager) Close() {
	close(tm.stopChan)

	tm.saveAllToDatabase()
}
