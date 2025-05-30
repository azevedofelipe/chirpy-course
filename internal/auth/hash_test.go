package auth

import (
	"testing"
)

func FuncTestHashPassword(t *testing.T) {
	password := "testing"
	hashed, err := HashPassword(password)
	err = CheckPasswordHash(hashed, "testing")
	if err != nil {
		t.Errorf("Password does not match hash: %n", err)
	}
}
