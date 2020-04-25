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
	healthmonitorAPIURL = "http://127.0.0.1:9000"
	registerDevicePath = "/healthmonitorapi/entities/devices/info"
	registerDataPath = "/healthmonitorapi/entities/devices/data"

	contentType = "application/json"

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

func (d *Device) Start(wg *sync.WaitGroup) {
	wg.Add(1)

	go func() {

		log.Infof("> device %v started with interval &v", d.did, d.interval)

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
	}

	bodyBytes, err := json.Marshal(deviceInfo)
	if err != nil {
		log.WithError(err).Errorln("> device %v error marshalling device info", d.did)
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
						Temperature: generateRandomFloat32(36, 38),
						Heartrate: generateRandomInt64(70, 90),
						Timestamp: time.Now().Unix(),
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
		case <- d.stopChan:
			d.ticker.Stop()
			return
		}
	}
}

func generateRandomFloat32(min float32, max float32) float32 {
	rand.Seed(time.Now().UnixNano())

	return min + rand.Float32() * (max - min)
}

func generateRandomInt64(min int64, max int64) int64 {
	rand.Seed(time.Now().UnixNano())

	return rand.Int63n(max - min + 1) + min
}
