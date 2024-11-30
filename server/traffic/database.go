package traffic

import (
	"log"
	"time"

	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/OnnaSoft/lipstick/server/db"
	"github.com/jinzhu/gorm"
)

func (tm *TrafficManager) updateDatabase(domain string, traffic int64) {
	conf, err := config.GetConfig()
	if err != nil {
		log.Fatalf("Error getting config: %v", err)
		return
	}
	connection, err := db.GetConnection(conf.Database)
	if err != nil {
		log.Printf("Error connecting to database: %v", err)
		return
	}

	today := time.Now().Truncate(24 * time.Hour)
	month := today.Format("2006-01")

	var dailyConsumption db.DailyConsumption

	err = connection.Where("domain = ? AND date = ?", domain, today).First(&dailyConsumption).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			dailyConsumption = db.DailyConsumption{
				Domain:    domain,
				Date:      today,
				Month:     month,
				BytesUsed: traffic,
			}
			if err = connection.Create(&dailyConsumption).Error; err != nil {
				log.Printf("Error creating DailyConsumption: %v", err)
			}
		} else {
			log.Printf("Error querying DailyConsumption: %v", err)
		}
		return
	}

	err = connection.Model(&dailyConsumption).Update("BytesUsed", dailyConsumption.BytesUsed+traffic).Error
	if err != nil {
		log.Printf("Error updating DailyConsumption: %v", err)
	}
}
