package handlers

import (
	"net/http"
	"time"

	"tiket-bps-api/database"
	"tiket-bps-api/utils"
)

// GET /api/admin/stats  (protected)
func GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.JSONResponse(w, http.StatusMethodNotAllowed, false, "Method tidak diizinkan", nil)
		return
	}

	today := time.Now().Format("2006-01-02")

	var (
		totalTiket      int
		tiketHariIni    int
		tiketBaru       int
		tiketDiproses   int
		tiketSelesai    int
		tiketDitolak    int
		totalPengguna   int
	)

	database.DB.QueryRow("SELECT COUNT(*) FROM tiket").Scan(&totalTiket)
	database.DB.QueryRow("SELECT COUNT(*) FROM tiket WHERE created_at::date = $1", today).Scan(&tiketHariIni)
	database.DB.QueryRow("SELECT COUNT(*) FROM tiket WHERE status = 'baru'").Scan(&tiketBaru)
	database.DB.QueryRow("SELECT COUNT(*) FROM tiket WHERE status = 'diproses'").Scan(&tiketDiproses)
	database.DB.QueryRow("SELECT COUNT(*) FROM tiket WHERE status = 'selesai'").Scan(&tiketSelesai)
	database.DB.QueryRow("SELECT COUNT(*) FROM tiket WHERE status = 'ditolak'").Scan(&tiketDitolak)
	database.DB.QueryRow("SELECT COUNT(*) FROM pengguna").Scan(&totalPengguna)

	// statistik per kategori
	rows, err := database.DB.Query(`
		SELECT kategori, COUNT(*) as jumlah
		FROM tiket
		GROUP BY kategori
		ORDER BY jumlah DESC`)
	perKategori := []map[string]interface{}{}
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var kat string
			var jml int
			rows.Scan(&kat, &jml)
			perKategori = append(perKategori, map[string]interface{}{
				"kategori": kat,
				"jumlah":   jml,
			})
		}
	}

	utils.JSONResponse(w, http.StatusOK, true, "OK", map[string]interface{}{
		"total_tiket":     totalTiket,
		"tiket_hari_ini":  tiketHariIni,
		"tiket_baru":      tiketBaru,
		"tiket_diproses":  tiketDiproses,
		"tiket_selesai":   tiketSelesai,
		"tiket_ditolak":   tiketDitolak,
		"total_pengguna":  totalPengguna,
		"per_kategori":    perKategori,
	})
}
