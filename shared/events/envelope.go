package events

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Envelope is the base event structure that all domain events must conform to
type Envelope struct {
	EventID         uuid.UUID              `json:"event_id"`
	EventType       string                 `json:"event_type"`
	AggregateID     uuid.UUID              `json:"aggregate_id"`
	AggregateType   string                 `json:"aggregate_type"`
	Version         int32                  `json:"version"`
	Timestamp       time.Time              `json:"timestamp"`
	CorrelationID   uuid.UUID              `json:"correlation_id"`
	ProducerService string                 `json:"producer_service"`
	Payload         json.RawMessage        `json:"payload"`
}

// NewEnvelope creates a new event envelope with default values
func NewEnvelope(
	eventType string,
	aggregateID uuid.UUID,
	aggregateType string,
	producerService string,
	payload interface{},
) (*Envelope, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return &Envelope{
		EventID:         uuid.New(),
		EventType:       eventType,
		AggregateID:     aggregateID,
		AggregateType:   aggregateType,
		Version:         1,
		Timestamp:       time.Now().UTC(),
		CorrelationID:   uuid.New(),
		ProducerService: producerService,
		Payload:         payloadBytes,
	}, nil
}

// WithCorrelationID sets the correlation ID for tracing
func (e *Envelope) WithCorrelationID(correlationID uuid.UUID) *Envelope {
	e.CorrelationID = correlationID
	return e
}

// Validate checks if the envelope conforms to the specification
func (e *Envelope) Validate() error {
	var errs []error

	if e.EventID == uuid.Nil {
		errs = append(errs, errors.New("event_id is required"))
	}
	if e.EventType == "" {
		errs = append(errs, errors.New("event_type is required"))
	}
	if e.AggregateID == uuid.Nil {
		errs = append(errs, errors.New("aggregate_id is required"))
	}
	if e.AggregateType == "" {
		errs = append(errs, errors.New("aggregate_type is required"))
	}
	if e.Version <= 0 {
		errs = append(errs, errors.New("version must be > 0"))
	}
	if e.Timestamp.IsZero() {
		errs = append(errs, errors.New("timestamp is required"))
	}
	if e.CorrelationID == uuid.Nil {
		errs = append(errs, errors.New("correlation_id is required"))
	}
	if e.ProducerService == "" {
		errs = append(errs, errors.New("producer_service is required"))
	}
	if len(e.Payload) == 0 {
		errs = append(errs, errors.New("payload is required"))
	}

	if len(errs) > 0 {
		return fmt.Errorf("envelope validation failed: %v", errs)
	}
	return nil
}

// ToJSON marshals the envelope to JSON
func (e *Envelope) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON unmarshals JSON into an Envelope
func FromJSON(data []byte) (*Envelope, error) {
	var e Envelope
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("failed to unmarshal envelope: %w", err)
	}
	return &e, nil
}
