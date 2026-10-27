<div align="center">
  <img src="../logo.svg" alt="Databasus Logo" width="250"/>

  <h3>Outil de sauvegarde PostgreSQL</h3>
  <p>Databasus est un outil gratuit, open source et auto-hébergé pour faire des backups de PostgreSQL. Créez des sauvegardes vers différents stockages (S3, Google Drive, FTP, etc.) avec des notifications sur leur progression (Slack, Discord, Telegram, etc.). Avec un focus sur le Point-in-Time Recovery à faible RPO/RTO</p>
  
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
    <b>Français</b> •
    <a href="README.id.md">Bahasa Indonesia</a>
  </p>

  <p>
    <a href="#-fonctionnalités">Fonctionnalités</a> •
    <a href="#-installation">Installation</a> •
    <a href="#-utilisation">Utilisation</a> •
    <a href="#-licence">Licence</a> •
    <a href="#-contribuer">Contribuer</a>
  </p>

  <p style="margin-top: 20px; margin-bottom: 20px; font-size: 1.2em;">
    <a href="https://databasus.com/fr/" target="_blank"><strong>🌐 Site de Databasus</strong></a>
  </p>
  
  <img src="../dashboard-dark.svg" alt="Databasus Dark Dashboard" width="800" style="margin-bottom: 10px;"/>

  <img src="../dashboard.svg" alt="Databasus Dashboard" width="800"/>
</div>

---

## ✨ Fonctionnalités

### 📦 **Types de sauvegarde**

- **Physique** : copie au niveau fichier de l'ensemble du cluster, appuyée sur le mécanisme natif de sauvegardes incrémentales de PostgreSQL (en savoir plus)
  - **Complète** : une copie intégrale et autonome du cluster
  - **Incrémentale** : ne stocke que ce qui a changé depuis la sauvegarde complète précédente, les fichiers restent donc petits et rapides à produire
  - **Streaming WAL** : capture en continu le flux d'écriture de la base et rend possible la récupération à un instant donné (PITR). Conçu pour la reprise après sinistre et une perte de données quasi nulle
- **Logique** : dump natif de la base dans le format binaire propre à son moteur (compressé, adapté à une restauration parallèle)

### 🔄 **Sauvegardes planifiées**

- **Planification souple** : toutes les heures, chaque jour, chaque semaine, chaque mois ou en cron
- **Horaires précis** : lancez les sauvegardes à l'heure de votre choix (par exemple 4 h du matin, quand le trafic est faible)
- **Compression intelligente** : des fichiers 4 à 8 fois plus petits avec une compression équilibrée (~20 % de surcoût)

### 🧪 **Vérification de restauration** <a href="https://databasus.com/fr/restore-verification/">(docs)</a>

Databasus effectue une vraie restauration pour confirmer que les sauvegardes sont exploitables, au lieu de se contenter de vérifier leur présence sur le disque ou leur somme de contrôle.

- **Déclencheurs** : après chaque sauvegarde ou selon une planification souple (toutes les heures, chaque jour, chaque semaine, chaque mois ou en cron)
- **Vraie restauration** : un conteneur de base de données est démarré, la restauration est exécutée et la taille obtenue est comparée à celle de la sauvegarde
- **Rapport** : chaque table est listée avec son nombre de lignes
- **Notifications facultatives** : envoyez le rapport, ou seulement les alertes d'échec, via n'importe quel canal configuré

### 🗑️ **Politiques de rétention**

- **Période** : conservez les sauvegardes pendant une durée fixe (7 jours, 3 mois, 1 an…)
- **Nombre** : conservez un nombre fixe de sauvegardes récentes (les 30 dernières, par exemple)
- **GFS (Grandfather-Father-Son)** : rétention par couches. Les sauvegardes horaires, quotidiennes, hebdomadaires, mensuelles et annuelles sont conservées indépendamment, pour un historique long terme finement découpé (une exigence des entreprises)
- **Limites de taille** : plafonnez la taille par sauvegarde et la taille totale pour maîtriser l'espace occupé

### 🗄️ **Plusieurs destinations de stockage** <a href="https://databasus.com/fr/storages/">(voir les stockages pris en charge)</a>

- **Stockage local** : gardez les sauvegardes sur votre VPS ou votre serveur
- **Stockage cloud** : S3, Cloudflare R2, Google Drive, NAS, Dropbox, SFTP, Rclone et d'autres
- **Sûr** : toutes les données restent sous votre contrôle

### 📱 **Notifications** <a href="https://databasus.com/fr/notifiers/">(voir les canaux pris en charge)</a>

- **Plusieurs canaux** : e-mail, Telegram, Slack, Discord, Teams, Mattermost, webhooks
- **En temps réel** : notifications de succès comme d'échec
- **Intégration d'équipe** : parfait pour les workflows DevOps

### 🔒 **Sécurité de niveau entreprise** <a href="https://databasus.com/fr/security/">(docs)</a>

- **Chiffrement AES-256-GCM** : une protection de niveau entreprise pour les fichiers de sauvegarde
- **Stockage zero-trust** : les sauvegardes sont chiffrées et restent inutilisables pour un attaquant, vous pouvez donc les déposer sans risque sur un stockage partagé comme S3, Azure Blob Storage, etc.
- **Chiffrement des secrets** : toute donnée sensible est chiffrée et n'est jamais exposée, pas même dans les logs ou les messages d'erreur
- **Utilisateur en lecture seule** : par défaut, Databasus sauvegarde via un utilisateur en lecture seule et ne conserve rien qui permette de modifier vos données
- **Authentification à deux facteurs** : la connexion par mot de passe peut exiger un code à six chiffres envoyé à l'adresse du compte. Se connecter par Google ou GitHub continue de s'appuyer sur les vérifications de ces fournisseurs

### 👥 **Adapté aux équipes** <a href="https://databasus.com/fr/access-management/">(docs)</a>

- **Espaces de travail** : regroupez bases, canaux de notification et stockages par projet ou par équipe
- **Gestion des accès** : décidez qui peut consulter ou gérer telle base, avec des permissions par rôle
- **Journaux d'audit** : suivez toutes les activités du système et les modifications faites par les utilisateurs
- **Rôles utilisateurs** : attribuez les rôles lecteur, membre, administrateur ou propriétaire dans un espace de travail
- **Logs OpenTelemetry** : exportez les logs applicatifs et d'audit vers un système externe (par défaut, ils sont aussi écrits dans un fichier local)

### 🎨 **Pensé pour l'utilisateur**

- **Interface soignée par un designer** : claire, intuitive, travaillée dans le détail
- **Thèmes sombre et clair** : choisissez l'apparence qui vous convient
- **Adapté au mobile** : consultez vos sauvegardes n'importe où, depuis n'importe quel appareil

### 💾 **Bases de données prises en charge**

- **PostgreSQL** : 14, 15, 16, 17 et 18 (physique et logique)
- **MySQL** : 5.7, 8.0, 8.4, 9 et 26 (logique uniquement)
- **MariaDB** : 5.5, 10, 11, 12 et 13 (logique uniquement)
- **MongoDB** : 4.2+, 5, 6, 7, 8 et 9 (logique uniquement)

### 🐳 **Auto-hébergé et sûr**

- **Basé sur Docker** : déploiement et administration simples
- **Vos données d'abord** : tout reste sur votre infrastructure
- **Open source** : sous licence Apache 2.0, chaque ligne de code est consultable
- **SSH intégré** : connectez-vous à votre Databasus via un tunnel SSH

### 📦 Installation <a href="https://databasus.com/fr/installation/">(docs)</a>

Vous avez quatre façons d'installer Databasus :

- Script automatisé (recommandé)
- Simple lancement Docker
- Configuration avec Docker Compose
- Kubernetes avec Helm

<img src="../healthchecks.svg" alt="Databasus Dashboard" width="800"/>

---

## 📦 Installation

Vous avez quatre façons d'installer Databasus : script automatisé (recommandé), simple lancement Docker ou configuration avec Docker Compose.

### Option 1 : script d'installation automatisé (recommandé, Linux uniquement)

Le script d'installation va :

- ✅ Installer Docker avec Docker Compose (s'ils ne sont pas déjà présents)
- ✅ Mettre en place Databasus
- ✅ Configurer le démarrage automatique au redémarrage du système

```bash
sudo apt-get install -y curl && \
sudo curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh \
| sudo bash
```

### Option 2 : simple lancement Docker

La façon la plus rapide de lancer Databasus :

```bash
docker run -d \
  --name databasus \
  -p 4005:4005 \
  -v ./databasus-data:/databasus-data \
  --restart unless-stopped \
  databasus/databasus:latest
```

_La même image est publiée sur le registre de GitHub : utilisez `ghcr.io/databasus/databasus:latest` si Docker Hub limite votre débit de téléchargement._

Cette unique commande va :

- ✅ Démarrer Databasus
- ✅ Stocker toutes les données dans le répertoire `./databasus-data`
- ✅ Redémarrer automatiquement au redémarrage du système

### Option 3 : configuration avec Docker Compose

Créez un fichier `docker-compose.yml` avec la configuration suivante :

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

Puis lancez :

```bash
docker compose up -d
```

### Option 4 : Kubernetes avec Helm

Pour un déploiement Kubernetes, installez directement depuis le registre OCI.

_Ajoutez `--set image.repository=ghcr.io/databasus/databasus` à n'importe quelle commande ci-dessous pour récupérer l'image depuis GHCR au lieu de Docker Hub._

**Avec ClusterIP + port-forward (développement et tests) :**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace
```

```bash
kubectl port-forward svc/databasus-service 4005:4005 -n databasus
# Access at http://localhost:4005
```

**Avec LoadBalancer (environnements cloud) :**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set service.type=LoadBalancer
```

```bash
kubectl get svc databasus-service -n databasus
# Access at http://<EXTERNAL-IP>:4005
```

**Avec Ingress (accès par nom de domaine) :**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=backup.example.com
```

Pour les autres options (NodePort, TLS, HTTPRoute pour Gateway API), consultez le [README du chart Helm](../../deploy/helm/README.md).

---

## 🚀 Utilisation

1. **Créez le premier compte** : rendez-vous sur `http://localhost:4005` et inscrivez-vous. Le premier compte créé sur une instance est celui qui l'administre
2. **Ajoutez votre première base à sauvegarder** : cliquez sur "New Database" et suivez l'assistant
3. **Configurez la planification** : toutes les heures, chaque jour, chaque semaine, chaque mois ou selon un intervalle cron
4. **Renseignez la connexion** : saisissez les identifiants et les paramètres de connexion de votre base
5. **Choisissez le stockage** : indiquez où déposer vos sauvegardes (local, S3, Google Drive, etc.)
6. **Configurez la rétention** : période, nombre ou GFS, selon la durée de conservation voulue
7. **Ajoutez des notifications** (facultatif) : e-mail, Telegram, Slack, Mattermost ou webhook
8. **Enregistrez et démarrez** : Databasus valide les paramètres et lance la planification des sauvegardes

### 🔑 Réinitialiser le mot de passe <a href="https://databasus.com/fr/password/">(docs)</a>

Si vous devez réinitialiser le mot de passe, utilisez la commande intégrée :

```bash
docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com"
```

Remplacez `owner@example.com` par l'adresse e-mail du compte dont vous voulez réinitialiser le mot de passe.

Si vous ne savez plus quelle adresse administre l'instance, listez les comptes administrateurs :

```bash
docker exec -it databasus ./main --list-admins
```

La sortie nomme chaque compte administrateur avec son adresse e-mail, son nom affiché, sa date de création et son état d'activité, et marque celui que l'instance reconnaît comme son administrateur. Aucun mot de passe ni empreinte de mot de passe n'est affiché.

Pour ouvrir un compte sans passer par l'inscription dans le navigateur, par exemple depuis un script de déploiement, créez-le depuis la console :

```bash
docker exec -it databasus ./main --create-user --email="ops@example.com" --password="YourSecurePassword123" --role=admin
```

`--role` vaut `admin` ou `member`, le mot de passe doit compter au moins 8 caractères, et `--name` permet de définir un nom affiché. Sur une instance qui n'a encore aucun compte, le premier doit utiliser `--role=admin` et administre l'instance. Relancer la commande pour le même compte ne change rien : un déploiement peut donc l'exécuter à chaque mise en production. Pour renouveler le mot de passe, utilisez `--new-password`.

### 💾 Sauvegarder Databasus lui-même

Après l'installation, il est aussi recommandé de <a href="https://databasus.com/fr/faq/#backup-databasus">sauvegarder Databasus lui-même</a> ou, au minimum, de copier la clé secrète utilisée pour le chiffrement (30 secondes suffisent). Vous pourrez ainsi restaurer vos sauvegardes chiffrées si vous perdez l'accès au serveur qui héberge Databasus ou s'il est corrompu.

---

## 🛡️ Ingénierie de sécurité et de fiabilité

Databasus manipule des données sensibles, donc la prévention des vulnérabilités, des accès non autorisés et des fuites de données est une préoccupation de premier plan. Nous y investissons des deux côtés du système : dans le code lui-même (contrôles de permissions, chiffrement, gestion soigneuse des secrets) et dans l'infrastructure qui l'entoure (analyse des dépendances, réponse aux CVE, pratiques DevSecOps). Le pipeline ci-dessous s'exécute automatiquement sur chaque commit et chaque PR. Aucune couche ne suffit à elle seule, mais ensemble elles réduisent le risque qu'un code vulnérable, des dépendances dangereuses, des images cassées ou des backups non restaurables atteignent une release.

L'analyse statique s'exécute en plusieurs passes indépendantes. CodeQL analyse l'ensemble du code à la recherche de problèmes de sécurité. Sur chaque PR, gitleaks recherche les secrets divulgués et semgrep vérifie le code selon des règles de sécurité. Les Dockerfiles et les workflows CI ont leurs propres règles supplémentaires (références d'actions épinglées, permissions de moindre privilège, images de base suspectes), si bien que les schémas non sûrs sont signalés avant même d'être fusionnés. En plus de ces contrôles par PR, Codex Security d'OpenAI effectue des audits réguliers et plus approfondis de tout le code. C'est un programme distinct qui détecte les problèmes architecturaux et transverses que les analyses limitées au moment de la PR laissent passer.

Côté dépendances, Dependabot surveille toutes nos dépendances par rapport à la GitHub Advisory Database et fait remonter les CVE quelques minutes après leur publication. Les mises à jour sont appliquées avec un délai, pour laisser aux versions fraîchement publiées le temps de mûrir avant que nous les adoptions. C'est une défense délibérée contre les incidents de paquets compromis, comme les attaques de la chaîne d'approvisionnement. La Dependency Review Action bloque catégoriquement toute PR qui introduit une nouvelle CVE HIGH ou CRITICAL.

Les images de conteneurs sont analysées avec Trivy à chaque build. Une passe Trivy distincte sur le Dockerfile détecte les erreurs de configuration avant qu'elles n'arrivent dans une image. Toutes les GitHub Actions sont épinglées sur des SHA de commit complets plutôt que sur des tags flottants comme `@v4` ou `@main`, qui ont été un vecteur d'attaque actif en 2025. Les workflows utilisent par défaut des permissions de moindre privilège et ne les élèvent par job que lorsque c'est réellement nécessaire.

Les chemins critiques sont couverts par des tests unitaires et d'intégration, exécutés contre de vrais conteneurs de bases de données pour chaque moteur et version majeure pris en charge. La restauration est le chemin qui compte le plus pour un outil de sauvegarde, donc nous la testons explicitement : chaque PR exécute des cycles complets de sauvegarde puis restauration contre ces mêmes conteneurs réels, en vérifiant que les backups peuvent réellement être restaurés de bout en bout, et pas seulement écrits avec succès. Le reste du pipeline CI/CD exécute le lint, la vérification de types, la suite de tests complète, des smoke tests d'image et des builds multi-architecture sur chaque PR. Une release ne sort que si tout passe.

Vous avez trouvé une vulnérabilité ? Signalez-la via l'onglet GitHub Security. Voir [SECURITY.md](https://github.com/databasus/databasus?tab=security-ov-file#readme). Les rapports de sécurité sont traités en priorité absolue. Pour la sécurité applicative à l'exécution (AES-256-GCM au repos, stockage zero-trust, secrets chiffrés, utilisateur de base en lecture seule par défaut), voir [Sécurité de niveau entreprise](#-sécurité-de-niveau-entreprise-docs) dans la section Fonctionnalités ci-dessus.

---

## 📝 Licence

Ce projet est distribué sous licence Apache 2.0 - voir le fichier [LICENSE](../../LICENSE) pour les détails

## 🤝 Contribuer

Les contributions sont les bienvenues ! Lisez le <a href="https://databasus.com/contribute">guide de contribution</a> pour les détails, les priorités et les règles. Si vous voulez contribuer sans savoir par où commencer, écrivez-moi sur Telegram [@rostislav_dugin](https://t.me/rostislav_dugin)

Vous pouvez aussi rejoindre notre grande communauté de développeurs, de DBA et d'ingénieurs DevOps sur Telegram [@databasus_community](https://t.me/databasus_community).
