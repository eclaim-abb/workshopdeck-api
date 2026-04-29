package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type DBProvider interface {
	GetDB() *gorm.DB
}

func GenerateRandomPassword(length int) (string, string, error) {
	b := make([]byte, length)
	rand.Read(b)

	finStr := base64.URLEncoding.EncodeToString(b)[:length]

	hashed, err := bcrypt.GenerateFromPassword([]byte(finStr), bcrypt.DefaultCost)
	if err != nil {
		return finStr, "", fmt.Errorf("failed to hash password: %v", err)
	}
	return finStr, string(hashed), nil
}

// WithTransaction executes the given function within a rollback-able database transaction
func WithTransaction(r DBProvider, fn func(tx *gorm.DB) error) error {
	tx := r.GetDB().Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}
