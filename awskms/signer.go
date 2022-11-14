package awskms

import (
	"context"
	"crypto/ecdsa"
	"encoding/asn1"
	"fmt"
	common2 "github.com/LampardNguyen234/evm-kms/common"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	"math/big"
)

const (
	signingAlgorithm   = "ECDSA_SHA_256"
	signingMessageType = "DIGEST"
)

// AmazonKMSClient implements basic functionalities of an Amazon Web Services' KMS client for signing transactions.
type AmazonKMSClient struct {
	kmsClient *kms.Client
	ctx       context.Context
	cfg       Config
	publicKey *ecdsa.PublicKey
	signer    types.Signer
}

// NewAmazonKMSClient creates a new AWS KMS client with the given config.
//
// If txSigner is not provided, the signer will be initiated as a types.NewLondonSigner(cfg.ChainID).
// Note that only the first value of txSigner is used.
//
// Example:
//
// 	ctx := context.Background()
//
// 	cfg := Config{
//		KeyID:   "YOUR_KEY_ID_HERE",
//		ChainID: 1,
// 	}
//
// 	awsCfg, err := config.LoadDefaultConfig(ctx,
//		config.WithRegion("AWS_REGION"),
//		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
//			"ACCESS_KEY_ID",
//			"SECRET_ACCESS_KEY",
//			"SESSION",
//		)),
// 	)
// 	if err != nil {
// 		panic(err)
//	}
//	kmsClient := kms.NewFromConfig(awsCfg)
//
//	c, err = NewAmazonKMSClient(ctx, cfg, kmsClient)
//	if err != nil {
//		panic(err)
//	}
func NewAmazonKMSClient(ctx context.Context, cfg Config, kmsClient *kms.Client, txSigner ...types.Signer) (*AmazonKMSClient, error) {
	if _, err := cfg.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid config")
	}

	signer := types.NewLondonSigner(new(big.Int).SetUint64(cfg.ChainID))
	if len(txSigner) > 0 {
		signer = txSigner[0]
	}

	c := &AmazonKMSClient{kmsClient: kmsClient, ctx: ctx, cfg: cfg, signer: signer}

	pubKey, err := c.getPublicKey()
	if err != nil {
		return nil, err
	}
	c.publicKey = pubKey

	return c, nil
}

// NewAmazonKMSClientWithStaticCredentials is an alternative of NewAmazonKMSClient but uses a StaticCredentialsConfig.
func NewAmazonKMSClientWithStaticCredentials(ctx context.Context, cfg StaticCredentialsConfig, txSigner ...types.Signer) (*AmazonKMSClient, error) {
	if _, err := cfg.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid config")
	}

	signer := types.NewLondonSigner(new(big.Int).SetUint64(cfg.ChainID))
	if len(txSigner) > 0 {
		signer = txSigner[0]
	}

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken)))
	if err != nil {
		panic(err)
	}
	kmsClient := kms.NewFromConfig(awsCfg)

	c := &AmazonKMSClient{kmsClient: kmsClient, ctx: ctx, cfg: cfg.Config, signer: signer}

	pubKey, err := c.getPublicKey()
	if err != nil {
		return nil, err
	}
	c.publicKey = pubKey

	return c, nil
}

// GetAddress returns the EVM address of the current signer.
func (c AmazonKMSClient) GetAddress() common.Address {
	return crypto.PubkeyToAddress(*c.publicKey)
}

// GetPublicKey returns the public Key corresponding to the given keyId.
func (c AmazonKMSClient) GetPublicKey() (*ecdsa.PublicKey, error) {
	return c.publicKey, nil
}

// SignHash calls the remote AWS KMS to sign a given digested message.
// Although the AWS KMS does not support keccak256 hash function (it uses SHA256 instead), it will not care about
// which hash function to use if you send the hash of message to the KMS.
func (c AmazonKMSClient) SignHash(digest common.Hash) ([]byte, error) {
	signInput := &kms.SignInput{
		KeyId:            &c.cfg.KeyID,
		Message:          digest[:],
		SigningAlgorithm: signingAlgorithm,
		MessageType:      signingMessageType,
	}

	result, err := c.kmsClient.Sign(c.ctx, signInput)
	if err != nil {
		return nil, fmt.Errorf("failed to sign digest: %v", err)
	}

	return c.parseKMSSignature(digest, result.Signature)
}

// GetDefaultEVMTransactor returns the default KMS-backed instance of bind.TransactOpts.
// Only `Context`, `From`, and `Signer` fields are set.
func (c AmazonKMSClient) GetDefaultEVMTransactor() *bind.TransactOpts {
	return &bind.TransactOpts{
		Context: c.ctx,
		From:    c.GetAddress(),
		Signer:  c.GetEVMSignerFn(),
	}
}

// GetEVMSignerFn returns the EVM signer using the AWS KMS.
func (c AmazonKMSClient) GetEVMSignerFn() bind.SignerFn {
	return func(addr common.Address, tx *types.Transaction) (*types.Transaction, error) {
		if addr != c.GetAddress() {
			return nil, bind.ErrNotAuthorized
		}

		sig, err := c.SignHash(c.signer.Hash(tx))
		if err != nil {
			return nil, fmt.Errorf("cannot sign transaction: %v", err)
		}

		ret, err := tx.WithSignature(c.signer, sig)
		if err != nil {
			return nil, err
		}

		if _, err = c.HasSignedTx(ret); err != nil {
			return nil, err
		}

		return ret, nil
	}
}

// HasSignedTx checks if the given tx is signed by the current AmazonKMSClient.
func (c AmazonKMSClient) HasSignedTx(tx *types.Transaction) (bool, error) {
	from, err := types.Sender(c.signer, tx)
	if err != nil {
		return false, fmt.Errorf("cannot get sender of the tx: %v", err)
	}

	if from != c.GetAddress() {
		return false, fmt.Errorf("expected signer: %v, got %v", c.GetAddress(), from)
	}

	return true, nil
}

// WithSigner assigns the given signer to the AmazonKMSClient.
func (c *AmazonKMSClient) WithSigner(signer types.Signer) {
	c.signer = signer
}

// WithChainID assigns given chainID (and updates the corresponding signer) to the AmazonKMSClient.
func (c *AmazonKMSClient) WithChainID(chainID *big.Int) {
	if c.cfg.ChainID != chainID.Uint64() {
		c.cfg.ChainID = chainID.Uint64()
		c.signer = types.NewLondonSigner(chainID)
	}
}

func (c AmazonKMSClient) getPublicKey() (*ecdsa.PublicKey, error) {
	getPubKeyOutput, err := c.kmsClient.GetPublicKey(c.ctx, &kms.GetPublicKeyInput{
		KeyId: aws.String(c.cfg.KeyID),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get public key from AWS KMS for KeyId=%v", c.cfg.KeyID)
	}

	return parseKMSPublicKey(getPubKeyOutput)
}

// parseKMSSignature parses a signature returned from the AWS KMS to a valid EVM-compatible signature.
// A valid EVM signature is a 65-byte long RLP-encoded of the form R || S || V (https://eips.ethereum.org/EIPS/eip-155).
func (c AmazonKMSClient) parseKMSSignature(digestedMsg common.Hash,
	kmsSignature []byte,
) ([]byte, error) {
	// recover r, s
	var sig common2.KmsSignature
	_, err := asn1.Unmarshal(kmsSignature, &sig)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal kms signature: %v", err)
	}

	// convert the signature into a valid EVM signature.
	return common2.KmsToEVMSignature(*c.publicKey, sig, digestedMsg)
}

// parseKMSPublicKey parses a public Key returned from the AWS KMS to a valid ecdsa.PublicKey.
func parseKMSPublicKey(kmsPubKey *kms.GetPublicKeyOutput) (*ecdsa.PublicKey, error) {
	type pubKeyHolder struct {
		EcPublicKeyInfo struct {
			Algorithm  asn1.ObjectIdentifier
			Parameters asn1.ObjectIdentifier
		}
		PublicKey asn1.BitString
	}
	var pubKeyInfo pubKeyHolder
	_, err := asn1.Unmarshal(kmsPubKey.PublicKey, &pubKeyInfo)
	if err != nil || len(pubKeyInfo.PublicKey.Bytes) == 0 {
		return nil, fmt.Errorf("cannot decode public key %x: %v", kmsPubKey.PublicKey, err)
	}

	return crypto.UnmarshalPubkey(pubKeyInfo.PublicKey.Bytes)
}
