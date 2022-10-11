package common

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"math/big"
)

// KmsSignature represents decoded signatures returned from the KMS.
//
// A KmsSignature only consists of 2 points: R, S. In order for it to be EVM-compatible, we need to manually convert
// it to the (r || s || v) form.
type KmsSignature struct {
	R, S *big.Int
}

// KmsToEVMSignature converts a KmsSignature into an EVM-compatible signature of the following form: r || s || v,
// with v either 0 or 1. The `WithSignature` function will adjust the value of v based on the Signer type (types.Signer).
// Reference: https://eips.ethereum.org/EIPS/eip-155.
func KmsToEVMSignature(pubKey ecdsa.PublicKey,
	kmsSig KmsSignature,
	digestedMsg common.Hash,
) ([]byte, error) {
	// For a signature to be valid, s must be less than n/2 + 1. Therefore, we first adjust s here.
	// https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.md
	if kmsSig.S.Cmp(CurveOrderHalf) > 0 {
		kmsSig.S = new(big.Int).Sub(CurveOrder, kmsSig.S)
	}

	// re-verify the signature
	if !ecdsa.Verify(&pubKey, digestedMsg[:], kmsSig.R, kmsSig.S) {
		return nil, fmt.Errorf("failed to verify signature")
	}

	// retrieve the bytes version of the public key for double-checking
	pubKeyBytes := secp256k1.S256().Marshal(pubKey.X, pubKey.Y)

	rsSig := append(pad(kmsSig.R.Bytes(), 32), pad(kmsSig.S.Bytes(), 32)...)

	v := uint64(0)

	// We try different value of v
	sig := append(rsSig, byte(v))
	recoveredPubKey, err := crypto.Ecrecover(digestedMsg[:], sig)
	if err != nil {
		return nil, fmt.Errorf("failed to recover pubKey with v = 0: %v", err)
	}

	if !bytes.Equal(recoveredPubKey, pubKeyBytes) {
		// try v = 1
		v = 1
		sig = append(rsSig, byte(v))
		recoveredPubKey, err = crypto.Ecrecover(digestedMsg[:], sig)
		if err != nil {
			return nil, fmt.Errorf("failed to recover pubKey with v = 1: %v", err)
		}
		if !bytes.Equal(recoveredPubKey, pubKeyBytes) {
			return nil, fmt.Errorf("cannot convert signature")
		}
	}

	return sig, nil
}

func pad(input []byte, paddedLength int) []byte {
	input = bytes.TrimLeft(input, "\x00")
	for len(input) < paddedLength {
		zeroBuf := []byte{0}
		input = append(zeroBuf, input...)
	}
	return input
}
