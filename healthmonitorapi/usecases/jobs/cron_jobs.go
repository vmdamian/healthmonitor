package jobs

import (
	"context"
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorapi/gateways"
	"time"
)

type CronJobRunner struct {
	messagingRepo   *gateways.MessagingRepo
	interval        time.Duration
	maxDatapointAge time.Duration

	quitChan chan struct{}
	doneChan chan struct{}
}

func NewCronJobRunner(messagingRepo *gateways.MessagingRepo, interval time.Duration, maxDatapointAge time.Duration) *CronJobRunner {
	return &CronJobRunner{
		messagingRepo:   messagingRepo,
		interval:        interval,
		maxDatapointAge: maxDatapointAge,

		quitChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

func (c *CronJobRunner) Start(ctx context.Context) {

	ticker := time.NewTicker(c.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				err := c.messagingRepo.SendCleanupRequest(ctx, time.Now().Add((-1)*c.maxDatapointAge).Format(time.RFC3339))
				if err != nil {
					log.WithError(err).Errorf("got error trying to send cleanup request")
				}
			case <-c.quitChan:
				ticker.Stop()
				close(c.doneChan)
				return
			}
		}
	}()
}

func (c *CronJobRunner) Stop() {
	close(c.quitChan)
	<-c.doneChan
}
