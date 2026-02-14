package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
)

// generateED25519Key creates a new ED25519 keypair.
func generateED25519Key() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}

// savePrivateKey writes an ED25519 private key to disk in PEM format.
func savePrivateKey(path string, key ed25519.PrivateKey) error {
	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	// Marshal the key to PKCS8
	marshaled, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return err
	}

	pemBlock := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: marshaled,
	}

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return pem.Encode(f, pemBlock)
}
