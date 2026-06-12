package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"tiket-bps-api/config"
	"tiket-bps-api/database"
	"tiket-bps-api/models"
	"tiket-bps-api/utils"
)

// POST /api/login
func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	var req models.LoginRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Format JSON tidak valid", nil)
		return
	}

	if req.Username == "" || req.Password == "" {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Username dan password wajib diisi", nil)
		return
	}

	var admin models.Admin
	err := database.DB.QueryRow(`
		SELECT id, username, password_hash, nama_lengkap, email, created_at
		FROM admin WHERE username = $1`, req.Username,
	).Scan(
		&admin.ID, &admin.Username, &admin.PasswordHash,
		&admin.NamaLengkap, &admin.Email, &admin.CreatedAt,
	)

	if err == sql.ErrNoRows {
		utils.JSONResponse(w, http.StatusUnauthorized, false, "Username atau password salah", nil)
		return
	}
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal memeriksa data admin", nil)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(req.Password)); err != nil {
		utils.JSONResponse(w, http.StatusUnauthorized, false, "Username atau password salah", nil)
		return
	}

	expiry := time.Now().Add(time.Duration(config.App.JWTExpiryHours) * time.Hour)
	claims := jwt.MapClaims{
		"admin_id": admin.ID,
		"username": admin.Username,
		"exp":      expiry.Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(config.App.JWTSecret))
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal membuat token", nil)
		return
	}

	utils.JSONResponse(w, http.StatusOK, true, "Login berhasil", models.LoginResponse{
		Token: tokenStr,
		Admin: &admin,
	})
}

// GET /api/admin/profile  (protected)
func GetAdminProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	adminID := r.Context().Value("admin_id")
	if adminID == nil {
		utils.JSONResponse(w, http.StatusUnauthorized, false, "Tidak terautentikasi", nil)
		return
	}

	var admin models.Admin
	err := database.DB.QueryRow(`
		SELECT id, username, nama_lengkap, email, created_at
		FROM admin WHERE id = $1`, adminID,
	).Scan(&admin.ID, &admin.Username, &admin.NamaLengkap, &admin.Email, &admin.CreatedAt)

	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal mengambil data profil", nil)
		return
	}

	utils.JSONResponse(w, http.StatusOK, true, "OK", admin)
}
