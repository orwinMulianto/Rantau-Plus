# Rantau+ — Panduan Desain Fitur

Rantau+ membantu mahasiswa rantau menjawab tiga pertanyaan sehari-hari. **Berapa uang yang bisa dipakai? Berapa batas belanja hari ini? Apakah dana cukup sampai pemasukan berikutnya?**

Gunakan mockup UTS sebagai acuan layar. Prioritas di bawah mengikuti blueprint MVP. Saat ini repo sudah berisi dashboard dengan data contoh dan API demo AI Companion untuk pertanyaan konsep finance.

## Alur utama

- **Pengguna baru** membuat akun, mengatur kondisi keuangan, lalu melihat pratinjau sebelum masuk ke Dashboard.
- **Penggunaan harian** dimulai dari Dashboard. Setelah pengguna menyimpan pemasukan atau pengeluaran, ringkasan diperbarui.

## Fitur yang dirancang

| Prioritas | Fitur / layar | Isi dan tindakan utama |
| --- | --- | --- |
| Inti | **Welcome, Daftar, Login, dan Profil** | Masuk atau buat akun dari Welcome. Di Profil, lihat nama dan email, atur preferensi dasar, atau keluar. |
| Inti | **Financial Setup** | Isi saldo awal, tanggal pemasukan berikutnya, kewajiban utama, dan dana cadangan. Tampilkan pratinjau dana tersedia dan batas harian sebelum pengguna menyimpan pengaturan. |
| Inti | **Home Dashboard dan Financial Insight** | Tampilkan dana tersedia, Safe-to-Spend, status risiko, dan hari menuju pemasukan berikutnya. Sediakan akses **Catat Cepat**, transaksi terbaru, tagihan terdekat, serta peringatan jika dana berpotensi habis lebih dulu. |
| Inti | **Transaksi dan Catat Cepat** | Tambah atau edit pemasukan dan pengeluaran dengan nominal, kategori, tanggal, serta catatan opsional. Sediakan riwayat per periode, pencarian, filter kategori atau jenis transaksi, dan konfirmasi hapus. |
| Inti | **Rencana dan Kewajiban** | Lihat daftar tagihan serta nominal, jatuh tempo, dan statusnya. Pengguna dapat menambah tagihan atau menandai tagihan yang sudah dibayar sebagai lunas. |
| Inti | **Detail Safe-to-Spend** | Jelaskan estimasi batas belanja harian dan tampilkan jumlah yang sudah digunakan serta sisa alokasi hari ini. Gunakan label `Rp/hari` dan tampilkan periode perhitungan. |
| Inti | **Detail Financial Runway** | Bandingkan perkiraan hari dana bertahan dengan sisa hari menuju pemasukan. Tampilkan rata-rata pengeluaran, proyeksi saldo, dan simulasi perubahan pengeluaran. |
| Berikutnya | **AI Companion** | Chat dengan contoh pertanyaan seperti “Berapa Safe-to-Spend saya?” atau “Kalau beli barang Rp300 ribu, uangku masih cukup?” Tampilkan hasil dari Financial Engine, penjelasan singkat, dan opsi tindakan yang relevan. Untuk jawaban dari panduan, tampilkan sumber bila tersedia. |
| Berikutnya | **Rekomendasi Makan** | Cari berdasarkan area dan anggaran. Kartu hasil memuat nama tempat, jarak dan kisaran harga jika tersedia, serta alasan kesesuaiannya dengan anggaran. Pengguna dapat memilih lokasi perangkat atau memasukkan area secara manual. |
| Tambahan | **Life Reminder** | Gabungkan dengan halaman Rencana. Pengguna dapat menambah, mengubah, menghapus, dan menandai pengingat tagihan atau aktivitas sebagai selesai. |

Mockup UTS juga memuat alokasi anggaran per kategori dan peringatan saat pengeluaran kategori meningkat. Keduanya bisa dieksplorasi pada desain lanjutan halaman Rencana.

## Kondisi tampilan untuk mockup

- **Belum ada data.** Arahkan pengguna untuk mengisi Financial Setup atau mencatat transaksi pertama. Tampilkan “Data belum cukup” ketika Financial Runway belum bisa dihitung.
- **Normal dan berisiko.** Pakai label **Aman**, **Waspada**, dan **Berisiko** bersama warna. Sebutkan alasannya, misalnya “Dana diperkirakan bertahan 16,25 hari, sementara pemasukan berikutnya 20 hari lagi.”
- **Input dan koneksi.** Letakkan pesan nominal atau tanggal yang salah di dekat kolomnya. Saat memuat data atau mengirim chat, tampilkan indikator proses. Sediakan **Coba lagi** jika gagal.
- **Pencarian kosong atau izin lokasi ditolak.** Beri opsi ubah area atau anggaran. Jika harga belum tersedia, tampilkan “Harga belum tersedia”.

**Contoh data konsisten untuk mockup** · Dana tersedia **Rp1.300.000** · Safe-to-Spend **Rp65.000/hari** · Financial Runway **16,25 hari** · Pemasukan berikutnya **20 hari lagi** · Status **Waspada**.

## Referensi proyek

- [UTS_RANTAU+(1).pdf](https://chatgpt.com/library?file_id=libfile_e2f4fcdae1fc8191a2184532d654e80d&account_id=personal) — User Flow dan UI Mockup, slide 12–14.
- [KELOMPOK 5_ISIT_I1_UTS_Rantau+(1).pdf](https://chatgpt.com/library?file_id=libfile_b7c8b198329c8191a2f172c2968f3369&account_id=personal) — Financial Engine, Insight & Alert, AI Companion, dan Contextual Recommendation.
- [Rantau+ Implementation Guide](https://chatgpt.com/library?file_id=libfile_c14db4dc258c8191b89f33abbd4a18f1&account_id=personal) — Cakupan dan prioritas MVP.
