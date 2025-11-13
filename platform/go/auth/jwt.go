package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"firebase.google.com/go/v4/auth"
)

type PalmyraClaims struct {
	IsAdmin bool
}

func VerifyUser(ctx context.Context, fbAuth *auth.Client, r *http.Request) (*auth.Token, error) {
	idToken, found := ExtractJWTToken(r)
	if !found {
		return nil, errors.New("Error. Auth token not found. Non authenticated calls are not allowed, How do we arrived here? !!!")
	}

	token, err := fbAuth.VerifyIDToken(ctx, idToken)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func ExtractJWTToken(r *http.Request) (string, bool) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", false
	}

	const prefix = "Bearer "
	// Case-insensitive prefix match.
	if len(authHeader) < len(prefix) || !strings.EqualFold(authHeader[:len(prefix)], prefix) {
		return "", false
	}

	return strings.TrimSpace(authHeader[len(prefix):]), true
}

func ExtractClaims(token auth.Token) PalmyraClaims {
	isAdmin, found := token.Claims["isAdmin"].(bool)

	claims := PalmyraClaims{
		IsAdmin: found && isAdmin,
	}

	return claims
}
