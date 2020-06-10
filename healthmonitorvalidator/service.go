package main

import (
	"context"
	"healthmonitor/healthmonitorvalidator/gateways"
)

type HealthMonitorValidatorService struct {
	MessagingRepo *gateways.MessagingRepo
	config *HealthMonitorValidatorServiceConfig
}

func NewHealthMonitorValidatorService(config *HealthMonitorValidatorServiceConfig) *HealthMonitorValidatorService {
	messagingRepo := gateways.NewMessagingRepo(config.KafkaBrokers)

	service := &HealthMonitorValidatorService{
		MessagingRepo: messagingRepo,
		config: config,
	}

	return service
}
func (s *HealthMonitorValidatorService) Start(ctx context.Context) error {

	// This has to be the last function here because it is the main loop of the service, receiving messages, validating
	// and generating errors.
	// TODO: Change the design to have a valid start/stop logic.
	s.MessagingRepo.Start(ctx)
	s.MessagingRepo.Stop()

	return nil
}