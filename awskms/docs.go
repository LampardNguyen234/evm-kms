// Package awskms uses the Amazon Web Services' Key Management Service to provide a signing interface for EVM-compatible
// transactions.
//
// Rather than directly accessing a private key to sign a transaction, the client makes calls to the remote
// AWS KMS to do so and the private key never leaves the KMS.
package awskms
