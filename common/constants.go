package common

import (
	secp256k1 "github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

var (
	// CurveOrder is the order of the secp256k1 elliptic curve.
	CurveOrder = secp256k1.S256().Params().N

	// CurveOrderHalf = CurveOrder / 2.
	CurveOrderHalf = new(big.Int).Div(CurveOrder, new(big.Int).SetUint64(2))
)
