package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Nodes NodesConfig `json:"nodes"`
}

type NodesConfig struct {
	None                    NodeConfig `json:"none"`
	MossFarm                NodeConfig `json:"moss_farm"`
	MossFermentationChamber NodeConfig `json:"moss_fermentation_chamber"`
}

type NodeConfig struct {
	Radius           float64  `json:"radius"`
	ResourceCapacity int      `json:"resource_capacity"`
	Consumes         []string `json:"consumes"`
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
