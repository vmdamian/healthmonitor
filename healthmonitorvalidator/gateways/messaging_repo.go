package gateways

import (
	"context"
	"github.com/segmentio/kafka-go"
	_ "github.com/segmentio/kafka-go/snappy"
	log "github.com/sirupsen/logrus"
	"strings"
	"sync"
	"time"
)

const (
	topicName         = "healthmonitor.validation.requests"
	consumerGroupName = "healthmonitorvalidator"
)

type MessageHandler func(context.Context, string) error

type MessagingRepo struct {
	brokers                  []string
	reader                   *kafka.Reader
	messageValidationHandler MessageHandler
	messageCleanupHandler    MessageHandler
	messageReportHandler     MessageHandler
	wg                       sync.WaitGroup
}

func NewMessagingRepo(brokers []string, messageValidationHandler MessageHandler, messageCleanupHandler MessageHandler, messageReportHandler MessageHandler) *MessagingRepo {
	return &MessagingRepo{
		brokers:                  brokers,
		messageValidationHandler: messageValidationHandler,
		messageCleanupHandler:    messageCleanupHandler,
		messageReportHandler:     messageReportHandler,
		wg:                       sync.WaitGroup{},
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
			message, err := mr.receiveValidationRequest(ctx)
			if err != nil {
				log.WithError(err).Errorln("failed to receive message")
				continue
			}

			switch {
			case strings.HasPrefix(message, "validation"):
				err = mr.messageValidationHandler(ctx, message)
				if err != nil {
					log.WithError(err).Errorf("failed to validate message=%v", message)
				}
			case strings.HasPrefix(message, "cleanup"):
				err = mr.messageCleanupHandler(ctx, message)
				if err != nil {
					log.WithError(err).Errorf("failed to cleanup for message=%v", message)
				}
			case strings.HasPrefix(message, "report-generation"):
				err = mr.messageReportHandler(ctx, message)
				if err != nil {
					log.WithError(err).Errorf("failed to do report generation message=%v", message)
				}
			default:
				log.Errorf("unknown message=%v", message)
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
