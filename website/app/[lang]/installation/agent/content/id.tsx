import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Instalasi Agen - Dokumentasi Databasus",
  description:
    "Pasang agen Databasus untuk backup fisik, backup incremental, pengarsipan WAL, dan Point-in-Time Recovery (PITR) database PostgreSQL.",
  keywords: [
    "agen Databasus",
    "backup fisik PostgreSQL",
    "pengarsipan WAL",
    "PITR",
    "Point-in-Time Recovery",
    "pg_basebackup",
    "backup incremental",
    "pemulihan bencana",
    "agen PostgreSQL",
    "agen backup database",
  ],
  openGraph: {
    title: "Instalasi Agen - Dokumentasi Databasus",
    description:
      "Pasang agen Databasus untuk backup fisik, backup incremental, pengarsipan WAL, dan Point-in-Time Recovery (PITR) database PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("id", "installation/agent"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Instalasi Agen - Dokumentasi Databasus",
    description:
      "Pasang agen Databasus untuk backup fisik, backup incremental, pengarsipan WAL, dan Point-in-Time Recovery (PITR) database PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "installation/agent"),
    languages: getLanguageAlternates("installation/agent"),
  },
  robots: "index, follow",
};

export default function AgentInstallationPage() {
  const downloadCommand = `curl -L -o databasus-agent "<DATABASUS_HOST>/api/v1/system/agent?arch=<ARCH>" && chmod +x databasus-agent`;
  const postgresqlConf = `wal_level = replica
archive_mode = on
archive_command = 'cp %p <WAL_QUEUE_DIR>/%f.tmp && mv <WAL_QUEUE_DIR>/%f.tmp <WAL_QUEUE_DIR>/%f'`;
  const postgresqlConfDocker = `wal_level = replica
archive_mode = on
archive_command = 'cp %p /wal-queue/%f.tmp && mv /wal-queue/%f.tmp /wal-queue/%f'`;
  const pgHbaEntry = `host    replication   all   127.0.0.1/32   md5`;
  const grantReplication = `ALTER ROLE <YOUR_PG_USER> WITH REPLICATION;`;
  const createWalDir = `mkdir -p /opt/databasus/wal-queue`;
  const walDirPermissions = `chown postgres:postgres /opt/databasus/wal-queue
chmod 755 /opt/databasus/wal-queue`;
  const dockerVolumeExample = `# In your docker run command:
docker run ... -v /opt/databasus/wal-queue:/wal-queue ...

# Or in docker-compose.yml:
volumes:
  - /opt/databasus/wal-queue:/wal-queue`;
  const dockerWalDirPermissions = `# Inside the container (or via docker exec):
chown postgres:postgres /wal-queue`;
  const startCommandHost = `./databasus-agent start \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --pg-host=localhost \\
  --pg-port=5432 \\
  --pg-user=<YOUR_PG_USER> \\
  --pg-password=<YOUR_PG_PASSWORD> \\
  --pg-type=host \\
  --pg-wal-dir=/opt/databasus/wal-queue`;
  const startCommandFolder = `./databasus-agent start \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --pg-host=localhost \\
  --pg-port=5432 \\
  --pg-user=<YOUR_PG_USER> \\
  --pg-password=<YOUR_PG_PASSWORD> \\
  --pg-type=host \\
  --pg-host-bin-dir=<PATH_TO_PG_BIN_DIR> \\
  --pg-wal-dir=/opt/databasus/wal-queue`;
  const startCommandDocker = `./databasus-agent start \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --pg-host=localhost \\
  --pg-port=5432 \\
  --pg-user=<YOUR_PG_USER> \\
  --pg-password=<YOUR_PG_PASSWORD> \\
  --pg-type=docker \\
  --pg-docker-container-name=<CONTAINER_NAME> \\
  --pg-wal-dir=/opt/databasus/wal-queue`;
  const restoreCommand = `./databasus-agent restore \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --backup-id=<BACKUP_ID> \\
  --target-dir=<PGDATA_DIR>`;
  const restoreCommandDocker = `./databasus-agent restore \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --backup-id=<BACKUP_ID> \\
  --pg-type=docker \\
  --target-dir=<HOST_PGDATA_PATH>`;
  const restoreCommandPitr = `./databasus-agent restore \\
  --databasus-host=<DATABASUS_HOST> \\
  --db-id=<DB_ID> \\
  --token=<YOUR_AGENT_TOKEN> \\
  --backup-id=<BACKUP_ID> \\
  --target-dir=<PGDATA_DIR> \\
  --target-time=<RFC3339_TIMESTAMP>`;
  const archiveCommandCleanup = `# In <PGDATA_DIR>/postgresql.auto.conf, remove or comment out:
# archive_mode = on
# archive_command = '...'`;
  const dockerVolumeMountExample = `# PostgreSQL 17 and earlier
docker run -d -v <HOST_PGDATA_PATH>:/var/lib/postgresql/data postgres:17

# PostgreSQL 18+
docker run -d -v <HOST_PGDATA_PATH>:/var/lib/postgresql/18/docker postgres:18`;

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Instalasi Agen - Dokumentasi Databasus",
            description:
              "Pasang agen Databasus untuk backup fisik, backup incremental, pengarsipan WAL, dan Point-in-Time Recovery (PITR) database PostgreSQL.",
            author: { "@type": "Organization", name: "Databasus" },
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: { "@type": "ImageObject", url: "https://databasus.com/logo.svg" },
            },
          }),
        }}
      />
      <DocsNavbarComponent lang="id" />
      <div className="flex min-h-screen bg-[#0F1115]">
        <DocsSidebarComponent lang="id" />
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="agent-installation">Mode agen</h1>
              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-red-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-red-500 mt-0.5 shrink-0"
                >
                  <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                  <path d="M12 9v4M12 17h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>Backup melalui agen sudah deprecated.</strong> Databasus kini
                    menjalankan backup fisik dan PITR dari jarak jauh menggunakan backup native
                    PostgreSQL 17, tanpa memasang agen di server database.{" "}
                    <a
                      href={getLocalizedUrl("id", "faq") + "#why-no-agent"}
                      className="text-blue-400 hover:text-blue-300 underline"
                    >
                      Baca alasannya dan cara kerja backup PITR sekarang
                    </a>
                    .
                  </p>
                </div>
              </div>
              <p className="text-lg text-gray-400">
                Agen Databasus memungkinkan backup fisik, backup incremental, pengarsipan WAL, dan
                Point-in-Time Recovery (PITR) untuk database PostgreSQL.
              </p>
              <h2 id="when-to-use">Kapan perlu menggunakan agen</h2>
              <p>
                Untuk kebanyakan database,{" "}
                <strong>backup jarak jauh adalah pilihan paling sederhana</strong>. Databasus
                terhubung langsung ke database melalui jaringan, menjalankan backup logis
                menggunakan pg_dump, dan tidak membutuhkan software tambahan di server database.
                Backup jarak jauh bisa digunakan untuk database yang dikelola penyedia cloud (RDS,
                Cloud SQL, Supabase) maupun instance self-hosted.
              </p>
              <p>Agen dirancang untuk situasi ketika backup jarak jauh tidak mencukupi:</p>
              <ul>
                <li>
                  <strong>Pemulihan bencana dengan PITR</strong>: restore ke detik mana pun di
                  antara backup dengan kehilangan data mendekati nol
                </li>
                <li>
                  <strong>Backup fisik</strong>: salinan seluruh cluster database pada tingkat file,
                  dengan backup dan restore lebih cepat untuk dataset besar
                </li>
                <li>
                  <strong>Database yang tidak terbuka ke publik</strong>: agen membuat koneksi
                  keluar ke Databasus, sehingga database tidak membutuhkan endpoint publik
                </li>
                <li>
                  <strong>Backup incremental</strong>: pengarsipan segmen WAL secara terus-menerus
                  yang dipadukan dengan base backup berkala
                </li>
              </ul>
              <h2 id="in-app-setup">Panduan pengaturan di aplikasi</h2>
              <p>
                Databasus menyediakan petunjuk instalasi dan restore interaktif langsung di
                antarmuka. Saat membuka pengaturan agen untuk suatu database, semua perintah sudah
                berisi nilai yang sesuai: arsitektur, ID database, token agen, host Databasus, dan
                jenis deployment PostgreSQL. Kamu bisa menyalin setiap perintah dan menjalankannya
                di server.
              </p>
              <p>
                Dokumentasi berikut mencakup langkah yang sama sebagai referensi dan untuk pengguna
                yang lebih suka mengikuti panduan di luar antarmuka.
              </p>
              <h2 id="requirements">Persyaratan</h2>
              <ul>
                <li>
                  <strong>PostgreSQL 15 atau lebih baru</strong>
                </li>
                <li>
                  <strong>Linux</strong> (amd64 atau arm64)
                </li>
                <li>
                  <strong>Akses jaringan</strong> dari agen ke instance Databasus kamu (hanya
                  koneksi keluar, database tidak perlu bisa diakses dari Databasus)
                </li>
              </ul>
              <h2 id="installation">Instalasi</h2>
              <h3 id="step-1-download">Langkah 1: Unduh agen</h3>
              <p>
                Unduh binary agen di server tempat PostgreSQL berjalan. Ganti{" "}
                <code>&lt;DATABASUS_HOST&gt;</code> dengan URL instance Databasus kamu dan{" "}
                <code>&lt;ARCH&gt;</code> dengan <code>amd64</code> atau <code>arm64</code>.
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={downloadCommand} lang="id" />
                </div>
              </div>
              <h3 id="step-2-postgresql-conf">Langkah 2: Atur postgresql.conf</h3>
              <p>
                Tambahkan atau perbarui pengaturan berikut di <code>postgresql.conf</code>, lalu{" "}
                <strong>restart PostgreSQL</strong>.
              </p>
              <p>
                <strong>Untuk instalasi di host</strong> (ganti <code>&lt;WAL_QUEUE_DIR&gt;</code>{" "}
                dengan path yang sebenarnya, misalnya <code>/opt/databasus/wal-queue</code>):
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConf}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={postgresqlConf} lang="id" />
                </div>
              </div>
              <p>
                <strong>Untuk instalasi Docker</strong>, path <code>archive_command</code> (
                <code>/wal-queue</code>) adalah path <strong>di dalam container</strong>. Path ini
                harus sesuai dengan tujuan mount volume, lihat Langkah 5.
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{postgresqlConfDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={postgresqlConfDocker} lang="id" />
                </div>
              </div>
              <h3 id="step-3-pg-hba">Langkah 3: Atur pg_hba.conf</h3>
              <p>
                Tambahkan baris berikut ke <code>pg_hba.conf</code>. Ini diperlukan agar{" "}
                <code>pg_basebackup</code> bisa membuat backup penuh, bukan untuk streaming
                replication. Sesuaikan alamat dan metode autentikasi sesuai kebutuhan, lalu reload
                PostgreSQL.
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{pgHbaEntry}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={pgHbaEntry} lang="id" />
                </div>
              </div>
              <h3 id="step-4-replication">Langkah 4: Berikan hak replication</h3>
              <p>
                Ini merupakan persyaratan PostgreSQL untuk menjalankan <code>pg_basebackup</code>,
                bukan untuk membuat replica.
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{grantReplication}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={grantReplication} lang="id" />
                </div>
              </div>
              <h3 id="step-5-wal-queue">Langkah 5: Buat direktori antrean WAL</h3>
              <p>PostgreSQL menaruh file arsip WAL di sini untuk diunggah oleh agen.</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createWalDir}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={createWalDir} lang="id" />
                </div>
              </div>
              <p>Pastikan PostgreSQL bisa menulis ke direktori ini dan agen bisa membacanya:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{walDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={walDirPermissions} lang="id" />
                </div>
              </div>
              <p>
                <strong>Untuk instalasi Docker</strong>, direktori antrean WAL harus berupa mount
                volume yang dibagikan antara container PostgreSQL dan host. Agen membaca file WAL
                dari path host, sementara PostgreSQL menulis ke path container melalui{" "}
                <code>archive_command</code>.
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerVolumeExample} lang="id" />
                </div>
              </div>
              <p>
                Pastikan direktori di dalam container dimiliki oleh pengguna <code>postgres</code>:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerWalDirPermissions}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerWalDirPermissions} lang="id" />
                </div>
              </div>
              <h3 id="step-6-start">Langkah 6: Jalankan agen</h3>
              <p>
                Ganti placeholder di dalam <code>&lt;ANGLE_BRACKETS&gt;</code> dengan nilai yang
                sebenarnya.
              </p>
              <p>
                <strong>PostgreSQL yang terpasang di sistem</strong> (pg_basebackup tersedia di
                PATH):
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandHost}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={startCommandHost} lang="id" />
                </div>
              </div>
              <p>
                <strong>PostgreSQL di folder tertentu</strong> (misalnya{" "}
                <code>/usr/lib/postgresql/17/bin</code>):
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandFolder}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={startCommandFolder} lang="id" />
                </div>
              </div>
              <p>
                <strong>Docker</strong> (gunakan port PostgreSQL <strong>di dalam container</strong>
                , biasanya 5432, bukan port yang dipetakan ke host):
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={startCommandDocker} lang="id" />
                </div>
              </div>
              <h3 id="after-installation">Setelah instalasi</h3>
              <ul>
                <li>
                  Agen berjalan di background setelah <code>start</code>
                </li>
                <li>
                  Periksa status: <code>./databasus-agent status</code>
                </li>
                <li>
                  Lihat log: <code>databasus.log</code> di direktori kerja
                </li>
                <li>
                  Hentikan agen: <code>./databasus-agent stop</code>
                </li>
              </ul>
              <h2 id="restore">Restore dari backup agen</h2>
              <p>
                Restore backup fisik atau incremental ke direktori tujuan. Untuk Point-in-Time
                Recovery, tambahkan flag <code>--target-time</code> agar restore dilakukan ke waktu
                tertentu.
              </p>
              <h3 id="restore-step-1">Langkah 1: Unduh agen</h3>
              <p>
                Unduh binary agen di server tempat kamu ingin melakukan restore (perintahnya sama
                dengan Langkah 1 instalasi).
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={downloadCommand} lang="id" />
                </div>
              </div>
              <h3 id="restore-step-2">Langkah 2: Hentikan PostgreSQL</h3>
              <p>PostgreSQL harus dihentikan sebelum restore. Direktori tujuan harus kosong.</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pg_ctl -D &lt;PGDATA_DIR&gt; stop</code>
                </pre>
              </div>
              <p>Untuk Docker:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker stop &lt;CONTAINER_NAME&gt;</code>
                </pre>
              </div>
              <h3 id="restore-step-3">Langkah 3: Jalankan restore</h3>
              <p>
                Ganti <code>&lt;YOUR_AGENT_TOKEN&gt;</code> dengan token agen kamu dan{" "}
                <code>&lt;PGDATA_DIR&gt;</code> dengan path direktori data PostgreSQL yang kosong.
              </p>
              <p>
                <strong>Instalasi di host:</strong>
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={restoreCommand} lang="id" />
                </div>
              </div>
              <p>
                <strong>Instalasi Docker</strong> (<code>&lt;HOST_PGDATA_PATH&gt;</code> adalah path
                di host yang akan di-mount sebagai volume pgdata container):
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandDocker}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={restoreCommandDocker} lang="id" />
                </div>
              </div>
              <p>
                Mount <code>&lt;HOST_PGDATA_PATH&gt;</code> ke path PGDATA container saat membuat
                atau membuat ulang container postgres. Path bergantung pada versi mayor: PostgreSQL
                18+ menggunakan <code>/var/lib/postgresql/&lt;major&gt;/docker</code>; PostgreSQL 17
                dan sebelumnya menggunakan <code>/var/lib/postgresql/data</code>.
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerVolumeMountExample}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerVolumeMountExample} lang="id" />
                </div>
              </div>
              <p>
                Untuk <strong>Point-in-Time Recovery</strong> (PITR), tambahkan{" "}
                <code>--target-time</code> dengan timestamp RFC 3339 (misalnya{" "}
                <code>2025-01-15T14:30:00Z</code>):
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{restoreCommandPitr}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={restoreCommandPitr} lang="id" />
                </div>
              </div>
              <h3 id="restore-step-4">Langkah 4: Tangani archive_command</h3>
              <p>
                Backup yang di-restore menyertakan konfigurasi <code>archive_command</code> asli.
                PostgreSQL akan gagal mengarsipkan file WAL setelah pemulihan kecuali kamu melakukan
                salah satu langkah berikut:
              </p>
              <ul>
                <li>
                  <strong>Hubungkan kembali agen</strong>: mount direktori antrean WAL dan jalankan
                  agen Databasus pada instance hasil restore, sama seperti pengaturan awal.
                </li>
                <li>
                  <strong>Nonaktifkan pengarsipan</strong>: jika belum membutuhkan backup
                  berkelanjutan, jadikan pengaturan arsip sebagai komentar atau reset pengaturan
                  tersebut di <code>postgresql.auto.conf</code>:
                </li>
              </ul>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{archiveCommandCleanup}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={archiveCommandCleanup} lang="id" />
                </div>
              </div>
              <h3 id="restore-step-5">Langkah 5: Jalankan PostgreSQL</h3>
              <p>
                Jalankan PostgreSQL untuk memulai pemulihan WAL. PostgreSQL akan menerapkan ulang
                segmen WAL secara otomatis.
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>pg_ctl -D &lt;PGDATA_DIR&gt; start</code>
                </pre>
              </div>
              <p>Untuk Docker:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker start &lt;CONTAINER_NAME&gt;</code>
                </pre>
              </div>
              <h3 id="restore-step-6">Langkah 6: Bersihkan file sementara</h3>
              <p>Setelah pemulihan selesai, hapus direktori restore WAL:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>rm -rf &lt;PGDATA_DIR&gt;/databasus-wal-restore/</code>
                </pre>
              </div>
              <h2 id="how-it-works">Cara kerja</h2>
              <p>
                Agen Databasus adalah binary Go ringan yang menjalankan dua proses secara bersamaan:
              </p>
              <ul>
                <li>
                  <strong>WAL streaming</strong>: mengambil file segmen WAL dari direktori antrean
                  kira-kira setiap 10 detik dan mengunggahnya ke Databasus
                </li>
                <li>
                  <strong>Base backup berkala</strong>: menjalankan <code>pg_basebackup</code>{" "}
                  sesuai jadwal untuk membuat backup fisik penuh cluster database
                </li>
              </ul>
              <p>
                Saat restore, agen mengunduh base backup dan semua segmen WAL yang diperlukan, lalu
                mengatur <code>recovery.signal</code> dan <code>restore_command</code> di{" "}
                <code>postgresql.auto.conf</code>. Saat PostgreSQL dijalankan, segmen WAL diterapkan
                ulang hingga mencapai titik pemulihan yang dituju.
              </p>
              <p>
                Agen selalu memulai koneksi ke Databasus (koneksi keluar). Server database tidak
                perlu menerima koneksi masuk dari Databasus, sehingga cocok untuk jaringan privat
                dan lingkungan di balik firewall.
              </p>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
