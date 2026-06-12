package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func Connect(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("gagal membuka koneksi DB: %w", err)
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("gagal ping DB: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	DB = db
	log.Println("[DB] Terhubung ke PostgreSQL")
	return nil
}

func Migrate() error {
	queries := []string{

		// ── Tabel pengguna ──────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS pengguna (
			id           SERIAL PRIMARY KEY,
			no_hp        VARCHAR(20)  NOT NULL UNIQUE,
			nama         VARCHAR(255) NOT NULL,
			jk           CHAR(1)      NOT NULL CHECK(jk IN ('L','P')),
			tahun_lahir  INTEGER      NOT NULL,
			pendidikan   VARCHAR(100) NOT NULL,
			pekerjaan    VARCHAR(100) NOT NULL,
			tempat_kerja VARCHAR(255) NOT NULL DEFAULT '',
			email        VARCHAR(255) NOT NULL,
			created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)`,

		// ── Tabel petugas_pst ────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS petugas_pst (
			id           SERIAL PRIMARY KEY,
			nama_lengkap VARCHAR(255) NOT NULL,
			nama_alias   VARCHAR(100) NOT NULL DEFAULT '',
			nip          VARCHAR(30)  NOT NULL DEFAULT '',
			aktif        BOOLEAN      NOT NULL DEFAULT TRUE,
			created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)`,

		// ── Tabel tiket ──────────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS tiket (
			id                SERIAL PRIMARY KEY,
			nomor_tiket       VARCHAR(30)  NOT NULL UNIQUE,
			no_hp             VARCHAR(20)  NOT NULL,
			nama_pemohon      VARCHAR(255) NOT NULL,
			kategori          VARCHAR(100) NOT NULL,
			subjek            VARCHAR(500) NOT NULL,
			uraian            TEXT         NOT NULL,
			status            VARCHAR(20)  NOT NULL DEFAULT 'baru'
			                               CHECK(status IN ('baru','diproses','selesai','ditolak')),
			catatan     TEXT         NOT NULL DEFAULT '',
			tanggal_pelayanan DATE,
			petugas_id        INTEGER      REFERENCES petugas_pst(id) ON DELETE SET NULL,
			silastik_pertanyaan TEXT       NOT NULL DEFAULT '',
			silastik_jawaban    TEXT       NOT NULL DEFAULT '',
			created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)`,

		// ── Tabel admin ──────────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS admin (
			id            SERIAL PRIMARY KEY,
			username      VARCHAR(100) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			nama_lengkap  VARCHAR(255) NOT NULL DEFAULT '',
			email         VARCHAR(255) NOT NULL DEFAULT '',
			created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)`,

		// ── Migrasi kolom baru (idempotent: ADD COLUMN IF NOT EXISTS) ────────
		`ALTER TABLE tiket ADD COLUMN IF NOT EXISTS tanggal_pelayanan DATE`,
		`ALTER TABLE tiket ADD COLUMN IF NOT EXISTS petugas_id INTEGER REFERENCES petugas_pst(id) ON DELETE SET NULL`,
		`ALTER TABLE tiket ADD COLUMN IF NOT EXISTS silastik_pertanyaan TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE tiket ADD COLUMN IF NOT EXISTS silastik_jawaban TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE petugas_pst ADD COLUMN IF NOT EXISTS aktif BOOLEAN NOT NULL DEFAULT TRUE`,

		// ── Index ─────────────────────────────────────────────────────────────
		`CREATE INDEX IF NOT EXISTS idx_tiket_no_hp      ON tiket(no_hp)`,
		`CREATE INDEX IF NOT EXISTS idx_tiket_status     ON tiket(status)`,
		`CREATE INDEX IF NOT EXISTS idx_tiket_created    ON tiket(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_tiket_petugas    ON tiket(petugas_id)`,
		`CREATE INDEX IF NOT EXISTS idx_tiket_tgl_pel    ON tiket(tanggal_pelayanan)`,
		`CREATE INDEX IF NOT EXISTS idx_pengguna_no_hp   ON pengguna(no_hp)`,
		`CREATE INDEX IF NOT EXISTS idx_pengguna_email   ON pengguna(email)`,
		`CREATE INDEX IF NOT EXISTS idx_petugas_nip      ON petugas_pst(nip)`,

		// ── Trigger updated_at ───────────────────────────────────────────────
		`CREATE OR REPLACE FUNCTION update_updated_at()
		 RETURNS TRIGGER AS $$
		 BEGIN
		   NEW.updated_at = NOW();
		   RETURN NEW;
		 END;
		 $$ LANGUAGE plpgsql`,

		`DO $$ BEGIN
		   IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_pengguna_updated_at') THEN
		     CREATE TRIGGER trg_pengguna_updated_at
		     BEFORE UPDATE ON pengguna
		     FOR EACH ROW EXECUTE FUNCTION update_updated_at();
		   END IF;
		 END $$`,

		`DO $$ BEGIN
		   IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_tiket_updated_at') THEN
		     CREATE TRIGGER trg_tiket_updated_at
		     BEFORE UPDATE ON tiket
		     FOR EACH ROW EXECUTE FUNCTION update_updated_at();
		   END IF;
		 END $$`,

		`DO $$ BEGIN
		   IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trg_petugas_updated_at') THEN
		     CREATE TRIGGER trg_petugas_updated_at
		     BEFORE UPDATE ON petugas_pst
		     FOR EACH ROW EXECUTE FUNCTION update_updated_at();
		   END IF;
		 END $$`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			return fmt.Errorf("migrasi gagal: %w\nQuery: %.120s...", err, q)
		}
	}

	log.Println("[DB] Migrasi selesai")
	return nil
}
