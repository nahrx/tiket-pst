package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"tiket-bps-api/config"
	"tiket-bps-api/utils"
)

type contextKey string

const AdminIDKey contextKey = "admin_id"

// ─── CORS ────────────────────────────────────────────────────────────────────

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		// cek apakah origin diizinkan
		allowed := false
		for _, o := range config.App.AllowedOrigins {
			if o == "*" || o == origin {
				allowed = true
				break
			}
		}

		if allowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if len(config.App.AllowedOrigins) > 0 {
			w.Header().Set("Access-Control-Allow-Origin", config.App.AllowedOrigins[0])
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ─── LOGGER ──────────────────────────────────────────────────────────────────

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{w, http.StatusOK}
		next.ServeHTTP(rw, r)
		duration := time.Since(start)
		// log sederhana ke stdout
		_ = duration
		// log.Printf("[%s] %s %s %d (%v)", time.Now().Format("2006-01-02 15:04:05"), r.Method, r.URL.Path, rw.status, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// ─── AUTH JWT ─────────────────────────────────────────────────────────────────

func JWTAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.JSONResponse(w, http.StatusUnauthorized, false, "Token tidak ditemukan", nil)
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			utils.JSONResponse(w, http.StatusUnauthorized, false, "Format token tidak valid (gunakan: Bearer <token>)", nil)
			return
		}

		tokenStr := parts[1]
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, jwt.ErrSignatureInvalid
			}
			return []byte(config.App.JWTSecret), nil
		})

		if err != nil || !token.Valid {
			utils.JSONResponse(w, http.StatusUnauthorized, false, "Token tidak valid atau sudah kadaluarsa", nil)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			utils.JSONResponse(w, http.StatusUnauthorized, false, "Token claims tidak valid", nil)
			return
		}

		adminID, ok := claims["admin_id"].(float64)
		if !ok {
			utils.JSONResponse(w, http.StatusUnauthorized, false, "Admin ID tidak ditemukan di token", nil)
			return
		}

		ctx := context.WithValue(r.Context(), AdminIDKey, int64(adminID))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
