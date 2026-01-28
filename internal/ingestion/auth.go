package ingestion

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"time"
)

type Auth struct {
	apiKeyID    string
	privateKey  *rsa.PrivateKey
}

type AuthHeaders struct {
	AccessKey      string
	AccessSignature string
	AccessTimestamp string
}

func NewAuth(apiKeyID, privateKeyPEM string) (*Auth, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return &Auth{
		apiKeyID:   apiKeyID,
		privateKey: privateKey,
	}, nil
}

func (a *Auth) SignRequest(method, path string, body []byte) (*AuthHeaders, error) {
	// Get current timestamp in milliseconds
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())

	// Create the string to sign
	// Format: method + path + timestamp + (body if present)
	stringToSign := method + path + timestamp
	if body != nil {
		stringToSign += string(body)
	}

	// Hash the string to sign
	hasher := sha256.New()
	hasher.Write([]byte(stringToSign))
	hashed := hasher.Sum(nil)

	// Sign with RSA-PSS
	signature, err := rsa.SignPSS(rand.Reader, a.privateKey, crypto.SHA256, hashed, &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}

	// Encode signature as base64
	signatureB64 := base64.StdEncoding.EncodeToString(signature)

	return &AuthHeaders{
		AccessKey:       a.apiKeyID,
		AccessSignature: signatureB64,
		AccessTimestamp: timestamp,
	}, nil
}

