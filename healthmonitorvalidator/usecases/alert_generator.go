package usecases

import (
	"context"
	"fmt"
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorvalidator/domain"
	"healthmonitor/healthmonitorvalidator/gateways"
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
	}
}

func (ag *AlertGenerator) GenerateUpdateAndSendAlertsForDevice(ctx context.Context, did string) error {

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

	// TODO: Update everything to support phone number for users, getting the users subscripted to a device, etc.
	err = ag.alertSender.SendAlerts("+40731322853", did, alertsToSend)
	if err != nil {
		return err
	}

	return nil
}

func (ag *AlertGenerator) generateAlertsForDevice(ctx context.Context, did string) ([]*domain.Alert, error) {
	dataSince := time.Now().Add((-1) * ag.validationPeriod)
	dataSet, err := ag.devicesRepo.GetDeviceData(ctx, did, dataSince)
	if err != nil {
		return nil, err
	}

	generatedAlerts := make([]*domain.Alert, 0)
	for _, validator := range ag.validators {
		alerts := validator.CheckData(dataSet)
		generatedAlerts = append(generatedAlerts, alerts...)
	}

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
