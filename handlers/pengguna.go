package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
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

// GET /api/admin/pengguna  (protected)
// Query: ?q=keyword&page=1&limit=20&sort=&order=
func ListPengguna(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	q := strings.TrimSpace(r.URL.Query().Get("q"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	sortColumns := map[string]string{
		"nama":        "nama",
		"no_hp":       "no_hp",
		"email":       "email",
		"pekerjaan":   "pekerjaan",
		"tahun_lahir": "tahun_lahir",
		"created_at":  "created_at",
	}
	orderByCol, ok := sortColumns[strings.TrimSpace(r.URL.Query().Get("sort"))]
	if !ok {
		orderByCol = "created_at"
	}
	orderDir := "DESC"
	if strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("order"))) == "ASC" {
		orderDir = "ASC"
	}

	args := []interface{}{}
	where := ""
	if q != "" {
		like := "%" + q + "%"
		where = "WHERE nama ILIKE $1 OR no_hp ILIKE $2 OR email ILIKE $3"
		args = append(args, like, like, like)
	}

	var total int
	database.DB.QueryRow("SELECT COUNT(*) FROM pengguna "+where, args...).Scan(&total)

	args = append(args, limit, offset)
	rows, err := database.DB.Query(
		`SELECT id, no_hp, nama, jk, tahun_lahir, pendidikan, pekerjaan, tempat_kerja, email, created_at, updated_at
		 FROM pengguna `+where+
			` ORDER BY `+orderByCol+` `+orderDir+
			` LIMIT $`+strconv.Itoa(len(args)-1)+` OFFSET $`+strconv.Itoa(len(args)),
		args...)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal mengambil data pengguna", nil)
		return
	}
	defer rows.Close()

	list := []models.Pengguna{}
	for rows.Next() {
		var p models.Pengguna
		if err := rows.Scan(
			&p.ID, &p.NoHP, &p.Nama, &p.JK, &p.TahunLahir,
			&p.Pendidikan, &p.Pekerjaan, &p.TempatKerja, &p.Email,
			&p.CreatedAt, &p.UpdatedAt,
		); err == nil {
			list = append(list, p)
		}
	}

	utils.JSONResponse(w, http.StatusOK, true, "OK", map[string]interface{}{
		"pengguna":    list,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + limit - 1) / limit,
	})
}

// POST /api/admin/pengguna  (protected)
func TambahPengguna(w http.ResponseWriter, r *http.Request) {
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

	var existingID int64
	err := database.DB.QueryRow("SELECT id FROM pengguna WHERE no_hp = $1", req.NoHP).Scan(&existingID)
	if err == nil {
		utils.JSONResponse(w, http.StatusConflict, false, "Nomor HP sudah terdaftar", nil)
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
		RETURNING id, no_hp, nama, jk, tahun_lahir, pendidikan, pekerjaan, tempat_kerja, email, created_at, updated_at`,
		req.NoHP, req.Nama, req.JK, req.TahunLahir, req.Pendidikan, req.Pekerjaan, req.TempatKerja, req.Email,
	).Scan(
		&p.ID, &p.NoHP, &p.Nama, &p.JK, &p.TahunLahir,
		&p.Pendidikan, &p.Pekerjaan, &p.TempatKerja, &p.Email,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal menyimpan data pengguna", nil)
		return
	}

	utils.JSONResponse(w, http.StatusCreated, true, "Pengguna berhasil ditambahkan", p)
}

// GET /api/admin/pengguna/{id}  (protected)
func DetailPengguna(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	id := extractLastPathSegment(r.URL.Path)

	var p models.Pengguna
	err := database.DB.QueryRow(`
		SELECT id, no_hp, nama, jk, tahun_lahir, pendidikan, pekerjaan, tempat_kerja, email, created_at, updated_at
		FROM pengguna WHERE id = $1`, id,
	).Scan(
		&p.ID, &p.NoHP, &p.Nama, &p.JK, &p.TahunLahir,
		&p.Pendidikan, &p.Pekerjaan, &p.TempatKerja, &p.Email,
		&p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		utils.JSONResponse(w, http.StatusNotFound, false, "Pengguna tidak ditemukan", nil)
		return
	}
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal mengambil data pengguna", nil)
		return
	}

	utils.JSONResponse(w, http.StatusOK, true, "OK", p)
}

// PUT /api/admin/pengguna/{id}  (protected)
func UpdatePengguna(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	id := extractLastPathSegment(r.URL.Path)

	var req models.RegisterRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Format JSON tidak valid: "+err.Error(), nil)
		return
	}
	if err := utils.ValidateRegisterRequest(&req); err != nil {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, err.Error(), nil)
		return
	}

	var existingID int64
	err := database.DB.QueryRow("SELECT id FROM pengguna WHERE no_hp = $1 AND id != $2", req.NoHP, id).Scan(&existingID)
	if err == nil {
		utils.JSONResponse(w, http.StatusConflict, false, "Nomor HP sudah digunakan pengguna lain", nil)
		return
	}
	if err != sql.ErrNoRows {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal memeriksa duplikat data", nil)
		return
	}

	var p models.Pengguna
	err = database.DB.QueryRow(`
		UPDATE pengguna
		SET no_hp = $1, nama = $2, jk = $3, tahun_lahir = $4, pendidikan = $5,
		    pekerjaan = $6, tempat_kerja = $7, email = $8, updated_at = NOW()
		WHERE id = $9
		RETURNING id, no_hp, nama, jk, tahun_lahir, pendidikan, pekerjaan, tempat_kerja, email, created_at, updated_at`,
		req.NoHP, req.Nama, req.JK, req.TahunLahir, req.Pendidikan, req.Pekerjaan, req.TempatKerja, req.Email, id,
	).Scan(
		&p.ID, &p.NoHP, &p.Nama, &p.JK, &p.TahunLahir,
		&p.Pendidikan, &p.Pekerjaan, &p.TempatKerja, &p.Email,
		&p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		utils.JSONResponse(w, http.StatusNotFound, false, "Pengguna tidak ditemukan", nil)
		return
	}
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal update pengguna", nil)
		return
	}

	utils.JSONResponse(w, http.StatusOK, true, "Pengguna berhasil diperbarui", p)
}

// DELETE /api/admin/pengguna/{id}  (protected)
func HapusPengguna(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	id := extractLastPathSegment(r.URL.Path)

	res, err := database.DB.Exec("DELETE FROM pengguna WHERE id = $1", id)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal menghapus pengguna", nil)
		return
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		utils.JSONResponse(w, http.StatusNotFound, false, "Pengguna tidak ditemukan", nil)
		return
	}

	utils.JSONResponse(w, http.StatusOK, true, "Pengguna berhasil dihapus", nil)
}
