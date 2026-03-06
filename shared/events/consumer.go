package events

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

// Consumer reads domain events from Kafka topics
type Consumer struct {
	reader *kafka.Reader
}

// NewConsumer creates a new Kafka event consumer
func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Topic:          topic,
			GroupID:        groupID,
			CommitInterval: kafka.DefaultCommitInterval,
			StartOffset:    kafka.LastOffset,
		}),
	}
}

// Consume reads the next message from the topic
func (c *Consumer) Consume(ctx context.Context) (*Envelope, error) {
	message, err := c.reader.FetchMessage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read message: %w", err)
	}

	event, err := FromJSON(message.Value)
	if err != nil {
		return nil, fmt.Errorf("failed to parse event: %w", err)
	}

	// Commit the message after successful processing
	if err := c.reader.CommitMessages(ctx, message); err != nil {
		// Log but don't fail - commit errors are non-critical
		fmt.Printf("failed to commit message: %v\n", err)
	}

	return event, nil
}

// Close closes the consumer
func (c *Consumer) Close() error {
	return c.reader.Close()
}
