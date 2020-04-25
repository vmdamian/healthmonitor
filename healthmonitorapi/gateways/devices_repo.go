package gateways

import (
	"context"
	"encoding/json"
	"errors"
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
	infoES := domain.DeviceInfoES{
		DID: deviceInfo.DID,
		LastSeenTimestamp: time.Unix(deviceInfo.LastSeenTimestamp, 0).Format(time.RFC3339),
	}

	_, err := dr.client.Index().Index(infoIndex).BodyJson(infoES).Do(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (dr *DevicesRepo) GetDeviceData(ctx context.Context, did string, since time.Time) (*domain.DeviceDataset, error) {
	rangeQuery := elastic.NewRangeQuery(timestampField).Gte(since.Format(time.RFC3339))
	termQuery := elastic.NewTermQuery(didField, did)
	query := elastic.NewBoolQuery().Filter(termQuery, rangeQuery)

	res, err := dr.client.Search(dataIndex).Query(query).Do(ctx)
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
	for _, data := range deviceData.Data {
		dataES := domain.DeviceDataES{
			DID: deviceData.DID,
			Temperature:  data.Temperature,
			Heartrate: data.Heartrate,
			Timestamp: time.Unix(data.Timestamp, 0).Format(time.RFC3339),
		}

		_, err := dr.client.Index().Index(dataIndex).BodyJson(dataES).Do(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
