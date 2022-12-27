package password

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

func Hash(plaintextPassword string) (string, error) {
	// hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	// if err != nil {
	// 	return "", err
	// }
	hasher := sha256.New()
	hasher.Write([]byte(plaintextPassword))
	hashedPassword := hex.EncodeToString(hasher.Sum(nil))
	fmt.Println(hashedPassword)
	return hashedPassword, nil
}

func Matches(plaintextPassword, hashedPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}

	return true, nil
}
