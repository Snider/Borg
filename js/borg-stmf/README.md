# @borg/stmf

Sovereign Form Encryption - Client-side form encryption using X25519 + ChaCha20-Poly1305.

## Overview

BorgSTMF encrypts HTML form data in the browser before submission, using the server's public key. Even if a MITM proxy intercepts the request, they only see encrypted data.

## Installation

```bash
npm install @borg/stmf
```

## Quick Start

```html
<!-- Load the WASM support -->
<script src="wasm_exec.js"></script>

<!-- Your form -->
<form id="login" action="/api/login" method="POST" data-stmf="YOUR_PUBLIC_KEY_BASE64">
  <input name="email" type="email" required>
  <input name="password" type="password" required>
  <button type="submit">Login</button>
</form>

<script type="module">
import { BorgSTMF } from '@borg/stmf';

const borg = new BorgSTMF({
  serverPublicKey: 'YOUR_PUBLIC_KEY_BASE64',
  wasmPath: '/wasm/stmf.wasm'
});

await borg.init();
borg.enableInterceptor();
</script>
```

## Manual Encryption

```typescript
import { BorgSTMF } from '@borg/stmf';

const borg = new BorgSTMF({
  serverPublicKey: 'YOUR_PUBLIC_KEY_BASE64'
});

await borg.init();

// Encrypt form element
const form = document.querySelector('form');
const result = await borg.encryptForm(form);
console.log(result.payload); // Base64 encrypted STMF

// Or encrypt key-value pairs directly
const result = await borg.encryptFields({
  email: 'user@example.com',
  password: 'secret'
});
```

## Server-Side Decryption

### Go Middleware

```go
import "github.com/Snider/Borg/pkg/stmf/middleware"

privateKey := os.Getenv("STMF_PRIVATE_KEY")
handler := middleware.Simple(privateKeyBytes)(yourHandler)

// In your handler, form values are automatically decrypted:
email := r.FormValue("email")
```

### PHP

```php
use Borg\STMF\STMF;

$stmf = new STMF($privateKeyBase64);
$formData = $stmf->decrypt($_POST['_stmf_payload']);

$email = $formData->get('email');
```

## Key Generation

Generate a keypair for your server:

```go
import "github.com/Snider/Borg/pkg/stmf"

kp, _ := stmf.GenerateKeyPair()
fmt.Println("Public key:", kp.PublicKeyBase64())  // Share this
fmt.Println("Private key:", kp.PrivateKeyBase64()) // Keep secret!
```

## Security

- **Hybrid encryption**: X25519 ECDH key exchange + ChaCha20-Poly1305
- **Forward secrecy**: Each form submission uses a new ephemeral keypair
- **Authenticated encryption**: Data integrity is verified on decryption
- **No passwords transmitted**: Only the public key is in the HTML
