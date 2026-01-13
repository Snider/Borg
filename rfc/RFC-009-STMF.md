# RFC-009: STMF Secure To-Me Form

**Status**: Draft
**Author**: [Snider](https://github.com/Snider/)
**Created**: 2026-01-13
**License**: EUPL-1.2

---

## Abstract

STMF (Secure To-Me Form) provides asymmetric encryption for web form submissions. It enables end-to-end encrypted form data where only the recipient can decrypt submissions, protecting sensitive data from server compromise.

## 1. Overview

STMF provides:
- Asymmetric encryption for form data
- X25519 key exchange
- ChaCha20-Poly1305 for payload encryption
- Browser-based encryption via WASM
- HTTP middleware for server-side decryption

## 2. Cryptographic Primitives

### 2.1 Key Exchange

X25519 (Curve25519 Diffie-Hellman)

| Parameter | Value |
|-----------|-------|
| Private key | 32 bytes |
| Public key | 32 bytes |
| Shared secret | 32 bytes |

### 2.2 Encryption

ChaCha20-Poly1305

| Parameter | Value |
|-----------|-------|
| Key | 32 bytes (SHA-256 of shared secret) |
| Nonce | 24 bytes (XChaCha variant) |
| Tag | 16 bytes |

## 3. Protocol

### 3.1 Setup (One-time)

```
Recipient (Server):
1. Generate X25519 keypair
2. Publish public key (embed in page or API)
3. Store private key securely
```

### 3.2 Encryption Flow (Browser)

```
1. Fetch recipient's public key
2. Generate ephemeral X25519 keypair
3. Compute shared secret: X25519(ephemeral_private, recipient_public)
4. Derive encryption key: SHA256(shared_secret)
5. Encrypt form data: ChaCha20-Poly1305(data, key, random_nonce)
6. Send: {ephemeral_public, nonce, ciphertext}
```

### 3.3 Decryption Flow (Server)

```
1. Receive {ephemeral_public, nonce, ciphertext}
2. Compute shared secret: X25519(recipient_private, ephemeral_public)
3. Derive encryption key: SHA256(shared_secret)
4. Decrypt: ChaCha20-Poly1305_Open(ciphertext, key, nonce)
```

## 4. Wire Format

### 4.1 Container (Trix-based)

```
[Magic: "STMF" (4 bytes)]
[Header: Gob-encoded JSON]
[Payload: ChaCha20-Poly1305 ciphertext]
```

### 4.2 Header Structure

```json
{
  "version": "1.0",
  "algorithm": "x25519-chacha20poly1305",
  "ephemeral_pk": "<base64 32-byte ephemeral public key>"
}
```

### 4.3 Transmission

- Default form field: `_stmf_payload`
- Encoding: Base64 string
- Content-Type: `application/x-www-form-urlencoded` or `multipart/form-data`

## 5. Data Structures

### 5.1 FormField

```go
type FormField struct {
    Name     string  // Field name
    Value    string  // Base64 for files, plaintext otherwise
    Type     string  // "text", "password", "file"
    Filename string  // For file uploads
    MimeType string  // For file uploads
}
```

### 5.2 FormData

```go
type FormData struct {
    Fields   []FormField           // Array of form fields
    Metadata map[string]string     // Arbitrary key-value metadata
}
```

### 5.3 Builder Pattern

```go
formData := NewFormData().
    AddField("email", "user@example.com").
    AddFieldWithType("password", "secret", "password").
    AddFile("document", base64Content, "report.pdf", "application/pdf").
    SetMetadata("timestamp", time.Now().String())
```

## 6. Key Management API

### 6.1 Key Generation

```go
// pkg/stmf/keypair.go
func GenerateKeyPair() (*KeyPair, error)

type KeyPair struct {
    privateKey *ecdh.PrivateKey
    publicKey  *ecdh.PublicKey
}
```

### 6.2 Key Loading

```go
// From raw bytes
func LoadPublicKey(data []byte) (*ecdh.PublicKey, error)
func LoadPrivateKey(data []byte) (*ecdh.PrivateKey, error)

// From base64
func LoadPublicKeyBase64(encoded string) (*ecdh.PublicKey, error)
func LoadPrivateKeyBase64(encoded string) (*ecdh.PrivateKey, error)

// Reconstruct keypair from private key
func LoadKeyPair(privateKeyBytes []byte) (*KeyPair, error)
```

### 6.3 Key Export

```go
func (kp *KeyPair) PublicKey() []byte        // Raw 32 bytes
func (kp *KeyPair) PrivateKey() []byte       // Raw 32 bytes
func (kp *KeyPair) PublicKeyBase64() string  // Base64 encoded
func (kp *KeyPair) PrivateKeyBase64() string // Base64 encoded
```

## 7. WASM API

### 7.1 BorgSTMF Namespace

```javascript
// Generate X25519 keypair
const keypair = await BorgSTMF.generateKeyPair();
// keypair.publicKey: base64 string
// keypair.privateKey: base64 string

// Encrypt form data
const encrypted = await BorgSTMF.encrypt(
    JSON.stringify(formData),
    serverPublicKeyBase64
);

// Encrypt with field-level control
const encrypted = await BorgSTMF.encryptFields(
    {email: "user@example.com", password: "secret"},
    serverPublicKeyBase64,
    {timestamp: Date.now().toString()}  // Optional metadata
);
```

## 8. HTTP Middleware

### 8.1 Simple Usage

```go
import "github.com/Snider/Borg/pkg/stmf/middleware"

// Create middleware with private key
mw := middleware.Simple(privateKeyBytes)

// Or from base64
mw, err := middleware.SimpleBase64(privateKeyB64)

// Apply to handler
http.Handle("/submit", mw(myHandler))
```

### 8.2 Advanced Configuration

```go
cfg := middleware.DefaultConfig(privateKeyBytes)
cfg.FieldName = "_custom_field"        // Custom field name (default: _stmf_payload)
cfg.PopulateForm = &true               // Auto-populate r.Form
cfg.OnError = customErrorHandler       // Custom error handling
cfg.OnMissingPayload = customHandler   // When field is absent

mw := middleware.Middleware(cfg)
```

### 8.3 Context Access

```go
func myHandler(w http.ResponseWriter, r *http.Request) {
    // Get decrypted form data
    formData := middleware.GetFormData(r)

    // Get metadata
    metadata := middleware.GetMetadata(r)

    // Access fields
    email := formData.Get("email")
    password := formData.Get("password")
}
```

### 8.4 Middleware Behavior

- Handles POST, PUT, PATCH requests only
- Parses multipart/form-data (32 MB limit) or application/x-www-form-urlencoded
- Looks for field `_stmf_payload` (configurable)
- Base64 decodes, then decrypts
- Populates `r.Form` and `r.PostForm` with decrypted fields
- Returns 400 Bad Request on decryption failure

## 9. Integration Example

### 9.1 HTML Form

```html
<form id="secure-form" data-stmf-pubkey="<base64-public-key>">
    <input name="name" type="text">
    <input name="email" type="email">
    <input name="ssn" type="password">
    <button type="submit">Send Securely</button>
</form>

<script>
document.getElementById('secure-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target;
    const pubkey = form.dataset.stmfPubkey;

    const formData = new FormData(form);
    const data = Object.fromEntries(formData);

    const encrypted = await BorgSTMF.encrypt(JSON.stringify(data), pubkey);

    await fetch('/api/submit', {
        method: 'POST',
        body: new URLSearchParams({_stmf_payload: encrypted}),
        headers: {'Content-Type': 'application/x-www-form-urlencoded'}
    });
});
</script>
```

### 9.2 Server Handler

```go
func main() {
    privateKey, _ := os.ReadFile("private.key")
    mw := middleware.Simple(privateKey)

    http.Handle("/api/submit", mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        formData := middleware.GetFormData(r)

        name := formData.Get("name")
        email := formData.Get("email")
        ssn := formData.Get("ssn")

        // Process securely...
        w.WriteHeader(http.StatusOK)
    })))

    http.ListenAndServeTLS(":443", "cert.pem", "key.pem", nil)
}
```

## 10. Security Properties

### 10.1 Forward Secrecy

- Fresh ephemeral keypair per encryption
- Compromised private key doesn't decrypt past messages
- Each ciphertext has unique shared secret

### 10.2 Authenticity

- Poly1305 MAC prevents tampering
- Decryption fails if ciphertext modified

### 10.3 Confidentiality

- ChaCha20 provides 256-bit security
- Nonces are random (24 bytes), collision unlikely
- Data encrypted before leaving browser

### 10.4 Key Isolation

- Private key never exposed to browser/JavaScript
- Public key can be safely distributed
- Ephemeral keys discarded after encryption

## 11. Error Handling

```go
var (
    ErrInvalidMagic        = errors.New("invalid STMF magic")
    ErrInvalidPayload      = errors.New("invalid STMF payload")
    ErrDecryptionFailed    = errors.New("decryption failed")
    ErrInvalidPublicKey    = errors.New("invalid public key")
    ErrInvalidPrivateKey   = errors.New("invalid private key")
    ErrKeyGenerationFailed = errors.New("key generation failed")
)
```

## 12. Implementation Reference

- Types: `pkg/stmf/types.go`
- Key management: `pkg/stmf/keypair.go`
- Encryption: `pkg/stmf/encrypt.go`
- Decryption: `pkg/stmf/decrypt.go`
- Middleware: `pkg/stmf/middleware/http.go`
- WASM: `pkg/wasm/stmf/main.go`

## 13. Security Considerations

1. **Public key authenticity**: Verify public key source (HTTPS, pinning)
2. **Private key protection**: Never expose to browser, store securely
3. **Nonce uniqueness**: Random generation ensures uniqueness
4. **HTTPS required**: Transport layer must be encrypted

## 14. Future Work

- [ ] Multiple recipients
- [ ] Key attestation
- [ ] Offline decryption app
- [ ] Hardware key support (WebAuthn)
- [ ] Key rotation support
