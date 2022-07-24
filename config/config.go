package config

import (
	"encoding/json"
	"os"

	"private/grow/blob"
	"private/grow/handler"
)

type Config struct {
	View handler.ViewConfig `json:"view"`
	Blob blob.BlobConfig    `json:"blob"`
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

type Save struct {
	View *handler.ViewJSON `json:"view"`
	Blob *blob.BlobJSON    `json:"blob"`
}

func LoadSave(filepath string) (*Save, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	save := &Save{}

	err = json.NewDecoder(f).Decode(&save)
	if err != nil {
		return nil, err
	}

	return save, nil
}

func RecordSave(filepath string, save *Save) error {
	f, err := os.Create(filepath)
	if err != nil {
		return err
	}

	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "    ")

	err = encoder.Encode(save)
	if err != nil {
		return err
	}

	return nil
}
