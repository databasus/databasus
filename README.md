<div align="center">
  <img src="assets/logo.svg" alt="Databasus Logo" width="250"/>

  <h3>PostgreSQL backup tool</h3>
  <p>Databasus is a free, open source and self-hosted tool to backup PostgreSQL. Make backups with different storages (S3, Google Drive, FTP, etc.) and notifications about progress (Slack, Discord, Telegram, etc.). With a focus on Point-in-Time Recovery at low RPO/RTO</p>
  
  <!-- Badges -->
   [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
  [![MySQL](https://img.shields.io/badge/MySQL-4479A1?logo=mysql&logoColor=white)](https://www.mysql.com/)
  [![MariaDB](https://img.shields.io/badge/MariaDB-003545?logo=mariadb&logoColor=white)](https://mariadb.org/)
  [![MongoDB](https://img.shields.io/badge/MongoDB-47A248?logo=mongodb&logoColor=white)](https://www.mongodb.com/)
  <br />
  [![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
  [![Docker Pulls](https://img.shields.io/docker/pulls/databasus/databasus?color=brightgreen)](https://hub.docker.com/r/databasus/databasus)
  [![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey)](https://github.com/databasus/databasus)
  [![Self Hosted](https://img.shields.io/badge/self--hosted-yes-brightgreen)](https://github.com/databasus/databasus)
  [![Open Source](https://img.shields.io/badge/open%20source-❤️-red)](https://github.com/databasus/databasus)

  <p>
    <b>English</b> •
    <a href="assets/readme/README.ru.md">Русский</a> •
    <a href="assets/readme/README.es.md">Español</a> •
    <a href="assets/readme/README.pt.md">Português</a> •
    <a href="assets/readme/README.zh.md">中文</a> •
    <a href="assets/readme/README.fr.md">Français</a> •
    <a href="assets/readme/README.id.md">Bahasa Indonesia</a>
  </p>

  <p>
    <a href="#-features">Features</a> •
    <a href="#-installation">Installation</a> •
    <a href="#-usage">Usage</a> •
    <a href="#-license">License</a> •
    <a href="#-contributing">Contributing</a>
  </p>

  <p style="margin-top: 20px; margin-bottom: 20px; font-size: 1.2em;">
    <a href="https://databasus.com" target="_blank"><strong>🌐 Databasus website</strong></a>
  </p>
  
  <img src="assets/dashboard-dark.svg" alt="Databasus Dark Dashboard" width="800" style="margin-bottom: 10px;"/>

  <img src="assets/dashboard.svg" alt="Databasus Dashboard" width="800"/>
</div>

---

## ✨ Features

### 📦 **Backup types**

- **Physical**: file-level copy of the entire database cluster over PostgreSQL native incremental backups mechanism (read more)
  - **Full**: a complete, self-contained copy of the cluster
  - **Incremental**: stores only what changed since the previous full backup, so backups stay small and fast
  - **WAL streaming**: continuously captures the database write stream, enabling Point-in-time recovery (PITR). Designed for disaster recovery and near-zero data loss
- **Logical**: native dump of the database in its engine-specific binary format (compressed, suitable for parallel restore)

### 🔄 **Scheduled backups**

- **Flexible scheduling**: hourly, daily, weekly, monthly or cron
- **Precise timing**: run backups at specific times (e.g., 4 AM during low traffic)
- **Smart compression**: 4-8x space savings with balanced compression (~20% overhead)

### 🧪 **Restore verification** <a href="https://databasus.com/restore-verification">(docs)</a>

Databasus performs a real restore to confirm backups are usable, not just intact on disk or checksum check.

- **Triggers**: after each backup or on a flexible schedule (hourly, daily, weekly, monthly or cron)
- **Real restore**: spins up a database container, runs the restore and checks the restored size against the backup
- **Report**: lists every table with its row count
- **Optional notifications**: send the report or failure-only alerts through any configured notifier

### 🗑️ **Retention policies**

- **Time period**: Keep backups for a fixed duration (e.g., 7 days, 3 months, 1 year)
- **Count**: Keep a fixed number of the most recent backups (e.g., last 30)
- **GFS (Grandfather-Father-Son)**: Layered retention — keep hourly, daily, weekly, monthly and yearly backups independently for fine-grained long-term history (enterprises requirement)
- **Size limits**: Set per-backup and total storage size caps to control storage usage

### 🗄️ **Multiple storage destinations** <a href="https://databasus.com/storages">(view supported)</a>

- **Local storage**: Keep backups on your VPS/server
- **Cloud storage**: S3, Cloudflare R2, Google Drive, NAS, Dropbox, SFTP, Rclone and more
- **Secure**: All data stays under your control

### 📱 **Notifications** <a href="https://databasus.com/notifiers">(view supported)</a>

- **Multiple channels**: Email, Telegram, Slack, Discord, Teams, Mattermost, webhooks
- **Real-time updates**: Success and failure notifications
- **Team integration**: Perfect for DevOps workflows

### 🔒 **Enterprise-grade security** <a href="https://databasus.com/security">(docs)</a>

- **AES-256-GCM encryption**: Enterprise-grade protection for backup files
- **Zero-trust storage**: Backups are encrypted and remain useless to attackers, so you can safely store them in shared storage like S3, Azure Blob Storage, etc.
- **Encryption for secrets**: Any sensitive data is encrypted and never exposed, even in logs or error messages
- **Read-only user**: Databasus uses a read-only user by default for backups and never stores anything that can modify your data
- **Two-factor authentication**: Password sign-in can require a six-digit code emailed to the account. Sign-in through Google or GitHub keeps relying on the provider's own checks

### 👥 **Suitable for teams** <a href="https://databasus.com/access-management">(docs)</a>

- **Workspaces**: Group databases, notifiers and storages for different projects or teams
- **Access management**: Control who can view or manage specific databases with role-based permissions
- **Audit logs**: Track all system activities and changes made by users
- **User roles**: Assign viewer, member, admin or owner roles within workspaces
- **OpenTelemetry logs**: Export application and audit logs to an external system (by default they are also written to a local file)

### 🎨 **UX-Friendly**

- **Designer-polished UI**: Clean, intuitive interface crafted with attention to detail
- **Dark & light themes**: Choose the look that suits your workflow
- **Mobile adaptive**: Check your backups from anywhere on any device

### 💾 **Supported databases**

- **PostgreSQL**: 14, 15, 16, 17 and 18 (physical and logical)
- **MySQL**: 5.7, 8.0, 8.4, 9 and 26 (logical only)
- **MariaDB**: 5.5, 10, 11, 12 and 13 (logical only)
- **MongoDB**: 4.2+, 5, 6, 7, 8 and 9 (logical only)

### 🐳 **Self-hosted & secure**

- **Docker-based**: Easy deployment and management
- **Privacy-first**: All your data stays on your infrastructure
- **Open source**: Apache 2.0 licensed, inspect every line of code
- **Build-in SSH**: Connect to your databasus via SSH tunnel

### 📦 Installation <a href="https://databasus.com/installation">(docs)</a>

You have four ways to install Databasus:

- Automated script (recommended)
- Simple Docker run
- Docker Compose setup
- Kubernetes with Helm

<img src="assets/healthchecks.svg" alt="Databasus Dashboard" width="800"/>

---

## 📦 Installation

You have four ways to install Databasus: automated script (recommended), simple Docker run, or Docker Compose setup.

### Option 1: Automated installation script (recommended, Linux only)

The installation script will:

- ✅ Install Docker with Docker Compose (if not already installed)
- ✅ Set up Databasus
- ✅ Configure automatic startup on system reboot

```bash
sudo apt-get install -y curl && \
sudo curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh \
| sudo bash
```

### Option 2: Simple Docker run

The easiest way to run Databasus:

```bash
docker run -d \
  --name databasus \
  -p 4005:4005 \
  -v ./databasus-data:/databasus-data \
  --restart unless-stopped \
  databasus/databasus:latest
```

_The same image lives on GitHub's registry — use `ghcr.io/databasus/databasus:latest` if Docker Hub rate-limits your pull._

This single command will:

- ✅ Start Databasus
- ✅ Store all data in `./databasus-data` directory
- ✅ Automatically restart on system reboot

### Option 3: Docker Compose setup

Create a `docker-compose.yml` file with the following configuration:

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

Then run:

```bash
docker compose up -d
```

### Option 4: Kubernetes with Helm

For Kubernetes deployments, install directly from the OCI registry.

_Add `--set image.repository=ghcr.io/databasus/databasus` to any of the commands below to pull image from GHCR instead of Docker Hub._

**With ClusterIP + port-forward (development/testing):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace
```

```bash
kubectl port-forward svc/databasus-service 4005:4005 -n databasus
# Access at http://localhost:4005
```

**With LoadBalancer (cloud environments):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set service.type=LoadBalancer
```

```bash
kubectl get svc databasus-service -n databasus
# Access at http://<EXTERNAL-IP>:4005
```

**With Ingress (domain-based access):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=backup.example.com
```

For more options (NodePort, TLS, HTTPRoute for Gateway API), see the [Helm chart README](deploy/helm/README.md).

---

## 🚀 Usage

1. **Create the first account**: Navigate to `http://localhost:4005` and sign up. The first account created on an instance administers it
2. **Add your first database for backup**: Click "New Database" and follow the setup wizard
3. **Configure schedule**: Choose from hourly, daily, weekly, monthly or cron intervals
4. **Set database connection**: Enter your database credentials and connection details
5. **Choose storage**: Select where to store your backups (local, S3, Google Drive, etc.)
6. **Configure retention policy**: Choose time period, count or GFS to control how long backups are kept
7. **Add notifications** (optional): Configure email, Telegram, Slack, Mattermost or webhook notifications
8. **Save and start**: Databasus will validate settings and begin the backup schedule

### 🔑 Resetting password <a href="https://databasus.com/password">(docs)</a>

If you need to reset the password, you can use the built-in password reset command:

```bash
docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com"
```

Replace `owner@example.com` with the actual email address of the account whose password you want to reset.

If you do not remember which address administers the instance, list the administrator accounts:

```bash
docker exec -it databasus ./main --list-admins
```

The output names every administrator account with its email address, display name, creation date and active state, and marks the one the instance recognizes as its administrator. It prints no password material.

To provision an account without signing up in the browser, for example from a deployment script, create it from the console:

```bash
docker exec -it databasus ./main --create-user --email="ops@example.com" --password="YourSecurePassword123" --role=admin
```

`--role` is `admin` or `member`, the password needs at least 8 characters, and `--name` optionally sets the display name. On an instance that holds no account yet, the first account must use `--role=admin` and administers the instance. Running the command again for the same account changes nothing, so a deployment can run it on every rollout; rotate the password with `--new-password`.

### 💾 Backuping Databasus itself

After installation, it is also recommended to <a href="https://databasus.com/faq#backup-databasus">backup your Databasus itself</a> or, at least, to copy secret key used for encryption (30 seconds is needed). So you are able to restore from your encrypted backups if you lose access to the server with Databasus or it is corrupted.

---

## 🛡️ Security & reliability engineering

Databasus works with sensitive data, so preventing vulnerabilities, unauthorised access and data leaks is a primary concern. We invest in this on both sides of the system: in the code itself (permission checks, encryption, careful handling of secrets) and in the infrastructure around it (dependency analysis, CVE response, DevSecOps best practices). The pipeline below runs automatically on every commit and PR. No single layer is enough on its own, but together they reduce the chance of vulnerable code, unsafe dependencies, broken images, or non-restorable backups reaching a release.

For static analysis we combine several independent passes. CodeQL scans the full codebase for security issues. On every PR, gitleaks scans for leaked secrets and semgrep checks the code against security rules. Dockerfiles and CI workflows get extra rules of their own (pinned action references, least-privilege permissions, suspicious base images), so insecure patterns are flagged before they ever merge. On top of these per-PR checks, Codex Security from OpenAI runs regular, deeper audits of the whole codebase. It's a separate program that catches architectural and cross-cutting issues narrow PR-time scans can miss.

On the dependency side, Dependabot watches all of our dependencies against the GitHub Advisory Database and surfaces CVEs within minutes of publication. Updates run through a cooldown so newly-published versions get a chance to mature before we adopt them. This is a deliberate defence against compromised-package incidents like supply-chain attack. The Dependency Review Action blocks any PR that introduces a new HIGH or CRITICAL CVE outright.

Container images are scanned with Trivy on every build. A separate Trivy pass on the Dockerfile catches misconfigurations before they make it into an image. All GitHub Actions are pinned to full commit SHAs rather than floating tags like `@v4` or `@main`, which have been an active attack vector in 2025. Workflows default to least-privilege permissions and only elevate per-job when genuinely needed.

Critical paths are covered by both unit and integration tests, run against real database containers for every supported engine and major version. Restore is the path that matters most for a backup tool, so we test it explicitly: every PR runs full backup-then-restore cycles against those same real containers, verifying that backups can actually be restored end-to-end, not just written successfully. The rest of the CI/CD pipeline runs lint, type-check, the full test suite, image smoke tests and multi-architecture builds on every PR. A release only ships if all of it passes.

Found a vulnerability? Report it via the GitHub Security tab. See [SECURITY.md](https://github.com/databasus/databasus?tab=security-ov-file#readme). Security reports are the highest-priority work queue. For runtime application security (AES-256-GCM at rest, zero-trust storage, encrypted secrets, read-only DB user by default) see [Enterprise-grade security](#-enterprise-grade-security-docs) in the Features section above.

---

## 📝 License

This project is licensed under the Apache 2.0 License - see the [LICENSE](LICENSE) file for details

## 🤝 Contributing

Contributions are welcome! Read the <a href="https://databasus.com/contribute">contributing guide</a> for more details, priorities and rules. If you want to contribute but don't know where to start, message me on Telegram [@rostislav_dugin](https://t.me/rostislav_dugin)

Also you can join our large community of developers, DBAs and DevOps engineers on Telegram [@databasus_community](https://t.me/databasus_community).
