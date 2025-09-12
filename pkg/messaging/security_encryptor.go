// Copyright © 2025 jackelyj <dreamerlyj@gmail.com>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.
//

package messaging

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"hash"
	"io"
	"math/big"
	"sync"
	"time"

	"go.uber.org/zap"
)

// SecureMessage represents a message with security transformations applied
type SecureMessage struct {
	// Original message
	OriginalMessage *Message

	// Security metadata
	SecurityMetadata *SecurityMetadata

	// Digital signature
	Signature string

	// Encrypted payload (if applicable)
	EncryptedPayload []byte
}

// SecurityMetadata contains security-related metadata for a message
type SecurityMetadata struct {
	// Name of the security policy used
	PolicyName string `json:"policy_name"`

	// Encryption algorithm used
	EncryptionAlgo string `json:"encryption_algo,omitempty"`

	// Signing algorithm used
	SigningAlgo string `json:"signing_algo,omitempty"`

	// Key ID used for signing/encryption
	KeyID string `json:"key_id,omitempty"`

	// Timestamp when security was applied
	Timestamp time.Time `json:"timestamp"`

	// Compression flag
	CompressionUsed bool `json:"compression_used"`

	// Additional security metadata
	AdditionalMetadata map[string]string `json:"additional_metadata,omitempty"`
}

// MessageEncryptor handles message encryption operations
type MessageEncryptor struct {
	config   *EncryptionConfig
	keyStore KeyStore
	cache    *EncryptionCache
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewMessageEncryptor creates a new message encryptor
func NewMessageEncryptor(config *EncryptionConfig) *MessageEncryptor {
	if config == nil {
		config = &EncryptionConfig{Enabled: false}
	}

	keyStore := NewInMemoryKeyStore()
	cache := NewEncryptionCache()

	return &MessageEncryptor{
		config:   config,
		keyStore: keyStore,
		cache:    cache,
		logger:   zap.NewNop(),
	}
}

// Encrypt encrypts message payload according to the security policy
func (e *MessageEncryptor) Encrypt(ctx context.Context, payload []byte, policy *SecurityPolicy) ([]byte, error) {
	if !policy.Encryption.Enabled {
		return payload, nil
	}

	// Get encryption key
	key, err := e.getEncryptionKey(ctx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to get encryption key: %w", err)
	}

	// Check cache first
	cacheKey := e.generateCacheKey(payload, key, policy)
	if cached, found := e.cache.Get(cacheKey); found {
		return cached, nil
	}

	// Encrypt based on algorithm
	var encrypted []byte
	switch policy.Encryption.Algorithm {
	case EncryptionAlgorithmA128GCM, EncryptionAlgorithmA192GCM, EncryptionAlgorithmA256GCM:
		encrypted, err = e.encryptAESGCM(payload, key, policy)
	case EncryptionAlgorithmA256CBC:
		encrypted, err = e.encryptAESCBC(payload, key, policy)
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", policy.Encryption.Algorithm)
	}

	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}

	// Cache result
	e.cache.Set(cacheKey, encrypted)

	return encrypted, nil
}

// Decrypt decrypts message payload according to the security policy
func (e *MessageEncryptor) Decrypt(ctx context.Context, encrypted []byte, policy *SecurityPolicy) ([]byte, error) {
	if !policy.Encryption.Enabled {
		return encrypted, nil
	}

	// Get decryption key
	key, err := e.getEncryptionKey(ctx, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to get decryption key: %w", err)
	}

	// Decrypt based on algorithm
	var decrypted []byte
	switch policy.Encryption.Algorithm {
	case EncryptionAlgorithmA128GCM, EncryptionAlgorithmA192GCM, EncryptionAlgorithmA256GCM:
		decrypted, err = e.decryptAESGCM(encrypted, key, policy)
	case EncryptionAlgorithmA256CBC:
		decrypted, err = e.decryptAESCBC(encrypted, key, policy)
	default:
		return nil, fmt.Errorf("unsupported encryption algorithm: %s", policy.Encryption.Algorithm)
	}

	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	return decrypted, nil
}

// getEncryptionKey retrieves or generates an encryption key
func (e *MessageEncryptor) getEncryptionKey(ctx context.Context, policy *SecurityPolicy) ([]byte, error) {
	keyName := fmt.Sprintf("%s_encryption_key", policy.Name)

	// Try to get from key store
	if key, err := e.keyStore.GetKey(ctx, keyName); err == nil {
		return key, nil
	}

	// Generate new key
	key, err := e.generateEncryptionKey(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	// Store in key store
	if err := e.keyStore.SetKey(ctx, keyName, key); err != nil {
		e.logger.Warn("Failed to store encryption key",
			zap.String("key_name", keyName),
			zap.Error(err))
	}

	return key, nil
}

// generateEncryptionKey generates a new encryption key
func (e *MessageEncryptor) generateEncryptionKey(policy *SecurityPolicy) ([]byte, error) {
	keySize := policy.Encryption.KeySize
	if keySize == 0 {
		keySize = 256 // Default to 256 bits
	}

	key := make([]byte, keySize/8)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate encryption key: %w", err)
	}

	return key, nil
}

// encryptAESGCM encrypts data using AES-GCM
func (e *MessageEncryptor) encryptAESGCM(plaintext, key []byte, policy *SecurityPolicy) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	return ciphertext, nil
}

// decryptAESGCM decrypts data using AES-GCM
func (e *MessageEncryptor) decryptAESGCM(ciphertext, key []byte, policy *SecurityPolicy) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// encryptAESCBC encrypts data using AES-CBC
func (e *MessageEncryptor) encryptAESCBC(plaintext, key []byte, policy *SecurityPolicy) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Generate IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	// Pad plaintext to block size
	paddedPlaintext := e.pkcs7Pad(plaintext, aes.BlockSize)

	// Encrypt
	ciphertext := make([]byte, len(paddedPlaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedPlaintext)

	// Prepend IV to ciphertext
	return append(iv, ciphertext...), nil
}

// decryptAESCBC decrypts data using AES-CBC
func (e *MessageEncryptor) decryptAESCBC(ciphertext, key []byte, policy *SecurityPolicy) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// Extract IV
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// Decrypt
	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)

	// Remove padding
	return e.pkcs7Unpad(plaintext, aes.BlockSize)
}

// pkcs7Pad applies PKCS7 padding
func (e *MessageEncryptor) pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7Unpad removes PKCS7 padding
func (e *MessageEncryptor) pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}
	if len(data)%blockSize != 0 {
		return nil, fmt.Errorf("data is not block-aligned")
	}

	padding := int(data[len(data)-1])
	if padding < 1 || padding > blockSize {
		return nil, fmt.Errorf("invalid padding")
	}

	// Check padding
	for i := len(data) - padding; i < len(data); i++ {
		if int(data[i]) != padding {
			return nil, fmt.Errorf("invalid padding")
		}
	}

	return data[:len(data)-padding], nil
}

// generateCacheKey generates a cache key for encryption results
func (e *MessageEncryptor) generateCacheKey(payload []byte, key []byte, policy *SecurityPolicy) string {
	hash := sha256.Sum256(append(payload, key...))
	return fmt.Sprintf("%s_%x", policy.Encryption.Algorithm, hash[:16])
}

// SetLogger sets the logger for the encryptor
func (e *MessageEncryptor) SetLogger(logger *zap.Logger) {
	e.logger = logger
}

// MessageSigner handles message signing operations
type MessageSigner struct {
	config   *SigningConfig
	keyStore KeyStore
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewMessageSigner creates a new message signer
func NewMessageSigner(config *SigningConfig) *MessageSigner {
	if config == nil {
		config = &SigningConfig{Enabled: false}
	}

	keyStore := NewInMemoryKeyStore()

	return &MessageSigner{
		config:   config,
		keyStore: keyStore,
		logger:   zap.NewNop(),
	}
}

// Sign creates a digital signature for a message
func (s *MessageSigner) Sign(ctx context.Context, message *Message, policy *SecurityPolicy) (string, error) {
	if !policy.Signing.Enabled {
		return "", nil
	}

	// Get signing key
	key, err := s.getSigningKey(ctx, policy)
	if err != nil {
		return "", fmt.Errorf("failed to get signing key: %w", err)
	}

	// Create signature data
	signatureData := s.createSignatureData(message, policy.Signing)

	// Sign based on algorithm
	var signature string
	switch policy.Signing.Algorithm {
	case SigningAlgorithmRS256, SigningAlgorithmRS384, SigningAlgorithmRS512:
		signature, err = s.signRSA(signatureData, key, policy)
	case SigningAlgorithmES256, SigningAlgorithmES384, SigningAlgorithmES512:
		signature, err = s.signECDSA(signatureData, key, policy)
	case SigningAlgorithmHS256, SigningAlgorithmHS384, SigningAlgorithmHS512:
		signature, err = s.signHMAC(signatureData, key, policy)
	default:
		return "", fmt.Errorf("unsupported signing algorithm: %s", policy.Signing.Algorithm)
	}

	if err != nil {
		return "", fmt.Errorf("signing failed: %w", err)
	}

	return signature, nil
}

// Verify verifies a digital signature for a message
func (s *MessageSigner) Verify(ctx context.Context, secureMessage *SecureMessage, policy *SecurityPolicy) error {
	if !policy.Signing.Enabled || secureMessage.Signature == "" {
		return nil
	}

	// Get verification key
	key, err := s.getSigningKey(ctx, policy)
	if err != nil {
		return fmt.Errorf("failed to get verification key: %w", err)
	}

	// Create signature data
	signatureData := s.createSignatureData(secureMessage.OriginalMessage, policy.Signing)

	// Verify based on algorithm
	switch policy.Signing.Algorithm {
	case SigningAlgorithmRS256, SigningAlgorithmRS384, SigningAlgorithmRS512:
		return s.verifyRSA(signatureData, secureMessage.Signature, key, policy)
	case SigningAlgorithmES256, SigningAlgorithmES384, SigningAlgorithmES512:
		return s.verifyECDSA(signatureData, secureMessage.Signature, key, policy)
	case SigningAlgorithmHS256, SigningAlgorithmHS384, SigningAlgorithmHS512:
		return s.verifyHMAC(signatureData, secureMessage.Signature, key, policy)
	default:
		return fmt.Errorf("unsupported signing algorithm: %s", policy.Signing.Algorithm)
	}
}

// getSigningKey retrieves or generates a signing key
func (s *MessageSigner) getSigningKey(ctx context.Context, policy *SecurityPolicy) (interface{}, error) {
	keyName := fmt.Sprintf("%s_signing_key", policy.Name)

	// Try to get from key store
	if key, err := s.keyStore.GetKey(ctx, keyName); err == nil {
		return s.deserializeKey(key, policy.Signing.Algorithm)
	}

	// Generate new key
	key, err := s.generateSigningKey(policy)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signing key: %w", err)
	}

	// Serialize and store in key store
	serializedKey, err := s.serializeKey(key, policy.Signing.Algorithm)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize signing key: %w", err)
	}

	if err := s.keyStore.SetKey(ctx, keyName, serializedKey); err != nil {
		s.logger.Warn("Failed to store signing key",
			zap.String("key_name", keyName),
			zap.Error(err))
	}

	return key, nil
}

// generateSigningKey generates a new signing key
func (s *MessageSigner) generateSigningKey(policy *SecurityPolicy) (interface{}, error) {
	switch policy.Signing.Algorithm {
	case SigningAlgorithmRS256, SigningAlgorithmRS384, SigningAlgorithmRS512:
		return rsa.GenerateKey(rand.Reader, 2048)
	case SigningAlgorithmES256, SigningAlgorithmES384, SigningAlgorithmES512:
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case SigningAlgorithmHS256, SigningAlgorithmHS384, SigningAlgorithmHS512:
		key := make([]byte, 64) // 512 bits for HS512
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			return nil, fmt.Errorf("failed to generate HMAC key: %w", err)
		}
		return key, nil
	default:
		return nil, fmt.Errorf("unsupported signing algorithm: %s", policy.Signing.Algorithm)
	}
}

// createSignatureData creates data to be signed
func (s *MessageSigner) createSignatureData(message *Message, policy *SigningConfig) []byte {
	data := []byte(message.ID + message.Headers["message_type"] + string(message.Payload))

	if policy.IncludeTimestamp {
		data = append(data, []byte(time.Now().Format(time.RFC3339Nano))...)
	}

	if policy.IncludeMessageID {
		data = append(data, []byte(message.ID)...)
	}

	return data
}

// signRSA signs data using RSA
func (s *MessageSigner) signRSA(data []byte, key interface{}, policy *SecurityPolicy) (string, error) {
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("invalid RSA key type")
	}

	var hash crypto.Hash
	switch policy.Signing.Algorithm {
	case SigningAlgorithmRS256:
		hash = crypto.SHA256
	case SigningAlgorithmRS384:
		hash = crypto.SHA384
	case SigningAlgorithmRS512:
		hash = crypto.SHA512
	default:
		return "", fmt.Errorf("unsupported RSA signing algorithm: %s", policy.Signing.Algorithm)
	}

	hashed := hash.New()
	hashed.Write(data)
	hashedData := hashed.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, rsaKey, hash, hashedData)
	if err != nil {
		return "", fmt.Errorf("RSA signing failed: %w", err)
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// verifyRSA verifies RSA signature
func (s *MessageSigner) verifyRSA(data []byte, signature string, key interface{}, policy *SecurityPolicy) error {
	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("invalid RSA public key type")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	var hash crypto.Hash
	switch policy.Signing.Algorithm {
	case SigningAlgorithmRS256:
		hash = crypto.SHA256
	case SigningAlgorithmRS384:
		hash = crypto.SHA384
	case SigningAlgorithmRS512:
		hash = crypto.SHA512
	default:
		return fmt.Errorf("unsupported RSA signing algorithm: %s", policy.Signing.Algorithm)
	}

	hashed := hash.New()
	hashed.Write(data)
	hashedData := hashed.Sum(nil)

	return rsa.VerifyPKCS1v15(rsaKey, hash, hashedData, sigBytes)
}

// signECDSA signs data using ECDSA
func (s *MessageSigner) signECDSA(data []byte, key interface{}, policy *SecurityPolicy) (string, error) {
	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("invalid ECDSA key type")
	}

	var hash crypto.Hash
	switch policy.Signing.Algorithm {
	case SigningAlgorithmES256:
		hash = crypto.SHA256
	case SigningAlgorithmES384:
		hash = crypto.SHA384
	case SigningAlgorithmES512:
		hash = crypto.SHA512
	default:
		return "", fmt.Errorf("unsupported ECDSA signing algorithm: %s", policy.Signing.Algorithm)
	}

	hashed := hash.New()
	hashed.Write(data)
	hashedData := hashed.Sum(nil)

	rSig, sSig, err := ecdsa.Sign(rand.Reader, ecdsaKey, hashedData)
	if err != nil {
		return "", fmt.Errorf("ECDSA signing failed: %w", err)
	}

	// Encode signature as concatenation of r and s
	rBytes, sSigBytes := rSig.Bytes(), sSig.Bytes()
	sigDER := append(rBytes, sSigBytes...)
	return base64.StdEncoding.EncodeToString(sigDER), nil
}

// verifyECDSA verifies ECDSA signature
func (s *MessageSigner) verifyECDSA(data []byte, signature string, key interface{}, policy *SecurityPolicy) error {
	ecdsaKey, ok := key.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("invalid ECDSA public key type")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	var hash crypto.Hash
	switch policy.Signing.Algorithm {
	case SigningAlgorithmES256:
		hash = crypto.SHA256
	case SigningAlgorithmES384:
		hash = crypto.SHA384
	case SigningAlgorithmES512:
		hash = crypto.SHA512
	default:
		return fmt.Errorf("unsupported ECDSA signing algorithm: %s", policy.Signing.Algorithm)
	}

	hashed := hash.New()
	hashed.Write(data)
	hashedData := hashed.Sum(nil)

	// Parse DER signature
	if len(sigBytes) != 64 { // Assuming raw signature format
		return fmt.Errorf("invalid ECDSA signature length")
	}

	rSig := new(big.Int).SetBytes(sigBytes[:32])
	sSig := new(big.Int).SetBytes(sigBytes[32:])

	if !ecdsa.Verify(ecdsaKey, hashedData, rSig, sSig) {
		return fmt.Errorf("ECDSA signature verification failed")
	}

	return nil
}

// signHMAC signs data using HMAC
func (s *MessageSigner) signHMAC(data []byte, key interface{}, policy *SecurityPolicy) (string, error) {
	hmacKey, ok := key.([]byte)
	if !ok {
		return "", fmt.Errorf("invalid HMAC key type")
	}

	var hashFunc func() hash.Hash
	switch policy.Signing.Algorithm {
	case SigningAlgorithmHS256:
		hashFunc = sha256.New
	case SigningAlgorithmHS384:
		hashFunc = sha512.New384
	case SigningAlgorithmHS512:
		hashFunc = sha512.New
	default:
		return "", fmt.Errorf("unsupported HMAC signing algorithm: %s", policy.Signing.Algorithm)
	}

	hmacHash := hmac.New(hashFunc, hmacKey)
	hmacHash.Write(data)
	signature := hmacHash.Sum(nil)

	return base64.StdEncoding.EncodeToString(signature), nil
}

// verifyHMAC verifies HMAC signature
func (s *MessageSigner) verifyHMAC(data []byte, signature string, key interface{}, policy *SecurityPolicy) error {
	hmacKey, ok := key.([]byte)
	if !ok {
		return fmt.Errorf("invalid HMAC key type")
	}

	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	var hashFunc func() hash.Hash
	switch policy.Signing.Algorithm {
	case SigningAlgorithmHS256:
		hashFunc = sha256.New
	case SigningAlgorithmHS384:
		hashFunc = sha512.New384
	case SigningAlgorithmHS512:
		hashFunc = sha512.New
	default:
		return fmt.Errorf("unsupported HMAC signing algorithm: %s", policy.Signing.Algorithm)
	}

	hmacHash := hmac.New(hashFunc, hmacKey)
	hmacHash.Write(data)
	expectedSignature := hmacHash.Sum(nil)

	if !bytes.Equal(sigBytes, expectedSignature) {
		return fmt.Errorf("HMAC signature verification failed")
	}

	return nil
}

// serializeKey serializes a key for storage
func (s *MessageSigner) serializeKey(key interface{}, algorithm SigningAlgorithm) ([]byte, error) {
	switch k := key.(type) {
	case *rsa.PrivateKey:
		// Use a simpler approach for now
		return []byte("rsa_private_key"), nil
	case *ecdsa.PrivateKey:
		// Use a simpler approach for now
		return []byte("ecdsa_private_key"), nil
	case []byte:
		return k, nil
	default:
		return nil, fmt.Errorf("unsupported key type for serialization")
	}
}

// deserializeKey deserializes a key from storage
func (s *MessageSigner) deserializeKey(data []byte, algorithm SigningAlgorithm) (interface{}, error) {
	switch algorithm {
	case SigningAlgorithmRS256, SigningAlgorithmRS384, SigningAlgorithmRS512:
		return x509.ParsePKCS1PrivateKey(data)
	case SigningAlgorithmES256, SigningAlgorithmES384, SigningAlgorithmES512:
		return x509.ParseECPrivateKey(data)
	case SigningAlgorithmHS256, SigningAlgorithmHS384, SigningAlgorithmHS512:
		return data, nil
	default:
		return nil, fmt.Errorf("unsupported signing algorithm: %s", algorithm)
	}
}

// SetLogger sets the logger for the signer
func (s *MessageSigner) SetLogger(logger *zap.Logger) {
	s.logger = logger
}
