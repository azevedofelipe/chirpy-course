package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), 8)
	if err != nil {
		return "", err
	}

	return string(hash), nil

}

func CheckPasswordHash(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func MakeJWT(userID uuid.UUID, tokenSecret string) (string, error) {

	claims := &jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
		ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(1 * time.Hour)),
		Subject:   userID.String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(tokenSecret))
	if err != nil {
		return "", err
	}

	return signed, nil
}

func ValidateJWT(tokenString, tokenSecret string) (userID uuid.UUID, err error) {
	type customClaims struct {
		jwt.RegisteredClaims
	}

	token, err := jwt.ParseWithClaims(tokenString, &customClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(tokenSecret), nil
	})

	if err != nil {
		return
	}

	userIDString, err := token.Claims.GetSubject()
	if err != nil {
		return
	}

	userID, err = uuid.Parse(userIDString)
	if err != nil {
		return
	}

	return

}

func GetBearerToken(headers http.Header) (string, error) {
	authHeader := headers.Get("Authorization")
	if authHeader == "" {
		return "", errors.New("Error getting Authorization header")
	}
	token := strings.TrimPrefix(authHeader, "Bearer ")

	return token, nil
}

func MakeRefreshToken() (string, error) {
	key := make([]byte, 32)
	rand.Read(key)
	keyString := hex.EncodeToString(key)

	return keyString, nil

}

func GetAPIKey(headers http.Header) (string, error) {
	apiHeader := headers.Get("Authorization")
	if apiHeader == "" {
		return "", errors.New("Error getting API Key")
	}
	apiKey := strings.TrimPrefix(apiHeader, "ApiKey ")

	return apiKey, nil
}
