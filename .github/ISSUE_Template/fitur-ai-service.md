---
name: Fitur AI Service
about: Skill, tool, knowledge, atau perbaikan di AI Service Node.js
title: "[AI] "
labels: ai-service, fitur
---

## Ringkasan

Jelaskan kemampuan baru AI Companion dan contoh pertanyaan pengguna yang harus terjawab.

## Contoh percakapan

| Pesan pengguna | Hasil yang diharapkan |
| --- | --- |
| | |

## Cakupan kode

- [ ] Skill atau tool di `ai-service/src/skills/`
- [ ] Dokumen referensi di `ai-service/knowledge/` (jika perlu)
- [ ] Perubahan RAG di `ai-service/src/rag/` atau provider di `ai-service/src/providers/`
- [ ] Sisi Go: validasi argumen dan klien di `backend/internal/integration/assistant/` serta `backend/internal/modules/assistant/`

## Aturan

Angka keuangan harus berasal dari Financial Engine di Go, bukan dihitung atau dikarang model. Jawaban dari panduan menampilkan sumber. Perilaku saat model offline dijelaskan di bawah.

Perilaku saat model offline:

## Acceptance criteria

- [ ] Nilai `mode` respons sesuai (`retrieval`, `rag`, `conversation`, atau `no_context`)
- [ ] Argumen tool divalidasi di Go
- [ ] Test ditambahkan di `ai-service/test/`
- [ ] `npm test` dan `go test ./...` lulus
- [ ] Diuji manual dengan model lokal (sebutkan model yang dipakai)

## Dependensi

Bergantung pada: #

## Catatan

PIC: @
Estimasi: S / M / L
