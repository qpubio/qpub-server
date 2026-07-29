package security

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultPasswordLength  = 16
	DefaultPasswordCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	DefaultHashCost        = 12
)

type Password struct {
	generator *Generator
}

func NewPassword() *Password {
	return &Password{
		generator: NewGenerator(),
	}
}

func (p *Password) Generate(length int) (string, error) {
	if length == 0 {
		length = DefaultPasswordLength
	}
	return p.generator.Generate(length, DefaultPasswordCharset)
}

func (p *Password) Hash(password string, cost int) (string, error) {
	if password == "" {
		return "", ErrEmptyPassword
	}
	if cost == 0 {
		cost = DefaultHashCost
	}
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return "", wrapError(ErrInvalidCost, "cost %d", cost)
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", wrapError(ErrHashPassword, err.Error())
	}
	return string(bytes), nil
}

func (p *Password) Verify(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return wrapError(ErrPasswordMismatch, "incorrect password")
		}
		return wrapError(err, "verification failed")
	}
	return nil
}
