package sinks

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/google/uuid"
)

const sqsMaxBytes = 256 * 1024

// SQS publica en una cola de Amazon SQS (o compatible, con SQS_ENDPOINT).
type SQS struct {
	client *sqs.Client
	url    string
	fifo   bool
}

// NewSQSFromEnv usa SQS_QUEUE_URL, SQS_REGION, SQS_ACCESS_KEY_ID,
// SQS_SECRET_ACCESS_KEY y SQS_ENDPOINT (las credenciales son opcionales: sin
// ellas se usa la cadena por defecto de AWS).
func NewSQSFromEnv(ctx context.Context) (*SQS, error) {
	url := os.Getenv("SQS_QUEUE_URL")
	if url == "" {
		return nil, errors.New("SQS_QUEUE_URL is required")
	}
	opts := []func(*awsconfig.LoadOptions) error{}
	if r := os.Getenv("SQS_REGION"); r != "" {
		opts = append(opts, awsconfig.WithRegion(r))
	}
	if k := os.Getenv("SQS_ACCESS_KEY_ID"); k != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(k, os.Getenv("SQS_SECRET_ACCESS_KEY"), "")))
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	endpoint := os.Getenv("SQS_ENDPOINT")
	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
	return &SQS{client: client, url: url, fifo: strings.HasSuffix(url, ".fifo")}, nil
}

func (s *SQS) Name() string { return "sqs" }

func (s *SQS) Send(ctx context.Context, m Message) error {
	if len(m.Payload) > sqsMaxBytes {
		return errors.New("event larger than 256 KB (SQS limit)")
	}
	in := &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.url),
		MessageBody: aws.String(string(m.Payload)),
		MessageAttributes: map[string]types.MessageAttributeValue{
			"event":      {DataType: aws.String("String"), StringValue: aws.String(m.Event)},
			"instanceId": {DataType: aws.String("String"), StringValue: aws.String(nonEmpty(m.InstanceID))},
			"instance":   {DataType: aws.String("String"), StringValue: aws.String(nonEmpty(m.InstanceName))},
		},
	}
	if s.fifo {
		in.MessageGroupId = aws.String(nonEmpty(m.InstanceID))
		in.MessageDeduplicationId = aws.String(uuid.NewString())
	}
	_, err := s.client.SendMessage(ctx, in)
	return err
}

func (s *SQS) Close() error { return nil }

// nonEmpty evita atributos vacíos, que SQS rechaza.
func nonEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
