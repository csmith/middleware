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
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func hmacHex(hashFunc func() hash.Hash, secret string, body []byte) string {
	mac := hmac.New(hashFunc, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature(t *testing.T) {
	innerHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Write(body)
	})

	sha256Body := []byte(`{"event": "push"}`)
	sha256Sig := hmacHex(sha256.New, "secret", sha256Body)

	sha1Body := []byte(`{"event": "release"}`)
	sha1Sig := hmacHex(sha1.New, "key", sha1Body)

	sha512Body := []byte(`{"event": "deploy"}`)
	sha512Sig := hmacHex(sha512.New, "pass", sha512Body)

	wrongBodySig := hmacHex(sha256.New, "secret", []byte(`{"event": "other"}`))

	tests := []struct {
		name           string
		headerName     string
		opts           []VerifySignatureOption
		body           []byte
		headerValue    string
		expectedStatus int
		expectedBody   string
	}{
		{"Valid signature without prefix", "X-Signature", []VerifySignatureOption{WithSignatureHeader("X-Signature"), WithSignatureSecret("secret")}, sha256Body, sha256Sig, http.StatusOK, string(sha256Body)},
		{"Valid signature with prefix", "X-Hub-Signature-256", []VerifySignatureOption{WithSignatureHeader("X-Hub-Signature-256"), WithSignatureSecret("secret")}, sha256Body, "sha256=" + sha256Sig, http.StatusOK, string(sha256Body)},
		{"Valid signature with uppercase prefix", "X-Sig", []VerifySignatureOption{WithSignatureHeader("X-Sig"), WithSignatureSecret("secret")}, sha256Body, "SHA256=" + sha256Sig, http.StatusOK, string(sha256Body)},
		{"Missing header", "X-Signature", []VerifySignatureOption{WithSignatureHeader("X-Signature"), WithSignatureSecret("secret")}, sha256Body, "", http.StatusForbidden, "Missing signature\n"},
		{"Invalid signature", "X-Signature", []VerifySignatureOption{WithSignatureHeader("X-Signature"), WithSignatureSecret("secret")}, sha256Body, "badsig", http.StatusForbidden, "Invalid signature\n"},
		{"Signature for wrong body", "X-Signature", []VerifySignatureOption{WithSignatureHeader("X-Signature"), WithSignatureSecret("secret")}, sha256Body, wrongBodySig, http.StatusForbidden, "Invalid signature\n"},
		{"Wrong secret", "X-Signature", []VerifySignatureOption{WithSignatureHeader("X-Signature"), WithSignatureSecret("wrongsecret")}, sha256Body, sha256Sig, http.StatusForbidden, "Invalid signature\n"},
		{"Wrong prefix", "X-Signature", []VerifySignatureOption{WithSignatureHeader("X-Signature"), WithSignatureSecret("secret")}, sha256Body, "sha1=" + sha256Sig, http.StatusForbidden, "Invalid signature\n"},
		{"SHA1 algorithm with prefix", "X-Hub-Signature", []VerifySignatureOption{WithSignatureHeader("X-Hub-Signature"), WithSignatureAlgorithm(SHA1), WithSignatureSecret("key")}, sha1Body, "sha1=" + sha1Sig, http.StatusOK, string(sha1Body)},
		{"SHA1 algorithm without prefix", "X-Hub-Signature", []VerifySignatureOption{WithSignatureHeader("X-Hub-Signature"), WithSignatureAlgorithm(SHA1), WithSignatureSecret("key")}, sha1Body, sha1Sig, http.StatusOK, string(sha1Body)},
		{"SHA512 algorithm", "X-Signature", []VerifySignatureOption{WithSignatureHeader("X-Signature"), WithSignatureAlgorithm(SHA512), WithSignatureSecret("pass")}, sha512Body, sha512Sig, http.StatusOK, string(sha512Body)},
		{"Empty body", "X-Sig", []VerifySignatureOption{WithSignatureHeader("X-Sig"), WithSignatureSecret("secret")}, []byte{}, hmacHex(sha256.New, "secret", []byte{}), http.StatusOK, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := VerifySignature(tt.opts...)(innerHandler)

			req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(tt.body))
			if tt.headerValue != "" {
				req.Header.Set(tt.headerName, tt.headerValue)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
			assert.Equal(t, tt.expectedBody, rr.Body.String())
		})
	}
}

func TestVerifySignature_UnsupportedAlgorithm(t *testing.T) {
	assert.Panics(t, func() {
		VerifySignature(WithSignatureAlgorithm(SignatureAlgorithm("md5")))
	})
}
