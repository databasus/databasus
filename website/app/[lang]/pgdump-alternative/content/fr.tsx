import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Alternative à pg_dump - Databasus, outil de sauvegarde PostgreSQL",
  description:
    "Databasus s'appuie sur pg_dump et étend ses fonctionnalités : gestion des backups, interface web, planification automatique, stockage cloud, notifications, travail en équipe et chiffrement.",
  keywords: [
    "alternative à pg_dump",
    "pg_dump GUI",
    "automatisation pg_dump",
    "interface web pg_dump",
    "outil de sauvegarde PostgreSQL",
    "planificateur pg_dump",
    "pg_dump stockage cloud",
    "chiffrement pg_dump",
    "automatisation des sauvegardes PostgreSQL",
    "wrapper pg_dump",
  ],
  openGraph: {
    title: "Alternative à pg_dump - Databasus, outil de sauvegarde PostgreSQL",
    description:
      "Databasus s'appuie sur pg_dump et étend ses fonctionnalités : gestion des backups, interface web, planification automatique, stockage cloud, notifications, travail en équipe et chiffrement.",
    type: "article",
    url: getLocalizedUrl("fr", "pgdump-alternative"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Alternative à pg_dump - Databasus, outil de sauvegarde PostgreSQL",
    description:
      "Databasus s'appuie sur pg_dump et étend ses fonctionnalités : gestion des backups, interface web, planification automatique, stockage cloud, notifications, travail en équipe et chiffrement.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "pgdump-alternative"),
    languages: getLanguageAlternates("pgdump-alternative"),
  },
  robots: "index, follow",
};

export default function PgDumpAlternativePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline:
              "Alternative à pg_dump - Databasus, outil de sauvegarde PostgreSQL",
            description:
              "Guide complet de Databasus comme alternative à pg_dump : comment il s'appuie sur pg_dump et étend ses capacités avec l'automatisation, le stockage cloud, les notifications et les fonctions d'équipe.",
            author: {
              "@type": "Organization",
              name: "Databasus",
            },
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: {
                "@type": "ImageObject",
                url: "https://databasus.com/logo.svg",
              },
            },
          }),
        }}
      />

      <DocsNavbarComponent lang="fr" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="fr" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="pgdump-alternative">Alternative à pg_dump</h1>

              <p className="text-lg text-gray-400">
                Pour les sauvegardes logiques, Databasus s&apos;appuie sur{" "}
                <code>pg_dump</code>. Plutôt que de remplacer{" "}
                <code>pg_dump</code>, Databasus étend ses capacités avec la
                gestion des backups, une interface web, la planification
                automatique, l&apos;intégration du stockage cloud, les
                notifications, le travail en équipe et le chiffrement intégré.
                Au-delà des sauvegardes logiques, Databasus prend aussi en
                charge les sauvegardes physiques, les sauvegardes incrémentales
                avec archivage des WAL et la restauration à un instant donné
                (Point-in-Time Recovery).
              </p>

              <h2 id="quick-comparison">Comparaison rapide</h2>

              <p>
                Voici un aperçu de la façon dont Databasus étend les
                fonctionnalités de base de <code>pg_dump</code> :
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Fonctionnalité</th>
                    <th>pg_dump</th>
                    <th>Databasus</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Moteur de sauvegarde</td>
                    <td data-label="pg_dump">pg_dump</td>
                    <td data-label="Databasus">Construit sur pg_dump</td>
                  </tr>
                  <tr>
                    <td>Gestion des backups</td>
                    <td data-label="pg_dump">❌ Non</td>
                    <td data-label="Databasus">✅ Oui</td>
                  </tr>
                  <tr>
                    <td>Autres bases de données</td>
                    <td data-label="pg_dump">PostgreSQL uniquement</td>
                    <td data-label="Databasus">
                      PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                  </tr>
                  <tr>
                    <td>Interface</td>
                    <td data-label="pg_dump">Ligne de commande</td>
                    <td data-label="Databasus">Interface web + API</td>
                  </tr>
                  <tr>
                    <td>Planification</td>
                    <td data-label="pg_dump">Manuelle ou scripts cron</td>
                    <td data-label="Databasus">✅ Planificateur intégré</td>
                  </tr>
                  <tr>
                    <td>Destinations de stockage</td>
                    <td data-label="pg_dump">Système de fichiers local</td>
                    <td data-label="Databasus">
                      Local, S3, R2, Azure, NAS, FTP
                    </td>
                  </tr>
                  <tr>
                    <td>Compression</td>
                    <td data-label="pg_dump">gzip, LZ4, zstd (manuelle)</td>
                    <td data-label="Databasus">
                      zstd (automatique, optimisée)
                    </td>
                  </tr>
                  <tr>
                    <td>Chiffrement</td>
                    <td data-label="pg_dump">Outils externes requis</td>
                    <td data-label="Databasus">✅ AES-256-GCM intégré</td>
                  </tr>
                  <tr>
                    <td>Notifications</td>
                    <td data-label="pg_dump">❌ Aucune</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, e-mail, webhooks
                    </td>
                  </tr>
                  <tr>
                    <td>Fonctions d&apos;équipe</td>
                    <td data-label="pg_dump">❌ Aucune</td>
                    <td data-label="Databasus">
                      ✅ Espaces de travail, RBAC, journaux d&apos;audit
                    </td>
                  </tr>
                  <tr>
                    <td>Politiques de rétention</td>
                    <td data-label="pg_dump">Scripts de nettoyage manuels</td>
                    <td data-label="Databasus">✅ Rétention automatique</td>
                  </tr>
                  <tr>
                    <td>Surveillance de l&apos;état</td>
                    <td data-label="pg_dump">❌ Aucune</td>
                    <td data-label="Databasus">✅ Health checks intégrés</td>
                  </tr>
                  <tr>
                    <td>Sauvegardes physiques</td>
                    <td data-label="pg_dump">❌ Non</td>
                    <td data-label="Databasus">✅ Oui</td>
                  </tr>
                  <tr>
                    <td>Sauvegardes incrémentales</td>
                    <td data-label="pg_dump">❌ Non</td>
                    <td data-label="Databasus">✅ Par blocs (PG 17+)</td>
                  </tr>
                  <tr>
                    <td>Point-in-Time Recovery</td>
                    <td data-label="pg_dump">❌ Non</td>
                    <td data-label="Databasus">✅ Oui</td>
                  </tr>
                  <tr>
                    <td>Sauvegardes distantes</td>
                    <td data-label="pg_dump">✅ Oui (CLI)</td>
                    <td data-label="Databasus">✅ Oui</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="what-is-pgdump">Qu&apos;est-ce que pg_dump ?</h2>

              <p>
                <code>pg_dump</code> est l&apos;utilitaire natif de PostgreSQL
                pour créer des sauvegardes logiques. Il fait partie de
                PostgreSQL depuis ses débuts et reste l&apos;outil standard pour
                exporter des bases de données.
              </p>

              <h3 id="pgdump-strengths">Points forts de pg_dump</h3>

              <ul>
                <li>
                  <strong>Sauvegardes portables</strong> : produit des dumps SQL
                  ou au format custom restaurables sur différentes versions de
                  PostgreSQL.
                </li>
                <li>
                  <strong>Sauvegardes sélectives</strong> : peut exporter des
                  tables ou des schémas précis, ou des bases entières.
                </li>
                <li>
                  <strong>Instantanés cohérents</strong> : utilise le MVCC de
                  PostgreSQL pour créer des sauvegardes cohérentes sans bloquer
                  les écritures.
                </li>
                <li>
                  <strong>Largement pris en charge</strong> : présent dans
                  chaque installation PostgreSQL, bien documenté et éprouvé.
                </li>
                <li>
                  <strong>Formats de sortie flexibles</strong> : SQL brut,
                  custom, répertoire ou tar.
                </li>
              </ul>

              <h3 id="pgdump-limitations">Limites de pg_dump</h3>

              <p>
                <code>pg_dump</code> est puissant, mais son utilisation en
                production demande en général des scripts supplémentaires :
              </p>

              <ul>
                <li>
                  <strong>Pas de planification intégrée</strong> : nécessite des
                  tâches cron ou des planificateurs externes.
                </li>
                <li>
                  <strong>Stockage local uniquement</strong> : écrit sur le
                  système de fichiers local ; l&apos;envoi vers le cloud demande
                  des scripts en plus.
                </li>
                <li>
                  <strong>Pas de chiffrement</strong> : les fichiers de backup
                  ne sont pas chiffrés par défaut ; il faut passer par gpg ou un
                  outil similaire.
                </li>
                <li>
                  <strong>Pas de notifications</strong> : aucun moyen
                  d&apos;être alerté du succès ou de l&apos;échec d&apos;une
                  sauvegarde sans scripts maison.
                </li>
                <li>
                  <strong>Pas de gestion de la rétention</strong> : les anciens
                  backups doivent être supprimés à la main ou par script.
                </li>
                <li>
                  <strong>Ligne de commande uniquement</strong> : aucune
                  interface visuelle pour la surveillance ou la gestion.
                </li>
              </ul>

              <h2 id="how-databasus-extends">
                Comment Databasus étend pg_dump
              </h2>

              <p>
                Databasus utilise <code>pg_dump</code> comme moteur de
                sauvegarde : il conserve tous les avantages des sauvegardes
                logiques et ajoute des fonctionnalités d&apos;entreprise
                par-dessus.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Sous le capot :</strong>{" "}
                  quand vous lancez une sauvegarde dans Databasus, il exécute{" "}
                  <code className="bg-[#374151] text-gray-200">pg_dump</code>{" "}
                  avec des paramètres optimisés, puis gère la compression, le
                  chiffrement et l&apos;envoi vers la destination de stockage
                  configurée.
                </p>
              </div>

              <h3 id="web-interface">Interface web</h3>

              <p>
                Au lieu de mémoriser les options de <code>pg_dump</code> en
                ligne de commande, Databasus fournit une interface web où vous
                pouvez :
              </p>

              <ul>
                <li>
                  Ajouter des bases de données avec un assistant de connexion
                  guidé
                </li>
                <li>
                  Configurer les plannings de sauvegarde avec des contrôles
                  visuels
                </li>
                <li>
                  Suivre l&apos;historique et le statut des backups d&apos;un
                  coup d&apos;œil
                </li>
                <li>Télécharger ou restaurer un backup en un clic</li>
                <li>
                  Consulter les graphiques d&apos;état et de disponibilité des
                  bases
                </li>
              </ul>

              <h3 id="optimized-compression">Compression optimisée</h3>

              <p>
                Databasus utilise par défaut la compression zstd (niveau 5), ce
                qui apporte :
              </p>

              <ul>
                <li>
                  <strong>Des fichiers 4 à 8 fois plus petits</strong> que les
                  dumps non compressés
                </li>
                <li>
                  <strong>~20 % de surcoût en temps</strong> seulement, bien
                  plus rapide que gzip
                </li>
                <li>
                  <strong>Une prise en charge automatique</strong>, sans avoir à
                  passer par des outils de compression
                </li>
              </ul>

              <h2 id="beyond-pgdump">
                Au-delà de pg_dump : sauvegardes physiques et PITR
              </h2>

              <p>
                Databasus s&apos;appuie sur <code>pg_dump</code> pour les
                sauvegardes logiques, mais va aussi au-delà de ce que{" "}
                <code>pg_dump</code> peut offrir :
              </p>

              <ul>
                <li>
                  <strong>Sauvegardes physiques</strong> : copies au niveau
                  fichier de l&apos;ensemble du cluster via{" "}
                  <code>pg_basebackup</code>. Sauvegarde et restauration plus
                  rapides pour les grosses bases.
                </li>
                <li>
                  <strong>Sauvegardes incrémentales et WAL</strong> :
                  sauvegardes incrémentales par blocs via{" "}
                  <code>pg_basebackup --incremental</code> (pilotées par les
                  résumés WAL côté serveur) plus streaming continu des WAL via{" "}
                  <code>pg_receivewal</code>, ce qui permet la restauration à un
                  instant donné : revenez à n&apos;importe quelle seconde entre
                  deux sauvegardes.
                </li>
                <li>
                  <strong>Reprise après sinistre</strong> : conçu pour des
                  exigences de perte de données quasi nulle, avec des
                  sauvegardes physiques de base et un streaming WAL continu.
                </li>
              </ul>

              <p>
                Ces sauvegardes reposent sur le mécanisme natif de PostgreSQL 17
                : Databasus réutilise l&apos;outillage éprouvé de PostgreSQL au
                lieu de le réinventer. Elles nécessitent PostgreSQL 17 ou plus
                récent ; sur les versions antérieures, seules les sauvegardes
                logiques <code>pg_dump</code> sont disponibles. Tout
                s&apos;exécute à distance depuis l&apos;hôte Databasus via le
                protocole de réplication, donc rien n&apos;est installé sur le
                serveur de base de données. Les réseaux fermés sont atteints par
                un tunnel SSH vers un hôte interne ou un bastion, si bien que la
                base n&apos;a jamais besoin d&apos;être exposée publiquement.{" "}
                <a
                  href="/fr/faq#pitr"
                  className="text-blue-400 hover:text-blue-600"
                >
                  Lisez comment fonctionnent les sauvegardes physiques et PITR
                </a>
                .
              </p>

              <h2 id="backup-automation">Automatisation des sauvegardes</h2>

              <p>
                L&apos;un des défis les plus courants avec <code>pg_dump</code>{" "}
                est la mise en place de sauvegardes automatiques fiables.
              </p>

              <h3 id="automation-pgdump">
                Automatisation traditionnelle de pg_dump
              </h3>

              <p>
                Un script d&apos;automatisation typique de <code>pg_dump</code>{" "}
                peut ressembler à ceci :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{`#!/bin/bash
# Backup script for pg_dump
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
DB_NAME="mydb"

# Create backup
pg_dump -Fc -h localhost -U postgres $DB_NAME > $BACKUP_DIR/$DB_NAME_$DATE.dump

# Compress (if not using custom format)
# gzip $BACKUP_DIR/$DB_NAME_$DATE.sql

# Encrypt
gpg --encrypt --recipient backup@company.com $BACKUP_DIR/$DB_NAME_$DATE.dump

# Upload to S3
aws s3 cp $BACKUP_DIR/$DB_NAME_$DATE.dump.gpg s3://my-bucket/backups/

# Cleanup old backups (keep last 7 days)
find $BACKUP_DIR -name "*.dump*" -mtime +7 -delete

# Send notification on failure
if [ $? -ne 0 ]; then
  curl -X POST https://hooks.slack.com/... -d '{"text":"Backup failed!"}'
fi`}</code>
                </pre>
              </div>

              <p>
                Ce script doit être maintenu, testé et surveillé. Chaque base de
                données demande sa propre entrée cron.
              </p>

              <h3 id="automation-databasus">Automatisation avec Databasus</h3>

              <p>Avec Databasus, les mêmes fonctionnalités sont intégrées :</p>

              <ul>
                <li>
                  <strong>Planificateur visuel</strong> : programmez des backups
                  horaires, quotidiens, hebdomadaires, mensuels ou cron à des
                  heures précises.
                </li>
                <li>
                  <strong>Compression automatique</strong> : la compression zstd
                  est appliquée automatiquement.
                </li>
                <li>
                  <strong>Chiffrement intégré</strong> : chiffrement AES-256-GCM
                  avec une clé unique par backup.
                </li>
                <li>
                  <strong>Envoi vers le cloud</strong> : envoi direct vers S3,
                  Cloudflare R2, Azure et d&apos;autres
                  destinations.
                </li>
                <li>
                  <strong>Politiques de rétention</strong> : nettoyage
                  automatique des anciens backups selon vos réglages.
                </li>
                <li>
                  <strong>Notifications</strong> : alertes vers Slack, Teams,
                  Telegram ou par e-mail en cas de succès ou d&apos;échec.
                </li>
              </ul>

              <h2 id="storage-options">Options de stockage</h2>

              <p>
                <code>pg_dump</code> écrit sur le système de fichiers local.
                Envoyer les backups vers un stockage cloud demande des outils et
                des scripts supplémentaires.
              </p>

              <h3 id="storage-databasus">
                Destinations de stockage de Databasus
              </h3>

              <p>
                Databasus prend en charge plusieurs destinations de stockage
                sans configuration supplémentaire :
              </p>

              <ul>
                <li>Stockage local</li>
                <li>Amazon S3 et les services compatibles S3</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (stockage en réseau)</li>
                <li>FTP</li>
              </ul>

              <p>
                Chaque base de données peut avoir sa propre destination de
                stockage, et vous pouvez en configurer plusieurs pour la
                redondance.
              </p>

              <p>
                <a
                  href="/fr/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Voir toutes les options de stockage →
                </a>
              </p>

              <h2 id="notifications">Notifications</h2>

              <p>
                Savoir quand une sauvegarde réussit ou échoue est essentiel pour
                protéger vos données.
              </p>

              <h3 id="notifications-pgdump">Notifications avec pg_dump</h3>

              <p>
                <code>pg_dump</code> n&apos;a aucun système de notification.
                Vous devez :
              </p>

              <ul>
                <li>
                  Écrire des scripts d&apos;encapsulation qui vérifient les
                  codes de retour
                </li>
                <li>Vous intégrer à des outils de supervision externes</li>
                <li>
                  Mettre en place des chaînes d&apos;alerte personnalisées
                </li>
              </ul>

              <h3 id="notifications-databasus">Notifications avec Databasus</h3>

              <p>Databasus inclut des notifications intégrées vers :</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>E-mail</li>
                <li>Webhooks (pour des intégrations personnalisées)</li>
              </ul>

              <p>
                Configurez les événements qui déclenchent les notifications :
                succès des sauvegardes, échec, ou les deux.
              </p>

              <p>
                <a
                  href="/fr/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Voir tous les canaux de notification →
                </a>
              </p>

              <h2 id="team-features">Fonctions d&apos;équipe</h2>

              <p>
                <code>pg_dump</code> est un outil en ligne de commande
                mono-utilisateur. Databasus ajoute des fonctions de
                collaboration pour les équipes :
              </p>

              <h3 id="team-databasus">Capacités d&apos;équipe de Databasus</h3>

              <ul>
                <li>
                  <strong>Espaces de travail</strong> : organisez bases de
                  données, notifications et stockages par projet ou par équipe.
                  Les utilisateurs ne voient que les espaces où ils sont
                  invités.
                </li>
                <li>
                  <strong>Contrôle d&apos;accès par rôles</strong> : attribuez
                  des permissions lecteur, éditeur ou administrateur pour
                  contrôler ce que chaque membre de l&apos;équipe peut faire.
                </li>
                <li>
                  <strong>Journaux d&apos;audit</strong> : tracez toutes les
                  activités et modifications du système. Indispensable pour la
                  conformité et la traçabilité.
                </li>
                <li>
                  <strong>Notifications partagées</strong> : les canaux
                  d&apos;équipe reçoivent automatiquement le statut des
                  sauvegardes.
                </li>
              </ul>

              <p>
                <a
                  href="/fr/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  En savoir plus sur la gestion des accès →
                </a>
              </p>

              <h2 id="security">Sécurité</h2>

              <p>
                La sécurité est le domaine où Databasus apporte le plus par
                rapport à l&apos;utilisation brute de <code>pg_dump</code>.
              </p>

              <h3 id="security-pgdump">Sécurité avec pg_dump</h3>

              <p>
                <code>pg_dump</code> crée des fichiers de backup non chiffrés.
                Les sécuriser demande de :
              </p>

              <ul>
                <li>
                  Faire passer la sortie par des outils de chiffrement (gpg,
                  openssl)
                </li>
                <li>Gérer les clés de chiffrement séparément</li>
                <li>Garantir un stockage et une rotation sécurisés des clés</li>
                <li>Configurer correctement les permissions des fichiers</li>
              </ul>

              <h3 id="security-databasus">Sécurité avec Databasus</h3>

              <p>Databasus applique la sécurité à plusieurs niveaux :</p>

              <ul>
                <li>
                  <strong>Chiffrement AES-256-GCM</strong> : tous les mots de
                  passe, jetons et identifiants sont chiffrés. La clé de
                  chiffrement est stockée séparément de la base de données.
                </li>
                <li>
                  <strong>Chiffrement unique par backup</strong> : chaque
                  fichier de backup est chiffré avec une clé unique dérivée de
                  la clé maîtresse, de l&apos;identifiant du backup et d&apos;un
                  sel aléatoire.
                </li>
                <li>
                  <strong>Accès en lecture seule à la base</strong> : seules les
                  permissions SELECT sont exigées, ce qui empêche toute
                  corruption des données même en cas de compromission.
                </li>
              </ul>

              <p>
                <a
                  href="/fr/security"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  En savoir plus sur la sécurité de Databasus →
                </a>
              </p>

              <h2 id="restore-process">Processus de restauration</h2>

              <p>
                Les deux outils permettent de restaurer des backups, mais avec
                des flux de travail différents.
              </p>

              <h3 id="restore-pgdump">Restaurer des backups pg_dump</h3>

              <p>
                Restaurer un backup <code>pg_dump</code> demande de :
              </p>

              <ol>
                <li>Localiser le fichier de backup</li>
                <li>Le déchiffrer s&apos;il est chiffré</li>
                <li>Le décompresser s&apos;il est compressé</li>
                <li>
                  Lancer <code>pg_restore</code> ou <code>psql</code> avec les
                  bons paramètres
                </li>
              </ol>

              <h3 id="restore-databasus">Restaurer des backups Databasus</h3>

              <p>Databasus simplifie la restauration :</p>

              <ul>
                <li>
                  <strong>Téléchargement en un clic</strong> : téléchargez
                  n&apos;importe quel backup directement depuis l&apos;interface
                  web.
                </li>
                <li>
                  <strong>Déchiffrement automatique</strong> : les backups sont
                  déchiffrés automatiquement au téléchargement.
                </li>
                <li>
                  <strong>Commandes de restauration fournies</strong> :
                  Databasus affiche la commande <code>pg_restore</code> exacte
                  pour chaque backup.
                </li>
                <li>
                  <strong>Restauration parallèle</strong> : exploitez plusieurs
                  cœurs CPU pour restaurer plus vite les grosses bases.
                </li>
              </ul>

              <h2 id="installation">Installation</h2>

              <h3 id="install-pgdump">Installation de pg_dump</h3>

              <p>
                <code>pg_dump</code> est fourni avec PostgreSQL. Si PostgreSQL
                est installé, vous avez déjà <code>pg_dump</code>.
              </p>

              <h3 id="install-databasus">Installation de Databasus</h3>

              <p>Databasus propose plusieurs méthodes d&apos;installation :</p>

              <ul>
                <li>
                  <strong>Script en une ligne</strong> : installe Docker (si
                  nécessaire), met en place Databasus et configure le démarrage
                  automatique.
                </li>
                <li>
                  <strong>Docker run</strong> : une seule commande pour démarrer
                  avec PostgreSQL embarqué.
                </li>
                <li>
                  <strong>Docker Compose</strong> : pour plus de contrôle sur le
                  déploiement.
                </li>
              </ul>

              <p>
                <a
                  href="/fr/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Voir le guide d&apos;installation →
                </a>
              </p>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                <code>pg_dump</code> est l&apos;utilitaire de sauvegarde éprouvé
                de PostgreSQL, et Databasus est construit directement dessus. Le
                choix entre utiliser <code>pg_dump</code> directement ou via
                Databasus dépend de vos besoins.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>Utilisez pg_dump directement si :</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Vous avez besoin d&apos;exports ponctuels ou occasionnels
                  </li>
                  <li>
                    Écrire et maintenir des scripts shell ne vous pose pas de
                    problème
                  </li>
                  <li>
                    Vous avez déjà une infrastructure d&apos;automatisation
                    (Ansible, Terraform, etc.)
                  </li>
                  <li>
                    Il vous faut seulement des backups locaux, sans stockage
                    cloud
                  </li>
                  <li>
                    Vous êtes un développeur seul avec des besoins simples
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Utilisez Databasus si :
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Vous voulez des sauvegardes automatiques et planifiées sans
                    écrire de scripts
                  </li>
                  <li>
                    Vous devez stocker les backups dans le cloud (S3, Azure Blob Storage, etc.)
                  </li>
                  <li>
                    Vous voulez un chiffrement intégré sans gérer les clés à la
                    main
                  </li>
                  <li>
                    Vous avez besoin de notifications de succès ou d&apos;échec
                  </li>
                  <li>
                    Vous travaillez en équipe et avez besoin de fonctions de
                    collaboration
                  </li>
                  <li>
                    Vous préférez une interface visuelle à la ligne de commande
                  </li>
                  <li>
                    Vous voulez des politiques de rétention et un nettoyage
                    automatiques
                  </li>
                  <li>
                    Vous avez besoin de sauvegardes physiques, de sauvegardes
                    incrémentales ou de Point-in-Time Recovery pour la reprise
                    après sinistre
                  </li>
                </ul>
              </div>

              <p>
                Databasus s&apos;appuie sur <code>pg_dump</code> pour les
                sauvegardes logiques et l&apos;étend avec l&apos;automatisation,
                la sécurité et les fonctions d&apos;équipe. Au-delà, Databasus
                prend aussi en charge les sauvegardes physiques, les sauvegardes
                incrémentales avec archivage des WAL et le Point-in-Time
                Recovery, des capacités que <code>pg_dump</code> ne peut tout
                simplement pas offrir.
              </p>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
