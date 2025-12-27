//go:build js && wasm

// Package main provides the WASM entry point for Borg encryption.
// This module exposes encryption/decryption functions to JavaScript for:
// - STMF: Client-side form encryption using server's public key
// - SMSG: Password-based secure message decryption
package main

import (
	"encoding/base64"
	"encoding/json"
	"syscall/js"

	"github.com/Snider/Borg/pkg/smsg"
	"github.com/Snider/Borg/pkg/stmf"
)

// Version of the WASM module
const Version = "1.1.0"

func main() {
	// Export the BorgSTMF object to JavaScript global scope
	js.Global().Set("BorgSTMF", js.ValueOf(map[string]interface{}{
		"encrypt":         js.FuncOf(encrypt),
		"encryptFields":   js.FuncOf(encryptFields),
		"generateKeyPair": js.FuncOf(generateKeyPair),
		"version":         Version,
		"ready":           true,
	}))

	// Export BorgSMSG for secure message handling
	js.Global().Set("BorgSMSG", js.ValueOf(map[string]interface{}{
		"decrypt":     js.FuncOf(smsgDecrypt),
		"encrypt":     js.FuncOf(smsgEncrypt),
		"getInfo":     js.FuncOf(smsgGetInfo),
		"quickDecrypt": js.FuncOf(smsgQuickDecrypt),
		"version":     Version,
		"ready":       true,
	}))

	// Dispatch a ready event
	dispatchReadyEvent()

	// Keep the WASM module alive
	select {}
}

// dispatchReadyEvent fires a custom event to notify JS that WASM is loaded
func dispatchReadyEvent() {
	event := js.Global().Get("CustomEvent").New("borgstmf:ready", map[string]interface{}{
		"detail": map[string]interface{}{
			"version": Version,
		},
	})
	js.Global().Get("document").Call("dispatchEvent", event)
}

// encrypt encrypts form data using the server's public key.
// JavaScript usage:
//
//	const result = await BorgSTMF.encrypt(formDataJSON, serverPublicKeyBase64);
//	// result is a base64-encoded STMF payload
func encrypt(this js.Value, args []js.Value) interface{} {
	// Return a Promise
	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			if len(args) < 2 {
				reject.Invoke(newError("encrypt requires 2 arguments: formDataJSON, serverPublicKeyBase64"))
				return
			}

			formDataJSON := args[0].String()
			serverPubKeyB64 := args[1].String()

			// Parse form data
			var formData stmf.FormData
			if err := json.Unmarshal([]byte(formDataJSON), &formData); err != nil {
				reject.Invoke(newError("invalid form data JSON: " + err.Error()))
				return
			}

			// Decode server public key
			serverPubKey, err := base64.StdEncoding.DecodeString(serverPubKeyB64)
			if err != nil {
				reject.Invoke(newError("invalid server public key base64: " + err.Error()))
				return
			}

			// Encrypt
			encryptedB64, err := stmf.EncryptBase64(&formData, serverPubKey)
			if err != nil {
				reject.Invoke(newError("encryption failed: " + err.Error()))
				return
			}

			resolve.Invoke(encryptedB64)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// encryptFields encrypts a simple key-value object of form fields.
// JavaScript usage:
//
//	const result = await BorgSTMF.encryptFields({
//	  email: 'user@example.com',
//	  password: 'secret'
//	}, serverPublicKeyBase64, metadata);
func encryptFields(this js.Value, args []js.Value) interface{} {
	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			if len(args) < 2 {
				reject.Invoke(newError("encryptFields requires at least 2 arguments: fields, serverPublicKeyBase64"))
				return
			}

			fieldsObj := args[0]
			serverPubKeyB64 := args[1].String()

			// Build FormData from JavaScript object
			formData := stmf.NewFormData()

			// Get field names
			keys := js.Global().Get("Object").Call("keys", fieldsObj)
			keysLen := keys.Length()

			for i := 0; i < keysLen; i++ {
				key := keys.Index(i).String()
				value := fieldsObj.Get(key)

				// Handle different value types
				if value.Type() == js.TypeString {
					formData.AddField(key, value.String())
				} else if value.Type() == js.TypeObject {
					// Check if it's a file-like object
					if !value.Get("name").IsUndefined() && !value.Get("value").IsUndefined() {
						field := stmf.FormField{
							Name:  key,
							Value: value.Get("value").String(),
						}
						if !value.Get("type").IsUndefined() {
							field.Type = value.Get("type").String()
						}
						if !value.Get("filename").IsUndefined() {
							field.Filename = value.Get("filename").String()
						}
						if !value.Get("mime").IsUndefined() {
							field.MimeType = value.Get("mime").String()
						}
						formData.Fields = append(formData.Fields, field)
					} else {
						// Convert to JSON string
						jsonStr := js.Global().Get("JSON").Call("stringify", value).String()
						formData.AddField(key, jsonStr)
					}
				} else {
					// Convert to string
					formData.AddField(key, value.String())
				}
			}

			// Handle optional metadata argument
			if len(args) >= 3 && args[2].Type() == js.TypeObject {
				metaObj := args[2]
				metaKeys := js.Global().Get("Object").Call("keys", metaObj)
				metaLen := metaKeys.Length()

				for i := 0; i < metaLen; i++ {
					key := metaKeys.Index(i).String()
					value := metaObj.Get(key).String()
					formData.SetMetadata(key, value)
				}
			}

			// Decode server public key
			serverPubKey, err := base64.StdEncoding.DecodeString(serverPubKeyB64)
			if err != nil {
				reject.Invoke(newError("invalid server public key base64: " + err.Error()))
				return
			}

			// Encrypt
			encryptedB64, err := stmf.EncryptBase64(formData, serverPubKey)
			if err != nil {
				reject.Invoke(newError("encryption failed: " + err.Error()))
				return
			}

			resolve.Invoke(encryptedB64)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// generateKeyPair generates a new X25519 keypair for testing/development.
// JavaScript usage:
//
//	const keypair = await BorgSTMF.generateKeyPair();
//	console.log(keypair.publicKey);  // base64 public key
//	console.log(keypair.privateKey); // base64 private key (keep secret!)
func generateKeyPair(this js.Value, args []js.Value) interface{} {
	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			kp, err := stmf.GenerateKeyPair()
			if err != nil {
				reject.Invoke(newError("key generation failed: " + err.Error()))
				return
			}

			result := map[string]interface{}{
				"publicKey":  kp.PublicKeyBase64(),
				"privateKey": kp.PrivateKeyBase64(),
			}

			resolve.Invoke(js.ValueOf(result))
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// newError creates a JavaScript Error object
func newError(message string) js.Value {
	return js.Global().Get("Error").New(message)
}

// smsgDecrypt decrypts a base64-encoded SMSG with a password.
// JavaScript usage:
//
//	const message = await BorgSMSG.decrypt(encryptedBase64, password);
//	console.log(message.body);
//	console.log(message.subject);
//	console.log(message.attachments);
func smsgDecrypt(this js.Value, args []js.Value) interface{} {
	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			if len(args) < 2 {
				reject.Invoke(newError("decrypt requires 2 arguments: encryptedBase64, password"))
				return
			}

			encryptedB64 := args[0].String()
			password := args[1].String()

			msg, err := smsg.DecryptBase64(encryptedB64, password)
			if err != nil {
				reject.Invoke(newError("decryption failed: " + err.Error()))
				return
			}

			// Convert message to JS object
			result := messageToJS(msg)
			resolve.Invoke(result)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// smsgEncrypt encrypts a message with a password.
// JavaScript usage:
//
//	const encrypted = await BorgSMSG.encrypt({
//	  body: 'Hello!',
//	  subject: 'Test',
//	  from: 'support@example.com'
//	}, password);
func smsgEncrypt(this js.Value, args []js.Value) interface{} {
	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			if len(args) < 2 {
				reject.Invoke(newError("encrypt requires 2 arguments: messageObject, password"))
				return
			}

			msgObj := args[0]
			password := args[1].String()

			// Build message from JS object
			msg := smsg.NewMessage(msgObj.Get("body").String())

			if !msgObj.Get("subject").IsUndefined() {
				msg.WithSubject(msgObj.Get("subject").String())
			}
			if !msgObj.Get("from").IsUndefined() {
				msg.WithFrom(msgObj.Get("from").String())
			}

			// Handle attachments
			attachments := msgObj.Get("attachments")
			if !attachments.IsUndefined() && attachments.Length() > 0 {
				for i := 0; i < attachments.Length(); i++ {
					att := attachments.Index(i)
					name := att.Get("name").String()
					content := att.Get("content").String()
					mimeType := ""
					if !att.Get("mime").IsUndefined() {
						mimeType = att.Get("mime").String()
					}
					msg.AddAttachment(name, content, mimeType)
				}
			}

			// Handle reply key
			replyKey := msgObj.Get("replyKey")
			if !replyKey.IsUndefined() {
				msg.WithReplyKey(replyKey.Get("publicKey").String())
			}

			// Handle metadata
			meta := msgObj.Get("meta")
			if !meta.IsUndefined() && meta.Type() == js.TypeObject {
				keys := js.Global().Get("Object").Call("keys", meta)
				for i := 0; i < keys.Length(); i++ {
					key := keys.Index(i).String()
					value := meta.Get(key).String()
					msg.SetMeta(key, value)
				}
			}

			// Get optional hint
			hint := ""
			if len(args) >= 3 && args[2].Type() == js.TypeString {
				hint = args[2].String()
			}

			var encrypted []byte
			var err error
			if hint != "" {
				encrypted, err = smsg.EncryptWithHint(msg, password, hint)
			} else {
				encrypted, err = smsg.Encrypt(msg, password)
			}

			if err != nil {
				reject.Invoke(newError("encryption failed: " + err.Error()))
				return
			}

			encryptedB64 := base64.StdEncoding.EncodeToString(encrypted)
			resolve.Invoke(encryptedB64)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// smsgGetInfo extracts header info from an SMSG without decrypting.
// JavaScript usage:
//
//	const info = await BorgSMSG.getInfo(encryptedBase64);
//	console.log(info.hint);  // password hint if set
//	console.log(info.version);
func smsgGetInfo(this js.Value, args []js.Value) interface{} {
	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			if len(args) < 1 {
				reject.Invoke(newError("getInfo requires 1 argument: encryptedBase64"))
				return
			}

			encryptedB64 := args[0].String()

			header, err := smsg.GetInfoBase64(encryptedB64)
			if err != nil {
				reject.Invoke(newError("failed to get info: " + err.Error()))
				return
			}

			result := map[string]interface{}{
				"version":   header.Version,
				"algorithm": header.Algorithm,
			}
			if header.Hint != "" {
				result["hint"] = header.Hint
			}

			resolve.Invoke(js.ValueOf(result))
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// smsgQuickDecrypt is a convenience function that just returns the body text.
// JavaScript usage:
//
//	const body = await BorgSMSG.quickDecrypt(encryptedBase64, password);
func smsgQuickDecrypt(this js.Value, args []js.Value) interface{} {
	handler := js.FuncOf(func(this js.Value, promiseArgs []js.Value) interface{} {
		resolve := promiseArgs[0]
		reject := promiseArgs[1]

		go func() {
			if len(args) < 2 {
				reject.Invoke(newError("quickDecrypt requires 2 arguments: encryptedBase64, password"))
				return
			}

			encryptedB64 := args[0].String()
			password := args[1].String()

			body, err := smsg.QuickDecrypt(encryptedB64, password)
			if err != nil {
				reject.Invoke(newError("decryption failed: " + err.Error()))
				return
			}

			resolve.Invoke(body)
		}()

		return nil
	})

	promiseConstructor := js.Global().Get("Promise")
	return promiseConstructor.New(handler)
}

// messageToJS converts an smsg.Message to a JavaScript object
func messageToJS(msg *smsg.Message) js.Value {
	result := map[string]interface{}{
		"body":      msg.Body,
		"timestamp": msg.Timestamp,
	}

	if msg.Subject != "" {
		result["subject"] = msg.Subject
	}
	if msg.From != "" {
		result["from"] = msg.From
	}

	// Convert attachments
	if len(msg.Attachments) > 0 {
		attachments := make([]interface{}, len(msg.Attachments))
		for i, att := range msg.Attachments {
			attachments[i] = map[string]interface{}{
				"name":    att.Name,
				"content": att.Content,
				"mime":    att.MimeType,
				"size":    att.Size,
			}
		}
		result["attachments"] = attachments
	}

	// Convert reply key
	if msg.ReplyKey != nil {
		result["replyKey"] = map[string]interface{}{
			"publicKey":   msg.ReplyKey.PublicKey,
			"keyId":       msg.ReplyKey.KeyID,
			"algorithm":   msg.ReplyKey.Algorithm,
			"fingerprint": msg.ReplyKey.Fingerprint,
		}
	}

	// Convert metadata
	if len(msg.Meta) > 0 {
		meta := make(map[string]interface{})
		for k, v := range msg.Meta {
			meta[k] = v
		}
		result["meta"] = meta
	}

	return js.ValueOf(result)
}
