package events

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// Producer publishes domain events to Kafka topics
type Producer struct {
	writer *kafka.Writer
}

// NewProducer creates a new Kafka event producer
func NewProducer(brokers []string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:        kafka.TCP(brokers...),
			Compression: kafka.Snappy,
		},
	}
}

// Publish publishes an event to a Kafka topic
func (p *Producer) Publish(ctx context.Context, topic string, event *Envelope) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	payload, err := event.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to serialize event: %w", err)
	}

	message := kafka.Message{
		Key:   []byte(event.AggregateID.String()),
		Value: payload,
		Headers: []kafka.Header{
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "correlation_id", Value: []byte(event.CorrelationID.String())},
		},
	}

	return p.writer.WriteMessages(ctx, message)
}

// Close closes the producer
func (p *Producer) Close() error {
	return p.writer.Close()
}
