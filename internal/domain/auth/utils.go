package auth

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

// generateOTP returns a cryptographically-random 6-digit string, e.g. "047291".
func generateOTP() (string, error) {
	const digits = "0123456789"
	result := make([]byte, 6)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		result[i] = digits[n.Int64()]
	}
	return string(result), nil
}

// generateRandomKey generates a hex-encoded random key of `length` bytes.
func generateRandomKey(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
