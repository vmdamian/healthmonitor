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
	dataIndex   = "device-data"
	infoIndex   = "device-info"
	alertsIndex = "device-alerts"

	didField               = "did"
	timestampField         = "timestamp"
	lastSeenTimestampField = "last_seen_timestamp"
	createdTimestampField  = "created_timestamp"
	subscribedPhonesField  = "subscribed_phones"
	patientNameField       = "patient_name"
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

func (dr *DevicesRepo) GetDeviceInfo(ctx context.Context, did string) (*domain.DeviceInfo, error) {
	query := elastic.NewMatchQuery(didField, did)

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

func (dr *DevicesRepo) RegisterDeviceInfo(ctx context.Context, did string, patient_name string) (*domain.DeviceInfo, error) {

	exists, err := dr.checkExistingDevice(ctx, did)
	if err != nil {
		return nil, err
	}

	if exists {
		log.Warningf("got register device info request for did=%v, but device info already exists", did)
		return nil, nil
	}

	timeNow := time.Now()
	timeZero := time.Time{}

	// Register new device. Device info document ids are the same as the device id.
	infoES := domain.DeviceInfoES{
		DID:                     did,
		LastSeenTimestamp:       timeNow.Format(time.RFC3339),
		LastValidationTimestamp: timeZero.Format(time.RFC3339),
		PatientName:             patient_name,
		SubscribedPhones:        []string{},
	}

	_, err = dr.client.Index().Index(infoIndex).Routing(did).Id(did).BodyJson(infoES).Do(ctx)
	if err != nil {
		return nil, err
	}

	info := &domain.DeviceInfo{
		DID:                     infoES.DID,
		LastSeenTimestamp:       timeNow,
		LastValidationTimestamp: timeZero,
		PatientName:             "",
		SubscribedPhones:        []string{},
	}

	return info, nil
}

func (dr *DevicesRepo) UpdateDeviceInfo(ctx context.Context, did string, patientName string) error {

	_, err := dr.GetDeviceInfo(ctx, did)
	if err != nil {
		return err
	}

	updatedData := map[string]interface{}{
		patientNameField: patientName,
	}

	_, err = dr.client.Update().Index(infoIndex).Routing(did).Id(did).Doc(updatedData).Do(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (dr *DevicesRepo) GetDeviceData(ctx context.Context, did string, since time.Time, toTime time.Time) (*domain.DeviceDataset, error) {
	rangeQuery := elastic.NewRangeQuery(timestampField).Gte(since.Format(time.RFC3339)).Lte(toTime.Format(time.RFC3339))
	termQuery := elastic.NewTermQuery(didField, did)
	query := elastic.NewBoolQuery().Filter(termQuery, rangeQuery)

	res, err := dr.client.Search(dataIndex).Routing(did).Size(1000).Sort(timestampField, true).Query(query).Do(ctx)
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
			DID:         dataES.DID,
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
			DID:         deviceData.DID,
			Temperature: data.Temperature,
			Heartrate:   data.Heartrate,
			Timestamp:   data.Timestamp.Format(time.RFC3339),
			ECG:         data.ECG,
			SPO2:        data.SPO2,
		}

		dataDocID := fmt.Sprintf("%v_%v", deviceData.DID, data.Timestamp)

		_, err := dr.client.Index().Routing(deviceData.DID).Index(dataIndex).Id(dataDocID).BodyJson(dataES).Do(ctx)
		if err != nil {
			return err
		}

		if data.Timestamp.Unix() > maxLastSeenTimestamp {
			maxLastSeenTimestamp = data.Timestamp.Unix()
		}
	}

	// Update the last seen timestamp field of the device info document with the last max seen timestamp.
	if maxLastSeenTimestamp > 0 {
		updatedData := map[string]interface{}{
			lastSeenTimestampField: time.Unix(maxLastSeenTimestamp, 0).Format(time.RFC3339),
		}

		_, err := dr.client.Update().Index(infoIndex).Routing(deviceData.DID).Id(deviceData.DID).Doc(updatedData).Do(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (dr *DevicesRepo) GetAlerts(ctx context.Context, did string) ([]*domain.Alert, error) {
	termQuery := elastic.NewTermQuery(didField, did)
	query := elastic.NewBoolQuery().Filter(termQuery)

	res, err := dr.client.Search(alertsIndex).Size(1000).Routing(did).Sort(createdTimestampField, false).Query(query).Do(ctx)
	if err != nil {
		return nil, err
	}

	var alerts []*domain.Alert
	for _, hit := range res.Hits.Hits {
		var alertES domain.AlertES

		err := json.Unmarshal(hit.Source, &alertES)
		if err != nil {
			log.WithError(err).Errorln("failed to unmarshall device alert")
			continue
		}

		createdTimestap, err := time.Parse(time.RFC3339, alertES.CreatedTimestamp)
		if err != nil {
			log.WithError(err).Errorln("failed to parse alert created timestamp")
			continue
		}

		lastActiveTimestamp, err := time.Parse(time.RFC3339, alertES.LastActiveTimestamp)
		if err != nil {
			log.WithError(err).Errorln("failed to parse alert last active timestamp")
			continue
		}

		resolvedTimestamp, err := time.Parse(time.RFC3339, alertES.ResolvedTimestamp)
		if err != nil {
			log.WithError(err).Errorln("failed to parse alert resolved timestamp")
			continue
		}

		alert := &domain.Alert{
			DID:                 alertES.DID,
			AlertType:           alertES.AlertType,
			Status:              alertES.Status,
			CreatedTimestamp:    createdTimestap,
			LastActiveTimestamp: lastActiveTimestamp,
			ResolvedTimestamp:   resolvedTimestamp,
		}

		alerts = append(alerts, alert)
	}

	return alerts, err
}

func (dr *DevicesRepo) AddSubscription(ctx context.Context, did string, phone string) error {
	deviceInfo, err := dr.GetDeviceInfo(ctx, did)
	if err != nil {
		return err
	}

	if stringInList(phone, deviceInfo.SubscribedPhones) {
		return nil
	}

	newSubscribedPhones := append(deviceInfo.SubscribedPhones, phone)

	updatedData := map[string]interface{}{
		subscribedPhonesField: newSubscribedPhones,
	}

	_, err = dr.client.Update().Index(infoIndex).Routing(did).Id(did).Doc(updatedData).Do(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (dr *DevicesRepo) DeleteSubscription(ctx context.Context, did string, phone string) error {
	deviceInfo, err := dr.GetDeviceInfo(ctx, did)
	if err != nil {
		return err
	}

	if !stringInList(phone, deviceInfo.SubscribedPhones) {
		return nil
	}

	newSubscribedPhones := make([]string, 0, len(deviceInfo.SubscribedPhones)-1)
	for _, existingPhone := range deviceInfo.SubscribedPhones {
		if existingPhone != phone {
			newSubscribedPhones = append(newSubscribedPhones, existingPhone)
		}
	}

	updatedData := map[string]interface{}{
		subscribedPhonesField: newSubscribedPhones,
	}

	_, err = dr.client.Update().Index(infoIndex).Routing(did).Id(did).Doc(updatedData).Do(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (dr *DevicesRepo) checkExistingDevice(ctx context.Context, did string) (bool, error) {
	_, err := dr.client.Get().Routing(did).Index(infoIndex).Id(did).Do(ctx)
	if err != nil {
		if elastic.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
