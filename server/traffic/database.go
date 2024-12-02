package traffic

import (
	"log"
	"time"

	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/OnnaSoft/lipstick/server/db"
	"gorm.io/gorm"
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
			newConsumption := db.DailyConsumption{
				Domain:    domain,
				Date:      today,
				Month:     month,
				BytesUsed: traffic,
			}
			if createErr := connection.Create(&newConsumption).Error; createErr != nil {
				log.Printf("Error creating DailyConsumption: %v", createErr)
			}
		} else {
			log.Printf("Error querying DailyConsumption: %v", err)
		}
		return
	}

	tx := connection.Model(&dailyConsumption).Update("bytes_used", gorm.Expr("bytes_used + ?", traffic))
	if tx.Error != nil {
		log.Printf("Error updating DailyConsumption: %v", tx.Error)
	}
}
