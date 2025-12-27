// +build ignore

// This program generates test vectors for PHP interoperability testing.
// Run with: go run generate_test_vectors.go > test_vectors.json
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Snider/Borg/pkg/stmf"
)

type TestVector struct {
	Name           string            `json:"name"`
	PrivateKey     string            `json:"private_key"`
	PublicKey      string            `json:"public_key"`
	EncryptedB64   string            `json:"encrypted_b64"`
	ExpectedFields map[string]string `json:"expected_fields"`
	ExpectedMeta   map[string]string `json:"expected_meta"`
}

func main() {
	var vectors []TestVector

	// Test 1: Simple form with two fields
	{
		kp, _ := stmf.GenerateKeyPair()
		formData := stmf.NewFormData().
			AddField("email", "test@example.com").
			AddFieldWithType("password", "secret123", "password")

		encrypted, _ := stmf.EncryptBase64(formData, kp.PublicKey())

		vectors = append(vectors, TestVector{
			Name:         "simple_form",
			PrivateKey:   kp.PrivateKeyBase64(),
			PublicKey:    kp.PublicKeyBase64(),
			EncryptedB64: encrypted,
			ExpectedFields: map[string]string{
				"email":    "test@example.com",
				"password": "secret123",
			},
			ExpectedMeta: nil,
		})
	}

	// Test 2: Form with metadata
	{
		kp, _ := stmf.GenerateKeyPair()
		formData := stmf.NewFormData().
			AddField("username", "johndoe").
			AddField("action", "login").
			SetMetadata("origin", "https://example.com").
			SetMetadata("timestamp", "1735265000")

		encrypted, _ := stmf.EncryptBase64(formData, kp.PublicKey())

		vectors = append(vectors, TestVector{
			Name:         "form_with_metadata",
			PrivateKey:   kp.PrivateKeyBase64(),
			PublicKey:    kp.PublicKeyBase64(),
			EncryptedB64: encrypted,
			ExpectedFields: map[string]string{
				"username": "johndoe",
				"action":   "login",
			},
			ExpectedMeta: map[string]string{
				"origin":    "https://example.com",
				"timestamp": "1735265000",
			},
		})
	}

	// Test 3: Unicode content
	{
		kp, _ := stmf.GenerateKeyPair()
		formData := stmf.NewFormData().
			AddField("name", "日本語テスト").
			AddField("emoji", "🔐🛡️✅").
			AddField("mixed", "Hello 世界 مرحبا")

		encrypted, _ := stmf.EncryptBase64(formData, kp.PublicKey())

		vectors = append(vectors, TestVector{
			Name:         "unicode_content",
			PrivateKey:   kp.PrivateKeyBase64(),
			PublicKey:    kp.PublicKeyBase64(),
			EncryptedB64: encrypted,
			ExpectedFields: map[string]string{
				"name":  "日本語テスト",
				"emoji": "🔐🛡️✅",
				"mixed": "Hello 世界 مرحبا",
			},
			ExpectedMeta: nil,
		})
	}

	// Test 4: Large form with many fields
	{
		kp, _ := stmf.GenerateKeyPair()
		formData := stmf.NewFormData()
		expectedFields := make(map[string]string)

		for i := 0; i < 20; i++ {
			key := fmt.Sprintf("field_%d", i)
			value := fmt.Sprintf("value_%d_with_some_content", i)
			formData.AddField(key, value)
			expectedFields[key] = value
		}

		encrypted, _ := stmf.EncryptBase64(formData, kp.PublicKey())

		vectors = append(vectors, TestVector{
			Name:           "large_form",
			PrivateKey:     kp.PrivateKeyBase64(),
			PublicKey:      kp.PublicKeyBase64(),
			EncryptedB64:   encrypted,
			ExpectedFields: expectedFields,
			ExpectedMeta:   nil,
		})
	}

	// Test 5: Special characters
	{
		kp, _ := stmf.GenerateKeyPair()
		formData := stmf.NewFormData().
			AddField("sql", "'; DROP TABLE users; --").
			AddField("html", "<script>alert('xss')</script>").
			AddField("json", `{"key": "value", "nested": {"a": 1}}`).
			AddField("newlines", "line1\nline2\nline3")

		encrypted, _ := stmf.EncryptBase64(formData, kp.PublicKey())

		vectors = append(vectors, TestVector{
			Name:         "special_characters",
			PrivateKey:   kp.PrivateKeyBase64(),
			PublicKey:    kp.PublicKeyBase64(),
			EncryptedB64: encrypted,
			ExpectedFields: map[string]string{
				"sql":      "'; DROP TABLE users; --",
				"html":     "<script>alert('xss')</script>",
				"json":     `{"key": "value", "nested": {"a": 1}}`,
				"newlines": "line1\nline2\nline3",
			},
			ExpectedMeta: nil,
		})
	}

	// Output as JSON
	output, err := json.MarshalIndent(vectors, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(output))
}
