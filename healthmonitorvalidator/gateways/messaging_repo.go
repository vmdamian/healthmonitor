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
	topicName         = "healthmonitor.validation.requests"
	consumerGroupName = "healthmonitorvalidator"
)

type MessageHandler func(context.Context, string) error

type MessagingRepo struct {
	brokers        []string
	reader         *kafka.Reader
	messageHandler MessageHandler
	wg             sync.WaitGroup
}

func NewMessagingRepo(brokers []string, messageHandler MessageHandler) *MessagingRepo {
	return &MessagingRepo{
		brokers:        brokers,
		messageHandler: messageHandler,
		wg:             sync.WaitGroup{},
	}
}

func (mr *MessagingRepo) Start(ctx context.Context) {

	dialer := &kafka.Dialer{
		Timeout:   10 * time.Second,
		DualStack: true,
	}

	config := kafka.ReaderConfig{
		Brokers: mr.brokers,
		Topic:   topicName,
		GroupID: consumerGroupName,
		Dialer:  dialer,
	}

	mr.reader = kafka.NewReader(config)

	mr.wg.Add(1)

	go func() {
		for {
			did, err := mr.receiveValidationRequest(ctx)
			if err != nil {
				log.WithError(err).Errorln("failed to receive message")
				continue
			}

			err = mr.messageHandler(ctx, did)
			if err != nil {
				log.WithError(err).Errorf("failed to validate message for did=%v", did)
			}
		}

		mr.wg.Done()
	}()

	mr.wg.Wait()
}

func (mr *MessagingRepo) Stop() {
	_ = mr.reader.Close()
}

func (mr *MessagingRepo) receiveValidationRequest(ctx context.Context) (string, error) {
	msg, err := mr.reader.FetchMessage(ctx)
	if err != nil {
		return "", err
	}

	err = mr.reader.CommitMessages(ctx, msg)
	if err != nil {
		return "", err
	}

	return string(msg.Value), nil
}
