package authapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

func signJWT(secret []byte, claims authClaims) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("marshal jwt header: %w", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt claims: %w", err)
	}

	encodedHeader := base64.RawURLEncoding.EncodeToString(headerJSON)
	encodedClaims := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsigned := encodedHeader + "." + encodedClaims

	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return unsigned + "." + signature, nil
}

func verifyJWT(secret []byte, token, expectedIssuer, expectedAudience string) (authClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return authClaims{}, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(unsigned))
	expectedSignature := mac.Sum(nil)

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return authClaims{}, ErrInvalidToken
	}
	if !hmac.Equal(signature, expectedSignature) {
		return authClaims{}, ErrInvalidToken
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return authClaims{}, ErrInvalidToken
	}

	var claims authClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return authClaims{}, ErrInvalidToken
	}
	if claims.Iss != expectedIssuer || claims.Aud != expectedAudience {
		return authClaims{}, ErrInvalidToken
	}
	if claims.Sub == "" || claims.Sid == "" || claims.Exp <= 0 {
		return authClaims{}, ErrInvalidToken
	}
	return claims, nil
}
