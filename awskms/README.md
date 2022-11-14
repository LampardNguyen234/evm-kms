# AWS KMS signer for go-ethereum
This package uses the Amazon Web Services' Key Management Service to provide a signing interface for EVM-compatible transactions. 
Rather than directly accessing a private key to sign a transaction, the client makes calls to the remote AWS KMS to do so 
and the private key never leaves the KMS.
## Import
```go
import "github.com/LampardNguyen234/evm-kms/awskms"
```

## Interact with the Code

### Create a KMSSigner
```go
ctx := context.Background()

cfg := Config{
    KeyID:   "KEY_ID",
    ChainID: 1,
}
awsCfg, err := config.LoadDefaultConfig(ctx,
    config.WithRegion("AWS_REGION"),
    config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
        "ACCESS_KEY", "ACCESS_SECRET", "SESSION")))
if err != nil {
    panic(err)
}
kmsClient := kms.NewFromConfig(awsCfg)

c, err = NewAmazonKMSClient(ctx, cfg, kmsClient)
if err != nil {
    panic(err)
}
```

Or one can create a KMSSigner directly from a given config file:
```go
cfg, err := LoadStaticCredentialsConfigConfigFromFile("./config-static-credentials-example.json")
c, err := NewAmazonKMSClientWithStaticCredentials(ctx, *cfg)
if err != nil {
    panic(err)
}
```
The config file looks like the following:
```json
{
    "KeyID": "KEY_ID",
    "ChainID": 1,
    "Region": "AWS_REGION",
    "AccessKeyID": "ACCESS_KEY",
    "SecretAccessKey": "ACCESS_SECRET",
    "SessionToken": "SESSION"
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
An example of this is [here](../common/erc20/ERC20.go). The `Transfer` function takes as input a `*bind.TransactOpts`, which
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
