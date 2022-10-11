// Package gcpkms uses the Google Cloud Platform's Key Management Service to provide a signing interface for EVM-compatible
// transactions.
//
// Rather than directly accessing a private key to sign a transaction, the client makes calls to the remote
// GCP KMS to do so and the private key never leaves the KMS.
package gcpkms
