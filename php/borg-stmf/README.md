# Borg STMF for PHP

Sovereign Form Encryption - Decrypt STMF payloads using X25519 + ChaCha20-Poly1305.

## Requirements

- PHP 7.2 or later
- `ext-sodium` (included in PHP 7.2+)
- `ext-json`

## Installation

```bash
composer require borg/stmf
```

## Quick Start

```php
<?php

use Borg\STMF\STMF;

// Initialize with your private key
$stmf = new STMF($privateKeyBase64);

// Decrypt the form payload from POST
$formData = $stmf->decrypt($_POST['_stmf_payload']);

// Access form fields
$email = $formData->get('email');
$password = $formData->get('password');

// Access all fields as array
$allFields = $formData->toArray();

// Access metadata
$origin = $formData->getOrigin();
$timestamp = $formData->getTimestamp();
```

## Laravel Integration

```php
// In a controller
public function handleForm(Request $request)
{
    $stmf = new STMF(config('app.stmf_private_key'));
    $formData = $stmf->decrypt($request->input('_stmf_payload'));

    // Use decrypted data
    $user = User::create([
        'email' => $formData->get('email'),
        'password' => Hash::make($formData->get('password')),
    ]);
}
```

## Key Generation

Generate a keypair in Go:

```go
import "github.com/Snider/Borg/pkg/stmf"

kp, _ := stmf.GenerateKeyPair()
fmt.Println("Public key:", kp.PublicKeyBase64())  // Put in HTML
fmt.Println("Private key:", kp.PrivateKeyBase64()) // Put in PHP config
```

Or generate in PHP (for testing):

```php
use Borg\STMF\KeyPair;

$keypair = KeyPair::generate();
echo "Public: " . $keypair->getPublicKeyBase64() . "\n";
echo "Private: " . $keypair->getPrivateKeyBase64() . "\n";
```

## API Reference

### STMF

```php
// Constructor
$stmf = new STMF(string $privateKeyBase64);

// Decrypt a base64-encoded payload
$formData = $stmf->decrypt(string $payloadBase64): FormData;

// Decrypt raw bytes
$formData = $stmf->decryptRaw(string $payload): FormData;

// Validate without decrypting
$isValid = $stmf->validate(string $payloadBase64): bool;

// Get payload info without decrypting
$info = $stmf->getInfo(string $payloadBase64): array;
```

### FormData

```php
// Get a single field value
$value = $formData->get(string $name): ?string;

// Get a field object (includes type, filename, mime)
$field = $formData->getField(string $name): ?FormField;

// Get all values for a field name
$values = $formData->getAll(string $name): array;

// Check if field exists
$exists = $formData->has(string $name): bool;

// Convert to associative array
$array = $formData->toArray(): array;

// Get all fields
$fields = $formData->fields(): array;

// Get metadata
$meta = $formData->getMetadata(): array;
$origin = $formData->getOrigin(): ?string;
$timestamp = $formData->getTimestamp(): ?int;
```

### FormField

```php
$field->name;     // Field name
$field->value;    // Field value
$field->type;     // Field type (text, password, file, etc.)
$field->filename; // Filename for file uploads
$field->mimeType; // MIME type for file uploads

$field->isFile(): bool;           // Check if this is a file field
$field->getFileContent(): ?string; // Get decoded file content
```

## Security

- **Hybrid encryption**: X25519 ECDH key exchange + ChaCha20-Poly1305
- **Forward secrecy**: Each form submission uses a new ephemeral keypair
- **Authenticated encryption**: Decryption fails if data was tampered with
- **Libsodium**: Uses PHP's built-in sodium extension
