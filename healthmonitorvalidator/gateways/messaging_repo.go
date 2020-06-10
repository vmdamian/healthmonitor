package gateways

import (
	"context"
	"github.com/segmentio/kafka-go"
	_ "github.com/segmentio/kafka-go/snappy"
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

const (
	topicName = "healthmonitor.validation.requests"
	consumerGroupName = "healthmonitorvalidator"
)

type MessagingRepo struct {
	brokers []string
	reader *kafka.Reader
	wg sync.WaitGroup
}

func NewMessagingRepo(brokers []string) *MessagingRepo {
	return &MessagingRepo{
		brokers: brokers,
		wg: sync.WaitGroup{},
	}
}

func (mr *MessagingRepo) Start(ctx context.Context) {

	dialer := &kafka.Dialer{
		Timeout: 10 * time.Second,
		DualStack: true,
	}

	config := kafka.ReaderConfig{
		Brokers: mr.brokers,
		Topic: topicName,
		GroupID: consumerGroupName,
		Dialer: dialer,
	}

	mr.reader = kafka.NewReader(config)

	mr.wg.Add(1)

	go func() {
		for {
			err := mr.receiveValidationRequest(ctx)
			if err != nil {
				log.WithError(err).Errorln("failed to receive message")
				break
			}
		}

		mr.wg.Done()
	}()

	mr.wg.Wait()
}

func (mr *MessagingRepo) Stop() {
	_ = mr.reader.Close()
}

func (mr *MessagingRepo) receiveValidationRequest(ctx context.Context) error {
	msg, err := mr.reader.FetchMessage(ctx)
	if err != nil {
		return err
	}

	log.Infof("got message %v with key %v on topic %v partition %v offset %v", string(msg.Value), string(msg.Key), msg.Topic, msg.Partition, msg.Offset)

	err = mr.reader.CommitMessages(ctx, msg)
	if err != nil {
		return err
	}

	return nil
}