package usecases

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorvalidator/domain"
	"healthmonitor/healthmonitorvalidator/gateways"
	"sync"
	"time"
)

type AlertGenerator struct {
	validators       []domain.Validator
	devicesRepo      *gateways.DevicesRepo
	alertSender      *gateways.AlertSender
	validationPeriod time.Duration

	alertCreated   bool
	alertContinued bool
	alertResolved  bool

	lock sync.Mutex
}

func NewAlertGenerator(validators []domain.Validator, devicesRepo *gateways.DevicesRepo, alertSender *gateways.AlertSender, validationPeriod time.Duration,
	alertCreated, alertContinued, alertResolved bool) *AlertGenerator {
	return &AlertGenerator{
		validators:       validators,
		devicesRepo:      devicesRepo,
		alertSender:      alertSender,
		validationPeriod: validationPeriod,
		alertCreated:     alertCreated,
		alertContinued:   alertContinued,
		alertResolved:    alertResolved,
		lock: sync.Mutex{},
	}
}

func (ag *AlertGenerator) GenerateUpdateAndSendAlertsForDevice(ctx context.Context, message string) error {

	var did string
	n, err := fmt.Sscanf(message, "validation_%v", &did)
	if err != nil || n != 1 {
		return fmt.Errorf("could not parse did from validation request = %v", message)
	}

	ag.lock.Lock()
	deviceInfo, err := ag.devicesRepo.GetDeviceInfo(ctx, did)
	if err != nil {
		return err
	}

	if time.Since(deviceInfo.LastValidationTimestamp) <= 5 * time.Second {
		ag.lock.Unlock()
		return nil
	}
	err = ag.devicesRepo.UpdateDeviceInfo(ctx, deviceInfo.DID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update last validation timestamp for device %v", deviceInfo.DID)
	}
	ag.lock.Unlock()

	newAlerts, err := ag.generateAlertsForDevice(ctx, did)
	if err != nil {
		return err
	}

	oldAlerts, err := ag.devicesRepo.GetDeviceAlerts(ctx, did, domain.ALERT_STATUS_ACTIVE)
	if err != nil {
		return err
	}

	alertUpdates := getAlertUpdates(oldAlerts, newAlerts)

	if len(alertUpdates) == 0 {
		return nil
	}

	err = ag.devicesRepo.UpsertAlerts(ctx, alertUpdates)
	if err != nil {
		return err
	}

	alertsToSend := make([]*domain.Alert, 0)
	for _, alertUpdate := range alertUpdates {
		if ag.alertCreated && alertUpdate.UpdateType == domain.ALERT_UPDATE_TYPE_CREATED {
			alertsToSend = append(alertsToSend, alertUpdate.Alert)
		} else if ag.alertContinued && alertUpdate.UpdateType == domain.ALERT_UPDATE_TYPE_CONTINUED {
			alertsToSend = append(alertsToSend, alertUpdate.Alert)
		} else if ag.alertResolved && alertUpdate.UpdateType == domain.ALERT_UPDATE_TYPE_RESOLVED {
			alertsToSend = append(alertsToSend, alertUpdate.Alert)
		}
	}

	if len(alertsToSend) == 0 {
		return nil
	}

	var failedPhones []string
	for _, subscribedPhone := range deviceInfo.SubscribedPhones {
		err = ag.alertSender.SendAlerts(subscribedPhone, did, alertsToSend)
		if err != nil {
			log.WithError(err).Errorf("failed to send notification to phoneNumber=%v", subscribedPhone)
			failedPhones = append(failedPhones, subscribedPhone)
		}
	}

	if len(failedPhones) > 0 {
		return fmt.Errorf("failed to send notifications to phoneNumbers=%v", failedPhones)
	}

	return nil
}

func (ag *AlertGenerator) generateAlertsForDevice(ctx context.Context, did string) ([]*domain.Alert, error) {
	dateSince := time.Now().Add((-1) * ag.validationPeriod)
	dataSet, err := ag.devicesRepo.GetDeviceData(ctx, did, dateSince)
	if err != nil {
		return nil, err
	}

	validateWg := sync.WaitGroup{}
	collectWg := sync.WaitGroup{}
	alertsChan := make(chan *domain.Alert)
	generatedAlerts := make([]*domain.Alert, 0)

	for _, validator := range ag.validators {
		validateWg.Add(1)
		go func(validator domain.Validator) {
			alerts := validator.CheckData(dataSet)
			for _, alert := range alerts {
				alertsChan <- alert
			}
			validateWg.Done()
		}(validator)
	}

	collectWg.Add(1)
	go func() {
		for alert := range alertsChan {
			generatedAlerts = append(generatedAlerts, alert)
		}
		collectWg.Done()
	}()

	validateWg.Wait()
	close(alertsChan)
	collectWg.Wait()

	return generatedAlerts, nil
}

func getAlertUpdates(activeAlerts []*domain.Alert, newAlerts []*domain.Alert) []*domain.AlertUpdate {
	// Four main cases here:
	// * active alert doesn't exist, create a new one -> UPDATE TYPE CREATED
	// * active alert exists and is not resolved, just update last active timestamp (last active timestamp is not the timestamp
	//   of the validation, but the timestamp of the last measurement where the value was alarming) -> UPDATE TYPE CONTINUED
	// * alert exists and is resolved, update last active timestamp and resolved
	//   timestamp (should be next measurement timestamp after last active) -> ALERT TYPE RESOLVED

	alertUpdates := make([]*domain.AlertUpdate, 0)

	// When creating the maps, we are assuming there are no two active alerts with the same type (would make no sense).
	activeAlertsMap := make(map[string]*domain.Alert)
	for _, alert := range activeAlerts {
		_, found := activeAlertsMap[alert.AlertType]
		if found {
			log.Warnf("got two active alerts with same type %v for did %v", alert.AlertType, alert.DID)
			continue
		}
		activeAlertsMap[alert.AlertType] = alert
	}

	newAlertsMap := make(map[string]*domain.Alert)
	for _, alert := range newAlerts {
		_, found := newAlertsMap[alert.AlertType]
		if found {
			log.Warnf("got two new alerts with same type %v for did %v", alert.AlertType, alert.DID)
			continue
		}
		newAlertsMap[alert.AlertType] = alert
	}

	for newAlertType, newAlert := range newAlertsMap {
		activeAlert, activeAlertFound := activeAlertsMap[newAlertType]
		if activeAlertFound && newAlert.Status == domain.ALERT_STATUS_ACTIVE {

			update := &domain.AlertUpdate{
				UpdateType: domain.ALERT_UPDATE_TYPE_CONTINUED,
				DocID:      fmt.Sprintf("%v_%v_%v", newAlert.DID, newAlert.AlertType, activeAlert.CreatedTimestamp.Unix()),
				Alert: &domain.Alert{
					DID:                 newAlert.DID,
					AlertType:           newAlert.AlertType,
					Status:              newAlert.Status,
					CreatedTimestamp:    activeAlert.CreatedTimestamp,
					LastActiveTimestamp: newAlert.LastActiveTimestamp,
					ResolvedTimestamp:   newAlert.ResolvedTimestamp,
				},
			}

			alertUpdates = append(alertUpdates, update)

		} else if activeAlertFound && newAlert.Status == domain.ALERT_STATUS_RESOLVED {

			update := &domain.AlertUpdate{
				UpdateType: domain.ALERT_UPDATE_TYPE_RESOLVED,
				DocID:      fmt.Sprintf("%v_%v_%v", newAlert.DID, newAlert.AlertType, activeAlert.CreatedTimestamp.Unix()),
				Alert: &domain.Alert{
					DID:                 newAlert.DID,
					AlertType:           newAlert.AlertType,
					Status:              newAlert.Status,
					CreatedTimestamp:    activeAlert.CreatedTimestamp,
					LastActiveTimestamp: newAlert.LastActiveTimestamp,
					ResolvedTimestamp:   newAlert.ResolvedTimestamp,
				},
			}

			alertUpdates = append(alertUpdates, update)

		} else if newAlert.Status == domain.ALERT_STATUS_ACTIVE {

			update := &domain.AlertUpdate{
				UpdateType: domain.ALERT_UPDATE_TYPE_CREATED,
				DocID:      fmt.Sprintf("%v_%v_%v", newAlert.DID, newAlert.AlertType, newAlert.CreatedTimestamp.Unix()),
				Alert: &domain.Alert{
					DID:                 newAlert.DID,
					AlertType:           newAlert.AlertType,
					Status:              newAlert.Status,
					CreatedTimestamp:    newAlert.CreatedTimestamp,
					LastActiveTimestamp: newAlert.LastActiveTimestamp,
					ResolvedTimestamp:   newAlert.ResolvedTimestamp,
				},
			}

			alertUpdates = append(alertUpdates, update)
		}
	}

	return alertUpdates
}
