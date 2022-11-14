package kms

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"github.com/LampardNguyen234/evm-kms/awskms"
	"github.com/LampardNguyen234/evm-kms/gcpkms"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"strings"
)

// KMSSigner specifies the required methods for a KMS signer
type KMSSigner interface {
	// GetAddress returns the EVM address of the current signer.
	GetAddress() common.Address

	// GetPublicKey returns the EVM public key of the current signer.
	GetPublicKey() (*ecdsa.PublicKey, error)

	// SignHash performs a signing operation for a given digested message.
	SignHash(hash common.Hash) ([]byte, error)

	// GetDefaultEVMTransactor returns the default KMS-backed instance of bind.TransactOpts.
	GetDefaultEVMTransactor() *bind.TransactOpts

	// GetEVMSignerFn returns the KMS-backed bind.SignerFn instance.
	GetEVMSignerFn() bind.SignerFn

	// HasSignedTx checks if the given transaction has been signed by the KMS.
	HasSignedTx(*types.Transaction) (bool, error)

	// WithSigner assigns the given signer to the current KMSSigner.
	WithSigner(types.Signer)

	// WithChainID assigns the given chainID to the current KMSSigner.
	WithChainID(*big.Int)
}

// NewKMSSignerFromConfig creates and returns a new KMSSigner with the given config.
func NewKMSSignerFromConfig(cfg Config) (KMSSigner, error) {
	if _, err := cfg.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	ctx := context.Background()
	switch strings.ToLower(cfg.Type) {
	case awsType:
		return awskms.NewAmazonKMSClientWithStaticCredentials(ctx, cfg.AwsConfig)
	case gcpType:
		return gcpkms.NewGoogleKMSClient(ctx, cfg.GcpConfig)
	}

	return nil, nil
}

// NewKMSSignerFromConfigFile creates and returns a new KMSSigner with the given config file.
func NewKMSSignerFromConfigFile(filePath string) (KMSSigner, error) {
	cfg, err := LoadConfigFromJSONFile(filePath)
	if err != nil {
		return nil, err
	}

	return NewKMSSignerFromConfig(*cfg)
}
