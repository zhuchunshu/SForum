package marketplace

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Ed25519Verifier verifies marketplace indexes with a standard Ed25519 public key.
type Ed25519Verifier struct {
	keyID string
	pub   ed25519.PublicKey
}

// NewEd25519Verifier builds a verifier from a 32-byte public key and stable key id.
func NewEd25519Verifier(keyID string, publicKey ed25519.PublicKey) (*Ed25519Verifier, error) {
	keyID = strings.TrimSpace(keyID)
	if keyID == "" || len(publicKey) != ed25519.PublicKeySize {
		return nil, ErrSignature
	}
	return &Ed25519Verifier{keyID: keyID, pub: append(ed25519.PublicKey(nil), publicKey...)}, nil
}

// PublicKeyID implements Verifier.
func (v *Ed25519Verifier) PublicKeyID() string {
	if v == nil {
		return ""
	}
	return v.keyID
}

// Verify implements Verifier.
func (v *Ed25519Verifier) Verify(canonicalBody []byte, signatureHex string) error {
	if v == nil || len(v.pub) != ed25519.PublicKeySize {
		return ErrSignature
	}
	sig, err := hex.DecodeString(strings.TrimSpace(signatureHex))
	if err != nil || len(sig) != ed25519.SignatureSize {
		return ErrSignature
	}
	if !ed25519.Verify(v.pub, canonicalBody, sig) {
		return ErrSignature
	}
	return nil
}

// SignIndexEd25519 signs the canonical JSON body (signature field cleared).
func SignIndexEd25519(privateKey ed25519.PrivateKey, index Index) (string, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return "", ErrSignature
	}
	body, err := canonicalIndexBytes(index)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(privateKey, body)
	return hex.EncodeToString(sig), nil
}

// canonicalIndexBytes marshals index with Signature cleared for signing/verify.
func canonicalIndexBytes(index Index) ([]byte, error) {
	body := cloneIndex(index)
	body.Signature = ""
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal: %v", ErrInvalid, err)
	}
	return raw, nil
}
