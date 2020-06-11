package gateways

import (
	"context"
	"encoding/json"
	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorvalidator/domain"
	"time"
)

const (
	dataIndex = "device-data"
	alertsIndex = "device-alerts"

	didField = "did"
	timestampField = "timestamp"
	lastActiveTimestampField = "last_active_timestamp"
	resolvedTimestampField = "resolved_timestamp"
	statusField = "status"
)

type DevicesRepo struct {
	host string
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

func (dr *DevicesRepo) GetDeviceData(ctx context.Context, did string, since time.Time) (*domain.DeviceDataset, error) {
	rangeQuery := elastic.NewRangeQuery(timestampField).Gte(since.Format(time.RFC3339))
	termQuery := elastic.NewTermQuery(didField, did)
	query := elastic.NewBoolQuery().Filter(termQuery, rangeQuery)

	res, err := dr.client.Search(dataIndex).Size(1000).Sort(timestampField, true).Query(query).Do(ctx)
	if err != nil {
		return nil, err
	}

	dataset := &domain.DeviceDataset{
		DID: did,
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
			Heartrate: dataES.Heartrate,
			Timestamp: dataTimestamp.Unix(),
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
			DID: alertES.DID,
			Status: alertES.Status,
			AlertType: alertES.AlertType,
			CreatedTimestamp: createdTimestmap,
			LastActiveTimestamp: lastActiveTimestamp,
			ResolvedTimestamp: resolvedTimestamp,
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

func (dr *DevicesRepo) UpsertAlerts(ctx context.Context, alertUpdates []*domain.AlertUpdate) error {
	for _, alertUpdate := range alertUpdates {
		if alertUpdate.UpdateType == domain.ALERT_UPDATE_TYPE_CREATED {
			alertES := domain.AlertES{
				DID: alertUpdate.Alert.DID,
				AlertType: alertUpdate.Alert.AlertType,
				Status: alertUpdate.Alert.Status,
				CreatedTimestamp: alertUpdate.Alert.CreatedTimestamp.Format(time.RFC3339),
				LastActiveTimestamp: alertUpdate.Alert.LastActiveTimestamp.Format(time.RFC3339),
				ResolvedTimestamp: alertUpdate.Alert.ResolvedTimestamp.Format(time.RFC3339),
			}

			_, err := dr.client.Index().Index(alertsIndex).Id(alertUpdate.DocID).BodyJson(alertES).Do(ctx)
			if err != nil {
				return err
			}
		} else if alertUpdate.UpdateType == domain.ALERT_UPDATE_TYPE_CONTINUED {
			updatedData := map[string]interface{} {
				lastActiveTimestampField: alertUpdate.Alert.LastActiveTimestamp.Format(time.RFC3339),
			}

			_, err := dr.client.Update().Index(alertsIndex).Id(alertUpdate.DocID).Doc(updatedData).Do(ctx)
			if err != nil {
				return err
			}
		} else if alertUpdate.UpdateType == domain.ALERT_UPDATE_TYPE_RESOLVED {
			updatedData := map[string]interface{} {
				statusField: alertUpdate.Alert.Status,
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