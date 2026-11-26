package sinks

import (
	"context"

	"github.com/gomessguii/logger"
)

// FromEnv crea los destinos activados con SQS_ENABLED, KAFKA_ENABLED y
// PUSHER_ENABLED. Un destino mal configurado se registra y se omite.
func FromEnv(ctx context.Context, o Options) *Hub {
	h := NewHub(o)
	if envBool("SQS_ENABLED") {
		if s, err := NewSQSFromEnv(ctx); err != nil {
			logger.LogError("[sinks] sqs disabled: %v", err)
		} else {
			h.Add(s, FilterFromEnv("SQS"))
		}
	}
	if envBool("KAFKA_ENABLED") {
		if k, err := NewKafkaFromEnv(); err != nil {
			logger.LogError("[sinks] kafka disabled: %v", err)
		} else {
			h.Add(k, FilterFromEnv("KAFKA"))
		}
	}
	if envBool("PUSHER_ENABLED") {
		if p, err := NewPusherFromEnv(); err != nil {
			logger.LogError("[sinks] pusher disabled: %v", err)
		} else {
			h.Add(p, FilterFromEnv("PUSHER"))
		}
	}
	return h
}
