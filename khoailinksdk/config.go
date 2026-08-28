package khoailinksdk

import (
	"encoding/json"
	"fmt"
	"os"
)

func LoadConfig(configPath string) (*Config, error) {
	var rawData []byte
	var err error
	if configPath == "" {
		rawData, err = os.ReadFile("khoai-config.json")
		if err != nil {
			return nil, fmt.Errorf("read config from khoai-config.json failed: %w", err)
		}
	} else {
		rawData, err = os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("read config from %s failed: %w", configPath, err)
		}
	}
	var data Config
	err = json.Unmarshal(rawData, &data)
	if err != nil {
		return nil, fmt.Errorf("invalid json format in config: %w", err)
	}

	err = data.Validate()
	if err != nil {
		return nil, err
	}

	return &data, nil
}
