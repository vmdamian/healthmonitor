package main

import (
	"context"
	"healthmonitor/healthmonitorvalidator/domain"
	"healthmonitor/healthmonitorvalidator/gateways"
	"healthmonitor/healthmonitorvalidator/usecases"
)

type HealthMonitorValidatorService struct {
	MessagingRepo *gateways.MessagingRepo
	DevicesRepo   *gateways.DevicesRepo
	config        *HealthMonitorValidatorServiceConfig
}

func NewHealthMonitorValidatorService(config *HealthMonitorValidatorServiceConfig) *HealthMonitorValidatorService {

	validators := make([]domain.Validator, 0)
	temperatureValidator := usecases.NewTemperatureValidator(35, 38)
	ecgValidator := usecases.NewECGValidator(200, 1000)
	spo2Validator := usecases.NewOxygenValidator(90, 100)
	heartrateValidator := usecases.NewPulseValidator(60, 120)

	validators = append(validators, temperatureValidator, ecgValidator, spo2Validator, heartrateValidator)

	devicesRepo := gateways.NewDevicesRepo(config.ElasticsearchHost)
	alertSender := gateways.NewAlertSender(config.AlertSenderAccountSID, config.AlertSenderToken, config.AlertSenderPhoneNumber)

	alertGenerator := usecases.NewAlertGenerator(validators, devicesRepo, alertSender, config.ValidationPeriod, config.SendCreatedAlert,
		config.SendContinuedAlert, config.SendResolvedAlert)
	reportGenerator := usecases.NewReportGenerator(devicesRepo)
	cleanupGenerator := usecases.NewCleanupGenerator(devicesRepo)

	messagingRepo := gateways.NewMessagingRepo(config.KafkaBrokers, config.KafkaWorkerCount,
		alertGenerator.GenerateUpdateAndSendAlertsForDevice,
		cleanupGenerator.GenerateCleanup,
		reportGenerator.GenerateReport)

	service := &HealthMonitorValidatorService{
		MessagingRepo: messagingRepo,
		DevicesRepo:   devicesRepo,
		config:        config,
	}

	return service
}
func (s *HealthMonitorValidatorService) Start(ctx context.Context) error {

	err := s.DevicesRepo.Start()
	if err != nil {
		return err
	}

	// This has to be the last function here because it is the main loop of the service, receiving messages, validating
	// and generating errors.
	// TODO: Change the design to have a valid start/stop logic.
	s.MessagingRepo.Start(ctx)
	s.MessagingRepo.Stop()

	return nil
}
