package utils

import "encoding/json"

func MapToStruct[T any](input map[string]any) (T, error) {
	var result T
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(jsonBytes, &result); err != nil {
		return result, err
	}
	return result, nil
}
