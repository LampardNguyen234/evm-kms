package kms

import (
	"encoding/json"
	"fmt"
	"github.com/LampardNguyen234/evm-kms/awskms"
	"github.com/LampardNguyen234/evm-kms/gcpkms"
	"io/ioutil"
	"os"
	"strings"
)

const (
	gcpType = "gcp"
	awsType = "aws"
)

// Config is the holder for the KMS service.
type Config struct {
	// Type indicates which service we are using ('gcp', 'aws').
	Type string `json:"type"`

	// GcpConfig is the detail of the GCP KMS Config.
	GcpConfig gcpkms.Config `json:"gcp"`

	// AwsConfig is the detail of the AWS KMS Config.
	AwsConfig awskms.Config `json:"aws"`
}

// IsValid checks if the current Config is valid.
func (cfg Config) IsValid() (bool, error) {
	switch cfg.Type {
	case awsType:
		return cfg.AwsConfig.IsValid()
	case gcpType:
		return cfg.GcpConfig.IsValid()
	}

	return false, fmt.Errorf("KMS Config type `%v` not supported", strings.ToLower(cfg.Type))
}

// LoadConfigFromJSONFile creates a Config from the given the json config file.
func LoadConfigFromJSONFile(filePath string) (*Config, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	bytesValue, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	err = json.Unmarshal(bytesValue, &res)
	if err != nil {
		return nil, err
	}

	return LoadConfig(res)
}

// LoadConfig creates a Config from the given raw config data.
func LoadConfig(rawConfig map[string]interface{}) (*Config, error) {
	jsb, err := json.Marshal(rawConfig)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = json.Unmarshal(jsb, &cfg)
	if err != nil {
		return nil, err
	}

	if _, err = cfg.IsValid(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
