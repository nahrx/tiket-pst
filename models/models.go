package models

import (
	"database/sql"
	"time"
)

// ─── PENGGUNA ────────────────────────────────────────────────────────────────

type Pengguna struct {
	ID          int64     `json:"id"`
	NoHP        string    `json:"no_hp"`
	Nama        string    `json:"nama"`
	JK          string    `json:"jk"`
	TahunLahir  int       `json:"tahun_lahir"`
	Pendidikan  string    `json:"pendidikan"`
	Pekerjaan   string    `json:"pekerjaan"`
	TempatKerja string    `json:"tempat_bekerja"`
	Email       string    `json:"email"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ─── PETUGAS PST ─────────────────────────────────────────────────────────────

type PetugasPST struct {
	ID          int64     `json:"id"`
	NamaLengkap string    `json:"nama_lengkap"`
	NamaAlias   string    `json:"nama_alias"`
	NIP         string    `json:"nip"`
	Aktif       bool      `json:"aktif"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ─── TIKET ───────────────────────────────────────────────────────────────────

type Tiket struct {
	ID               int64          `json:"id"`
	NomorTiket       string         `json:"nomor_tiket"`
	NoHP             string         `json:"no_hp"`
	NamaPemohon      string         `json:"nama_pemohon"`
	Kategori         string         `json:"kategori"`
	Subjek           string         `json:"subjek"`
	Uraian           string         `json:"uraian"`
	Status           string         `json:"status"` // baru | diproses | selesai | ditolak
	Catatan     string         `json:"catatan,omitempty"`
	TanggalPelayanan sql.NullTime   `json:"tanggal_pelayanan"` // nullable
	PetugasID        sql.NullInt64  `json:"petugas_id"`        // FK ke petugas_pst, nullable
	PetugasNama      sql.NullString `json:"petugas_nama"`      // joined dari petugas_pst
	SilastikPertanyaan string       `json:"silastik_pertanyaan"`
	SilastikJawaban    string       `json:"silastik_jawaban"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// TiketJSON adalah versi Tiket yang JSON-serializable (tanpa sql.Null*)
type TiketJSON struct {
	ID               int64      `json:"id"`
	NomorTiket       string     `json:"nomor_tiket"`
	NoHP             string     `json:"no_hp"`
	NamaPemohon      string     `json:"nama_pemohon"`
	Kategori         string     `json:"kategori"`
	Subjek           string     `json:"subjek"`
	Uraian           string     `json:"uraian"`
	Status           string     `json:"status"`
	Catatan     string     `json:"catatan"`
	TanggalPelayanan *time.Time `json:"tanggal_pelayanan"` // null kalau belum diisi
	PetugasID        *int64     `json:"petugas_id"`        // null kalau belum ditugaskan
	PetugasNama      *string    `json:"petugas_nama"`      // null kalau belum ditugaskan
	SilastikPertanyaan string   `json:"silastik_pertanyaan"`
	SilastikJawaban    string   `json:"silastik_jawaban"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// ToJSON konversi Tiket (dengan sql.Null*) ke TiketJSON
func (t *Tiket) ToJSON() TiketJSON {
	tj := TiketJSON{
		ID:           t.ID,
		NomorTiket:   t.NomorTiket,
		NoHP:         t.NoHP,
		NamaPemohon:  t.NamaPemohon,
		Kategori:     t.Kategori,
		Subjek:       t.Subjek,
		Uraian:       t.Uraian,
		Status:       t.Status,
		Catatan: t.Catatan,
		SilastikPertanyaan: t.SilastikPertanyaan,
		SilastikJawaban:    t.SilastikJawaban,
		CreatedAt:    t.CreatedAt,
		UpdatedAt:    t.UpdatedAt,
	}
	if t.TanggalPelayanan.Valid {
		tj.TanggalPelayanan = &t.TanggalPelayanan.Time
	}
	if t.PetugasID.Valid {
		tj.PetugasID = &t.PetugasID.Int64
	}
	if t.PetugasNama.Valid {
		tj.PetugasNama = &t.PetugasNama.String
	}
	return tj
}

// ─── ADMIN ───────────────────────────────────────────────────────────────────

type Admin struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	NamaLengkap  string    `json:"nama_lengkap"`
	Email        string    `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
}

// ─── REQUEST / RESPONSE DTOs ─────────────────────────────────────────────────

type RegisterRequest struct {
	Nama        string `json:"nama"`
	JK          string `json:"jk"`
	TahunLahir  int    `json:"tahun_lahir"`
	Pendidikan  string `json:"pendidikan"`
	Pekerjaan   string `json:"pekerjaan"`
	TempatKerja string `json:"tempat_bekerja"`
	Email       string `json:"email"`
	NoHP        string `json:"no_hp"`
}

type TiketRequest struct {
	NoHP      string `json:"no_hp"`
	Nama      string `json:"nama"`
	Kategori  string `json:"kategori"`
	Subjek    string `json:"subjek"`
	Uraian    string `json:"uraian"`
}

// UpdateTiketRequest dipakai admin untuk update status + assignment petugas
type UpdateTiketRequest struct {
	Status           string  `json:"status"`
	Catatan          string  `json:"catatan"`
	TanggalPelayanan string  `json:"tanggal_pelayanan"` // format: "2006-01-02" atau kosong
	PetugasID        *int64  `json:"petugas_id"`        // null = hapus assignment
}

// EditTiketRequest dipakai admin untuk edit isi tiket (subjek, uraian, kategori, dll)
type EditTiketRequest struct {
	NoHP             string  `json:"no_hp"`
	NamaPemohon      string  `json:"nama_pemohon"`
	Kategori         string  `json:"kategori"`
	Subjek           string  `json:"subjek"`
	Uraian           string  `json:"uraian"`
	Status           string  `json:"status"`
	Catatan          string  `json:"catatan"`
	TanggalPelayanan string  `json:"tanggal_pelayanan"`
	PetugasID        *int64  `json:"petugas_id"`
	SilastikPertanyaan string `json:"silastik_pertanyaan"`
	SilastikJawaban    string `json:"silastik_jawaban"`
}

// AdminTiketRequest dipakai admin membuat tiket atas nama pemohon
type AdminTiketRequest struct {
	NoHP        string `json:"no_hp"`
	NamaPemohon string `json:"nama_pemohon"`
	Kategori    string `json:"kategori"`
	Subjek      string `json:"subjek"`
	Uraian      string `json:"uraian"`
}

type PetugasRequest struct {
	NamaLengkap string `json:"nama_lengkap"`
	NamaAlias   string `json:"nama_alias"`
	NIP         string `json:"nip"`
	Aktif       bool   `json:"aktif"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	Admin *Admin `json:"admin"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
