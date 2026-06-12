package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"tiket-bps-api/config"
	"tiket-bps-api/database"
	"tiket-bps-api/handlers"
	"tiket-bps-api/middleware"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	// ── 1. Load .env jika ada ──────────────────────────────────────────────
	loadDotEnv()

	// ── 2. Konfigurasi ────────────────────────────────────────────────────
	config.Load()
	log.Printf("[APP] Tiket PST BPS Kaltim — env=%s port=%s", config.App.AppEnv, config.App.AppPort)

	// ── 3. Database ───────────────────────────────────────────────────────
	if err := database.Connect(config.App.DBDSN); err != nil {
		log.Fatalf("[FATAL] Koneksi DB gagal: %v", err)
	}
	if err := database.Migrate(); err != nil {
		log.Fatalf("[FATAL] Migrasi DB gagal: %v", err)
	}

	// ── 4. Seed admin default ─────────────────────────────────────────────
	seedAdmin()

	// ── 5. Router ─────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	// Public endpoints
	mux.HandleFunc("/api/cek-hp",    handlers.CekHP)
	mux.HandleFunc("/api/register",  handlers.Register)
	mux.HandleFunc("/api/tiket",     handlers.BuatTiket)
	mux.HandleFunc("/api/login",     handlers.Login)

	// Health check
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"ok","service":"Tiket PST BPS Kaltim"}`)
	})

	// Protected endpoints (butuh JWT)
	protected := http.NewServeMux()
	protected.HandleFunc("/api/admin/profile",    handlers.GetAdminProfile)
	protected.HandleFunc("/api/admin/stats",       handlers.GetStats)
	protected.HandleFunc("/api/admin/tiket",       routeTiketAdminRoot)
	protected.HandleFunc("/api/admin/tiket/",      routeTiketAdmin)
	protected.HandleFunc("/api/admin/petugas",     routePetugasAdminRoot)
	protected.HandleFunc("/api/admin/petugas/",    routePetugasAdmin)
	protected.HandleFunc("/api/admin/pengguna",    routePenggunaAdminRoot)
	protected.HandleFunc("/api/admin/pengguna/",   routePenggunaAdmin)

	mux.Handle("/api/admin/", middleware.JWTAuth(protected))

	// Serve frontend HTML (opsional — sajikan file statis)
	mux.Handle("/", http.FileServer(http.Dir("./public")))

	// ── 6. Stack middleware ───────────────────────────────────────────────
	handler := middleware.Logger(middleware.CORS(mux))

	// ── 7. Start server ───────────────────────────────────────────────────
	addr := ":" + config.App.AppPort
	log.Printf("[APP] Server berjalan di http://localhost%s", addr)
	log.Printf("[APP] API Docs:")
	log.Printf("[APP]   GET    /api/health")
	log.Printf("[APP]   GET    /api/cek-hp?no_hp=08xx")
	log.Printf("[APP]   POST   /api/register")
	log.Printf("[APP]   POST   /api/tiket              ← publik")
	log.Printf("[APP]   POST   /api/login")
	log.Printf("[APP]   ---  Protected (JWT)  ---")
	log.Printf("[APP]   GET    /api/admin/stats")
	log.Printf("[APP]   GET    /api/admin/profile")
	log.Printf("[APP]   GET    /api/admin/tiket[?status&petugas_id&q&page&limit]")
	log.Printf("[APP]   POST   /api/admin/tiket         ← admin buat tiket")
	log.Printf("[APP]   GET    /api/admin/tiket/{nomor}")
	log.Printf("[APP]   PUT    /api/admin/tiket/{nomor} ← admin edit tiket")
	log.Printf("[APP]   DELETE /api/admin/tiket/{nomor} ← admin hapus tiket")
	log.Printf("[APP]   PATCH  /api/admin/tiket/{nomor}/status")
	log.Printf("[APP]   GET    /api/admin/petugas")
	log.Printf("[APP]   POST   /api/admin/petugas")
	log.Printf("[APP]   GET    /api/admin/petugas/{id}")
	log.Printf("[APP]   PUT    /api/admin/petugas/{id}")
	log.Printf("[APP]   DELETE /api/admin/petugas/{id}")

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("[FATAL] Server gagal start: %v", err)
	}
}
// routeTiketAdminRoot menangani /api/admin/tiket (tanpa trailing slash)
// GET → list, POST → admin buat tiket
func routePetugasAdminRoot(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handlers.ListPetugas(w, r)
	case http.MethodPost:
		handlers.TambahPetugas(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"success":false,"message":"Method tidak diizinkan"}`, http.StatusMethodNotAllowed)
	}
}

// routeTiketAdminRoot menangani /api/admin/tiket (tanpa trailing slash)
// GET → list, POST → admin buat tiket
func routeTiketAdminRoot(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handlers.ListTiket(w, r)
	case http.MethodPost:
		handlers.AdminBuatTiket(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"success":false,"message":"Method tidak diizinkan"}`, http.StatusMethodNotAllowed)
	}
}

// routeTiketAdmin menangani sub-path /api/admin/tiket/*
func routeTiketAdmin(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/tiket/")
	parts := strings.Split(strings.Trim(path, "/"), "/")

	switch {
	case len(parts) == 1 && parts[0] != "":
		// /api/admin/tiket/{nomor} — GET, PUT, DELETE
		switch r.Method {
		case http.MethodGet:
			handlers.DetailTiket(w, r)
		case http.MethodPut:
			handlers.EditTiket(w, r)
		case http.MethodDelete:
			handlers.HapusTiket(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"success":false,"message":"Method tidak diizinkan"}`, http.StatusMethodNotAllowed)
		}
	case len(parts) == 2 && parts[1] == "status":
		// /api/admin/tiket/{nomor}/status — PATCH
		handlers.UpdateStatusTiket(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"success":false,"message":"Endpoint tidak ditemukan"}`, http.StatusNotFound)
	}
}

// routePetugasAdmin menangani sub-path /api/admin/petugas/{id}
func routePetugasAdmin(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/petugas/")
	id := strings.Trim(path, "/")

	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"success":false,"message":"ID petugas diperlukan"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handlers.DetailPetugas(w, r)
	case http.MethodPut:
		handlers.UpdatePetugas(w, r)
	case http.MethodDelete:
		handlers.HapusPetugas(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"success":false,"message":"Method tidak diizinkan"}`, http.StatusMethodNotAllowed)
	}
}

// routePenggunaAdminRoot menangani /api/admin/pengguna (tanpa trailing slash)
// GET → list, POST → admin tambah pengguna
func routePenggunaAdminRoot(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		handlers.ListPengguna(w, r)
	case http.MethodPost:
		handlers.TambahPengguna(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"success":false,"message":"Method tidak diizinkan"}`, http.StatusMethodNotAllowed)
	}
}

// routePenggunaAdmin menangani sub-path /api/admin/pengguna/{id}
func routePenggunaAdmin(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/admin/pengguna/")
	id := strings.Trim(path, "/")

	if id == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"success":false,"message":"ID pengguna diperlukan"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		handlers.DetailPengguna(w, r)
	case http.MethodPut:
		handlers.UpdatePengguna(w, r)
	case http.MethodDelete:
		handlers.HapusPengguna(w, r)
	default:
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"success":false,"message":"Method tidak diizinkan"}`, http.StatusMethodNotAllowed)
	}
}

// seedAdmin membuat admin default jika belum ada
func seedAdmin() {
	var count int
	database.DB.QueryRow("SELECT COUNT(*) FROM admin").Scan(&count)
	if count > 0 {
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(config.App.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("[FATAL] Gagal hash password admin: %v", err)
	}

	_, err = database.DB.Exec(`
		INSERT INTO admin (username, password_hash, nama_lengkap, email)
		VALUES ($1, $2, $3, $4)`,
		config.App.AdminUsername,
		string(hash),
		"Administrator BPS Kaltim",
		"admin@bpskaltim.go.id",
	)
	if err != nil {
		log.Fatalf("[FATAL] Gagal seed admin: %v", err)
	}
	log.Printf("[APP] Admin default dibuat — username: %s", config.App.AdminUsername)
}

// loadDotEnv membaca file .env secara manual (tanpa library eksternal)
func loadDotEnv() {
	data, err := os.ReadFile(".env")
	if err != nil {
		return // .env tidak wajib
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// hapus komentar inline
		if idx := strings.Index(val, " #"); idx > 0 {
			val = strings.TrimSpace(val[:idx])
		}
		if key != "" && val != "" {
			os.Setenv(key, val)
		}
	}
	log.Println("[APP] File .env dimuat")
}
