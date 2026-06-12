package utils

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"tiket-bps-api/models"
)

// ─── HTTP HELPERS ────────────────────────────────────────────────────────────

func JSONResponse(w http.ResponseWriter, statusCode int, success bool, message string, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := models.APIResponse{
		Success: success,
		Message: message,
		Data:    data,
	}
	json.NewEncoder(w).Encode(resp)
}

func DecodeJSON(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body kosong")
	}
	return json.NewDecoder(r.Body).Decode(dst)
}

// ─── NOMOR TIKET ─────────────────────────────────────────────────────────────

func GenerateNomorTiket() string {
	year := time.Now().Year()
	random := rand.Intn(900000) + 100000
	return fmt.Sprintf("BPS-KT-%d-%d", year, random)
}

// ─── VALIDASI ────────────────────────────────────────────────────────────────

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var hpRegex = regexp.MustCompile(`^0[0-9]{8,13}$`)

func ValidateEmail(email string) bool {
	return emailRegex.MatchString(strings.TrimSpace(email))
}

func ValidateHP(hp string) bool {
	return hpRegex.MatchString(strings.TrimSpace(hp))
}

func ValidateRegisterRequest(req *models.RegisterRequest) error {
	req.Nama = strings.TrimSpace(req.Nama)
	req.Email = strings.TrimSpace(req.Email)
	req.NoHP = strings.TrimSpace(req.NoHP)
	req.Pekerjaan = strings.TrimSpace(req.Pekerjaan)
	req.Pendidikan = strings.TrimSpace(req.Pendidikan)
	req.TempatKerja = strings.TrimSpace(req.TempatKerja)

	if req.Nama == "" {
		return fmt.Errorf("nama wajib diisi")
	}
	if len(req.Nama) < 2 {
		return fmt.Errorf("nama minimal 2 karakter")
	}
	if req.JK != "L" && req.JK != "P" {
		return fmt.Errorf("jenis kelamin harus L atau P")
	}
	currentYear := time.Now().Year()
	if req.TahunLahir < 1950 || req.TahunLahir > currentYear-10 {
		return fmt.Errorf("tahun lahir tidak valid")
	}
	if req.Pendidikan == "" {
		return fmt.Errorf("pendidikan wajib diisi")
	}
	if req.Pekerjaan == "" {
		return fmt.Errorf("pekerjaan wajib diisi")
	}
	if req.TempatKerja == "" {
		return fmt.Errorf("tempat bekerja wajib diisi")
	}
	if !ValidateEmail(req.Email) {
		return fmt.Errorf("format email tidak valid")
	}
	if !ValidateHP(req.NoHP) {
		return fmt.Errorf("nomor HP tidak valid (format: 08xxxxxxxx)")
	}
	return nil
}

func ValidateTiketRequest(req *models.TiketRequest) error {
	req.NoHP = strings.TrimSpace(req.NoHP)
	req.Nama = strings.TrimSpace(req.Nama)
	req.Kategori = strings.TrimSpace(req.Kategori)
	req.Subjek = strings.TrimSpace(req.Subjek)
	req.Uraian = strings.TrimSpace(req.Uraian)

	validKategori := map[string]bool{
		"Permintaan Data": true,
		"Konsultasi":      true,
		"Pengaduan":       true,
		"Lainnya":         true,
	}
	if !validKategori[req.Kategori] {
		return fmt.Errorf("kategori tidak valid")
	}
	if req.Subjek == "" {
		return fmt.Errorf("subjek wajib diisi")
	}
	if len(req.Subjek) < 5 {
		return fmt.Errorf("subjek terlalu pendek")
	}
	if req.Uraian == "" {
		return fmt.Errorf("uraian wajib diisi")
	}
	if len(req.Uraian) < 10 {
		return fmt.Errorf("uraian terlalu pendek")
	}
	return nil
}
