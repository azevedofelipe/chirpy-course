package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHashPassword(t *testing.T) {
	password := "testing"
	hashed, err := HashPassword(password)
	err = CheckPasswordHash(hashed, "testing")
	if err != nil {
		t.Errorf("Password does not match hash: %n", err)
	}
}

func TestJWT(t *testing.T) {
	userID := uuid.New()
	tokenSecret := "potato"
	expires := 10 * time.Minute

	tokenString, err := MakeJWT(userID, tokenSecret, expires)
	if err != nil {
		t.Errorf("Erro ao criar JWT: %v", err)
	}

	validatedID, err := ValidateJWT(tokenString, tokenSecret)
	if err != nil {
		t.Errorf("Erro ao validar JWT: %v", err)
	}

	if validatedID != userID {
		t.Errorf("IDs não coincidem: esperado %v, obtido %v", userID, validatedID)
	}
}
