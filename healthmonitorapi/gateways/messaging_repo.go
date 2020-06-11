package gateways

import (
	"context"
	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/snappy"
)

const (
	topicName = "healthmonitor.validation.requests"
)

type MessagingRepo struct {
	brokers []string
	writer  *kafka.Writer
}

func NewMessagingRepo(brokers []string) *MessagingRepo {
	return &MessagingRepo{
		brokers: brokers,
	}
}

func (mr *MessagingRepo) Start() {

	config := kafka.WriterConfig{
		Brokers:          mr.brokers,
		Topic:            topicName,
		Balancer:         &kafka.Murmur2Balancer{},
		CompressionCodec: snappy.NewCompressionCodec(),
	}

	mr.writer = kafka.NewWriter(config)
}

func (mr *MessagingRepo) Stop() {
	_ = mr.writer.Close()
}

func (mr *MessagingRepo) SendValidationRequest(ctx context.Context, did string) error {
	message := kafka.Message{
		Key:   []byte(did),
		Value: []byte(did),
	}

	return mr.writer.WriteMessages(ctx, message)
}
