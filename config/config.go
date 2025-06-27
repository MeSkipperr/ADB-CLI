package config

import (
	"encoding/json"
	"errors"
	"os"
)

func LoadJSON[T any](filepath string) (T, error) {
	var result T

	data, err := os.ReadFile(filepath)
	if err != nil {
		return result, err
	}

	if len(data) == 0 {
		return result, errors.New("file JSON kosong")
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}
