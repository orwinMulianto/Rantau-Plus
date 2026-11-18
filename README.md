# Rantau+

Rantau+ membantu mahasiswa rantau memantau keuangan sampai periode pemasukan berikutnya. Workspace berisi aplikasi Flutter, API Go/Gin, PostgreSQL, dan AI Service Node.js.

| Komponen | Tanggung jawab |
| --- | --- |
| `mobile/lib/` | Antarmuka pengguna dan konsumsi API Go |
| `backend/internal/domain/finance/` | Saldo tersedia, Safe-to-Spend, Financial Runway, proyeksi saldo, dan status risiko |
| `backend/internal/integration/assistant/` | Klien dari API Go ke AI Service |
| `backend/database/` | Koneksi, model GORM, dan migrasi skema PostgreSQL |
| `ai-service/src/skills/finance/` | Skill penjelasan konsep keuangan Rantau+ |
| `ai-service/src/rag/` | Pencarian dokumen dan penyusunan jawaban dengan konteks |
| `ai-service/src/providers/` | Adapter model opsional untuk API Chat Completions |
| `ai-service/knowledge/` | Dokumen referensi untuk tiap skill |

## Struktur backend

Backend mengelompokkan kode per fitur dan memakai pola Controller–Service–Repository dari [go-gin-clean-starter](https://github.com/Caknoooo/go-gin-clean-starter/).

| Lokasi di `backend/` | Tanggung jawab |
| --- | --- |
| `cmd/api/` dan `internal/httpserver/` | Merangkai dependensi, menjalankan Gin, dan mendaftarkan route umum seperti `/healthz` dan `/readyz` |
| `internal/modules/finance/` | Financial Setup, dashboard pribadi, dan ringkasan demo. Perhitungan memakai `internal/domain/finance/`. |
| `internal/modules/assistant/` | Route, controller, dan service AI Companion. Service memvalidasi pesan dan memanggil klien di `internal/integration/assistant/`. |
| `internal/modules/auth/` | Route, service, repository, dan middleware untuk akun, sesi, dan pemulihan password. |
| `database/entities/` | Model GORM untuk tabel PostgreSQL. Nominal disimpan sebagai `int64` dalam rupiah bulat. |
| `database/postgres/` | Membuka koneksi GORM dengan driver PostgreSQL |
| `database/migration.go` dan `database/schema_steps.go` | Menjalankan `AutoMigrate` dan perubahan khusus PostgreSQL per versi lewat `cmd/migrate/` |

Controller membaca request dan mengirim respons JSON. Service mengatur alur fitur. Repository autentikasi menyimpan akun dan sesi di PostgreSQL.

## Autentikasi

Backend menyediakan pendaftaran dengan verifikasi email, login, pengelolaan sesi, ganti password, dan pemulihan akun. Panduan menjalankan SMTP lokal dan kontrak API untuk Flutter ada di [autentikasi-backend.md](docs/autentikasi-backend.md).

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

## Dashboard Flutter

Pilih **Financial Setup** di dashboard, lalu masuk atau daftar dengan akun Rantau+. Isi saldo awal, tanggal pemasukan berikutnya, dana cadangan, dan kewajiban utama. Pilih **Lihat pratinjau** untuk memeriksa dana tersedia dan batas harian, lalu **Simpan pengaturan** untuk kembali ke dashboard pribadi. Pengaturan disimpan di PostgreSQL. Sesi masuk Flutter digunakan selama aplikasi berjalan.

Aktifkan autentikasi dengan [panduan SMTP lokal](docs/autentikasi-backend.md) dan terapkan migrasi `000004_financial_setup` lewat `go run ./cmd/migrate`. API pribadi memakai bearer token.

| Endpoint | Kegunaan |
| --- | --- |
| `GET /api/v1/finance/setup` | Mengambil tanggal hari ini menurut zona waktu akun dan pengaturan periode aktif |
| `POST /api/v1/finance/setup/preview` | Menghitung dampak perubahan pengaturan tanpa menulis ke database |
| `PUT /api/v1/finance/setup` | Menyimpan periode dan daftar kewajiban setup dalam satu transaksi database |
| `GET /api/v1/finance/summary` | Mengambil ringkasan periode aktif untuk dashboard pribadi |

Jatuh tempo yang dikosongkan mengikuti tanggal pemasukan berikutnya. Perhitungan juga menyisihkan kewajiban tercatat lain yang belum dibayar dan jatuh tempo paling lambat pada tanggal itu.

Endpoint `GET /api/v1/demo/finance/summary` berjalan dengan data sintetis. Contoh ini menghasilkan Rp1.300.000 dana tersedia, Rp65.000 per hari Safe-to-Spend, serta 16,25 hari Financial Runway.

Untuk menyiapkan platform Flutter sekali dari root proyek, jalankan `flutter create --empty --platforms=android,ios --project-name rantau_plus --org id.rantauplus mobile`. Jalankan `flutter pub get` dari `mobile/` setelahnya. Tambahkan elemen `<application android:usesCleartextTraffic="true" />` ke manifest Android debug untuk mengizinkan demo HTTP lokal, lalu jalankan `flutter run` di emulator Android. Alamat backend dapat diatur melalui `--dart-define=API_BASE_URL=...`.

## Database PostgreSQL dan API Gin

Dari root repo, jalankan PostgreSQL lokal dengan Docker Compose. Setelah database siap, gunakan GORM untuk menerapkan migrasi berdasarkan model Go, lalu jalankan API Gin dengan koneksi yang sama.

```powershell
docker compose up -d --wait db
Set-Location backend
$env:DATABASE_URL = 'postgres://rantau:rantau_dev@127.0.0.1:5433/rantau_plus?sslmode=disable'
go run ./cmd/migrate
```

Jika migrasi berhasil, jalankan `go run ./cmd/api` di terminal yang sama.

Docker Compose memakai port host `5433` secara default. Jika memakai PostgreSQL yang sudah terpasang, buat database kosong `rantau_plus` dan isi `DATABASE_URL` dengan akun dan port server tersebut. API dan perintah migrasi membaca variabel lingkungan. Kredensial pada contoh khusus pengembangan lokal.

| Tabel | Isi |
| --- | --- |
| `users`, `refresh_tokens` | Akun, preferensi dasar, dan sesi |
| `auth_sessions`, `auth_email_jobs`, `auth_rate_limits` | Sesi aktif, antrean email autentikasi, dan batas percobaan |
| `financial_periods` | Saldo awal, dana cadangan, dan tanggal pemasukan berikutnya |
| `categories`, `transactions` | Kategori bawaan atau pribadi serta catatan pemasukan dan pengeluaran dalam rupiah bulat |
| `obligations` | Tagihan, jatuh tempo, dan transaksi pengeluaran yang terkait saat lunas |
| `reminders` | Pengingat dan status penyelesaian |
| `schema_migrations` | Riwayat versi migrasi yang sudah diterapkan |

Transaksi hanya dapat merujuk ke kategori bawaan atau kategori milik pengguna yang sama. Saat menyimpan transaksi, backend mengisi `category_is_shared` berdasarkan kategori di database. Tagihan lunas harus terhubung ke transaksi pengeluaran milik pengguna tersebut.

Periksa `http://127.0.0.1:8080/healthz` untuk status server dan `http://127.0.0.1:8080/readyz` untuk kesiapan database. Endpoint `/readyz` mengembalikan HTTP 200 saat PostgreSQL terhubung dan seluruh migrasi tersedia. API demo tetap dapat dipakai tanpa PostgreSQL dengan `RANTAU_DEMO=true`.

Perintah migrasi bisa dijalankan ulang dan membaca riwayat `schema_migrations` yang sudah ada. Jika kamu pernah memasang skema awal versi 000001 secara manual, gunakan `go run ./cmd/migrate -adopt-initial` untuk mencatat versi tersebut dan melanjutkan ke versi berikutnya.

## Pemeriksaan

Jalankan `npm test` dari `ai-service/`, serta `go test ./...` dan `go vet ./...` dari `backend/`.
