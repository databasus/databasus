<div align="center">
  <img src="../logo.svg" alt="Databasus Logo" width="250"/>

  <h3>Alat backup PostgreSQL</h3>
  <p>Databasus adalah alat gratis, open source, dan self-hosted untuk backup PostgreSQL. Buat backup ke berbagai penyimpanan (S3, Google Drive, FTP, dan lain-lain) dan dapatkan notifikasi progresnya (Slack, Discord, Telegram, dan lain-lain). Fokusnya Point-in-Time Recovery dengan RPO/RTO rendah</p>

  <!-- Badges -->

[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![MySQL](https://img.shields.io/badge/MySQL-4479A1?logo=mysql&logoColor=white)](https://www.mysql.com/)
[![MariaDB](https://img.shields.io/badge/MariaDB-003545?logo=mariadb&logoColor=white)](https://mariadb.org/)
[![MongoDB](https://img.shields.io/badge/MongoDB-47A248?logo=mongodb&logoColor=white)](https://www.mongodb.com/)
<br />
[![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../../LICENSE)
[![Docker Pulls](https://img.shields.io/docker/pulls/databasus/databasus?color=brightgreen)](https://hub.docker.com/r/databasus/databasus)
[![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey)](https://github.com/databasus/databasus)
[![Self Hosted](https://img.shields.io/badge/self--hosted-yes-brightgreen)](https://github.com/databasus/databasus)
[![Open Source](https://img.shields.io/badge/open%20source-❤️-red)](https://github.com/databasus/databasus)

  <p>
    <a href="../../README.md">English</a> •
    <a href="README.ru.md">Русский</a> •
    <a href="README.es.md">Español</a> •
    <a href="README.pt.md">Português</a> •
    <a href="README.zh.md">中文</a> •
    <a href="README.fr.md">Français</a> •
    <b>Bahasa Indonesia</b>
  </p>

  <p>
    <a href="#-fitur">Fitur</a> •
    <a href="#-instalasi">Instalasi</a> •
    <a href="#-penggunaan">Penggunaan</a> •
    <a href="#-lisensi">Lisensi</a> •
    <a href="#-berkontribusi">Berkontribusi</a>
  </p>

  <p style="margin-top: 20px; margin-bottom: 20px; font-size: 1.2em;">
    <a href="https://databasus.com/id/" target="_blank"><strong>🌐 Situs web Databasus</strong></a>
  </p>

  <img src="../dashboard-dark.svg" alt="Databasus Dark Dashboard" width="800" style="margin-bottom: 10px;"/>

  <img src="../dashboard.svg" alt="Databasus Dashboard" width="800"/>
</div>

---

## ✨ Fitur

### 📦 **Jenis backup**

- **Fisik**: salinan tingkat file dari seluruh cluster database lewat mekanisme backup inkremental native PostgreSQL (selengkapnya)
  - **Penuh**: salinan lengkap dan mandiri dari cluster
  - **Inkremental**: hanya menyimpan perubahan sejak backup penuh sebelumnya, jadi backup tetap kecil dan cepat
  - **WAL streaming**: menangkap aliran tulis database secara terus-menerus, sehingga Point-in-Time Recovery (PITR) bisa dilakukan. Dirancang untuk disaster recovery dan kehilangan data mendekati nol
- **Logis**: dump native database dalam format biner khas engine-nya (terkompresi, cocok untuk restore paralel)

### 🔄 **Backup terjadwal**

- **Penjadwalan fleksibel**: per jam, harian, mingguan, bulanan, atau cron
- **Waktu presisi**: jalankan backup pada waktu tertentu (misalnya pukul 4 pagi saat trafik rendah)
- **Kompresi pintar**: hemat ruang 4-8x dengan kompresi seimbang (overhead sekitar 20%)

### 🧪 **Verifikasi restore** <a href="https://databasus.com/id/restore-verification/">(dokumentasi)</a>

Databasus melakukan restore sungguhan untuk memastikan backup bisa dipakai, bukan sekadar utuh di disk atau lolos pemeriksaan checksum.

- **Pemicu**: setelah setiap backup atau sesuai jadwal fleksibel (per jam, harian, mingguan, bulanan, atau cron)
- **Restore sungguhan**: menjalankan container database, melakukan restore, lalu membandingkan ukuran hasil restore dengan backup
- **Laporan**: mencantumkan setiap tabel beserta jumlah barisnya
- **Notifikasi opsional**: kirim laporan atau hanya alert kegagalan lewat kanal notifikasi yang sudah dikonfigurasi

### 🗑️ **Kebijakan retensi**

- **Periode waktu**: simpan backup untuk durasi tetap (misalnya 7 hari, 3 bulan, 1 tahun)
- **Jumlah**: simpan sejumlah backup terbaru (misalnya 30 terakhir)
- **GFS (Grandfather-Father-Son)**: retensi berlapis. Backup per jam, harian, mingguan, bulanan, dan tahunan disimpan terpisah untuk riwayat jangka panjang yang rinci (kebutuhan enterprise)
- **Batas ukuran**: tetapkan batas ukuran per backup dan total penyimpanan untuk mengendalikan pemakaian ruang

### 🗄️ **Berbagai tujuan penyimpanan** <a href="https://databasus.com/id/storages/">(lihat penyimpanan yang didukung)</a>

- **Penyimpanan lokal**: simpan backup di VPS/server kamu
- **Penyimpanan cloud**: S3, Cloudflare R2, Google Drive, NAS, Dropbox, SFTP, Rclone, dan lainnya
- **Aman**: semua data tetap dalam kendali kamu

### 📱 **Notifikasi** <a href="https://databasus.com/id/notifiers/">(lihat kanal yang didukung)</a>

- **Banyak kanal**: Email, Telegram, Slack, Discord, Teams, Mattermost, webhook
- **Update real-time**: notifikasi sukses dan gagal
- **Integrasi tim**: pas untuk alur kerja DevOps

### 🔒 **Keamanan kelas enterprise** <a href="https://databasus.com/id/security/">(dokumentasi)</a>

- **Enkripsi AES-256-GCM**: perlindungan kelas enterprise untuk file backup
- **Penyimpanan zero-trust**: backup terenkripsi dan tetap tidak berguna bagi penyerang, jadi kamu bisa menyimpannya dengan aman di penyimpanan bersama seperti S3, Azure Blob Storage, dan lain-lain
- **Enkripsi untuk secret**: semua data sensitif dienkripsi dan tidak pernah terekspos, bahkan di log atau pesan error
- **Pengguna read-only**: secara default Databasus memakai pengguna read-only untuk backup dan tidak pernah menyimpan apa pun yang bisa mengubah data kamu
- **Autentikasi dua faktor**: login dengan password bisa mewajibkan kode enam digit yang dikirim ke email akun. Login lewat Google atau GitHub tetap mengandalkan pemeriksaan dari penyedianya

### 👥 **Cocok untuk tim** <a href="https://databasus.com/id/access-management/">(dokumentasi)</a>

- **Ruang kerja**: kelompokkan database, kanal notifikasi, dan penyimpanan untuk proyek atau tim berbeda
- **Manajemen akses**: atur siapa yang bisa melihat atau mengelola database tertentu dengan izin berbasis peran
- **Audit log**: lacak semua aktivitas sistem dan perubahan yang dibuat pengguna
- **Peran pengguna**: tetapkan peran viewer, member, admin, atau owner di dalam ruang kerja
- **Log OpenTelemetry**: ekspor log aplikasi dan audit ke sistem eksternal (secara default log juga ditulis ke file lokal)

### 🎨 **Ramah pengguna**

- **UI hasil polesan desainer**: antarmuka bersih dan intuitif, dibuat dengan perhatian pada detail
- **Tema gelap dan terang**: pilih tampilan yang cocok dengan alur kerja kamu
- **Adaptif di mobile**: cek backup dari mana saja dan perangkat apa pun

### 💾 **Database yang didukung**

- **PostgreSQL**: 14, 15, 16, 17 dan 18 (fisik dan logis)
- **MySQL**: 5.7, 8.0, 8.4, 9 dan 26 (hanya logis)
- **MariaDB**: 5.5, 10, 11, 12 dan 13 (hanya logis)
- **MongoDB**: 4.2+, 5, 6, 7, 8 dan 9 (hanya logis)

### 🐳 **Self-hosted dan aman**

- **Berbasis Docker**: deployment dan pengelolaan mudah
- **Privasi dulu**: semua data tetap di infrastruktur kamu
- **Open source**: lisensi Apache 2.0, setiap baris kode bisa diperiksa
- **SSH bawaan**: hubungkan ke Databasus lewat tunnel SSH

### 📦 Instalasi <a href="https://databasus.com/id/installation/">(dokumentasi)</a>

Ada empat cara memasang Databasus:

- Skrip otomatis (direkomendasikan)
- Docker run sederhana
- Setup Docker Compose
- Kubernetes dengan Helm

<img src="../healthchecks.svg" alt="Databasus Dashboard" width="800"/>

---

## 📦 Instalasi

Ada empat cara memasang Databasus: skrip otomatis (direkomendasikan), Docker run sederhana, atau setup Docker Compose.

### Opsi 1: Skrip instalasi otomatis (direkomendasikan, khusus Linux)

Skrip instalasi akan:

- ✅ Memasang Docker dengan Docker Compose (jika belum terpasang)
- ✅ Menyiapkan Databasus
- ✅ Mengatur startup otomatis saat sistem reboot

```bash
sudo apt-get install -y curl && \
sudo curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh \
| sudo bash
```

### Opsi 2: Docker run sederhana

Cara termudah menjalankan Databasus:

```bash
docker run -d \
  --name databasus \
  -p 4005:4005 \
  -v ./databasus-data:/databasus-data \
  --restart unless-stopped \
  databasus/databasus:latest
```

_Image yang sama ada di registry GitHub, gunakan `ghcr.io/databasus/databasus:latest` kalau Docker Hub membatasi laju pull kamu._

Satu perintah ini akan:

- ✅ Menjalankan Databasus
- ✅ Menyimpan semua data di direktori `./databasus-data`
- ✅ Restart otomatis saat sistem reboot

### Opsi 3: Setup Docker Compose

Buat file `docker-compose.yml` dengan konfigurasi berikut:

```yaml
services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "databasus", "healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 60s
```

Lalu jalankan:

```bash
docker compose up -d
```

### Opsi 4: Kubernetes dengan Helm

Untuk deployment Kubernetes, pasang langsung dari registry OCI.

_Tambahkan `--set image.repository=ghcr.io/databasus/databasus` ke salah satu perintah di bawah untuk mengambil image dari GHCR, bukan Docker Hub._

**Dengan ClusterIP + port-forward (pengembangan/pengujian):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace
```

```bash
kubectl port-forward svc/databasus-service 4005:4005 -n databasus
# Access at http://localhost:4005
```

**Dengan LoadBalancer (lingkungan cloud):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set service.type=LoadBalancer
```

```bash
kubectl get svc databasus-service -n databasus
# Access at http://<EXTERNAL-IP>:4005
```

**Dengan Ingress (akses berbasis domain):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=backup.example.com
```

Untuk opsi lain (NodePort, TLS, HTTPRoute untuk Gateway API), lihat [README chart Helm](../../deploy/helm/README.md).

---

## 🚀 Penggunaan

1. **Buat akun pertama**: buka `http://localhost:4005` dan daftar. Akun pertama yang dibuat di sebuah instance akan mengelolanya
2. **Tambahkan database pertama untuk backup**: klik "New Database" dan ikuti wizard setup
3. **Atur jadwal**: pilih interval per jam, harian, mingguan, bulanan, atau cron
4. **Isi koneksi database**: masukkan kredensial dan detail koneksi database kamu
5. **Pilih penyimpanan**: tentukan tempat menyimpan backup (lokal, S3, Google Drive, dan lain-lain)
6. **Atur kebijakan retensi**: pilih periode waktu, jumlah, atau GFS untuk mengatur berapa lama backup disimpan
7. **Tambahkan notifikasi** (opsional): atur notifikasi email, Telegram, Slack, Mattermost, atau webhook
8. **Simpan dan mulai**: Databasus akan memvalidasi pengaturan dan memulai jadwal backup

### 🔑 Reset password <a href="https://databasus.com/id/password/">(dokumentasi)</a>

Kalau perlu reset password, gunakan perintah reset password bawaan:

```bash
docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com"
```

Ganti `owner@example.com` dengan alamat email akun yang password-nya ingin kamu reset.

Kalau kamu tidak ingat alamat mana yang mengelola instance, daftar akun administrator:

```bash
docker exec -it databasus ./main --list-admins
```

Output-nya mencantumkan setiap akun administrator beserta alamat email, nama tampilan, tanggal pembuatan, dan status aktifnya, lalu menandai akun yang dikenali instance sebagai administrator. Tidak ada materi password yang dicetak.

Untuk membuat akun tanpa mendaftar lewat browser, misalnya dari skrip deployment, buat dari konsol:

```bash
docker exec -it databasus ./main --create-user --email="ops@example.com" --password="YourSecurePassword123" --role=admin
```

`--role` berisi `admin` atau `member`, password minimal 8 karakter, dan `--name` opsional untuk mengatur nama tampilan. Di instance yang belum punya akun, akun pertama harus memakai `--role=admin` dan akan mengelola instance. Menjalankan perintah yang sama untuk akun yang sama tidak mengubah apa pun, jadi deployment bisa menjalankannya di setiap rollout. Ganti password dengan `--new-password`.

### 💾 Backup Databasus itu sendiri

Setelah instalasi, sebaiknya kamu juga <a href="https://databasus.com/id/faq/#backup-databasus">backup Databasus itu sendiri</a> atau setidaknya menyalin secret key yang dipakai untuk enkripsi (butuh 30 detik). Dengan begitu kamu bisa restore dari backup terenkripsi kalau kehilangan akses ke server Databasus atau server-nya rusak.

---

## 🛡️ Rekayasa keamanan dan keandalan

Databasus bekerja dengan data sensitif, jadi mencegah kerentanan, akses tidak sah, dan kebocoran data menjadi prioritas utama. Kami berinvestasi di kedua sisi sistem: di kode itu sendiri (pemeriksaan izin, enkripsi, penanganan secret yang hati-hati) dan di infrastruktur sekitarnya (analisis dependensi, respons CVE, praktik DevSecOps). Pipeline di bawah berjalan otomatis pada setiap commit dan PR. Tidak ada satu lapisan pun yang cukup sendirian, tetapi bersama-sama lapisan ini mengurangi peluang kode rentan, dependensi tidak aman, image rusak, atau backup yang tidak bisa di-restore masuk ke rilis.

Untuk analisis statis, kami menggabungkan beberapa pemeriksaan independen. CodeQL memindai seluruh codebase untuk masalah keamanan. Di setiap PR, gitleaks memindai secret yang bocor dan semgrep memeriksa kode terhadap aturan keamanan. Dockerfile dan workflow CI mendapat aturan tambahan sendiri (referensi action yang di-pin, izin least-privilege, image dasar yang mencurigakan), sehingga pola tidak aman ditandai sebelum sempat di-merge. Selain pemeriksaan per-PR itu, Codex Security dari OpenAI menjalankan audit berkala yang lebih dalam pada seluruh codebase. Ini program terpisah yang menangkap masalah arsitektural dan lintas area yang bisa terlewat oleh pemindaian sempit saat PR.

Di sisi dependensi, Dependabot memantau semua dependensi kami terhadap GitHub Advisory Database dan memunculkan CVE dalam hitungan menit setelah dipublikasikan. Update berjalan dengan masa tunggu supaya versi yang baru terbit punya kesempatan matang sebelum kami adopsi. Ini pertahanan yang disengaja terhadap insiden paket yang dikompromikan, seperti serangan supply chain. Dependency Review Action memblokir setiap PR yang memasukkan CVE HIGH atau CRITICAL baru.

Image container dipindai dengan Trivy pada setiap build. Pemeriksaan Trivy terpisah pada Dockerfile menangkap kesalahan konfigurasi sebelum masuk ke image. Semua GitHub Actions di-pin ke SHA commit lengkap, bukan tag mengambang seperti `@v4` atau `@main`, yang menjadi vektor serangan aktif pada 2025. Workflow secara default memakai izin least-privilege dan hanya menaikkannya per job saat benar-benar perlu.

Jalur kritis dicakup oleh unit test dan integration test, dijalankan terhadap container database sungguhan untuk setiap engine dan versi mayor yang didukung. Restore adalah jalur yang paling penting bagi alat backup, jadi kami mengujinya secara eksplisit: setiap PR menjalankan siklus backup lalu restore penuh terhadap container nyata yang sama, memastikan backup benar-benar bisa di-restore end-to-end, bukan hanya berhasil ditulis. Sisa pipeline CI/CD menjalankan lint, type-check, seluruh test suite, smoke test image, dan build multi-arsitektur pada setiap PR. Rilis hanya terbit kalau semuanya lolos.

Menemukan kerentanan? Laporkan lewat tab Security di GitHub. Lihat [SECURITY.md](https://github.com/databasus/databasus?tab=security-ov-file#readme). Laporan keamanan adalah antrean kerja dengan prioritas tertinggi. Untuk keamanan aplikasi saat berjalan (AES-256-GCM saat disimpan, penyimpanan zero-trust, secret terenkripsi, pengguna database read-only secara default), lihat [Keamanan kelas enterprise](#-keamanan-kelas-enterprise-dokumentasi) di bagian Fitur di atas.

---

## 📝 Lisensi

Proyek ini dilisensikan di bawah Apache 2.0 License. Lihat file [LICENSE](../../LICENSE) untuk detailnya

## 🤝 Berkontribusi

Kontribusi selalu diterima! Baca <a href="https://databasus.com/contribute">panduan kontribusi</a> untuk detail, prioritas, dan aturan. Kalau kamu ingin berkontribusi tapi bingung mulai dari mana, kirim pesan ke saya di Telegram [@rostislav_dugin](https://t.me/rostislav_dugin)

Kamu juga bisa bergabung dengan komunitas besar developer, DBA, dan engineer DevOps kami di Telegram [@databasus_community](https://t.me/databasus_community).
