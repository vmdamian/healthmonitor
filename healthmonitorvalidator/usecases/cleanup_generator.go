package usecases

import (
	"context"
	"fmt"
	"healthmonitor/healthmonitorvalidator/gateways"
	"time"
)

type CleanupGenerator struct {
	devicesRepo *gateways.DevicesRepo
}

func NewCleanupGenerator(devicesRepo *gateways.DevicesRepo) *CleanupGenerator {
	return &CleanupGenerator{
		devicesRepo: devicesRepo,
	}
}

func (ag *CleanupGenerator) GenerateCleanup(ctx context.Context, message string) error {

	var maxTimeString string
	n, err := fmt.Sscanf(message, "cleanup_%v", &maxTimeString)
	if err != nil || n != 1 {
		return fmt.Errorf("could not parse maxAge from cleanup request = %v", message)
	}

	maxTime, err := time.Parse(time.RFC3339, maxTimeString)
	if err != nil {
		return err
	}

	return ag.devicesRepo.CleanupData(ctx, maxTime)
}
