package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"tiket-bps-api/database"
	"tiket-bps-api/models"
	"tiket-bps-api/utils"
)

// helper: scan satu row tiket (termasuk LEFT JOIN petugas)
func scanTiket(row interface {
	Scan(...interface{}) error
}) (models.Tiket, error) {
	var t models.Tiket
	err := row.Scan(
		&t.ID, &t.NomorTiket, &t.NoHP, &t.NamaPemohon,
		&t.Kategori, &t.Subjek, &t.Uraian,
		&t.Status, &t.Catatan,
		&t.TanggalPelayanan, &t.PetugasID, &t.PetugasNama,
		&t.SilastikPertanyaan, &t.SilastikJawaban,
		&t.CreatedAt, &t.UpdatedAt,
	)
	return t, err
}

const tiketSelectCols = `
	t.id, t.nomor_tiket, t.no_hp, t.nama_pemohon,
	t.kategori, t.subjek, t.uraian,
	t.status, t.catatan,
	t.tanggal_pelayanan, t.petugas_id, p.nama_alias,
	t.silastik_pertanyaan, t.silastik_jawaban,
	t.created_at, t.updated_at
FROM tiket t
LEFT JOIN petugas_pst p ON p.id = t.petugas_id`

// POST /api/tiket
func BuatTiket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	var req models.TiketRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Format JSON tidak valid: "+err.Error(), nil)
		return
	}
	if err := utils.ValidateTiketRequest(&req); err != nil {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, err.Error(), nil)
		return
	}

	var nomorTiket string
	for i := 0; i < 5; i++ {
		candidate := utils.GenerateNomorTiket()
		var tmp string
		err := database.DB.QueryRow("SELECT nomor_tiket FROM tiket WHERE nomor_tiket = $1", candidate).Scan(&tmp)
		if err == sql.ErrNoRows {
			nomorTiket = candidate
			break
		}
	}
	if nomorTiket == "" {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal generate nomor tiket", nil)
		return
	}

	var newID int64
	err := database.DB.QueryRow(`
		INSERT INTO tiket (nomor_tiket, no_hp, nama_pemohon, kategori, subjek, uraian, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'baru')
		RETURNING id`,
		nomorTiket, req.NoHP, req.Nama, req.Kategori, req.Subjek, req.Uraian,
	).Scan(&newID)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal menyimpan tiket", nil)
		return
	}

	t, err := scanTiket(database.DB.QueryRow(`SELECT `+tiketSelectCols+` WHERE t.id = $1`, newID))
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal mengambil data tiket baru", nil)
		return
	}

	utils.JSONResponse(w, http.StatusCreated, true, "Tiket berhasil dibuat", t.ToJSON())
}

// GET /api/admin/tiket  (protected)
// Query: ?status=baru&petugas_id=1&page=1&limit=20&q=keyword
func ListTiket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	status    := strings.TrimSpace(r.URL.Query().Get("status"))
	q         := strings.TrimSpace(r.URL.Query().Get("q"))
	petugasID := strings.TrimSpace(r.URL.Query().Get("petugas_id"))
	page, _   := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _  := strconv.Atoi(r.URL.Query().Get("limit"))
	if page < 1  { page = 1 }
	if limit < 1 || limit > 100 { limit = 20 }
	offset := (page - 1) * limit

	sortColumns := map[string]string{
		"nomor_tiket":       "t.nomor_tiket",
		"nama_pemohon":      "t.nama_pemohon",
		"kategori":          "t.kategori",
		"subjek":            "t.subjek",
		"status":            "t.status",
		"petugas_nama":      "p.nama_alias",
		"tanggal_pelayanan": "t.tanggal_pelayanan",
		"created_at":        "t.created_at",
	}
	orderByCol, ok := sortColumns[strings.TrimSpace(r.URL.Query().Get("sort"))]
	if !ok {
		orderByCol = "t.created_at"
	}
	orderDir := "DESC"
	if strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("order"))) == "ASC" {
		orderDir = "ASC"
	}

	args       := []interface{}{}
	conditions := []string{}
	paramIdx   := 1

	if status != "" {
		conditions = append(conditions, "t.status = $"+strconv.Itoa(paramIdx))
		args = append(args, status)
		paramIdx++
	}
	if petugasID != "" {
		if petugasID == "null" || petugasID == "unassigned" {
			conditions = append(conditions, "t.petugas_id IS NULL")
		} else {
			conditions = append(conditions, "t.petugas_id = $"+strconv.Itoa(paramIdx))
			args = append(args, petugasID)
			paramIdx++
		}
	}
	if q != "" {
		like := "%" + q + "%"
		conditions = append(conditions,
			"(t.nama_pemohon ILIKE $"+strconv.Itoa(paramIdx)+
			" OR t.nomor_tiket ILIKE $"+strconv.Itoa(paramIdx+1)+
			" OR t.subjek ILIKE $"+strconv.Itoa(paramIdx+2)+
			" OR t.no_hp ILIKE $"+strconv.Itoa(paramIdx+3)+")")
		args = append(args, like, like, like, like)
		paramIdx += 4
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	database.DB.QueryRow(
		"SELECT COUNT(*) FROM tiket t LEFT JOIN petugas_pst p ON p.id = t.petugas_id "+where,
		countArgs...,
	).Scan(&total)

	args = append(args, limit, offset)
	rows, err := database.DB.Query(
		`SELECT `+tiketSelectCols+` `+where+
		` ORDER BY `+orderByCol+` `+orderDir+` LIMIT $`+strconv.Itoa(paramIdx)+
		` OFFSET $`+strconv.Itoa(paramIdx+1),
		args...)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal mengambil data tiket", nil)
		return
	}
	defer rows.Close()

	tikets := []models.TiketJSON{}
	for rows.Next() {
		t, err := scanTiket(rows)
		if err == nil {
			tikets = append(tikets, t.ToJSON())
		}
	}

	utils.JSONResponse(w, http.StatusOK, true, "OK", map[string]interface{}{
		"tikets":      tikets,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": (total + limit - 1) / limit,
	})
}

// GET /api/admin/tiket/{nomor_tiket}  (protected)
func DetailTiket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	nomorTiket := parts[len(parts)-1]

	t, err := scanTiket(database.DB.QueryRow(
		`SELECT `+tiketSelectCols+` WHERE t.nomor_tiket = $1`, nomorTiket,
	))
	if err == sql.ErrNoRows {
		utils.JSONResponse(w, http.StatusNotFound, false, "Tiket tidak ditemukan", nil)
		return
	}
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal mengambil data tiket", nil)
		return
	}

	utils.JSONResponse(w, http.StatusOK, true, "OK", t.ToJSON())
}

// PATCH /api/admin/tiket/{nomor_tiket}/status  (protected)
// Body: UpdateTiketRequest — bisa update status, catatan, tanggal_pelayanan, petugas_id sekaligus
func UpdateStatusTiket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 5 {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Path tidak valid", nil)
		return
	}
	nomorTiket := parts[len(parts)-2]

	var body models.UpdateTiketRequest
	if err := utils.DecodeJSON(r, &body); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Format JSON tidak valid", nil)
		return
	}

	validStatus := map[string]bool{"baru": true, "diproses": true, "selesai": true, "ditolak": true}
	if body.Status != "" && !validStatus[body.Status] {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false,
			"Status tidak valid (baru|diproses|selesai|ditolak)", nil)
		return
	}

	// Parse tanggal_pelayanan jika ada
	var tanggalPelayanan interface{} = nil // NULL by default
	if body.TanggalPelayanan != "" {
		tgl, err := time.Parse("2006-01-02", body.TanggalPelayanan)
		if err != nil {
			utils.JSONResponse(w, http.StatusUnprocessableEntity, false,
				"Format tanggal_pelayanan tidak valid (gunakan YYYY-MM-DD)", nil)
			return
		}
		tanggalPelayanan = tgl
	}

	// Bangun SET clause secara dinamis
	setClauses := []string{"updated_at = NOW()"}
	args       := []interface{}{}
	paramIdx   := 1

	if body.Status != "" {
		setClauses = append(setClauses, "status = $"+strconv.Itoa(paramIdx))
		args = append(args, body.Status)
		paramIdx++
	}
	// catatan selalu diupdate (boleh kosong)
	setClauses = append(setClauses, "catatan = $"+strconv.Itoa(paramIdx))
	args = append(args, body.Catatan)
	paramIdx++

	// tanggal_pelayanan: kalau string kosong → set NULL, kalau ada → set tanggal
	setClauses = append(setClauses, "tanggal_pelayanan = $"+strconv.Itoa(paramIdx))
	args = append(args, tanggalPelayanan)
	paramIdx++

	// petugas_id: kalau nil dalam JSON → set NULL, kalau ada nilai → set ID
	setClauses = append(setClauses, "petugas_id = $"+strconv.Itoa(paramIdx))
	args = append(args, body.PetugasID) // *int64: nil jadi NULL di postgres
	paramIdx++

	args = append(args, nomorTiket)
	query := "UPDATE tiket SET " + strings.Join(setClauses, ", ") +
		" WHERE nomor_tiket = $" + strconv.Itoa(paramIdx)

	res, err := database.DB.Exec(query, args...)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal update tiket: "+err.Error(), nil)
		return
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		utils.JSONResponse(w, http.StatusNotFound, false, "Tiket tidak ditemukan", nil)
		return
	}

	// Return tiket terbaru
	t, err := scanTiket(database.DB.QueryRow(
		`SELECT `+tiketSelectCols+` WHERE t.nomor_tiket = $1`, nomorTiket,
	))
	if err != nil {
		utils.JSONResponse(w, http.StatusOK, true, "Tiket berhasil diperbarui", nil)
		return
	}
	utils.JSONResponse(w, http.StatusOK, true, "Tiket berhasil diperbarui", t.ToJSON())
}

// POST /api/admin/tiket  (protected) — admin buat tiket atas nama pemohon
func AdminBuatTiket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	var req models.AdminTiketRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Format JSON tidak valid: "+err.Error(), nil)
		return
	}

	req.NoHP        = strings.TrimSpace(req.NoHP)
	req.NamaPemohon = strings.TrimSpace(req.NamaPemohon)
	req.Kategori    = strings.TrimSpace(req.Kategori)
	req.Subjek      = strings.TrimSpace(req.Subjek)
	req.Uraian      = strings.TrimSpace(req.Uraian)

	if req.NamaPemohon == "" {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, "Nama pemohon wajib diisi", nil)
		return
	}
	if req.Kategori == "" {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, "Kategori wajib diisi", nil)
		return
	}
	if req.Subjek == "" {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, "Subjek wajib diisi", nil)
		return
	}
	if req.Uraian == "" {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, "Uraian wajib diisi", nil)
		return
	}

	var nomorTiket string
	for i := 0; i < 5; i++ {
		candidate := utils.GenerateNomorTiket()
		var tmp string
		err := database.DB.QueryRow("SELECT nomor_tiket FROM tiket WHERE nomor_tiket = $1", candidate).Scan(&tmp)
		if err == sql.ErrNoRows {
			nomorTiket = candidate
			break
		}
	}
	if nomorTiket == "" {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal generate nomor tiket", nil)
		return
	}

	var newID int64
	err := database.DB.QueryRow(`
		INSERT INTO tiket (nomor_tiket, no_hp, nama_pemohon, kategori, subjek, uraian, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'baru')
		RETURNING id`,
		nomorTiket, req.NoHP, req.NamaPemohon, req.Kategori, req.Subjek, req.Uraian,
	).Scan(&newID)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal menyimpan tiket", nil)
		return
	}

	t, err := scanTiket(database.DB.QueryRow(`SELECT `+tiketSelectCols+` WHERE t.id = $1`, newID))
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal mengambil data tiket baru", nil)
		return
	}

	utils.JSONResponse(w, http.StatusCreated, true, "Tiket berhasil dibuat oleh admin", t.ToJSON())
}

// PUT /api/admin/tiket/{nomor}  (protected) — admin edit isi tiket secara penuh
func EditTiket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	nomorTiket := parts[len(parts)-1]

	var req models.EditTiketRequest
	if err := utils.DecodeJSON(r, &req); err != nil {
		utils.JSONResponse(w, http.StatusBadRequest, false, "Format JSON tidak valid: "+err.Error(), nil)
		return
	}

	req.NamaPemohon = strings.TrimSpace(req.NamaPemohon)
	req.Kategori    = strings.TrimSpace(req.Kategori)
	req.Subjek      = strings.TrimSpace(req.Subjek)
	req.Uraian      = strings.TrimSpace(req.Uraian)
	req.SilastikPertanyaan = strings.TrimSpace(req.SilastikPertanyaan)
	req.SilastikJawaban    = strings.TrimSpace(req.SilastikJawaban)

	if req.NamaPemohon == "" {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, "Nama pemohon wajib diisi", nil)
		return
	}
	if req.Kategori == "" {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, "Kategori wajib diisi", nil)
		return
	}
	if req.Subjek == "" {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, "Subjek wajib diisi", nil)
		return
	}
	if req.Uraian == "" {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false, "Uraian wajib diisi", nil)
		return
	}

	validStatus := map[string]bool{"baru": true, "diproses": true, "selesai": true, "ditolak": true}
	if req.Status != "" && !validStatus[req.Status] {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false,
			"Status tidak valid (baru|diproses|selesai|ditolak)", nil)
		return
	}
	status := req.Status
	if status == "" {
		status = "baru"
	}

	if (status == "selesai" || status == "ditolak") &&
		(req.SilastikPertanyaan == "" || req.SilastikJawaban == "") {
		utils.JSONResponse(w, http.StatusUnprocessableEntity, false,
			"Pertanyaan dan jawaban Silastik wajib diisi untuk mengakhiri tiket", nil)
		return
	}

	// parse tanggal_pelayanan
	var tanggalPelayanan interface{} = nil
	if req.TanggalPelayanan != "" {
		tgl, err := time.Parse("2006-01-02", req.TanggalPelayanan)
		if err != nil {
			utils.JSONResponse(w, http.StatusUnprocessableEntity, false,
				"Format tanggal_pelayanan tidak valid (gunakan YYYY-MM-DD)", nil)
			return
		}
		tanggalPelayanan = tgl
	}

	res, err := database.DB.Exec(`
		UPDATE tiket
		SET no_hp               = $1,
		    nama_pemohon        = $2,
		    kategori            = $3,
		    subjek              = $4,
		    uraian              = $5,
		    status              = $6,
		    catatan             = $7,
		    tanggal_pelayanan   = $8,
		    petugas_id          = $9,
		    silastik_pertanyaan = $10,
		    silastik_jawaban    = $11,
		    updated_at          = NOW()
		WHERE nomor_tiket = $12`,
		req.NoHP, req.NamaPemohon, req.Kategori, req.Subjek, req.Uraian,
		status, req.Catatan, tanggalPelayanan, req.PetugasID,
		req.SilastikPertanyaan, req.SilastikJawaban,
		nomorTiket,
	)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal mengupdate tiket: "+err.Error(), nil)
		return
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		utils.JSONResponse(w, http.StatusNotFound, false, "Tiket tidak ditemukan", nil)
		return
	}

	t, err := scanTiket(database.DB.QueryRow(`SELECT `+tiketSelectCols+` WHERE t.nomor_tiket = $1`, nomorTiket))
	if err != nil {
		utils.JSONResponse(w, http.StatusOK, true, "Tiket berhasil diperbarui", nil)
		return
	}
	utils.JSONResponse(w, http.StatusOK, true, "Tiket berhasil diperbarui", t.ToJSON())
}

// DELETE /api/admin/tiket/{nomor}  (protected)
func HapusTiket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	nomorTiket := parts[len(parts)-1]

	res, err := database.DB.Exec("DELETE FROM tiket WHERE nomor_tiket = $1", nomorTiket)
	if err != nil {
		utils.JSONResponse(w, http.StatusInternalServerError, false, "Gagal menghapus tiket", nil)
		return
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		utils.JSONResponse(w, http.StatusNotFound, false, "Tiket tidak ditemukan", nil)
		return
	}

	utils.JSONResponse(w, http.StatusOK, true, "Tiket berhasil dihapus", map[string]string{
		"nomor_tiket": nomorTiket,
	})
}
