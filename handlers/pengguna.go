package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"tiket-bps-api/database"
	"tiket-bps-api/models"
	"tiket-bps-api/utils"
)

// GET /api/cek-hp?no_hp=08xx
func CekHP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	noHP := strings.TrimSpace(r.URL.Query().Get("no_hp"))
	if noHP == "" {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Parameter no_hp wajib diisi", nil)
		return
	}
	if !utils.ValidateHP(noHP) {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Format nomor HP tidak valid", nil)
		return
	}

	var p models.Pengguna
	err := database.DB.QueryRow(`
		SELECT id, no_hp, nama, jk, tahun_lahir, pendidikan, pekerjaan,
		       tempat_kerja, email, created_at, updated_at
		FROM pengguna WHERE no_hp = $1`, noHP,
	).Scan(
		&p.ID, &p.NoHP, &p.Nama, &p.JK, &p.TahunLahir,
		&p.Pendidikan, &p.Pekerjaan, &p.TempatKerja, &p.Email,
		&p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		utils.JSONResponse(w, http.StatusOK, true, "Nomor HP belum terdaftar", map[string]interface{}{
			"terdaftar": false,
		})
		return
	}
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal memeriksa data", nil)
		return
	}

	utils.JSONResponse(w, http.StatusOK, true, "Nomor HP sudah terdaftar", map[string]interface{}{
		"terdaftar": true,
		"data":      p,
	})
}

// POST /api/register
func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	var req models.RegisterRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Format JSON tidak valid: "+err.Error(), nil)
		return
	}

	if err := utils.ValidateRegisterRequest(&req); err != nil {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, err.Error(), nil)
		return
	}

	// cek duplikat HP
	var existingID int64
	err := database.DB.QueryRow("SELECT id FROM pengguna WHERE no_hp = $1", req.NoHP).Scan(&existingID)
	if err == nil {
		utils.JSONResponse(w, http.StatusConflict, false, "Nomor HP sudah terdaftar", map[string]interface{}{
			"no_hp": req.NoHP,
		})
		return
	}
	if err != sql.ErrNoRows {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal memeriksa duplikat data", nil)
		return
	}

	var p models.Pengguna
	err = database.DB.QueryRow(`
		INSERT INTO pengguna (no_hp, nama, jk, tahun_lahir, pendidikan, pekerjaan, tempat_kerja, email)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, no_hp, nama, jk, tahun_lahir, pendidikan, pekerjaan,
		          tempat_kerja, email, created_at, updated_at`,
		req.NoHP, req.Nama, req.JK, req.TahunLahir,
		req.Pendidikan, req.Pekerjaan, req.TempatKerja, req.Email,
	).Scan(
		&p.ID, &p.NoHP, &p.Nama, &p.JK, &p.TahunLahir,
		&p.Pendidikan, &p.Pekerjaan, &p.TempatKerja, &p.Email,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal menyimpan data pendaftaran", nil)
		return
	}

	utils.JSONResponse(w, http.StatusCreated, true, "Registrasi berhasil", p)
}
