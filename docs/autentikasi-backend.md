# Autentikasi backend Rantau+

Rantau+ memakai Go/Gin, GORM, dan PostgreSQL untuk autentikasi email dan password. Route, controller, service, dan repository ada di `backend/internal/modules/auth/`. Migrasi Go versi `000003_authentication` menambahkan status verifikasi, sesi, antrean email, dan pembatasan percobaan.

## Alur pengguna

- **Daftar** dengan email, buka token verifikasi, lalu isi nama dan password. Akun menjadi aktif setelah email terverifikasi. Baris pengguna lama yang belum terverifikasi dapat diaktifkan melalui alur yang sama.
- **Login** untuk menerima access token dan refresh token. Access token berlaku 15 menit. Refresh token berganti setiap kali dipakai dan memakai batas usia sesi 30 hari sejak login. Penggunaan ulang refresh token lama mencabut sesi tersebut.
- **Lupa password** dengan mengirim email. Respons awal sama untuk alamat terdaftar dan alamat yang belum terdaftar. Token reset berlaku 30 menit dan dapat dipakai sekali. Reset atau perubahan password mencabut seluruh sesi dan mengantrekan pemberitahuan email.
- **Kelola sesi** lewat daftar sesi, pencabutan sesi tertentu, logout, dan logout semua perangkat.

## Kontrak API

Semua route berawalan `/api/v1/auth`. Request dengan body memakai `Content-Type: application/json`. Route privat memakai `Authorization: Bearer <access_token>`.

| Method dan path | Input | Sukses |
| --- | --- | --- |
| `POST /register` | `email` | `202` mulai verifikasi |
| `POST /resend-verification` | `email` | `202` minta ulang token verifikasi |
| `POST /verify-email` | `token`, `name`, `password` | `204` aktifkan akun, lalu login |
| `POST /login` | `email`, `password` | `200` pasangan token |
| `POST /refresh` | `refresh_token` | `200` pasangan token baru |
| `POST /forgot-password` | `email` | `202` respons generik |
| `POST /reset-password` | `token`, `password` | `204` reset dan cabut sesi |
| `GET /me` | Bearer | `200` profil akun |
| `GET /sessions` | Bearer | `200` daftar sesi akun |
| `DELETE /sessions/:id` | Bearer, ID sesi | `204` cabut sesi |
| `POST /logout` | Bearer | `204` cabut sesi saat ini |
| `POST /logout-all` | Bearer | `204` cabut semua sesi akun |
| `POST /change-password` | Bearer, `current_password`, `new_password` | `204` ganti password dan cabut sesi |

Respons sukses dengan body memakai `data`. Login dan refresh mengembalikan `access_token`, `refresh_token`, `token_type`, `access_expires_at`, dan `refresh_expires_at`. Respons gagal memakai `error.code` dan `error.message`. Contoh kode mencakup `INVALID_CREDENTIALS`, `INVALID_SESSION`, `INVALID_TOKEN`, `PASSWORD_POLICY`, dan `RATE_LIMITED`.

Middleware `RequireSession` menyediakan ID akun terverifikasi lewat `SessionIdentity`. Repository fitur privat dapat memakai ID itu untuk membatasi kueri berdasarkan pemilik.

## Jalankan lokal

Jalankan dari root proyek di PowerShell.

```powershell
docker compose --profile auth up -d --wait db mailpit
Set-Location backend
$env:DATABASE_URL = 'postgres://rantau:rantau_dev@127.0.0.1:5433/rantau_plus?sslmode=disable'
$env:AUTH_SMTP_ADDR = '127.0.0.1:1025'
$env:AUTH_SMTP_FROM = 'rantau@example.test'
$env:AUTH_SMTP_LOCAL_PLAINTEXT = 'true'
$key = New-Object byte[] 32
$rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
try { $rng.GetBytes($key) } finally { $rng.Dispose() }
$env:AUTH_EMAIL_TOKEN_KEY = [Convert]::ToBase64String($key)
go run ./cmd/migrate
go run ./cmd/api
```

Buka [Mailpit lokal](http://127.0.0.1:8025) untuk melihat token. Pertahankan `AUTH_EMAIL_TOKEN_KEY` yang sama selama memakai database yang sama agar antrean email lama dapat dikirim setelah restart.

Contoh permintaan dari terminal kedua.

```powershell
$auth = 'http://127.0.0.1:8080/api/v1/auth'
Invoke-RestMethod -Method Post -Uri "$auth/register" -ContentType 'application/json' `
  -Body (@{ email = 'student@example.test' } | ConvertTo-Json)
```

Masukkan token dari Mailpit bersama nama dan password pada `POST /verify-email`, lalu gunakan `POST /login`. Uji pemulihan dengan `POST /forgot-password`, ambil token terbaru dari Mailpit, lalu panggil `POST /reset-password`.

## Keamanan dan integrasi klien

- Password di-hash dengan Argon2id dan salt acak. Pendaftaran, reset, dan perubahan password menerima 15–128 karakter. Isi `AUTH_PASSWORD_BLOCKLIST_FILE` dengan satu password umum atau bocor per baris untuk melengkapi daftar awal.
- PostgreSQL menyimpan hash access token, refresh token, dan token email. Antrean menyimpan nonce acak sehingga token email dapat dibuat kembali memakai kunci server saat pengiriman dicoba ulang.
- Verifikasi, reset, dan rotasi refresh memakai transaksi database. Login dan permintaan email dibatasi berdasarkan IP dan akun. Default menggunakan alamat koneksi langsung. Isi `AUTH_TRUSTED_PROXIES` dengan IP atau CIDR proxy yang dikendalikan deployment, dipisahkan spasi.
- Untuk produksi, gunakan HTTPS dan SMTP dengan STARTTLS. Simpan kunci email dan kredensial SMTP di secret manager. Gunakan kunci email yang sama pada setiap instance backend. Atur `AUTH_VERIFY_URL` dan `AUTH_RESET_URL` ke halaman HTTPS atau App Links yang kamu kendalikan. Klien membaca token dari fragment `#token=...` dan mengirimkannya lewat POST.
- Di Flutter, simpan refresh token lewat Keychain/Keystore, pegang access token di memori, dan jalankan satu refresh pada satu waktu. Jika sesi ditolak setelah refresh, arahkan pengguna ke login.

## Uji integrasi

Jalankan dari folder `backend`. Tes membuat schema sementara untuk menguji integrasi PostgreSQL dan menangkap email di memori.

```powershell
$env:AUTH_TEST_DATABASE_URL = 'postgres://rantau:rantau_dev@127.0.0.1:5433/rantau_plus?sslmode=disable'
go test ./internal/modules/auth -run TestAuthPostgresLifecycle -v
```

## Referensi

- [OWASP Password Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
- [OWASP Forgot Password Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Forgot_Password_Cheat_Sheet.html)
- [NIST SP 800-63B-4](https://pages.nist.gov/800-63-4/sp800-63b.html)
- [RFC 9700, refresh token protection](https://www.rfc-editor.org/rfc/rfc9700.html#section-4.14)
