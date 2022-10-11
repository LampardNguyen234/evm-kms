package awskms

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Config represents required information to create an AWS KMS client.
type Config struct {
	// KeyID is the ID of the working AWS KMS key.
	KeyID string `json:"KeyID"`

	// ChainID is the ID of the target EVM chain.
	//
	// See https://chainlist.org.
	ChainID uint64 `json:"ChainID"`
}

// IsValid checks if a Config is valid.
func (cfg Config) IsValid() (bool, error) {
	if cfg.KeyID == "" {
		return false, fmt.Errorf("empty KeyID")
	}

	return true, nil
}

// LoadConfigFromFile loads the config from the given config file.
func LoadConfigFromFile(filePath string) (*Config, error) {
	f, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(f, &cfg)
	if err != nil {
		return nil, err
	}

	if _, err = cfg.IsValid(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
