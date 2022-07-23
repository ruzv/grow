package config

import (
	"encoding/json"
	"os"

	"private/grow/blob"
)

type Config struct {
	Blob blob.BlobConfig `json:"blob"`
}

func LoadConfig(filepath string) (*Config, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	conf := &Config{}

	err = json.NewDecoder(f).Decode(&conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
