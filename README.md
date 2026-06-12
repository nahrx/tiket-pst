# Tiket PST BPS Kalimantan Timur — Backend API

REST API untuk sistem tiket layanan PST BPS Kalimantan Timur.  
Dibangun dengan **Go 1.22** · **PostgreSQL** · **JWT Auth**

---

## Quick Start

### Cara 1 — Docker Compose (paling mudah)

```bash
# Start PostgreSQL + API sekaligus
docker compose up -d --build

# API siap di http://localhost:8080
# PostgreSQL di localhost:5432
```

### Cara 2 — Manual (PostgreSQL sudah berjalan)

```bash
# 1. Siapkan database PostgreSQL
psql -U postgres -c "CREATE DATABASE tiket_bps;"

# 2. Copy & edit konfigurasi
cp .env.example .env
# Edit .env: DB_PASSWORD, JWT_SECRET, dll.

# 3. Jalankan
go run main.go
# atau: make run
```

---

## Struktur Project

```
tiket-bps-api/
├── main.go              # Entry point, routing, seed admin
├── config/
│   └── config.go        # Konfigurasi dari environment
├── database/
│   └── db.go            # Koneksi PostgreSQL + migrasi otomatis
├── models/
│   └── models.go        # Struct data & DTO
├── handlers/
│   ├── auth.go          # Login, profil admin
│   ├── pengguna.go      # Cek HP, registrasi
│   ├── tiket.go         # CRUD tiket
│   └── stats.go         # Statistik dashboard
├── middleware/
│   └── middleware.go    # CORS, Logger, JWT Auth
├── utils/
│   └── utils.go         # Helper: validasi, response, generate nomor tiket
├── public/              # (opsional) Frontend HTML statis
├── .env.example
├── docker-compose.yml   # PostgreSQL + API
├── Dockerfile
├── Makefile
└── README.md
```

---

## Environment Variables

| Variable           | Default                     | Keterangan                        |
|--------------------|-----------------------------|-----------------------------------|
| `APP_PORT`         | `8080`                      | Port server                       |
| `APP_ENV`          | `development`               | `development` / `production`      |
| `DB_HOST`          | `localhost`                 | Host PostgreSQL                   |
| `DB_PORT`          | `5432`                      | Port PostgreSQL                   |
| `DB_NAME`          | `tiket_bps`                 | Nama database                     |
| `DB_USER`          | `postgres`                  | Username PostgreSQL               |
| `DB_PASSWORD`      | `postgres`                  | Password PostgreSQL               |
| `DB_SSLMODE`       | `disable`                   | `disable` / `require` / `verify-full` |
| `DATABASE_URL`     | *(kosong)*                  | Override semua DB field (format postgres://...) |
| `JWT_SECRET`       | *(default — ganti!)*        | Secret key untuk JWT              |
| `JWT_EXPIRY_HOURS` | `24`                        | Masa berlaku token (jam)          |
| `ADMIN_USERNAME`   | `admin`                     | Username admin default            |
| `ADMIN_PASSWORD`   | `Admin@BPSKaltim2025`       | Password admin default (di-hash)  |
| `ALLOWED_ORIGINS`  | `http://localhost:8080`     | CORS origins (pisah koma)         |
| `WA_NUMBER`        | *(kosong)*                  | Nomor WA notifikasi               |

---

## API Endpoints

### Public (tanpa autentikasi)

| Method | Endpoint              | Deskripsi                     |
|--------|-----------------------|-------------------------------|
| GET    | `/api/health`         | Health check                  |
| GET    | `/api/cek-hp`         | Cek apakah nomor HP terdaftar |
| POST   | `/api/register`       | Daftarkan pengguna baru       |
| POST   | `/api/tiket`          | Buat tiket layanan            |
| POST   | `/api/login`          | Login admin                   |

### Protected (butuh `Authorization: Bearer <token>`)

| Method | Endpoint                               | Deskripsi                  |
|--------|----------------------------------------|----------------------------|
| GET    | `/api/admin/profile`                   | Profil admin login         |
| GET    | `/api/admin/stats`                     | Statistik dashboard        |
| GET    | `/api/admin/tiket`                     | Daftar semua tiket         |
| GET    | `/api/admin/tiket/{nomor_tiket}`       | Detail satu tiket          |
| PATCH  | `/api/admin/tiket/{nomor_tiket}/status`| Update status tiket        |

---

## Contoh Request

### Cek HP
```bash
curl "http://localhost:8080/api/cek-hp?no_hp=081234567890"
```

### Registrasi
```bash
curl -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{
    "nama": "Ahmad Fauzi",
    "jk": "L",
    "tahun_lahir": 1990,
    "pendidikan": "D4/S1",
    "pekerjaan": "ASN/TNI/Polri",
    "tempat_bekerja": "Dinas Pertanian Kaltim",
    "email": "ahmad@email.com",
    "no_hp": "081234567890"
  }'
```

### Buat Tiket
```bash
curl -X POST http://localhost:8080/api/tiket \
  -H "Content-Type: application/json" \
  -d '{
    "no_hp": "081234567890",
    "nama": "Ahmad Fauzi",
    "kategori": "Permintaan Data",
    "subjek": "Data Kemiskinan Kaltim 2024",
    "uraian": "Saya membutuhkan data kemiskinan Kalimantan Timur tahun 2024..."
  }'
```

### Login Admin
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username": "admin", "password": "Admin@BPSKaltim2025"}'
```

### Update Status Tiket (protected)
```bash
curl -X PATCH http://localhost:8080/api/admin/tiket/BPS-KT-2025-123456/status \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"status": "diproses", "catatan_admin": "Sedang diproses oleh seksi statistik sosial"}'
```

---

## Format Response

Semua endpoint mengembalikan format JSON seragam:

```json
{
  "success": true,
  "message": "Deskripsi hasil",
  "data": { ... }
}
```

---

## Deploy dengan Docker Compose

```bash
# Development
docker compose up -d --build

# Production — buat file .env atau set environment langsung
JWT_SECRET=secret_panjang_min32 \
DB_PASSWORD=password_kuat \
APP_ENV=production \
docker compose up -d --build
```

---

## Serve Frontend dari Backend

```bash
mkdir -p public
cp tiket-bps-kaltim.html public/index.html
go run main.go
# buka http://localhost:8080
```

---

## Database Schema

### `pengguna`
| Kolom         | Tipe         | Keterangan                    |
|---------------|--------------|-------------------------------|
| id            | SERIAL       | Primary key                   |
| no_hp         | VARCHAR(20)  | Nomor HP (unik)               |
| nama          | VARCHAR(255) | Nama lengkap                  |
| jk            | CHAR(1)      | L / P                         |
| tahun_lahir   | INTEGER      | Tahun lahir                   |
| pendidikan    | VARCHAR(100) | Pendidikan terakhir           |
| pekerjaan     | VARCHAR(100) | Pekerjaan                     |
| tempat_kerja  | VARCHAR(255) | Instansi / tempat bekerja     |
| email         | VARCHAR(255) | Email                         |
| created_at    | TIMESTAMPTZ  |                               |
| updated_at    | TIMESTAMPTZ  | Auto-update via trigger       |

### `tiket`
| Kolom          | Tipe         | Keterangan                      |
|----------------|--------------|---------------------------------|
| id             | SERIAL       | Primary key                     |
| nomor_tiket    | VARCHAR(30)  | Format BPS-KT-YYYY-XXXXXX       |
| no_hp          | VARCHAR(20)  | Nomor HP pemohon                |
| nama_pemohon   | VARCHAR(255) | Nama pemohon                    |
| kategori       | VARCHAR(100) | Permintaan Data/Konsultasi/dll  |
| subjek         | VARCHAR(500) | Judul permohonan                |
| uraian         | TEXT         | Detail permohonan               |
| status         | VARCHAR(20)  | baru/diproses/selesai/ditolak   |
| catatan_admin  | TEXT         | Catatan dari admin              |
| created_at     | TIMESTAMPTZ  |                                 |
| updated_at     | TIMESTAMPTZ  | Auto-update via trigger         |

### `admin`
| Kolom         | Tipe         | Keterangan        |
|---------------|--------------|-------------------|
| id            | SERIAL       | Primary key       |
| username      | VARCHAR(100) | Username (unik)   |
| password_hash | VARCHAR(255) | bcrypt hash       |
| nama_lengkap  | VARCHAR(255) | Nama admin        |
| email         | VARCHAR(255) | Email admin       |
| created_at    | TIMESTAMPTZ  |                   |
