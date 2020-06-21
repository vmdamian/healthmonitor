package gateways

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorvalidator/domain"
	"io"
	"time"
)

const (
	dataIndex   = "device-data"
	infoIndex   = "device-info"
	alertsIndex = "device-alerts"

	internalTasksIndex  = ".tasks"
	internalTaskDocName = "task"

	scrollSize       = 100
	conflictsProceed = "proceed"

	didField                 = "did"
	timestampField           = "timestamp"
	lastActiveTimestampField = "last_active_timestamp"
	resolvedTimestampField   = "resolved_timestamp"
	statusField              = "status"

	lastValidationTimestampField = "last_validation_timestamp"
)

var (
	ErrDeviceInfoNotFound = errors.New("device info not found")
)

type DevicesRepo struct {
	host   string
	client *elastic.Client
}

func NewDevicesRepo(host string) *DevicesRepo {
	return &DevicesRepo{
		host: host,
	}
}

func (dr *DevicesRepo) Start() error {
	client, err := elastic.NewClient(elastic.SetURL(dr.host), elastic.SetSniff(false), elastic.SetHealthcheck(false))
	if err != nil {
		return err
	}

	dr.client = client

	return nil
}

func (dr *DevicesRepo) ScrollWriteData(ctx context.Context, did string, dataWriter func(*domain.DeviceData) error) error {

	termQuery := elastic.NewTermQuery(didField, did)
	query := elastic.NewBoolQuery().Filter(termQuery)

	scrollService := dr.client.Scroll(dataIndex).Routing(did).Size(1000).Query(query)
	defer func() {
		_ = scrollService.Clear(ctx)
	}()

	for {
		result, err := scrollService.Do(ctx)
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		for _, hit := range result.Hits.Hits {
			var dataES domain.DeviceDataES

			err := json.Unmarshal(hit.Source, &dataES)
			if err != nil {
				return err
			}

			dataTimestamp, err := time.Parse(time.RFC3339, dataES.Timestamp)
			if err != nil {
				return err
			}

			data := &domain.DeviceData{
				Temperature: dataES.Temperature,
				Heartrate:   dataES.Heartrate,
				ECG:         dataES.ECG,
				SPO2:        dataES.SPO2,
				Timestamp:   dataTimestamp,
			}

			err = dataWriter(data)
			if err != nil {
				return err
			}
		}
	}
}

func (dr *DevicesRepo) ScrollWriteAlerts(ctx context.Context, did string, alertWriter func(*domain.Alert) error) error {

	termQuery := elastic.NewTermQuery(didField, did)
	query := elastic.NewBoolQuery().Filter(termQuery)

	scrollService := dr.client.Scroll(alertsIndex).Routing(did).Size(scrollSize).Query(query)
	defer func() {
		_ = scrollService.Clear(ctx)
	}()

	for {
		result, err := scrollService.Do(ctx)
		if err == io.EOF {
			return nil
		}

		if err != nil {
			return err
		}

		for _, hit := range result.Hits.Hits {
			var alertES domain.AlertES

			err := json.Unmarshal(hit.Source, &alertES)
			if err != nil {
				log.WithError(err).Errorln("failed to unmarshall device alert")
				continue
			}

			createdTimestmap, err := time.Parse(time.RFC3339, alertES.CreatedTimestamp)
			if err != nil {
				log.WithError(err).Errorln("failed to parse alert created timestamp")
				continue
			}

			lastActiveTimestamp, err := time.Parse(time.RFC3339, alertES.LastActiveTimestamp)
			if err != nil {
				log.WithError(err).Errorln("failed to parse alert created timestamp")
				continue
			}

			resolvedTimestamp, err := time.Parse(time.RFC3339, alertES.ResolvedTimestamp)
			if err != nil {
				log.WithError(err).Errorln("failed to parse alert created timestamp")
				continue
			}

			alert := &domain.Alert{
				DID:                 alertES.DID,
				Status:              alertES.Status,
				AlertType:           alertES.AlertType,
				CreatedTimestamp:    createdTimestmap,
				LastActiveTimestamp: lastActiveTimestamp,
				ResolvedTimestamp:   resolvedTimestamp,
			}

			err = alertWriter(alert)
			if err != nil {
				return err
			}
		}
	}
}

func (dr *DevicesRepo) CleanupData(ctx context.Context, maxTime time.Time) error {
	rangeQuery := elastic.NewRangeQuery(timestampField).Lte(maxTime.Format(time.RFC3339))
	query := elastic.NewBoolQuery().Filter(rangeQuery)

	task, err := dr.client.DeleteByQuery(dataIndex).Conflicts(conflictsProceed).Query(query).DoAsync(ctx)
	if err != nil {
		return err
	}
	defer dr.deleteESTask(ctx, task.TaskId)

	completionCheckTicker := time.NewTicker(5 * time.Second)
	defer completionCheckTicker.Stop()

	for {
		select {
		case <-completionCheckTicker.C:
			resp, err := dr.client.TasksGetTask().TaskId(task.TaskId).Do(ctx)
			if err != nil {
				return err
			}
			if resp.Completed {
				return nil
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (dr *DevicesRepo) deleteESTask(ctx context.Context, taskID string) {
	_, err := dr.client.Delete().Index(internalTasksIndex).Type(internalTaskDocName).Id(taskID).Do(ctx)
	if err != nil {
		log.WithError(err).Errorf("failed to delete cleanup task with id=%v", taskID)
	}
}

func (dr *DevicesRepo) GetDeviceInfo(ctx context.Context, did string) (*domain.DeviceInfo, error) {
	termQuery := elastic.NewTermQuery(didField, did)
	query := elastic.NewBoolQuery().Filter(termQuery)

	res, err := dr.client.Search(infoIndex).Routing(did).Query(query).Do(ctx)
	if err != nil {
		return nil, err
	}

	if len(res.Hits.Hits) == 0 {
		return nil, ErrDeviceInfoNotFound
	}

	if len(res.Hits.Hits) > 1 {
		log.Errorf("got multiple results when getting device info for did = %v", did)
	}

	var infoES domain.DeviceInfoES
	err = json.Unmarshal(res.Hits.Hits[0].Source, &infoES)
	if err != nil {
		log.WithError(err).Errorln("got error trying to unmarshall device info")
	}

	infoLastSeenTimestamp, err := time.Parse(time.RFC3339, infoES.LastSeenTimestamp)
	if err != nil {
		log.WithError(err).Errorf("failed to parse device info timestamp=%v", infoES.LastSeenTimestamp)
		return nil, err
	}

	infoLastValidationTimestamp, err := time.Parse(time.RFC3339, infoES.LastValidationTimestamp)
	if err != nil {
		log.WithError(err).Errorf("failed to parse device info timestamp=%v", infoES.LastSeenTimestamp)
		return nil, err
	}

	return &domain.DeviceInfo{
		DID:                     did,
		LastSeenTimestamp:       infoLastSeenTimestamp,
		LastValidationTimestamp: infoLastValidationTimestamp,
		PatientName:             infoES.PatientName,
		SubscribedPhones:        infoES.SubscribedPhones,
	}, nil
}

func (dr *DevicesRepo) UpdateDeviceInfo(ctx context.Context, did string, lastValidated time.Time) error {

	_, err := dr.GetDeviceInfo(ctx, did)
	if err != nil {
		return err
	}

	updatedData := map[string]interface{}{
		lastValidationTimestampField: lastValidated.Format(time.RFC3339),
	}

	_, err = dr.client.Update().Index(infoIndex).Routing(did).Id(did).Doc(updatedData).Do(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (dr *DevicesRepo) GetDeviceData(ctx context.Context, did string, since time.Time) (*domain.DeviceDataset, error) {
	rangeQuery := elastic.NewRangeQuery(timestampField).Gte(since.Format(time.RFC3339))
	termQuery := elastic.NewTermQuery(didField, did)
	query := elastic.NewBoolQuery().Filter(termQuery, rangeQuery)

	res, err := dr.client.Search(dataIndex).Size(1000).Sort(timestampField, true).Query(query).Do(ctx)
	if err != nil {
		return nil, err
	}

	dataset := &domain.DeviceDataset{
		DID:  did,
		Data: make([]*domain.DeviceData, 0, len(res.Hits.Hits)),
	}
	for _, hit := range res.Hits.Hits {
		var dataES domain.DeviceDataES

		err := json.Unmarshal(hit.Source, &dataES)
		if err != nil {
			log.WithError(err).Errorln("failed to unmarshall device data")
			continue
		}

		dataTimestamp, err := time.Parse(time.RFC3339, dataES.Timestamp)
		if err != nil {
			log.WithError(err).Errorln("failed to parse device data timestamp")
			continue
		}

		data := &domain.DeviceData{
			Temperature: dataES.Temperature,
			Heartrate:   dataES.Heartrate,
			ECG:         dataES.ECG,
			SPO2:        dataES.SPO2,
			Timestamp:   dataTimestamp,
		}

		dataset.Data = append(dataset.Data, data)
	}

	return dataset, nil
}

func (dr *DevicesRepo) GetDeviceAlerts(ctx context.Context, did string, status string) ([]*domain.Alert, error) {
	didTermQuery := elastic.NewMatchQuery(didField, did)
	statusTermQuery := elastic.NewMatchQuery(statusField, status)
	query := elastic.NewBoolQuery().Must(didTermQuery, statusTermQuery)

	res, err := dr.client.Search(alertsIndex).Size(10).Query(query).Do(ctx)
	if err != nil {
		return nil, err
	}

	alerts := make([]*domain.Alert, 0, len(res.Hits.Hits))
	for _, hit := range res.Hits.Hits {
		var alertES domain.AlertES

		err := json.Unmarshal(hit.Source, &alertES)
		if err != nil {
			log.WithError(err).Errorln("failed to unmarshall device alert")
			continue
		}

		createdTimestmap, err := time.Parse(time.RFC3339, alertES.CreatedTimestamp)
		if err != nil {
			log.WithError(err).Errorln("failed to parse alert created timestamp")
			continue
		}

		lastActiveTimestamp, err := time.Parse(time.RFC3339, alertES.LastActiveTimestamp)
		if err != nil {
			log.WithError(err).Errorln("failed to parse alert created timestamp")
			continue
		}

		resolvedTimestamp, err := time.Parse(time.RFC3339, alertES.ResolvedTimestamp)
		if err != nil {
			log.WithError(err).Errorln("failed to parse alert created timestamp")
			continue
		}

		alert := &domain.Alert{
			DID:                 alertES.DID,
			Status:              alertES.Status,
			AlertType:           alertES.AlertType,
			CreatedTimestamp:    createdTimestmap,
			LastActiveTimestamp: lastActiveTimestamp,
			ResolvedTimestamp:   resolvedTimestamp,
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (dr *DevicesRepo) UpsertAlerts(ctx context.Context, alertUpdates []*domain.AlertUpdate) error {
	for _, alertUpdate := range alertUpdates {
		if alertUpdate.UpdateType == domain.ALERT_UPDATE_TYPE_CREATED {
			alertES := domain.AlertES{
				DID:                 alertUpdate.Alert.DID,
				AlertType:           alertUpdate.Alert.AlertType,
				Status:              alertUpdate.Alert.Status,
				CreatedTimestamp:    alertUpdate.Alert.CreatedTimestamp.Format(time.RFC3339),
				LastActiveTimestamp: alertUpdate.Alert.LastActiveTimestamp.Format(time.RFC3339),
				ResolvedTimestamp:   alertUpdate.Alert.ResolvedTimestamp.Format(time.RFC3339),
			}

			_, err := dr.client.Index().Index(alertsIndex).Id(alertUpdate.DocID).BodyJson(alertES).Do(ctx)
			if err != nil {
				return err
			}
		} else if alertUpdate.UpdateType == domain.ALERT_UPDATE_TYPE_CONTINUED {
			updatedData := map[string]interface{}{
				lastActiveTimestampField: alertUpdate.Alert.LastActiveTimestamp.Format(time.RFC3339),
			}

			_, err := dr.client.Update().Index(alertsIndex).Id(alertUpdate.DocID).Doc(updatedData).Do(ctx)
			if err != nil {
				return err
			}
		} else if alertUpdate.UpdateType == domain.ALERT_UPDATE_TYPE_RESOLVED {
			updatedData := map[string]interface{}{
				statusField:            alertUpdate.Alert.Status,
				resolvedTimestampField: alertUpdate.Alert.ResolvedTimestamp.Format(time.RFC3339),
			}

			_, err := dr.client.Update().Index(alertsIndex).Id(alertUpdate.DocID).Doc(updatedData).Do(ctx)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
