package httpapi

import "fmt"

func normalizeOptionalPositiveThreshold(input *int) *int {
	if input == nil {
		return nil
	}
	value := *input
	if value <= 0 {
		return nil
	}
	copy := value
	return &copy
}

func validateOptionalPositiveThreshold(field string, input *int) error {
	if input == nil {
		return nil
	}
	if *input <= 0 {
		return fmt.Errorf("%s must be a positive integer", field)
	}
	return nil
}
