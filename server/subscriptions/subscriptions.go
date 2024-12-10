package subscriptions

import (
	"fmt"
	"sync"

	"github.com/OnnaSoft/lipstick/logger"
	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/nats-io/nats.go"
)

var managerOnce sync.Once

type SubscriptionManager struct {
	conn *nats.Conn
}

var DefaultSubscriptionManager *SubscriptionManager

func GetSubscriptionManager() (*SubscriptionManager, error) {
	var err error
	managerOnce.Do(func() {
		conf, err := config.GetConfig()
		if err != nil {
			logger.Default.Error("Error getting config:", err)
			return
		}
		DefaultSubscriptionManager, err = NewSubscriptionManager(conf.Nats.URL)
	})

	return DefaultSubscriptionManager, err
}

func NewSubscriptionManager(url string) (*SubscriptionManager, error) {
	conn, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("error connecting to NATS: %v", err)
	}

	return &SubscriptionManager{conn: conn}, nil
}

func (s *SubscriptionManager) Subscribe(topic string, callback func(msg *nats.Msg)) (*nats.Subscription, error) {
	sub, err := s.conn.QueueSubscribe(topic, "all", callback)
	if err != nil {
		return nil, fmt.Errorf("error subscribing to topic %s: %v", topic, err)
	}

	logger.Default.Info("Subscribed to topic: ", topic)
	return sub, nil
}

func (s *SubscriptionManager) Unsubscribe() {
	s.conn.Drain()
	logger.Default.Info("All subscriptions drained and connection closed")
}

func (s *SubscriptionManager) Publish(topic string, message []byte) error {
	err := s.conn.Publish(topic, message)
	if err != nil {
		return fmt.Errorf("error publishing to topic %s: %v", topic, err)
	}

	logger.Default.Debug("Published message to topic: ", topic)
	return nil
}

func (s *SubscriptionManager) Close() {
	s.conn.Close()
	logger.Default.Info("NATS connection closed")
}
