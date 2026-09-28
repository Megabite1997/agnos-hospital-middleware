package auth

import "golang.org/x/crypto/bcrypt"

// PasswordHasher is an interface so tests can swap in a cheap implementation.
type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) bool
}

type bcryptHasher struct {
	cost int
}

func NewBcryptHasher() PasswordHasher {
	return bcryptHasher{cost: bcrypt.DefaultCost}
}

func (h bcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	return string(b), err
}

func (h bcryptHasher) Compare(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
