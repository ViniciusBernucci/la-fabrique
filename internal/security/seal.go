package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
)

// ErrSealed indica texto cifrado inválido, adulterado ou de outra chave/contexto.
var ErrSealed = errors.New("dado cifrado inválido")

// Sealer cifra com AES-256-GCM usando uma chave que vive fora do banco
// (VAULT_KEY). O contexto adicional (AAD) amarra o texto cifrado ao registro:
// copiar o cifrado de uma linha para outra faz a abertura falhar.
type Sealer struct {
	aead  cipher.AEAD
	keyID string
}

// NewSealer exige chave de 32 bytes.
func NewSealer(key []byte, keyID string) (*Sealer, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("chave do cofre deve ter 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Sealer{aead: aead, keyID: keyID}, nil
}

// KeyID identifica a chave usada (para rotação futura da chave mestra).
func (s *Sealer) KeyID() string { return s.keyID }

// Seal devolve nonce||texto-cifrado.
func (s *Sealer) Seal(plaintext, aad []byte) ([]byte, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return s.aead.Seal(nonce, nonce, plaintext, aad), nil
}

// Open reverte Seal.
func (s *Sealer) Open(sealed, aad []byte) ([]byte, error) {
	n := s.aead.NonceSize()
	if len(sealed) < n+s.aead.Overhead() {
		return nil, ErrSealed
	}
	pt, err := s.aead.Open(nil, sealed[:n], sealed[n:], aad)
	if err != nil {
		return nil, ErrSealed
	}
	return pt, nil
}
