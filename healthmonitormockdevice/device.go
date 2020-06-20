package main

import (
	"bytes"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"healthmonitor/healthmonitorapi/domain"
	"math/rand"
	"net/http"
	"sync"
	"time"
)

const (
	didPrefix = "testdevice"
)

type Device struct {
	did string
	interval time.Duration
	ticker *time.Ticker
	stopChan chan struct{}
}

func NewDevice(did string, interval time.Duration) *Device {
	return &Device{
		did: did,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

func (d *Device) GetDID() string {
	return d.did
}

func (d *Device) Start(wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {

		log.Infof("> device %v started with interval %v", d.did, d.interval)

		if d.createDevice() {
			d.generateDeviceData()
		}

		log.Infof("> device %v stopped", d.did)

		wg.Done()
	}()
}

func (d *Device) Stop() {
	close(d.stopChan)
}

func (d *Device) createDevice() bool {
	deviceInfo := &domain.DeviceInfo{
		DID: d.did,
		LastSeenTimestamp: time.Now(),
	}

	bodyBytes, err := json.Marshal(deviceInfo)
	if err != nil {
		log.WithError(err).Errorf("> device %v error marshalling device info", d.did)
		return false
	}
	reader := bytes.NewReader(bodyBytes)

	resp, err := http.Post(healthmonitorAPIURL + registerDevicePath, contentType, reader)
	if err != nil {
		log.WithError(err).Errorf("> device %v error sending request to healthmonitorapi service to register device", d.did)
		return false
	}

	if resp.StatusCode != 200 {
		log.Errorf("> device %v register device response was not OK = %v", d.did, resp.StatusCode)
		return false
	}

	return true
}

func (d *Device) generateDeviceData() {

	d.ticker = time.NewTicker(d.interval)

	for {
		select {
		case <- d.ticker.C:
			deviceDataset := domain.DeviceDataset{
				DID: d.did,
				Data: []*domain.DeviceData{
					{
						Temperature: generateRandomFloat64(36.5, 41),
						Heartrate: generateRandomFloat64(70, 90),
						ECG:  generateRandomFloat64(100, 500),
						SPO2: generateRandomFloat64(90, 100),
						Timestamp: time.Now(),
					},
				},
			}

			bodyBytes, err := json.Marshal(deviceDataset)
			if err != nil {
				log.WithError(err).Errorf("> device %v error marshalling device data", d.did)
				return
			}
			reader := bytes.NewReader(bodyBytes)

			resp, err := http.Post(healthmonitorAPIURL + registerDataPath, contentType, reader)
			if err != nil {
				log.WithError(err).Errorf("> device %v error sending request to healthmonitorapi service to register device data", d.did)
				return
			}

			if resp.StatusCode != 200 {
				log.Errorf("> device %v register device data response was not OK = %v", d.did, resp.StatusCode)
				return
			}

			resp.Body.Close()
		case <- d.stopChan:
			d.ticker.Stop()
			return
		}
	}
}

func generateRandomFloat64(min float64, max float64) float64 {
	rand.Seed(time.Now().UnixNano())

	return min + rand.Float64() * (max - min)
}