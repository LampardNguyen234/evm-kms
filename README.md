[![Go Report Card](https://goreportcard.com/badge/github.com/LampardNguyen234/evm-kms)](https://goreportcard.com/report/github.com/LampardNguyen234/evm-kms)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/LampardNguyen234/evm-kms/blob/main/LICENSE)

# EVM-Compatible KMS
Key Management Service's client for EVM applications.

## Dependencies
See [go.mod](./go.mod)

## Status
This product is currently in _beta quality_, take your own risk. 

### TODOs
- [X] [Google Cloud Platform KMS](./gcpkms/README.md)
- [X] [Amazon Web Services KMS](./awskms/README.md)

### Tutorial
#### Create a config file
Create a json file consisting of the following information:
```json
{
  "type": "gcp",
  "gcp": {
    "ProjectID": "evm-kms",
    "LocationID": "us-west1",
    "CredentialLocation": "/Users/SomeUser/.cred/gcp-credential.json",
    "Key": {
      "Keyring": "my-keying-name",
      "Name": "evm-ecdsa",
      "Version": "1"
    },
    "ChainID": 1
  },
  "aws": {
    "KeyID": "KEY_ID",
    "ChainID": 1,
    "Region": "AWS_REGION",
    "AccessKeyID": "ACCESS_KEY_ID",
    "SecretAccessKey": "SECRET_ACCESS_KEY",
    "SessionToken": "SESSION_TOKEN"
  }
}
```
- If `type = "gcp"`, the `aws` field is not needed.
- If `type = "aws"`, the `gcp` field is not needed.

#### Create a KMSSigner from the config file
```go
kmsSigner, err := NewKMSSignerFromConfigFile("kms-config.json")
if err != nil {
	panic(err)
}
```

## Contributions
You are encouraged to open an [issue](https://github.com/LampardNguyen234/evm-kms/issues/new) if you encounter a problem
while using this code. Even better, you can create [PRs](https://github.com/LampardNguyen234/evm-kms/compare) to the
`main` branch if you think these are necessary functions. 
