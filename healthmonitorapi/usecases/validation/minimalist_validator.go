package validation

import (
	"fmt"
	"healthmonitor/healthmonitorapi/domain"
)

const (
	OK_BOUND            = 0
	NOT_RESPECTED_BOUND = 1
)

type ValidationBound struct {
	minValue float64
	maxValue float64
}

func NewValidationBound(minValue float64, maxValue float64) *ValidationBound {
	return &ValidationBound{
		minValue: minValue,
		maxValue: maxValue,
	}
}

type MinimalistValidator struct {
	temperatureValidationBound *ValidationBound
	heartrateValidationBound   *ValidationBound
	ecgValidationBound         *ValidationBound
	spo2ValidationBound        *ValidationBound
}

func NewMinimalistValidator(temperatureValidationBound, heartrateValidationBound, ecgValidationBound, spo2ValidationBound *ValidationBound) *MinimalistValidator {
	return &MinimalistValidator{
		temperatureValidationBound: temperatureValidationBound,
		heartrateValidationBound:   heartrateValidationBound,
		ecgValidationBound:         ecgValidationBound,
		spo2ValidationBound:        spo2ValidationBound,
	}
}

func (mv *MinimalistValidator) CheckDataset(dataset *domain.DeviceDataset) []int {
	var validationErrors []int

	for _, datapoint := range dataset.Data {
		validationErrors = append(validationErrors,
			checkValueAgainstBound(datapoint.Temperature, mv.temperatureValidationBound),
			checkValueAgainstBound(datapoint.Heartrate, mv.heartrateValidationBound),
			checkValueAgainstBound(datapoint.ECG, mv.ecgValidationBound),
			checkValueAgainstBound(datapoint.SPO2, mv.spo2ValidationBound))
	}

	fmt.Println(validationErrors)

	validationErrors = removeDuplicates(validationErrors)
	validationErrors = removeValueFromUniquesList(OK_BOUND, validationErrors)

	return validationErrors
}

func removeDuplicates(list []int) []int {
	uniquesMap := make(map[int]struct{})
	uniquesList := make([]int, 0)

	for _, elem := range list {
		if _, found := uniquesMap[elem]; !found {
			uniquesMap[elem] = struct{}{}
			uniquesList = append(uniquesList, elem)
		}
	}

	return uniquesList
}

func removeValueFromUniquesList(value int, list []int) []int {
	resultList := make([]int, 0, len(list)-1)

	for _, elem := range list {
		if elem != value {
			resultList = append(resultList, elem)
		}
	}

	return resultList
}

func checkValueAgainstBound(value float64, bound *ValidationBound) int {
	if bound == nil {
		return OK_BOUND
	}

	if bound.minValue < value && value < bound.maxValue {
		return OK_BOUND
	}

	return NOT_RESPECTED_BOUND
}
