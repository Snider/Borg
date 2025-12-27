<?php

declare(strict_types=1);

namespace Borg\STMF;

/**
 * STMF - Sovereign Form Encryption
 *
 * Decrypts STMF payloads that were encrypted client-side using the server's public key.
 * Uses X25519 ECDH key exchange + ChaCha20-Poly1305 authenticated encryption.
 *
 * @example
 * ```php
 * $stmf = new STMF($privateKeyBase64);
 * $formData = $stmf->decrypt($_POST['_stmf_payload']);
 *
 * $email = $formData->get('email');
 * $password = $formData->get('password');
 * ```
 */
class STMF
{
    private const MAGIC = 'STMF';

    private string $privateKey;

    /**
     * @param string $privateKeyBase64 Base64-encoded X25519 private key
     */
    public function __construct(string $privateKeyBase64)
    {
        $privateKey = base64_decode($privateKeyBase64, true);
        if ($privateKey === false || strlen($privateKey) !== SODIUM_CRYPTO_BOX_SECRETKEYBYTES) {
            throw new \InvalidArgumentException('Invalid private key');
        }
        $this->privateKey = $privateKey;
    }

    /**
     * Decrypt an STMF payload
     *
     * @param string $payloadBase64 Base64-encoded STMF payload
     * @return FormData Decrypted form data
     * @throws InvalidPayloadException If the payload format is invalid
     * @throws DecryptionException If decryption fails
     */
    public function decrypt(string $payloadBase64): FormData
    {
        // Decode base64
        $payload = base64_decode($payloadBase64, true);
        if ($payload === false) {
            throw new InvalidPayloadException('Invalid base64 payload');
        }

        return $this->decryptRaw($payload);
    }

    /**
     * Decrypt raw STMF bytes
     *
     * @param string $payload Raw STMF bytes
     * @return FormData Decrypted form data
     */
    public function decryptRaw(string $payload): FormData
    {
        // Verify magic
        if (strlen($payload) < 4 || substr($payload, 0, 4) !== self::MAGIC) {
            throw new InvalidPayloadException('Invalid STMF magic');
        }

        // Parse trix container
        $trix = $this->parseTrixContainer($payload);

        // Extract ephemeral public key from header
        if (!isset($trix['header']['ephemeral_pk'])) {
            throw new InvalidPayloadException('Missing ephemeral_pk in header');
        }

        $ephemeralPKBase64 = $trix['header']['ephemeral_pk'];
        $ephemeralPK = base64_decode($ephemeralPKBase64, true);
        if ($ephemeralPK === false || strlen($ephemeralPK) !== SODIUM_CRYPTO_BOX_PUBLICKEYBYTES) {
            throw new InvalidPayloadException('Invalid ephemeral public key');
        }

        // Perform X25519 ECDH key exchange
        $sharedSecret = sodium_crypto_scalarmult($this->privateKey, $ephemeralPK);

        // Derive symmetric key using SHA-256 (same as Go implementation)
        $symmetricKey = hash('sha256', $sharedSecret, true);

        // Decrypt the payload with ChaCha20-Poly1305
        $decrypted = $this->chachaDecrypt($trix['payload'], $symmetricKey);
        if ($decrypted === null) {
            throw new DecryptionException('Decryption failed (wrong key?)');
        }

        // Parse JSON
        $data = json_decode($decrypted, true);
        if ($data === null) {
            throw new InvalidPayloadException('Invalid JSON in decrypted payload');
        }

        return FormData::fromArray($data);
    }

    /**
     * Validate an STMF payload without decrypting
     *
     * @param string $payloadBase64 Base64-encoded STMF payload
     * @return bool True if the payload appears valid
     */
    public function validate(string $payloadBase64): bool
    {
        try {
            $payload = base64_decode($payloadBase64, true);
            if ($payload === false) {
                return false;
            }

            if (strlen($payload) < 4 || substr($payload, 0, 4) !== self::MAGIC) {
                return false;
            }

            $trix = $this->parseTrixContainer($payload);
            return isset($trix['header']['ephemeral_pk']);
        } catch (\Exception $e) {
            return false;
        }
    }

    /**
     * Get payload info without decrypting
     *
     * @param string $payloadBase64 Base64-encoded STMF payload
     * @return array{version: ?string, algorithm: ?string, ephemeral_pk: ?string}
     */
    public function getInfo(string $payloadBase64): array
    {
        $payload = base64_decode($payloadBase64, true);
        if ($payload === false) {
            throw new InvalidPayloadException('Invalid base64 payload');
        }

        $trix = $this->parseTrixContainer($payload);

        return [
            'version' => $trix['header']['version'] ?? null,
            'algorithm' => $trix['header']['algorithm'] ?? null,
            'ephemeral_pk' => $trix['header']['ephemeral_pk'] ?? null,
        ];
    }

    /**
     * Parse a Trix container
     *
     * Enchantrix Trix format:
     * - Magic (4 bytes): "STMF"
     * - Version (4 bytes, little-endian): 2
     * - Header length (1 byte or varint)
     * - Header (JSON)
     * - Payload
     *
     * @return array{header: array, payload: string}
     */
    private function parseTrixContainer(string $data): array
    {
        $offset = 4; // Skip magic

        // Skip version (4 bytes)
        if (strlen($data) < $offset + 4) {
            throw new InvalidPayloadException('Payload too short for version');
        }
        $offset += 4;

        // Read header length (varint - for now just handle 1-2 byte cases)
        if (strlen($data) < $offset + 1) {
            throw new InvalidPayloadException('Payload too short for header length');
        }

        $firstByte = ord($data[$offset]);
        $headerLen = 0;

        if ($firstByte < 128) {
            // Single byte length
            $headerLen = $firstByte;
            $offset += 1;
        } else {
            // Two byte length (varint continuation)
            if (strlen($data) < $offset + 2) {
                throw new InvalidPayloadException('Payload too short for header length');
            }
            $secondByte = ord($data[$offset + 1]);
            $headerLen = ($firstByte & 0x7F) | ($secondByte << 7);
            $offset += 2;
        }

        // Read header
        if (strlen($data) < $offset + $headerLen) {
            throw new InvalidPayloadException('Payload too short for header');
        }

        $headerJson = substr($data, $offset, $headerLen);
        $header = json_decode($headerJson, true);
        if ($header === null) {
            throw new InvalidPayloadException('Invalid header JSON: ' . json_last_error_msg());
        }

        $offset += $headerLen;

        // Rest is payload
        $payload = substr($data, $offset);

        return [
            'header' => $header,
            'payload' => $payload,
        ];
    }

    /**
     * Decrypt data encrypted by Go's Enchantrix ChaChaPolySigil
     *
     * Enchantrix format:
     * - Nonce (24 bytes for XChaCha20-Poly1305)
     * - Ciphertext + Auth tag (16 bytes)
     *
     * Enchantrix also applies XOR pre-obfuscation before encryption.
     * After decryption, we must deobfuscate using the nonce as entropy.
     */
    private function chachaDecrypt(string $ciphertext, string $key): ?string
    {
        $nonceLen = SODIUM_CRYPTO_AEAD_XCHACHA20POLY1305_IETF_NPUBBYTES; // 24

        if (strlen($ciphertext) < $nonceLen + SODIUM_CRYPTO_AEAD_XCHACHA20POLY1305_IETF_ABYTES) {
            return null;
        }

        $nonce = substr($ciphertext, 0, $nonceLen);
        $encrypted = substr($ciphertext, $nonceLen);

        try {
            $obfuscated = sodium_crypto_aead_xchacha20poly1305_ietf_decrypt(
                $encrypted,
                '', // Additional data
                $nonce,
                $key
            );

            if ($obfuscated === false) {
                return null;
            }

            // Deobfuscate using XOR with nonce-derived key stream (Enchantrix pattern)
            return $this->xorDeobfuscate($obfuscated, $nonce);
        } catch (\SodiumException $e) {
            return null;
        }
    }

    /**
     * Deobfuscate data using XOR with entropy-derived key stream.
     * This matches Enchantrix's XORObfuscator.
     *
     * The key stream is derived by hashing: SHA256(entropy || blockNumber)
     * for each 32-byte block needed.
     */
    private function xorDeobfuscate(string $data, string $entropy): string
    {
        if (strlen($data) === 0) {
            return $data;
        }

        $keyStream = $this->deriveKeyStream($entropy, strlen($data));
        $result = '';

        for ($i = 0; $i < strlen($data); $i++) {
            $result .= chr(ord($data[$i]) ^ ord($keyStream[$i]));
        }

        return $result;
    }

    /**
     * Derive a key stream from entropy using SHA-256.
     * Matches Enchantrix's XORObfuscator.deriveKeyStream.
     */
    private function deriveKeyStream(string $entropy, int $length): string
    {
        $stream = '';
        $blockNum = 0;

        while (strlen($stream) < $length) {
            // SHA256(entropy || blockNumber as big-endian uint64)
            $blockBytes = pack('J', $blockNum); // J = unsigned 64-bit big-endian
            $block = hash('sha256', $entropy . $blockBytes, true);

            $copyLen = min(32, $length - strlen($stream));
            $stream .= substr($block, 0, $copyLen);
            $blockNum++;
        }

        return $stream;
    }

    /**
     * Create STMF instance from a KeyPair
     */
    public static function fromKeyPair(KeyPair $keyPair): self
    {
        return new self($keyPair->getPrivateKeyBase64());
    }
}
