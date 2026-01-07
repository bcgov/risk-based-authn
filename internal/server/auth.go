package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"rba/rules"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

func validateJWT(rawToken string) bool {
	jwksURL := os.Getenv("JWKS_URL")
	expectedAud := os.Getenv("JWT_AUD")

	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		log.Printf("Failed to create JWK Set from resource at the given URL.\nError: %s", err)
		return false
	}

	// Parse the JWT.
	token, err := jwt.Parse(rawToken, jwks.Keyfunc)
	if err != nil {
		log.Printf("Failed to parse the JWT.\nError: %s", err)
		return false
	}

	aud, err := token.Claims.GetAudience()

	if err != nil || !slices.Contains(aud, expectedAud) {
		return false
	}

	if !token.Valid {
		return false
	}

	return true
}

func parseSkew() time.Duration {
	defaultSkew := 5 * time.Minute
	val := os.Getenv("ALLOWED_SKEW_MINUTES")
	minutes, err := strconv.Atoi(val)
	if err != nil {
		return defaultSkew
	}

	if minutes <= 0 {
		return 0 // disabled in local dev
	}

	return time.Duration(minutes) * time.Minute
}

func verifyHMAC(r *http.Request) bool {
	sig := r.Header.Get("X-Signature")
	ts := r.Header.Get("X-Timestamp")

	allowedSkew := parseSkew()
	// --- Timestamp validation ---
	tsInt, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return false
	}

	t := time.Unix(tsInt, 0)
	now := time.Now()
	if allowedSkew != 0 && t.Before(now.Add(-allowedSkew)) || t.After(now.Add(allowedSkew)) {
		return false // stale or future request
	}

	clientKey := r.Header.Get("X-Key-ID")
	clientSecret := os.Getenv(clientKey)

	message := ts // or ts + body, depending on your scheme
	mac := hmac.New(sha256.New, []byte(clientSecret))
	mac.Write([]byte(message))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expectedSig))
}

/*
Middleware factory is used to pass in config
*/
func AuthMiddleware(authCfg rules.AuthConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if authCfg.Enabled {

				if authCfg.Method == "jwt" {
					authHeader := r.Header.Get("Authorization")
					parts := strings.Split(authHeader, " ")

					if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
						http.Error(w, "forbidden", http.StatusForbidden)
						return
					}

					if !validateJWT(parts[1]) {
						http.Error(w, "forbidden", http.StatusForbidden)
						return
					}
				}

				if authCfg.Method == "hmac" {
					if !verifyHMAC(r) {
						http.Error(w, "forbidden", http.StatusForbidden)
						return
					}
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
