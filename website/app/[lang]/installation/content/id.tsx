import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Instalasi - Dokumentasi Databasus",
  description:
    "Pelajari cara memasang Databasus dengan script otomatis, Docker run, Docker Compose, Helm untuk Kubernetes, atau reverse proxy Caddy. Instalasi sederhana tanpa konfigurasi untuk sistem backup PostgreSQL self-hosted kamu.",
  keywords: [
    "instalasi Databasus",
    "instalasi Docker",
    "pengaturan backup PostgreSQL",
    "backup self-hosted",
    "Docker Compose",
    "instalasi backup database",
    "pengaturan pg_dump",
    "Kubernetes",
    "Helm chart",
    "deployment K8s",
    "reverse proxy Caddy",
    "pengaturan HTTPS",
    "pemeriksaan ketersediaan",
    "pemantauan",
    "liveness probe",
  ],
  openGraph: {
    title: "Instalasi - Dokumentasi Databasus",
    description:
      "Pelajari cara memasang Databasus dengan script otomatis, Docker run, Docker Compose, Helm untuk Kubernetes, atau reverse proxy Caddy. Instalasi sederhana tanpa konfigurasi untuk sistem backup PostgreSQL self-hosted kamu.",
    type: "article",
    url: getLocalizedUrl("id", "installation"),
    locale: OG_LOCALES.id,
  },
  twitter: {
    card: "summary",
    title: "Instalasi - Dokumentasi Databasus",
    description:
      "Pelajari cara memasang Databasus dengan script otomatis, Docker run, Docker Compose, Helm untuk Kubernetes, atau reverse proxy Caddy. Instalasi sederhana tanpa konfigurasi untuk sistem backup PostgreSQL self-hosted kamu.",
  },
  alternates: {
    canonical: getLocalizedUrl("id", "installation"),
    languages: getLanguageAlternates("installation"),
  },
  robots: "index, follow",
};

export default function InstallationPage() {
  const installScript = `sudo apt-get install -y curl && \\
sudo curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh | sudo bash`;
  const dockerRun = `docker run -d \\
  --name databasus \\
  -p 4005:4005 \\
  -v ./databasus-data:/databasus-data \\
  --restart unless-stopped \\
  databasus/databasus:latest`;
  const dockerCompose = `services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped`;
  const helmInstallClusterIP = `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace`;
  const helmPortForward = `kubectl port-forward svc/databasus-service 4005:4005 -n databasus
# Access at http://localhost:4005`;
  const helmInstallLoadBalancer = `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace \\
  --set service.type=LoadBalancer`;
  const helmGetSvc = `kubectl get svc databasus-service -n databasus
# Access at http://<EXTERNAL-IP>:4005`;
  const helmInstallIngress = `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace \\
  --set ingress.enabled=true \\
  --set ingress.hosts[0].host=backup.example.com`;
  const helmUpgrade = `helm upgrade databasus oci://ghcr.io/databasus/charts/databasus -n databasus`;
  const dockerComposeCaddy = `services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped
    # No port exposed - Caddy handles external access

  caddy:
    container_name: caddy
    image: caddy:latest
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - ./caddy-data:/data
      - ./caddy-config:/config
    restart: unless-stopped
    depends_on:
      - databasus`;
  const caddyfile = `backup.example.com {
    reverse_proxy databasus:4005
}`;
  const healthEndpoint = `GET http://<host>:4005/api/v1/system/health`;

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Instalasi - Dokumentasi Databasus",
            description:
              "Pelajari cara memasang Databasus dengan script otomatis, Docker run, Docker Compose, Helm untuk Kubernetes, atau reverse proxy Caddy. Instalasi sederhana tanpa konfigurasi untuk sistem backup PostgreSQL self-hosted kamu.",
            author: { "@type": "Organization", name: "Databasus" },
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: { "@type": "ImageObject", url: "https://databasus.com/logo.svg" },
            },
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cara memasang Databasus",
            description:
              "Panduan langkah demi langkah untuk memasang Databasus, alat backup PostgreSQL",
            step: [
              {
                "@type": "HowToStep",
                name: "Script instalasi otomatis",
                text: "Jalankan script instalasi otomatis untuk memasang Docker dan menyiapkan Databasus agar berjalan otomatis saat sistem dinyalakan.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Jalankan perintah curl untuk mengunduh dan menjalankan script instalasi",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Docker Run",
                text: "Gunakan perintah Docker run untuk menjalankan container Databasus dengan cepat dan menyimpan datanya secara persisten.",
              },
              {
                "@type": "HowToStep",
                name: "Docker Compose",
                text: "Buat file docker-compose.yml dan gunakan Docker Compose untuk mengelola deployment.",
              },
              {
                "@type": "HowToStep",
                name: "Kubernetes dengan Helm",
                text: "Gunakan Helm chart resmi untuk menjalankan Databasus di Kubernetes dengan StatefulSet, penyimpanan persisten, dan ingress opsional.",
              },
              {
                "@type": "HowToStep",
                name: "Menjalankan dengan reverse proxy Caddy",
                text: "Gunakan Docker Compose dengan Caddy untuk deployment production dengan sertifikat HTTPS otomatis.",
              },
            ],
          }),
        }}
      />
      <DocsNavbarComponent lang="id" />
      <div className="flex min-h-screen bg-[#0F1115]">
        <DocsSidebarComponent lang="id" />
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="installation">Instalasi</h1>
              <p className="text-lg text-gray-400">
                Ada beberapa cara memasang Databasus: script otomatis (direkomendasikan), Docker
                run, Docker Compose, Helm untuk Kubernetes, atau Docker Compose dengan Caddy untuk
                deployment production.
              </p>
              <h2 id="system-requirements">Kebutuhan sistem</h2>
              <p>
                Databasus membutuhkan sumber daya minimum berikut agar bisa berjalan dengan baik:
              </p>
              <ul>
                <li>
                  <strong>CPU</strong>: Setidaknya 1 core CPU
                </li>
                <li>
                  <strong>RAM</strong>: Minimum 500 MB RAM
                </li>
                <li>
                  <strong>Penyimpanan</strong>: 5 GB untuk instalasi, ditambah ruang sesuai
                  kebutuhan backup kamu
                </li>
                <li>
                  <strong>Docker</strong>: Docker Engine 20.10+ dan Docker Compose v2.0+
                </li>
              </ul>
              <h2 id="option-1-automated-script">
                Opsi 1: script instalasi (direkomendasikan, khusus Linux)
              </h2>
              <p>Script instalasi akan:</p>
              <ul>
                <li>✅ Memasang Docker beserta Docker Compose (jika belum terpasang)</li>
                <li>✅ Menyiapkan Databasus</li>
                <li>✅ Mengatur agar Databasus berjalan otomatis setelah sistem dimulai ulang</li>
              </ul>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{installScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={installScript} lang="id" />
                </div>
              </div>
              <p>
                Dengan cara ini, Databasus dipasang di direktori <code>/opt/databasus</code>.
              </p>
              <h2 id="option-2-docker-run">Opsi 2: Docker run sederhana</h2>
              <p>Cara paling mudah menjalankan Databasus:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRun}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerRun} lang="id" />
                </div>
              </div>
              <p>Satu perintah ini akan:</p>
              <ul>
                <li>✅ Menjalankan Databasus</li>
                <li>
                  ✅ Menyimpan semua data di direktori <code>./databasus-data</code>
                </li>
                <li>✅ Menjalankan ulang Databasus secara otomatis setelah sistem dimulai ulang</li>
              </ul>
              <h2 id="option-3-docker-compose">Opsi 3: pengaturan Docker Compose</h2>
              <p>
                Buat file <code>docker-compose.yml</code> dengan konfigurasi berikut:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerCompose}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerCompose} lang="id" />
                </div>
              </div>
              <p>Lalu jalankan:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text="docker compose up -d" lang="id" />
                </div>
              </div>
              <p>Proses awal menjalankan layanan bisa memakan waktu hingga ~2 menit.</p>
              <h2 id="option-4-helm">Opsi 4: Kubernetes dengan Helm</h2>
              <p>
                Untuk deployment Kubernetes, pasang langsung dari OCI registry. Pilih metode akses
                yang sesuai dengan lingkungan kamu.
              </p>
              <h3 id="helm-clusterip">Dengan ClusterIP + port-forward (development)</h3>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallClusterIP}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={helmInstallClusterIP} lang="id" />
                </div>
              </div>
              <p>Akses melalui port-forward:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmPortForward}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={helmPortForward} lang="id" />
                </div>
              </div>
              <h3 id="helm-loadbalancer">Dengan LoadBalancer (lingkungan cloud)</h3>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallLoadBalancer}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={helmInstallLoadBalancer} lang="id" />
                </div>
              </div>
              <p>Dapatkan IP eksternal, lalu akses Databasus:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmGetSvc}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={helmGetSvc} lang="id" />
                </div>
              </div>
              <h3 id="helm-ingress">Dengan Ingress (akses melalui domain)</h3>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallIngress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={helmInstallIngress} lang="id" />
                </div>
              </div>
              <p>
                Untuk opsi lain (NodePort, TLS, HTTPRoute untuk Gateway API), lihat{" "}
                <a
                  href="https://github.com/databasus/databasus/tree/main/deploy/helm"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  dokumentasi Helm chart
                </a>
                .
              </p>
              <h2 id="caddy-reverse-proxy">Menjalankan dengan reverse proxy Caddy</h2>
              <p>
                Untuk deployment production, kamu bisa menggunakan{" "}
                <a
                  href="https://caddyserver.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Caddy
                </a>{" "}
                sebagai reverse proxy agar mendapatkan sertifikat HTTPS otomatis dan akses aman ke
                Databasus. Berikut konfigurasi lengkap Docker Compose dengan Caddy.
              </p>
              <h3 id="caddy-docker-compose">Docker Compose dengan Caddy</h3>
              <p>
                Buat file <code>docker-compose.yml</code>:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeCaddy}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={dockerComposeCaddy} lang="id" />
                </div>
              </div>
              <p>
                Buat <code>Caddyfile</code> di direktori yang sama:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{caddyfile}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={caddyfile} lang="id" />
                </div>
              </div>
              <p>Lalu jalankan layanan:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text="docker compose up -d" lang="id" />
                </div>
              </div>
              <p>Konfigurasi ini menyediakan:</p>
              <ul>
                <li>✅ HTTPS otomatis dengan sertifikat Let&apos;s Encrypt</li>
                <li>✅ Pengalihan HTTP ke HTTPS</li>
                <li>✅ Reverse proxy ke Databasus</li>
                <li>✅ Data persisten untuk Caddy dan Databasus</li>
              </ul>
              <p>
                Ganti <code>backup.example.com</code> dengan domain kamu. Pastikan DNS domain
                mengarah ke alamat IP server sebelum menjalankan layanan.
              </p>
              <h2 id="getting-started">Mulai menggunakan Databasus</h2>
              <p>Setelah instalasi:</p>
              <ol>
                <li>
                  <strong>Jalankan dan akses Databasus</strong>: Jalankan Databasus, lalu buka{" "}
                  <code>http://localhost:4005</code>
                </li>
                <li>
                  <strong>Buat tugas backup pertama</strong>: Klik &quot;New Backup&quot;, lalu atur
                  koneksi database PostgreSQL kamu
                </li>
                <li>
                  <strong>Atur jadwal</strong>: Tentukan jadwal backup (per jam, harian, mingguan,
                  bulanan, atau cron)
                </li>
                <li>
                  <strong>Pilih tujuan penyimpanan</strong>: Pilih tempat menyimpan backup (lokal,
                  S3, Google Drive, dan sebagainya)
                </li>
                <li>
                  <strong>Atur notifikasi</strong>: Tambahkan kanal notifikasi (Slack, Telegram,
                  Discord) untuk menerima pemberitahuan status backup
                </li>
                <li>
                  <strong>Mulai backup</strong>: Simpan konfigurasi dan pantau backup pertama kamu!
                </li>
              </ol>
              <h2 id="health-checks">Pemeriksaan ketersediaan</h2>
              <h3 id="docker-health-check">Pemeriksaan ketersediaan Docker</h3>
              <p>
                Pemeriksaan ketersediaan bawaan aktif otomatis untuk <code>docker run</code> dan
                Docker Compose. Container dilaporkan <code>healthy</code> setelah Databasus mulai
                melayani request (setelah masa tenggang singkat saat startup). Pemeriksaan ini hanya
                memastikan aplikasi merespons, sehingga kondisi yang tidak kritis, seperti ruang
                disk menipis, tidak memicu restart container.
              </p>
              <h3 id="monitoring-endpoint">Endpoint pemantauan / status</h3>
              <p>Untuk pemantauan uptime dan dashboard status:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{healthEndpoint}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={healthEndpoint} lang="id" />
                </div>
              </div>
              <p>
                Mengembalikan <code>200</code> jika sehat, atau <code>503</code> beserta alasannya
                jika ada yang perlu diperhatikan: database internal, cache, penggunaan disk (di atas
                95%), alat client database, penjadwal backup, dan keaktifan agen verifikasi. Tidak
                memerlukan autentikasi, dan CORS terbuka untuk alat pemantauan berbasis browser.
              </p>
              <p>
                <strong>⚠️ Penting:</strong> gunakan hanya untuk pemantauan dan pemberitahuan, bukan
                sebagai liveness probe container atau Kubernetes. Endpoint ini mengembalikan{" "}
                <code>503</code> ketika kondisi memburuk tetapi layanan masih berjalan (misalnya
                disk hampir penuh), sehingga bisa memicu restart container yang sebenarnya masih
                berfungsi.
              </p>
              <h3 id="kubernetes-health-check">Kubernetes</h3>
              <p>
                Gunakan liveness/readiness probe yang menjalankan <code>databasus healthcheck</code>
                ; gunakan endpoint{" "}
                <a href="#monitoring-endpoint" className="text-blue-400 hover:text-blue-300">
                  /api/v1/system/health
                </a>{" "}
                hanya untuk pemantauan eksternal.
              </p>
              <h2 id="how-to-update">Cara memperbarui Databasus</h2>
              <h3 id="update-docker">Memperbarui instalasi Docker</h3>
              <p>
                Untuk memperbarui Databasus yang berjalan melalui Docker, hentikan layanan,
                bersihkan cache Docker, lalu jalankan kembali container.
              </p>
              <ol>
                <li>
                  Buka direktori instalasi Databasus (biasanya <code>/opt/databasus</code>)
                </li>
                <li>
                  Hentikan container: <code>docker compose stop</code>
                </li>
                <li>
                  Bersihkan cache Docker: <code>docker system prune -a</code>
                </li>
                <li>
                  Jalankan kembali container: <code>docker compose up -d</code>
                </li>
              </ol>
              <p>
                Langkah ini akan mengambil versi terbaru Databasus dari Docker Hub (jika kamu tidak
                menetapkan versi tertentu di file <code>docker-compose.yml</code>).
              </p>
              <h3 id="update-helm">Memperbarui instalasi Helm</h3>
              <p>
                Untuk memperbarui Databasus yang berjalan di Kubernetes melalui Helm, gunakan
                perintah upgrade:
              </p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmUpgrade}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={helmUpgrade} lang="id" />
                </div>
              </div>
              <p>
                Jika menggunakan nilai khusus, tambahkan <code>-f values.yaml</code> atau flag{" "}
                <code>--set</code> agar konfigurasi kamu tetap digunakan. Helm akan melakukan
                rolling update ke versi baru.
              </p>
              <h2 id="postgresus-migration">Migrasi dari Postgresus</h2>
              <p>
                Databasus adalah nama baru Postgresus. Jika saat ini menggunakan Postgresus, kamu
                bisa tetap menggunakannya atau bermigrasi ke Databasus.
              </p>
              <p>
                <strong>Penting:</strong> Mengganti nama image Docker saja tidak cukup karena
                Postgresus dan Databasus menggunakan folder data dan nama database internal yang
                berbeda.
              </p>
              <p>Untuk bermigrasi:</p>
              <ol>
                <li>
                  Hentikan container Postgresus: <code>docker compose stop</code>
                </li>
                <li>
                  Pasang Databasus dengan salah satu metode di atas (gunakan path volume yang
                  berbeda, <code>./databasus-data</code>)
                </li>
                <li>
                  Buat ulang database, penyimpanan, dan kanal notifikasi secara manual di Databasus
                </li>
              </ol>
              <p>
                Kamu bisa menjalankan Postgresus dan Databasus bersamaan selama migrasi dengan
                menggunakan port dan path volume yang berbeda.
              </p>
              <h2 id="troubleshooting">Pemecahan masalah</h2>
              <h3 id="container-wont-start">Container tidak bisa dijalankan</h3>
              <p>Jika container gagal dijalankan, periksa log:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker logs databasus</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text="docker logs databasus" lang="id" />
                </div>
              </div>
              <h3 id="port-already-in-use">Port sudah digunakan</h3>
              <p>Jika port 4005 sudah digunakan, kamu bisa menggantinya di docker-compose.yml:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>
                    ports:{"\n  "}- &quot;8080:4005&quot; # Change 8080 to any available port
                  </code>
                </pre>
              </div>
              <h3 id="permission-denied">Error permission denied</h3>
              <p>Jika ada masalah izin pada direktori data:</p>
              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>
                    sudo chown -R $USER:$USER ./databasus-data{"\n"}chmod -R 755 ./databasus-data
                  </code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton
                    text={`sudo chown -R $USER:$USER ./databasus-data\nchmod -R 755 ./databasus-data`}
                    lang="id"
                  />
                </div>
              </div>
            </article>
          </div>
        </main>
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
