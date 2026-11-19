## Ringkasan

Jelaskan perubahan dan alasannya.

Closes #

## Jenis perubahan

- [ ] Fitur backend
- [ ] Fitur AI Service
- [ ] Perbaikan bug
- [ ] Teknis (refactor, CI, dokumentasi)

## Checklist

- [ ] Branch dibuat dari `main` terbaru dengan nama `feat/<fitur>`
- [ ] `go test ./...` dan `go vet ./...` lulus (jika menyentuh `backend/`)
- [ ] `npm test` lulus (jika menyentuh `ai-service/`)
- [ ] Migrasi baru memakai nomor yang sudah dipesan di issue dan sudah dijalankan ulang tanpa error
- [ ] Tidak ada `.env`, token, atau kunci API yang ikut ter-commit
- [ ] README atau `docs/` diperbarui jika ada endpoint atau konfigurasi baru

## Cara menguji

Tuliskan perintah atau contoh request yang bisa dijalankan reviewer.

## Catatan untuk reviewer
