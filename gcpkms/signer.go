package gcpkms

import (
	kms "cloud.google.com/go/kms/apiv1"
	"context"
	"crypto/ecdsa"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	common2 "github.com/LampardNguyen234/evm-kms/common"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	kmspb "google.golang.org/genproto/googleapis/cloud/kms/v1"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"hash/crc32"
	"math/big"
)

// GoogleKMSClient implements basic functionalities of a Google KMS client for signing transactions.
type GoogleKMSClient struct {
	kmsClient *kms.KeyManagementClient
	ctx       context.Context
	cfg       Config
	publicKey *ecdsa.PublicKey
	signer    types.Signer
}

// NewGoogleKMSClient creates a new GCP KMS client with the given config.
//
// If txSigner is not provided, the signer will be initiated as a types.NewLondonSigner(cfg.ChainID).
// Note that only the first value of txSigner is used.
func NewGoogleKMSClient(ctx context.Context, cfg Config, txSigner ...types.Signer) (*GoogleKMSClient, error) {
	if _, err := cfg.IsValid(); err != nil {
		return nil, fmt.Errorf("invalid config")
	}
	client, err := kms.NewKeyManagementClient(ctx, option.WithCredentialsFile(cfg.CredentialLocation))
	if err != nil {
		return nil, err
	}

	signer := types.NewLondonSigner(new(big.Int).SetUint64(cfg.ChainID))
	if len(txSigner) > 0 {
		signer = txSigner[0]
	}

	c := &GoogleKMSClient{kmsClient: client, ctx: ctx, cfg: cfg, signer: signer}

	pubKey, err := c.getPublicKey()
	if err != nil {
		return nil, err
	}
	c.publicKey = pubKey

	return c, nil
}

// GetAddress returns the EVM address of the current signer.
func (c GoogleKMSClient) GetAddress() common.Address {
	return crypto.PubkeyToAddress(*c.publicKey)
}

// GetPublicKey returns the public Key corresponding to the given keyId.
func (c GoogleKMSClient) GetPublicKey() (*ecdsa.PublicKey, error) {
	return c.publicKey, nil
}

// SignHash calls the remote GCP KMS to sign a given digested message.
// Although the GCP KMS does not support keccak256 hash function (it uses SHA256 instead), it will not care about
// which hash function to use if you send the hash of message to the KMS.
func (c GoogleKMSClient) SignHash(digest common.Hash) ([]byte, error) {
	// calculate the digest of the message

	// compute digest's CRC32C
	crc32c := func(data []byte) uint32 {
		t := crc32.MakeTable(crc32.Castagnoli)
		return crc32.Checksum(data, t)

	}
	digestCRC32C := crc32c(digest[:])

	// build the signing request
	req := &kmspb.AsymmetricSignRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s/cryptoKeyVersions/%s",
			c.cfg.ProjectID, c.cfg.LocationID, c.cfg.Key.Keyring, c.cfg.Key.Name, c.cfg.Key.Version),
		Digest: &kmspb.Digest{
			// we send the hash to the remote KMS, not the actual data
			Digest: &kmspb.Digest_Sha256{
				Sha256: digest[:],
			},
		},
		DigestCrc32C: wrapperspb.Int64(int64(digestCRC32C)),
	}

	// call the API
	result, err := c.kmsClient.AsymmetricSign(c.ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to sign digest: %v", err)
	}

	// perform integrity verification on result
	if result.VerifiedDigestCrc32C == false {
		return nil, fmt.Errorf("AsymmetricSign: request corrupted in-transit")
	}
	if int64(crc32c(result.Signature)) != result.SignatureCrc32C.Value {
		return nil, fmt.Errorf("AsymmetricSign: response corrupted in-transit")
	}

	return c.parseKMSSignature(digest, result.Signature)
}

// GetDefaultEVMTransactor returns the default KMS-backed instance of bind.TransactOpts.
// Only `Context`, `From`, and `Signer` fields are set.
func (c GoogleKMSClient) GetDefaultEVMTransactor() *bind.TransactOpts {
	return &bind.TransactOpts{
		Context: c.ctx,
		From:    c.GetAddress(),
		Signer:  c.GetEVMSignerFn(),
	}
}

// GetEVMSignerFn returns the EVM signer using the GCP KMS.
func (c GoogleKMSClient) GetEVMSignerFn() bind.SignerFn {
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

// HasSignedTx checks if the given tx is signed by the current GoogleKMSClient.
func (c GoogleKMSClient) HasSignedTx(tx *types.Transaction) (bool, error) {
	from, err := types.Sender(c.signer, tx)
	if err != nil {
		return false, fmt.Errorf("cannot get sender of the tx: %v", err)
	}

	if from != c.GetAddress() {
		return false, fmt.Errorf("expected signer: %v, got %v", c.GetAddress(), from)
	}

	return true, nil
}

// WithSigner assigns the given signer to the GoogleKMSClient.
func (c *GoogleKMSClient) WithSigner(signer types.Signer) {
	c.signer = signer
}

func (c GoogleKMSClient) getPublicKey() (*ecdsa.PublicKey, error) {
	req := &kmspb.GetPublicKeyRequest{
		Name: fmt.Sprintf("projects/%s/locations/%s/keyRings/%s/cryptoKeys/%s/cryptoKeyVersions/%s",
			c.cfg.ProjectID, c.cfg.LocationID, c.cfg.Key.Keyring, c.cfg.Key.Name, c.cfg.Key.Version),
	}
	pubKey, err := c.kmsClient.GetPublicKey(c.ctx, req)
	if err != nil {
		return nil, err
	}

	return parseKMSPublicKey(pubKey)
}

// parseKMSSignature parses a signature returned from the GCP KMS to a valid EVM-compatible signature.
// A valid EVM signature is a 65-byte long RLP-encoded of the form R || S || V (https://eips.ethereum.org/EIPS/eip-155).
func (c GoogleKMSClient) parseKMSSignature(digestedMsg common.Hash,
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

func (c GoogleKMSClient) describe() error {
	// Create the request to list KeyRings.
	listKeyRingsReq := &kmspb.ListKeyRingsRequest{
		Parent: fmt.Sprintf("projects/%s/locations/%s", c.cfg.ProjectID, c.cfg.LocationID),
	}

	// List the KeyRings.
	it := c.kmsClient.ListKeyRings(c.ctx, listKeyRingsReq)

	// Iterate and print the results.
	for {
		resp, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to list Key rings: %v", err)
		}

		fmt.Printf("Key ring: %s\n", resp.Name)
	}

	return nil
}

// parseKMSPublicKey parses a public Key returned from the GCP KMS to a valid ecdsa.PublicKey.
func parseKMSPublicKey(kmsPubKey *kmspb.PublicKey) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(kmsPubKey.Pem))
	if block == nil || block.Type != "PUBLIC KEY" || len(block.Bytes) < 64 {
		return nil, fmt.Errorf("cannot decode public Key %v", kmsPubKey.Pem)
	}

	// last 64 bytes of block.Bytes are: x, y
	pubKeyBytes := block.Bytes[len(block.Bytes)-64:]
	x := new(big.Int).SetBytes(pubKeyBytes[:32])
	y := new(big.Int).SetBytes(pubKeyBytes[32:])

	// check if the point is on the secp256k1 curve
	if !secp256k1.S256().IsOnCurve(x, y) {
		return nil, fmt.Errorf("invalid secp256k1 public Key %v", kmsPubKey.Pem)
	}
	pubKey := ecdsa.PublicKey{
		Curve: secp256k1.S256(),
		X:     x,
		Y:     y,
	}

	return &pubKey, nil
}
