package gcpkms

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// Key consists of required information to retrieve the CGP KMS Key path.
type Key struct {
	// Keyring is the name of your KMS keyring.
	Keyring string `json:"Keyring"`

	// Name is the name of the key in the Keyring.
	Name string `json:"Name"`

	// Version is the of the current key.
	Version string `json:"Version"`
}

func (k Key) isValid() bool {
	return k.Keyring != "" && k.Name != "" && k.Version != ""
}

// Config represents required information to create a Google Cloud KMS client.
type Config struct {
	// ProjectID is the ID of the working GCP project.
	ProjectID string `json:"ProjectID"`

	// LocationID is the region ID of the project.
	//
	// Example: us-west1.
	LocationID string `json:"LocationID"`

	// CredentialLocation is the absolute path of the credential file downloaded from the GCP.
	//
	// Example: "/Users/SomeUser/.cred/gcp-credential.json".
	// Leave this field empty if the environment varialbe `GOOGLE_APPLICATION_CREDENTIALS` has been set.
	CredentialLocation string `json:"CredentialLocation,omitempty"`

	// Key is the detail of the GCP KMS key.
	Key Key `json:"Key"`

	// ChainID is the ID of the target EVM chain.
	//
	// See https://chainlist.org.
	ChainID uint64 `json:"ChainID"`
}

// IsValid checks if a Config is valid.
func (cfg Config) IsValid() (bool, error) {
	if cfg.ProjectID == "" {
		return false, fmt.Errorf("empty ProjectID")
	}

	if cfg.LocationID == "" {
		return false, fmt.Errorf("empty LocationID")
	}

	if !cfg.Key.isValid() {
		return false, fmt.Errorf("invalid Key")
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

	if cfg.CredentialLocation == "" {
		cfg.CredentialLocation = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	}

	if _, err = cfg.IsValid(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
