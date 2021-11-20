package config

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/pkg/errors"
)

type Config struct {
	ApiPort  string `json:"api_port"`
	LocalURL string `json:"local_url"`
}

// returns a parsed json formatted configuration
func Parse(filepath string) (*Config, error) {
	config := Config{}
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, errors.Wrap(err, "could not read config file")
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal config file")
	}

	log.Println("CONFIGURATION loaded:")

	//setting a default value for api port if empty
	if config.ApiPort == "" {
		config.ApiPort = "8080"
	}

	//setting a default value for local url if empty
	if config.LocalURL == "" {
		config.LocalURL = "http://localhost"
	}

	return &config, nil
}
