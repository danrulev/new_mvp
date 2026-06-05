package hasher

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

type Hasher struct {
}

func NewHasher() *Hasher {
	return &Hasher{}
}

func (h *Hasher) GenerateHash(password string) (string, error) {
	if password == "" {
		return "", fmt.Errorf("empty password")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

func (h *Hasher) ComparePassword(hash string, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}