package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"os"
	"strconv"
	"time"
)

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

func verifyHMAC(r *http.Request, secret []byte) bool {
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

	message := ts // or ts + body, depending on your scheme
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(message))
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(sig), []byte(expectedSig))
}

/*
Middleware factory is used to pass in the secret auth keys
*/
func AuthMiddleware(secrets map[string][]byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			keyID := r.Header.Get("X-Key-ID")
			secret, ok := secrets[keyID]
			if !ok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if !verifyHMAC(r, secret) {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
