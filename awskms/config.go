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

// StaticCredentialsConfig consists of AWS KMS Config with static credentials.
//
// Example:
//   scConfig = StaticCredentialsConfig{
//  	KeyID: "KEY_ID",
// 		ChainID: 0,
//		Region: "REGION_ID",
//		AccessKeyID: "ACCESS_KEY_ID",
//		SecretAccessKey: "SECRET_ACCESS_KEY",
//		SessionToken: "SESSION_TOKEN",
//  }
type StaticCredentialsConfig struct {
	Config

	// Region is the region of the AWS KMS Key.
	Region string `json:"Region"`

	// AccessKeyID is the access key ID of the given account for the sake of connecting to the remote AWS client.
	AccessKeyID string `json:"AccessKeyID"`

	// SecretAccessKey is the secret key for the AccessKeyID.
	SecretAccessKey string `json:"SecretAccessKey"`

	// SessionToken is the session ID.
	SessionToken string `json:"SessionToken,omitempty"`
}

func (cfg StaticCredentialsConfig) IsValid() (bool, error) {
	if cfg.Region == "" {
		return false, fmt.Errorf("empty Region")
	}

	if cfg.AccessKeyID == "" {
		return false, fmt.Errorf("empty AccessKeyID")
	}

	if cfg.SecretAccessKey == "" {
		return false, fmt.Errorf("empty SecretAccessKey")
	}

	return cfg.Config.IsValid()
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

// LoadStaticCredentialsConfigConfigFromFile loads a static credential config from the given config file.
func LoadStaticCredentialsConfigConfigFromFile(filePath string) (*StaticCredentialsConfig, error) {
	f, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var cfg StaticCredentialsConfig
	err = json.Unmarshal(f, &cfg)
	if err != nil {
		return nil, err
	}

	if _, err = cfg.IsValid(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
