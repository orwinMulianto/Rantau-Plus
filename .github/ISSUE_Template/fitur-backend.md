---
name: Fitur Backend
about: Fitur atau endpoint baru di API Go/Gin
title: "[Backend] "
labels: backend, fitur
---

## Ringkasan

Jelaskan fitur ini dan masalah pengguna yang diselesaikannya dalam dua atau tiga kalimat.

## Cakupan kode

Modul: `backend/internal/modules/<nama-fitur>/`

- [ ] `routes`, `controller`, `service`, `repository`, dan `dto`
- [ ] Entity di `database/entities/` dan migrasi di `database/schema_steps.go` (nomor migrasi: `0000__`)
- [ ] Perhitungan di `internal/domain/finance/` (jika ada)
- [ ] Route didaftarkan di `internal/httpserver/`

## Kontrak API

| Method | Endpoint | Auth | Keterangan |
| --- | --- | --- | --- |
| `GET` | `/api/v1/...` | Bearer | |

Contoh request:

```json
{}
```

Contoh respons:

```json
{}
```

## Aturan bisnis

Nominal disimpan `int64` dalam rupiah bulat. Data selalu dibatasi pada akun dari sesi login. Tuliskan aturan lain di sini, misalnya validasi, batas nilai, atau status yang mungkin.

## Acceptance criteria

- [ ] Endpoint berjalan sesuai kontrak di atas dan menolak akses tanpa bearer token
- [ ] Kasus kosong, input salah, dan data milik akun lain ditangani
- [ ] Unit test service dan controller ditambahkan
- [ ] `go test ./...` dan `go vet ./...` lulus
- [ ] README atau `docs/` diperbarui jika ada endpoint baru

## Dependensi

Bergantung pada: #
Memblokir: #

## Catatan

PIC: @
Estimasi: S / M / L
