package gateways

import (
	"healthmonitor/healthmonitorapi/domain"
	"math/rand"
	"time"
)

//TODO: Actually implement this.

type DevicesRepo struct {
}

func NewDevicesRepo() *DevicesRepo {
	return &DevicesRepo{
	}
}

func (dr *DevicesRepo) Start() error {
	return nil
}

func (dr *DevicesRepo) GetDeviceInfo(did string) (*domain.DeviceInfo, error) {
	return &domain.DeviceInfo{
		DID: did,
		LastSeenTimestamp: time.Now().Unix(),
	}, nil
}

func (dr *DevicesRepo) GetDeviceData(did string) (*domain.DeviceDataset, error) {

	startTimestamp := time.Now().Unix()
	data := make([]*domain.DeviceData, 0)
	for i := 0; i < 10; i++ {
		data = append(data, &domain.DeviceData{
			Temperature: 36.5 + rand.Float32() * 2,
			Heartrate: 70 + rand.Int63() % 20,
			Timestamp: startTimestamp + int64(i),
		})
	}
	return &domain.DeviceDataset{
		DID: did,
		Data: data,
	}, nil
}
