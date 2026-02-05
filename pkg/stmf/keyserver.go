package stmf

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Snider/Enchantrix/pkg/keyserver"
	"github.com/Snider/Enchantrix/pkg/keystore"
)

// DecryptKS decrypts a STMF payload using the keyserver. The server's X25519
// private key never leaves the keyserver — ECDH and decryption happen internally.
func DecryptKS(ctx context.Context, stmfData []byte, keyID string, ks keyserver.KeyServer) (*FormData, error) {
	// Keyserver performs ECDH + decrypt internally — private key never leaves
	plaintext, err := ks.DecryptSTMF(ctx, keyID, stmfData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	var formData FormData
	if err := json.Unmarshal(plaintext, &formData); err != nil {
		return nil, fmt.Errorf("%w: invalid JSON payload: %v", ErrInvalidPayload, err)
	}

	return &formData, nil
}

// GenerateKeyPairKS generates an X25519 keypair in the keyserver and returns
// the public key for distribution. The private key stays in the keyserver.
func GenerateKeyPairKS(ctx context.Context, label string, ks keyserver.KeyServer) (publicKey []byte, keyID string, err error) {
	keyID, err = ks.GenerateKey(ctx, keystore.X25519, label)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate keypair: %w", err)
	}

	publicKey, err = ks.GetPublicKey(ctx, keyID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get public key: %w", err)
	}

	return publicKey, keyID, nil
}
