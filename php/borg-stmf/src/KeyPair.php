<?php

declare(strict_types=1);

namespace Borg\STMF;

/**
 * X25519 keypair for STMF encryption/decryption
 */
class KeyPair
{
    private string $publicKey;
    private string $privateKey;

    /**
     * @param string $publicKey Raw public key bytes (32 bytes)
     * @param string $privateKey Raw private key bytes (32 bytes)
     */
    public function __construct(string $publicKey, string $privateKey)
    {
        if (strlen($publicKey) !== SODIUM_CRYPTO_BOX_PUBLICKEYBYTES) {
            throw new \InvalidArgumentException(
                'Public key must be ' . SODIUM_CRYPTO_BOX_PUBLICKEYBYTES . ' bytes'
            );
        }
        if (strlen($privateKey) !== SODIUM_CRYPTO_BOX_SECRETKEYBYTES) {
            throw new \InvalidArgumentException(
                'Private key must be ' . SODIUM_CRYPTO_BOX_SECRETKEYBYTES . ' bytes'
            );
        }

        $this->publicKey = $publicKey;
        $this->privateKey = $privateKey;
    }

    /**
     * Generate a new X25519 keypair
     */
    public static function generate(): self
    {
        $keypair = sodium_crypto_box_keypair();
        return new self(
            sodium_crypto_box_publickey($keypair),
            sodium_crypto_box_secretkey($keypair)
        );
    }

    /**
     * Load keypair from base64-encoded private key
     */
    public static function fromPrivateKeyBase64(string $privateKeyBase64): self
    {
        $privateKey = base64_decode($privateKeyBase64, true);
        if ($privateKey === false) {
            throw new \InvalidArgumentException('Invalid base64 private key');
        }

        // Derive public key from private key
        $publicKey = sodium_crypto_scalarmult_base($privateKey);

        return new self($publicKey, $privateKey);
    }

    /**
     * Get the raw public key bytes
     */
    public function getPublicKey(): string
    {
        return $this->publicKey;
    }

    /**
     * Get the raw private key bytes
     */
    public function getPrivateKey(): string
    {
        return $this->privateKey;
    }

    /**
     * Get the public key as base64
     */
    public function getPublicKeyBase64(): string
    {
        return base64_encode($this->publicKey);
    }

    /**
     * Get the private key as base64
     */
    public function getPrivateKeyBase64(): string
    {
        return base64_encode($this->privateKey);
    }
}
