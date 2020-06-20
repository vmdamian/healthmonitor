package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"healthmonitor/healthmonitorvalidator/domain"
	"healthmonitor/healthmonitorvalidator/gateways"
	"os"
	"sync"
)

const (
	infoFileSuffix   = "info"
	dataFileSuffix   = "data"
	alertsFileSuffix = "alerts"
)

type ReportGenerator struct {
	devicesRepo *gateways.DevicesRepo
}

func NewReportGenerator(devicesRepo *gateways.DevicesRepo) *ReportGenerator {
	return &ReportGenerator{
		devicesRepo: devicesRepo,
	}
}

func (ag *ReportGenerator) GenerateReport(ctx context.Context, message string) error {

	var reportName, did, username, uuid string

	n, err := fmt.Sscanf(message, "report-generation_%v", &reportName)
	if err != nil || n != 1 {
		return fmt.Errorf("could not parse reportName from validation request = %v", message)
	}

	n, err = fmt.Sscanf(message, "%v_%v_%v", &username, &did, &uuid)
	if err != nil || n != 1 {
		return fmt.Errorf("could not parse did from reportName = %v", reportName)
	}

	wg := sync.WaitGroup{}
	var infoWriteErr, dataWriteErr, alertsWriteErr error

	deviceInfo, err := ag.devicesRepo.GetDeviceInfo(ctx, did)
	if err != nil {
		return err
	}

	wg.Add(3)
	go func() {
		defer wg.Done()
		fileName := fmt.Sprintf("%v_%v", reportName, infoFileSuffix)

		file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			infoWriteErr = err
			return
		}

		bytes, err := json.Marshal(deviceInfo)
		if err != nil {
			infoWriteErr = err
			return
		}

		if _, err := file.Write(bytes); err != nil {
			infoWriteErr = err
			return
		}

		if err := file.Close(); err != nil {
			infoWriteErr = err
			return
		}
	}()

	go func() {
		defer wg.Done()
		fileName := fmt.Sprintf("%v_%v", reportName, dataFileSuffix)

		file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			dataWriteErr = err
			return
		}

		dataWriteErr = ag.devicesRepo.ScrollWriteData(ctx, did, createWriteDataWithFile(file))
		if dataWriteErr != nil {
			return
		}

		if err := file.Close(); err != nil {
			dataWriteErr = err
			return
		}
	}()

	go func() {
		defer wg.Done()
		fileName := fmt.Sprintf("%v_%v", reportName, alertsFileSuffix)

		file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			alertsWriteErr = err
			return
		}

		alertsWriteErr = ag.devicesRepo.ScrollWriteAlerts(ctx, did, createWriteAlertWithFile(file))
		if alertsWriteErr != nil {
			return
		}

		if err := file.Close(); err != nil {
			alertsWriteErr = err
			return
		}
	}()

	wg.Wait()

	if dataWriteErr != nil {
		return dataWriteErr
	}

	if alertsWriteErr != nil {
		return alertsWriteErr
	}

	return nil
}

func createWriteDataWithFile(file *os.File) func(data *domain.DeviceData) error {
	return func(data *domain.DeviceData) error {
		bytes, err := json.Marshal(data)
		if err != nil {
			return err
		}

		if _, err := file.Write(bytes); err != nil {
			return err
		}

		return nil
	}
}

func createWriteAlertWithFile(file *os.File) func(data *domain.Alert) error {
	return func(data *domain.Alert) error {
		bytes, err := json.Marshal(data)
		if err != nil {
			return err
		}

		if _, err := file.Write(bytes); err != nil {
			return err
		}

		return nil
	}
}
