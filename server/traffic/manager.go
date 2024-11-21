package traffic

import (
	"log"
	"sync"
	"time"

	"github.com/OnnaSoft/lipstick/server/db"
)

type TrafficManager struct {
	mu          sync.Mutex
	trafficData map[string]int64 // Maps Domain (name) to bytes used
	threshold   int64            // Threshold for updating the database
	dbChan      chan TrafficEvent
}

type TrafficEvent struct {
	Domain string
	Bytes  int64
}

// NewTrafficManager initializes a new TrafficManager
func NewTrafficManager(threshold int64) *TrafficManager {
	manager := &TrafficManager{
		trafficData: make(map[string]int64),
		threshold:   threshold,
		dbChan:      make(chan TrafficEvent, 100),
	}
	go manager.processTraffic()
	return manager
}

// AddTraffic adds traffic data for a specific domain
func (tm *TrafficManager) AddTraffic(domain string, bytes int64) {
	//fmt.Println("AddTraffic", domain, bytes)
	//fmt.Println(TrafficEvent{Domain: domain, Bytes: bytes})
	tm.dbChan <- TrafficEvent{Domain: domain, Bytes: bytes}
}

// processTraffic listens to the channel and processes traffic events
func (tm *TrafficManager) processTraffic() {
	for event := range tm.dbChan {
		tm.mu.Lock()
		tm.trafficData[event.Domain] += event.Bytes
		currentTraffic := tm.trafficData[event.Domain]
		tm.mu.Unlock()

		if currentTraffic >= tm.threshold {
			tm.updateDatabase(event.Domain)
		}
	}
}

// updateDatabase writes the accumulated traffic data to the database
func (tm *TrafficManager) updateDatabase(domain string) {
	tm.mu.Lock()
	traffic := tm.trafficData[domain]
	tm.trafficData[domain] = 0
	tm.mu.Unlock()

	connection, err := db.GetConnection()
	if err != nil {
		log.Printf("Error connecting to database: %v", err)
		return
	}

	// Get or create the DailyConsumption record
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

	// Update BytesUsed
	err = connection.Model(&dailyConsumption).Update("BytesUsed", dailyConsumption.BytesUsed+traffic).Error
	if err != nil {
		log.Printf("Error updating DailyConsumption: %v", err)
		return
	}
}

// Close stops the TrafficManager by closing the channel
func (tm *TrafficManager) Close() {
	close(tm.dbChan)
}
