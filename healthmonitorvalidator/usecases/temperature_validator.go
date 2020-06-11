package usecases

import (
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorvalidator/domain"
	"time"
)

type TemperatureValidator struct {
	HighMargin float32
	LowMargin float32
}

func NewTemperatureValidator(lowMargin float32, highMargin float32) *TemperatureValidator {
	return &TemperatureValidator{
		HighMargin: highMargin,
		LowMargin: lowMargin,
	}
}

func (tv *TemperatureValidator) CheckData(dataSet *domain.DeviceDataset) []*domain.Alert {
	resultAlerts := make([]*domain.Alert, 0)
	highValueAlerts := make([]*domain.Alert, 0)
	lowValueAlerts := make([]*domain.Alert, 0)

	var currentAlert *domain.Alert

	for _, dataPoint := range dataSet.Data {
		if dataPoint.Temperature >= tv.HighMargin {
			if currentAlert == nil {
				currentAlert = &domain.Alert{
					DID: dataSet.DID,
					AlertType: domain.ALERT_TYPE_TEMP_HIGH,
					Status: domain.ALERT_STATUS_ACTIVE,
					CreatedTimestamp: time.Unix(dataPoint.Timestamp, 0),
					LastActiveTimestamp: time.Unix(dataPoint.Timestamp, 0),
				}
			} else {
				currentAlert.LastActiveTimestamp = time.Unix(dataPoint.Timestamp, 0)
			}
		} else {
			if currentAlert != nil {
				currentAlert.Status = domain.ALERT_STATUS_RESOLVED
				currentAlert.ResolvedTimestamp = time.Unix(dataPoint.Timestamp, 0)
				highValueAlerts = append(highValueAlerts, currentAlert)
				currentAlert = nil
			}
		}
	}

	if currentAlert != nil {
		highValueAlerts = append(highValueAlerts, currentAlert)
		currentAlert = nil
	}

	for _, dataPoint := range dataSet.Data {
		if dataPoint.Temperature <= tv.LowMargin {
			if currentAlert == nil {
				currentAlert = &domain.Alert{
					DID: dataSet.DID,
					AlertType: domain.ALERT_TYPE_TEMP_LOW,
					Status: domain.ALERT_STATUS_ACTIVE,
					CreatedTimestamp: time.Unix(dataPoint.Timestamp, 0),
					LastActiveTimestamp: time.Unix(dataPoint.Timestamp, 0),
				}
			} else {
				currentAlert.LastActiveTimestamp = time.Unix(dataPoint.Timestamp, 0)
			}
		} else {
			if currentAlert != nil {
				currentAlert.Status = domain.ALERT_STATUS_RESOLVED
				currentAlert.ResolvedTimestamp = time.Unix(dataPoint.Timestamp, 0)
				lowValueAlerts = append(lowValueAlerts, currentAlert)
				currentAlert = nil
			}
		}
	}

	if currentAlert != nil {
		lowValueAlerts = append(lowValueAlerts, currentAlert)
	}

	if len(highValueAlerts) > 0 {
		resultAlerts = append(resultAlerts, highValueAlerts[len(highValueAlerts) - 1])
	}

	if len(lowValueAlerts) > 0 {
		resultAlerts = append(resultAlerts, lowValueAlerts[len(lowValueAlerts) - 1])
	}

	if len(highValueAlerts) > 0 && len(lowValueAlerts) > 0 {
		lowAlert := lowValueAlerts[len(lowValueAlerts) - 1]
		highAlert := highValueAlerts[len(highValueAlerts) - 1]

		if lowAlert.Status == domain.ALERT_STATUS_ACTIVE && highAlert.Status == domain.ALERT_STATUS_ACTIVE {
			log.Warnf("got two active alerts for temperature for low and high values for did %v", dataSet.DID)
		}
	}

	return resultAlerts
}