package gateways

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/olivere/elastic/v7"
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorapi/domain"
	"time"
)

const (
	dataIndex = "device-data"
	infoIndex = "device-info"

	didField = "did"
	timestampField = "timestamp"
	lastSeenTimestampField = "last_seen_timestamp"
)

var (
	ErrDeviceInfoNotFound = errors.New("device info not found")
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

func (dr *DevicesRepo) GetDeviceInfo(ctx context.Context, did string) (*domain.DeviceInfo, error) {
	termQuery := elastic.NewTermQuery(didField, did)
	query := elastic.NewBoolQuery().Filter(termQuery)

	res, err := dr.client.Search(infoIndex).Query(query).Do(ctx)
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
		log.WithError(err).Errorln("failed to parse device info timestamp")
		return nil, err
	}

	return &domain.DeviceInfo{
		DID: did,
		LastSeenTimestamp: infoLastSeenTimestamp.Unix(),
	}, nil
}

func (dr *DevicesRepo) RegisterDeviceInfo(ctx context.Context, deviceInfo domain.DeviceInfo) error {

	exists, err := dr.checkExistingDevice(ctx, deviceInfo.DID)
	if err != nil {
		return err
	}

	if exists {
		log.Warningf("got register device info request for did=%v, but device info already exists", deviceInfo.DID)
		return nil
	}

	// Register new device. Device info document ids are the same as the device id.
	infoES := domain.DeviceInfoES{
		DID: deviceInfo.DID,
		LastSeenTimestamp: time.Unix(deviceInfo.LastSeenTimestamp, 0).Format(time.RFC3339),
	}

	_, err = dr.client.Index().Index(infoIndex).Id(deviceInfo.DID).BodyJson(infoES).Do(ctx)
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

func (dr *DevicesRepo) RegisterDeviceData(ctx context.Context, deviceData domain.DeviceDataset) error {

	var maxLastSeenTimestamp int64

	// Check if the device exists.
	exists, err := dr.checkExistingDevice(ctx, deviceData.DID)
	if err != nil {
		return err
	}

	if !exists {
		return errors.New("can't insert data for device that does not exist")
	}

	// Insert new data. Data documents ids are of the form "DID_TIMESTAMP"
	for _, data := range deviceData.Data {
		dataES := domain.DeviceDataES{
			DID: deviceData.DID,
			Temperature:  data.Temperature,
			Heartrate: data.Heartrate,
			Timestamp: time.Unix(data.Timestamp, 0).Format(time.RFC3339),
		}

		dataDocID := fmt.Sprintf("%v_%v", deviceData.DID, data.Timestamp)

		_, err := dr.client.Index().Index(dataIndex).Id(dataDocID).BodyJson(dataES).Do(ctx)
		if err != nil {
			return err
		}

		if data.Timestamp > maxLastSeenTimestamp {
			maxLastSeenTimestamp = data.Timestamp
		}
	}

	// Update the last seen timestamp field of the device info document with the last max seen timestamp.
	if maxLastSeenTimestamp > 0 {
		updatedData := map[string]interface{} {
			lastSeenTimestampField: maxLastSeenTimestamp,
		}

		_, err := dr.client.Update().Index(infoIndex).Id(deviceData.DID).Doc(updatedData).Do(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dr *DevicesRepo) checkExistingDevice(ctx context.Context, did string) (bool, error) {
	_, err := dr.client.Get().Index(infoIndex).Id(did).Do(ctx)
	if err != nil {
		if elastic.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}