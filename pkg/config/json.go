package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// UnmarshalJSONFile десериализует json конфиг из файла
func UnmarshalJSONFile(res any, filePath string) error {
	bytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	err = json.Unmarshal(bytes, res)
	if err != nil {
		return fmt.Errorf("failed to unmarshal: %w", err)
	}

	return nil
}
