package usecases

import (
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorvalidator/domain"
)

type PulseValidator struct {
	HighMargin float64
	LowMargin  float64
}

func NewPulseValidator(lowMargin float64, highMargin float64) *PulseValidator {
	return &PulseValidator{
		HighMargin: highMargin,
		LowMargin:  lowMargin,
	}
}

func (tv *PulseValidator) CheckData(dataSet *domain.DeviceDataset) []*domain.Alert {
	resultAlerts := make([]*domain.Alert, 0)
	highValueAlerts := make([]*domain.Alert, 0)
	lowValueAlerts := make([]*domain.Alert, 0)

	var currentAlert *domain.Alert

	for _, dataPoint := range dataSet.Data {
		if dataPoint.Heartrate >= tv.HighMargin {
			if currentAlert == nil {
				currentAlert = &domain.Alert{
					DID:                 dataSet.DID,
					AlertType:           domain.ALERT_TYPE_SPO2_HIGH,
					Status:              domain.ALERT_STATUS_ACTIVE,
					CreatedTimestamp:    dataPoint.Timestamp,
					LastActiveTimestamp: dataPoint.Timestamp,
				}
			} else {
				currentAlert.LastActiveTimestamp = dataPoint.Timestamp
			}
		} else {
			if currentAlert != nil {
				currentAlert.Status = domain.ALERT_STATUS_RESOLVED
				currentAlert.ResolvedTimestamp = dataPoint.Timestamp
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
		if dataPoint.Heartrate <= tv.LowMargin {
			if currentAlert == nil {
				currentAlert = &domain.Alert{
					DID:                 dataSet.DID,
					AlertType:           domain.ALERT_TYPE_SP02_LOW,
					Status:              domain.ALERT_STATUS_ACTIVE,
					CreatedTimestamp:    dataPoint.Timestamp,
					LastActiveTimestamp: dataPoint.Timestamp,
				}
			} else {
				currentAlert.LastActiveTimestamp = dataPoint.Timestamp
			}
		} else {
			if currentAlert != nil {
				currentAlert.Status = domain.ALERT_STATUS_RESOLVED
				currentAlert.ResolvedTimestamp = dataPoint.Timestamp
				lowValueAlerts = append(lowValueAlerts, currentAlert)
				currentAlert = nil
			}
		}
	}

	if currentAlert != nil {
		lowValueAlerts = append(lowValueAlerts, currentAlert)
	}

	if len(highValueAlerts) > 0 {
		resultAlerts = append(resultAlerts, highValueAlerts[len(highValueAlerts)-1])
	}

	if len(lowValueAlerts) > 0 {
		resultAlerts = append(resultAlerts, lowValueAlerts[len(lowValueAlerts)-1])
	}

	if len(highValueAlerts) > 0 && len(lowValueAlerts) > 0 {
		lowAlert := lowValueAlerts[len(lowValueAlerts)-1]
		highAlert := highValueAlerts[len(highValueAlerts)-1]

		if lowAlert.Status == domain.ALERT_STATUS_ACTIVE && highAlert.Status == domain.ALERT_STATUS_ACTIVE {
			log.Warnf("got two active alerts for ecg for low and high values for did %v", dataSet.DID)
		}
	}

	return resultAlerts
}
