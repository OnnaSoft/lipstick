package subscriptions

import (
	"context"
	"fmt"
	"log"

	"github.com/OnnaSoft/lipstick/server/config"
	"github.com/streadway/amqp"
)

type RabbitMQManager struct {
	conn          *amqp.Connection
	channel       *amqp.Channel
	subscriptions map[string]chan struct{}
	ctx           context.Context
	cancel        context.CancelFunc
}

var DefaultRabbitMQManager *RabbitMQManager

func GetRabbitMQManager(cfg ...config.RabbitMQConfig) (*RabbitMQManager, error) {
	if DefaultRabbitMQManager == nil {
		if len(cfg) == 0 {
			return nil, fmt.Errorf("RabbitMQ configuration not provided")
		}

		rmq, err := NewRabbitMQManager(cfg[0])
		if err != nil {
			return nil, err
		}
		DefaultRabbitMQManager = rmq
	}

	return DefaultRabbitMQManager, nil
}

func NewRabbitMQManager(cfg config.RabbitMQConfig) (*RabbitMQManager, error) {
	url := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.VHost,
	)

	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open a channel: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &RabbitMQManager{
		conn:          conn,
		channel:       channel,
		subscriptions: make(map[string]chan struct{}),
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

func (rmq *RabbitMQManager) PublishMessage(queueName, message string) error {
	_, err := rmq.channel.QueueDeclare(queueName, false, false, false, false, nil)
	if err != nil {
		return err
	}

	err = rmq.channel.Publish(
		"",        // Exchange
		queueName, // Routing key
		false,     // Mandatory
		true,      // Immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(message),
		},
	)

	if err != nil {
		log.Printf("Error publishing message: %v", err)
		return err
	}

	log.Printf("Message published to queue '%s': %s", queueName, message)
	return nil
}

// SubscribeToQueue suscribe un consumidor a una cola
func (rmq *RabbitMQManager) SubscribeToQueue(queueName string, handler func(message string) error) error {
	_, err := rmq.channel.QueueDeclare(queueName, false, false, false, false, nil)
	if err != nil {
		return err
	}

	messages, err := rmq.channel.Consume(
		queueName, // Queue
		"",        // Consumer
		true,      // Auto-Ack
		false,     // Exclusive
		false,     // No-Local
		false,     // No-Wait
		nil,       // Args
	)
	if err != nil {
		return err
	}

	stopChan := make(chan struct{})
	rmq.subscriptions[queueName] = stopChan

	go func() {
		for {
			select {
			case <-stopChan:
				log.Printf("Unsubscribed from queue '%s'\n", queueName)
				return
			case msg := <-messages:
				if err := handler(string(msg.Body)); err != nil {
					log.Printf("Error processing message: %v\n", err)
				}
			}
		}
	}()

	return nil
}

// UnsubscribeFromQueue permite detener una suscripción activa
func (rmq *RabbitMQManager) UnsubscribeFromQueue(queueName string) {
	if stopChan, ok := rmq.subscriptions[queueName]; ok {
		close(stopChan)
		delete(rmq.subscriptions, queueName)
		log.Printf("Unsubscribed successfully from queue '%s'\n", queueName)
	} else {
		log.Printf("No active subscription found for queue '%s'\n", queueName)
	}
}

// Close cierra todas las conexiones activas y canales
func (rmq *RabbitMQManager) Close() {
	rmq.cancel()
	for queueName := range rmq.subscriptions {
		rmq.UnsubscribeFromQueue(queueName)
	}

	if err := rmq.channel.Close(); err != nil {
		log.Printf("Error closing channel: %v", err)
	}

	if err := rmq.conn.Close(); err != nil {
		log.Printf("Error closing connection: %v", err)
	}
	log.Println("RabbitMQ connection closed.")
}
