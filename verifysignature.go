package middleware

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"io"
	"net/http"
	"strings"
)

type SignatureAlgorithm string

const (
	SHA1   SignatureAlgorithm = "sha1"
	SHA256 SignatureAlgorithm = "sha256"
	SHA512 SignatureAlgorithm = "sha512"
)

var supportedAlgorithms = map[SignatureAlgorithm]func() hash.Hash{
	SHA1:   sha1.New,
	SHA256: sha256.New,
	SHA512: sha512.New,
}

type verifySignatureConfig struct {
	headerName string
	algorithm  SignatureAlgorithm
	secret     string
}

type VerifySignatureOption func(*verifySignatureConfig)

func WithSignatureHeader(headerName string) VerifySignatureOption {
	return func(config *verifySignatureConfig) {
		config.headerName = headerName
	}
}

func WithSignatureAlgorithm(algorithm SignatureAlgorithm) VerifySignatureOption {
	return func(config *verifySignatureConfig) {
		config.algorithm = algorithm
	}
}

func WithSignatureSecret(secret string) VerifySignatureOption {
	return func(config *verifySignatureConfig) {
		config.secret = secret
	}
}

// VerifySignature returns a middleware that verifies request signatures using
// HMAC. It reads the request body, computes the HMAC using the given algorithm
// and secret, and compares it to the value in the specified header.
//
// The signature header may contain just the hex-encoded hash, or be prefixed
// with the algorithm name followed by an equals sign (e.g. "sha256=abcdef...").
// Both formats are accepted.
//
// Requests with a missing or invalid signature receive a 403 Forbidden response.
func VerifySignature(opts ...VerifySignatureOption) func(http.Handler) http.Handler {
	config := &verifySignatureConfig{
		algorithm: SHA256,
	}
	for _, opt := range opts {
		opt(config)
	}

	hashFunc, ok := supportedAlgorithms[config.algorithm]
	if !ok {
		panic("middleware: unsupported signature algorithm: " + string(config.algorithm))
	}

	algoStr := strings.ToLower(string(config.algorithm))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headerValue := r.Header.Get(config.headerName)
			if headerValue == "" {
				http.Error(w, "Missing signature", http.StatusForbidden)
				return
			}

			body, err := io.ReadAll(r.Body)
			r.Body.Close()
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))

			mac := hmac.New(hashFunc, []byte(config.secret))
			mac.Write(body)
			expected := hex.EncodeToString(mac.Sum(nil))

			signature := headerValue
			if prefix, sig, found := strings.Cut(headerValue, "="); found {
				if strings.EqualFold(prefix, algoStr) {
					signature = sig
				}
			}

			if !hmac.Equal([]byte(signature), []byte(expected)) {
				http.Error(w, "Invalid signature", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
