package kms

import (
	"crypto/ecdsa"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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
	WithSigner(signer types.Signer)
}
