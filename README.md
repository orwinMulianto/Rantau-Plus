# Rantau+

Rantau+ membantu mahasiswa rantau memantau keuangan sampai periode pemasukan berikutnya. Workspace berisi aplikasi Flutter, API Go/Gin, PostgreSQL, dan AI Service Node.js.

| Komponen | Tanggung jawab |
| --- | --- |
| `mobile/lib/` | Antarmuka pengguna dan konsumsi API Go |
| `backend/internal/domain/finance/` | Saldo tersedia, Safe-to-Spend, Financial Runway, proyeksi saldo, dan status risiko |
| `backend/internal/integration/assistant/` | Klien dari API Go ke AI Service |
| `backend/migrations/` | Skema awal PostgreSQL dan SQL rollback |
| `ai-service/src/skills/finance/` | Skill penjelasan konsep keuangan Rantau+ |
| `ai-service/src/rag/` | Pencarian dokumen dan penyusunan jawaban dengan konteks |
| `ai-service/src/providers/` | Adapter model opsional untuk API Chat Completions |
| `ai-service/knowledge/` | Dokumen referensi untuk tiap skill |

## Jalankan AI Companion lokal

Dari root workspace, jalankan AI Service di terminal pertama. Node.js 22 atau lebih baru cukup untuk demo lokal.

```powershell
Set-Location ai-service
npm start
```

Jalankan API di terminal kedua dengan Go 1.25 atau lebih baru.

```powershell
Set-Location backend
$env:RANTAU_DEMO = 'true'
$env:AI_SERVICE_URL = 'http://127.0.0.1:8081'
go run ./cmd/api
```

Jalankan contoh permintaan melalui Go.

```powershell
Invoke-RestMethod -Method Post `
  -Uri 'http://127.0.0.1:8080/api/v1/demo/assistant/chat' `
  -ContentType 'application/json' `
  -Body '{"skill":"finance","message":"Apa itu Safe-to-Spend?"}' |
  ConvertTo-Json -Depth 6
```

AI Service mencari cuplikan dokumen berdasarkan kata dalam pertanyaan. Respons memuat sumber yang diambil. Nilai `mode` adalah `retrieval` ketika jawaban menampilkan cuplikan, `rag` ketika model menghasilkan jawaban dengan konteks dokumen, atau `no_context` jika topik belum ditemukan.

Untuk generasi dengan model, atur `AI_LLM_URL`, `AI_LLM_MODEL`, dan `AI_LLM_API_KEY` di lingkungan AI Service sebelum memulainya. Adapter menerima endpoint HTTPS yang menggunakan kontrak Chat Completions. Panduan kontrak tersedia di [dokumentasi API](https://developers.openai.com/api/reference/resources/chat). Demo lokal memakai panduan Rantau+ dan pertanyaan contoh. Koneksi database, kredensial pengguna, dan transaksi pribadi berada di jalur terpisah dari korpus dan jalur demo AI.

## Tambah skill atau hubungkan data personal

1. Siapkan korpus milik skill di `ai-service/knowledge/` dengan ID sumber yang stabil. Daftarkan skill di `ai-service/src/server.js` dan gunakan kembali retriever serta resolver jawaban. Jika volume dokumen meningkat, retriever dapat diganti dengan pencarian embedding tanpa mengubah kontrak HTTP Go.
2. Untuk pertanyaan personal seperti “Berapa Safe-to-Spend saya?”, buat use case Go yang memvalidasi sesi dan mengambil hasil financial engine. AI Service dapat memakai hasil terstruktur tersebut sebagai konteks jawaban setelah jalur privat tersedia. Endpoint demo saat ini menjelaskan konsep dan terpisah dari `GET /api/v1/demo/finance/summary`.
3. Flutter menggunakan Go untuk memanggil AI Companion ketika layar chat ditambahkan. AI Service berjalan di loopback dan memiliki endpoint internal `POST /internal/v1/chat`.

## Dashboard Flutter dan PostgreSQL

Endpoint `GET /api/v1/demo/finance/summary` berjalan dengan data sintetis. Contoh ini menghasilkan Rp1.300.000 dana tersedia, Rp65.000 per hari Safe-to-Spend, serta 16,25 hari Financial Runway.

Untuk menyiapkan platform Flutter sekali dari root proyek, jalankan `flutter create --empty --platforms=android,ios --project-name rantau_plus --org id.rantauplus mobile`. Jalankan `flutter pub get` dari `mobile/` setelahnya. Tambahkan elemen `<application android:usesCleartextTraffic="true" />` ke manifest Android debug untuk mengizinkan demo HTTP lokal, lalu jalankan `flutter run` di emulator Android. Alamat backend dapat diatur melalui `--dart-define=API_BASE_URL=...`.

Pengembangan dengan PostgreSQL memakai `docker compose up -d --wait db`. Pada database baru, jalankan migrasi `backend/migrations/000001_initial_schema.up.sql` menggunakan `psql`. Konfigurasi koneksi Go melalui `DATABASE_URL`.

## Pemeriksaan

Jalankan `npm test` dari `ai-service/`, serta `go test ./...` dan `go vet ./...` dari `backend/`.
