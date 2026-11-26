package sinks

import (
	"context"
	"crypto/tls"
	"errors"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// Kafka publica en un tema (o en uno por evento) con la línea como clave.
type Kafka struct {
	w        *kafka.Writer
	topic    string
	perEvent bool
}

// NewKafkaFromEnv usa KAFKA_BROKERS, KAFKA_TOPIC (evolution-events),
// KAFKA_TOPIC_PER_EVENT, KAFKA_SASL_MECHANISM (plain, scram-sha-256,
// scram-sha-512), KAFKA_SASL_USERNAME, KAFKA_SASL_PASSWORD y KAFKA_TLS.
func NewKafkaFromEnv() (*Kafka, error) {
	brokers := envList("KAFKA_BROKERS")
	if len(brokers) == 0 {
		return nil, errors.New("KAFKA_BROKERS is required")
	}
	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "evolution-events"
	}
	tr := &kafka.Transport{DialTimeout: 10 * time.Second}
	if envBool("KAFKA_TLS") {
		tr.TLS = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	user, pass := os.Getenv("KAFKA_SASL_USERNAME"), os.Getenv("KAFKA_SASL_PASSWORD")
	if user != "" {
		var mech sasl.Mechanism
		var err error
		switch strings.ToLower(os.Getenv("KAFKA_SASL_MECHANISM")) {
		case "scram-sha-256":
			mech, err = scram.Mechanism(scram.SHA256, user, pass)
		case "scram-sha-512":
			mech, err = scram.Mechanism(scram.SHA512, user, pass)
		default:
			mech = plain.Mechanism{Username: user, Password: pass}
		}
		if err != nil {
			return nil, err
		}
		tr.SASL = mech
	}
	w := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Balancer:               &kafka.Hash{},
		Transport:              tr,
		AllowAutoTopicCreation: true,
		RequiredAcks:           kafka.RequireOne,
		BatchTimeout:           50 * time.Millisecond,
	}
	return &Kafka{w: w, topic: topic, perEvent: envBool("KAFKA_TOPIC_PER_EVENT")}, nil
}

func (k *Kafka) Name() string { return "kafka" }

// TopicFor es el tema de un evento.
func (k *Kafka) TopicFor(event string) string {
	if !k.perEvent {
		return k.topic
	}
	return k.topic + "." + sanitize(strings.ToLower(event))
}

func (k *Kafka) Send(ctx context.Context, m Message) error {
	return k.w.WriteMessages(ctx, kafka.Message{
		Topic: k.TopicFor(m.Event),
		Key:   []byte(m.InstanceID),
		Value: m.Payload,
		Headers: []kafka.Header{
			{Key: "event", Value: []byte(m.Event)},
			{Key: "instance", Value: []byte(m.InstanceName)},
		},
	})
}

func (k *Kafka) Close() error { return k.w.Close() }
