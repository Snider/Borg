package stmf

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// KeyPair represents an X25519 keypair for STMF encryption
type KeyPair struct {
	privateKey *ecdh.PrivateKey
	publicKey  *ecdh.PublicKey
}

// GenerateKeyPair generates a new X25519 keypair
func GenerateKeyPair() (*KeyPair, error) {
	curve := ecdh.X25519()
	privateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
	}

	return &KeyPair{
		privateKey: privateKey,
		publicKey:  privateKey.PublicKey(),
	}, nil
}

// PublicKey returns the raw public key bytes (32 bytes)
func (k *KeyPair) PublicKey() []byte {
	return k.publicKey.Bytes()
}

// PrivateKey returns the raw private key bytes (32 bytes)
func (k *KeyPair) PrivateKey() []byte {
	return k.privateKey.Bytes()
}

// PublicKeyBase64 returns the public key as a base64-encoded string
func (k *KeyPair) PublicKeyBase64() string {
	return base64.StdEncoding.EncodeToString(k.publicKey.Bytes())
}

// PrivateKeyBase64 returns the private key as a base64-encoded string
func (k *KeyPair) PrivateKeyBase64() string {
	return base64.StdEncoding.EncodeToString(k.privateKey.Bytes())
}

// LoadPublicKey loads a public key from raw bytes
func LoadPublicKey(data []byte) (*ecdh.PublicKey, error) {
	curve := ecdh.X25519()
	pub, err := curve.NewPublicKey(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPublicKey, err)
	}
	return pub, nil
}

// LoadPublicKeyBase64 loads a public key from a base64-encoded string
func LoadPublicKeyBase64(encoded string) (*ecdh.PublicKey, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64: %v", ErrInvalidPublicKey, err)
	}
	return LoadPublicKey(data)
}

// LoadPrivateKey loads a private key from raw bytes
func LoadPrivateKey(data []byte) (*ecdh.PrivateKey, error) {
	curve := ecdh.X25519()
	priv, err := curve.NewPrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPrivateKey, err)
	}
	return priv, nil
}

// LoadPrivateKeyBase64 loads a private key from a base64-encoded string
func LoadPrivateKeyBase64(encoded string) (*ecdh.PrivateKey, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64: %v", ErrInvalidPrivateKey, err)
	}
	return LoadPrivateKey(data)
}

// LoadKeyPair loads a keypair from raw private key bytes
func LoadKeyPair(privateKeyBytes []byte) (*KeyPair, error) {
	priv, err := LoadPrivateKey(privateKeyBytes)
	if err != nil {
		return nil, err
	}
	return &KeyPair{
		privateKey: priv,
		publicKey:  priv.PublicKey(),
	}, nil
}

// LoadKeyPairBase64 loads a keypair from a base64-encoded private key
func LoadKeyPairBase64(privateKeyBase64 string) (*KeyPair, error) {
	data, err := base64.StdEncoding.DecodeString(privateKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid base64: %v", ErrInvalidPrivateKey, err)
	}
	return LoadKeyPair(data)
}
