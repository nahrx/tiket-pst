package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"tiket-bps-api/database"
	"tiket-bps-api/models"
	"tiket-bps-api/utils"
)

// GET /api/admin/petugas  (protected)
func ListPetugas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	rows, err := database.DB.Query(`
		SELECT id, nama_lengkap, nama_alias, nip, aktif, created_at, updated_at
		FROM petugas_pst
		ORDER BY nama_lengkap ASC`)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal mengambil data petugas", nil)
		return
	}
	defer rows.Close()

	list := []models.PetugasPST{}
	for rows.Next() {
		var p models.PetugasPST
		if err := rows.Scan(&p.ID, &p.NamaLengkap, &p.NamaAlias, &p.NIP, &p.Aktif, &p.CreatedAt, &p.UpdatedAt); err == nil {
			list = append(list, p)
		}
	}

	utils.JSONResponse(w, http.StatusOK, true, "OK", list)
}

// POST /api/admin/petugas  (protected)
func TambahPetugas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	var req models.PetugasRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Format JSON tidak valid: "+err.Error(), nil)
		return
	}

	req.NamaLengkap = strings.TrimSpace(req.NamaLengkap)
	req.NamaAlias   = strings.TrimSpace(req.NamaAlias)
	req.NIP         = strings.TrimSpace(req.NIP)

	if req.NamaLengkap == "" {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, "Nama lengkap wajib diisi", nil)
		return
	}

	var p models.PetugasPST
	err := database.DB.QueryRow(`
		INSERT INTO petugas_pst (nama_lengkap, nama_alias, nip, aktif)
		VALUES ($1, $2, $3, $4)
		RETURNING id, nama_lengkap, nama_alias, nip, aktif, created_at, updated_at`,
		req.NamaLengkap, req.NamaAlias, req.NIP, req.Aktif,
	).Scan(&p.ID, &p.NamaLengkap, &p.NamaAlias, &p.NIP, &p.Aktif, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal menyimpan petugas", nil)
		return
	}

	utils.JSONResponse(w, http.StatusCreated, true, "Petugas berhasil ditambahkan", p)
}

// GET /api/admin/petugas/{id}  (protected)
func DetailPetugas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	id := extractLastPathSegment(r.URL.Path)

	var p models.PetugasPST
	err := database.DB.QueryRow(`
		SELECT id, nama_lengkap, nama_alias, nip, aktif, created_at, updated_at
		FROM petugas_pst WHERE id = $1`, id,
	).Scan(&p.ID, &p.NamaLengkap, &p.NamaAlias, &p.NIP, &p.Aktif, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		utils.JSONResponse(w, http.StatusNotFound, false, "Petugas tidak ditemukan", nil)
		return
	}
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal mengambil data petugas", nil)
		return
	}

	utils.JSONResponse(w, http.StatusOK, true, "OK", p)
}

// PUT /api/admin/petugas/{id}  (protected)
func UpdatePetugas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	id := extractLastPathSegment(r.URL.Path)

	var req models.PetugasRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Format JSON tidak valid: "+err.Error(), nil)
		return
	}

	req.NamaLengkap = strings.TrimSpace(req.NamaLengkap)
	req.NamaAlias   = strings.TrimSpace(req.NamaAlias)
	req.NIP         = strings.TrimSpace(req.NIP)

	if req.NamaLengkap == "" {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, "Nama lengkap wajib diisi", nil)
		return
	}

	var p models.PetugasPST
	err := database.DB.QueryRow(`
		UPDATE petugas_pst
		SET nama_lengkap = $1, nama_alias = $2, nip = $3, aktif = $4, updated_at = NOW()
		WHERE id = $5
		RETURNING id, nama_lengkap, nama_alias, nip, aktif, created_at, updated_at`,
		req.NamaLengkap, req.NamaAlias, req.NIP, req.Aktif, id,
	).Scan(&p.ID, &p.NamaLengkap, &p.NamaAlias, &p.NIP, &p.Aktif, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		utils.JSONResponse(w, http.StatusNotFound, false, "Petugas tidak ditemukan", nil)
		return
	}
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal update petugas", nil)
		return
	}

	utils.JSONResponse(w, http.StatusOK, true, "Petugas berhasil diperbarui", p)
}

// DELETE /api/admin/petugas/{id}  (protected)
func HapusPetugas(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	id := extractLastPathSegment(r.URL.Path)

	res, err := database.DB.Exec("DELETE FROM petugas_pst WHERE id = $1", id)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal menghapus petugas", nil)
		return
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		utils.JSONResponse(w, http.StatusNotFound, false, "Petugas tidak ditemukan", nil)
		return
	}

	utils.JSONResponse(w, http.StatusOK, true, "Petugas berhasil dihapus", nil)
}

// extractLastPathSegment mengambil segmen terakhir dari URL path
// misal: "/api/admin/petugas/5" → "5"
func extractLastPathSegment(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}
