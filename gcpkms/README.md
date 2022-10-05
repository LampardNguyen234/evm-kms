# GCP KMS signer for go-ethereum
Despite having algorithms for asymmetric signing, Google Cloud Platform's Key Management Service (GCP KMS) does not support signing Ethereum transactions by default.
While the Ethereum network (or EVM-compatible networks) uses ECDSA-SECP256K1-KECCAK256 for signing, the GCP KMS only supports 
[ECDSA-SECP256K1-SHA256](https://cloud.google.com/kms/docs/algorithms#elliptic_curve_signing_algorithms) to the nearest. 
This limits the ability of the GCP KMS to directly provide secure key management for blockchains,
especially Ethereum.

Fortunately, with some tricks and extensions, we are able to convert a signature from the GCP KMS to an EVM-compatible signature.
This package is dedicated to doing so.
## Import
```go
import "github.com/LampardNguyen234/evm-kms/gcpkms"
```

## Dependencies
```go
go 1.18

require (
	cloud.google.com/go/kms v1.4.0
	github.com/ethereum/go-ethereum v1.10.25
	google.golang.org/api v0.98.0
	google.golang.org/genproto v0.0.0-20220930163606-c98284e70a91
	google.golang.org/protobuf v1.28.1
)
```

## Prerequisites
### Create a KMS Key
In order to sign Ethereum transactions using this package, you need to create a
KMS key in GCP. To do this, please follow the instruction [here](https://cloud.google.com/kms/docs/creating-asymmetric-keys).
Remember to choose `Purpose = Asymmetric sign` and `Algorithm = Elliptic Curve secp256k1 - SHA256 Digest` in the 
`Create key` screen. 

### Download Credential
Head over [the `IAM` page of the GCP](https://cloud.google.com/iam/docs/service-accounts), create a service account to use to API and download the credential.
The credential file looks like the following:
```json
{
  "type": "service_account",
  "project_id": "__REDACTED__",
  "private_key_id": "__REDACTED__",
  "private_key": "-----BEGIN PRIVATE KEY-----\n__REDACTED__\n-----END PRIVATE KEY-----\n",
  "client_email": "__REDACTED__",
  "client_id": "__REDACTED__",
  "auth_uri": "https://accounts.google.com/o/oauth2/auth",
  "token_uri": "https://oauth2.googleapis.com/token",
  "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
  "client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/__REDACTED__"
}
```
Save this file to your machine, you will need it later. Example location: `/Users/SomeUser/.cred/gcp-credential.json`.

## Interact with the Code

### Prepare the config
To create a new GoogleKMSClient, we need to know the detail of the key we are using. This can be initiated via a config
of the following form:
```go
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
```

### Create the client
Here is an example config:
```json
{
  "ProjectID": "evm-kms",
  "LocationID": "us-west1",
  "CredentialLocation": "/Users/SomeUser/.cred/gcp-credential.json",
  "Key": {
    "Keyring": "my-keying-name",
    "Name": "evm-ecdsa",
    "Version": "1"
  },
  "ChainID": 1
}
```

Then, create a client using the `NewGoogleKMSClient` function.
```go
var err error
cfg = &Config{
    ProjectID:          "evm-kms",
    LocationID:         "us-west1",
    CredentialLocation: "/Users/SomeUser/.cred/gcp-credential.json",
    Key: Key{
        Keyring: "my-keying-name",
        Name:    "evm-ecdsa",
        Version: "1",
    },
    ChainID: 1,
}

c, err = NewGoogleKMSClient(context.Background(), *cfg)
if err != nil {
    panic(err)
}
```

### Send ETH
#### Create a transaction
```go
testTx := types.NewTx(&types.LegacyTx{
    To:       &common.HexToAddress("0x243e9517a24813a2d73e9a74cd2c1c699d0ff7a5"),
    Nonce:    9090,
    GasPrice: big.NewInt(1000000),
    Gas:      50000,
    Value:    big.NewInt(100),
    Data:     []byte{1, 2, 3},
})
```

#### Sign the transaction
```go
signedTx, err := c.GetDefaultEVMTransactor().Signer(c.GetAddress(), testTx)
if err != nil {
    panic(err)
}
jsb, _ := json.Marshal(signedTx)
fmt.Println("signedTx", string(jsb))
```

#### Broadcast the transaction
```go
err = evmClient.SendTransaction(ctx, signedTx)
if err != nil {
    panic(err)
}
```

See [TestSendETH](signer_test.go).

### Send ERC20
The [abigen](https://geth.ethereum.org/docs/dapp/native-bindings) tool generates `.go` binding files that are able to directly operate with the `*bind.TransactOpts` type. 
An example of this is [here](./test/erc20/ERC20.go). The `Transfer` function takes as input a `*bind.TransactOpts`, which
can be retrieved via the `GetDefaultEVMTransactor` function of the client, or can be constructed manually, as long as 
a `bind.SignerFn` is supplied.
```go
transactor := &bind.TransactOpts{
    From:      c.GetAddress(),
    Nonce:     new(big.Int).SetUint64(90909),
    Signer:    c.GetEVMSignerFn(),
    GasPrice:  big.NewInt(1000000),
    GasLimit:  50000,
    Context:   ctx,
}

tx, err := erc20Instance.Transfer(transactor, common.HexToAddress("0x243e9517a24813a2d73e9a74cd2c1c699d0ff7a5"), new(big.Int).SetUint64(1000))
if err != nil {
    panic(err)
}
```

See [TestSendERC20](signer_test.go).