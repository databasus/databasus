import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "Sauvegarde MySQL avec prise en charge de MariaDB",
  description:
    "Outil gratuit et open source pour les backups planifiés de MySQL et MariaDB. Alternative à mysqldump avec interface web, stockage cloud (S3, Azure Blob Storage), notifications (Slack, Discord, Telegram) et chiffrement AES-256.",
  keywords:
    "sauvegarde MySQL, sauvegarde MariaDB, alternative mysqldump, mysqldump GUI, automatisation sauvegarde MySQL, outil de sauvegarde MySQL, outil de sauvegarde MariaDB, sauvegarde MySQL planifiée, sauvegarde MySQL cloud, sauvegarde MySQL S3, sauvegarde MySQL Docker, chiffrement sauvegarde MySQL",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("fr", "mysql-backup"),
    languages: getLanguageAlternates("mysql-backup"),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("fr", "mysql-backup"),
    title: "Sauvegarde MySQL avec prise en charge de MariaDB",
    description:
      "Outil gratuit et open source pour les backups planifiés de MySQL et MariaDB. Alternative à mysqldump avec interface web, stockage cloud, notifications et chiffrement.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Interface du tableau de bord Databasus pour la gestion des sauvegardes MySQL",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary_large_image",
    title: "Sauvegarde MySQL avec prise en charge de MariaDB",
    description:
      "Outil gratuit et open source pour les backups planifiés de MySQL et MariaDB. Alternative à mysqldump avec interface web, stockage cloud, notifications et chiffrement.",
    images: ["https://databasus.com/images/index/dashboard.png"],
  },
  applicationName: "Databasus",
  appleWebApp: {
    title: "Databasus",
    capable: true,
  },
  icons: {
    icon: [
      { url: "/favicon.ico", type: "image/x-icon" },
      { url: "/favicon.svg", type: "image/svg+xml" },
    ],
    apple: "/favicon.svg",
    shortcut: "/favicon.ico",
  },
};

export default function MysqlBackupPage() {
  return (
    <div className="overflow-x-hidden">
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "SoftwareApplication",
            name: "Databasus",
            description:
              "Outil gratuit et open source pour les backups planifiés de MySQL et MariaDB. Alternative à mysqldump avec interface web, stockage cloud, notifications et chiffrement.",
            url: "https://databasus.com/fr/mysql-backup/",
            image: "https://databasus.com/images/index/dashboard.png",
            logo: "https://databasus.com/logo.svg",
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: {
                "@type": "ImageObject",
                url: "https://databasus.com/logo.svg",
              },
            },
            featureList: [
              "Sauvegardes planifiées de MySQL et MariaDB",
              "Plusieurs destinations de stockage (S3, Azure Blob Storage, FTP, SFTP, NAS, etc.)",
              "Notifications en temps réel (Slack, Telegram, Discord, Webhook, email, etc.)",
              "Supervision de l'état des bases de données MySQL",
              "Auto-hébergé via Docker",
              "Open source et gratuit",
              "Prise en charge de MySQL 5.7, 8.0, 8.4, 9, 26",
              "Prise en charge de MariaDB 5.5, 10, 11, 12, 13",
              "Compression des backups et chiffrement AES-256-GCM",
              "Alternative à mysqldump avec interface web",
            ],
            screenshot: "https://databasus.com/images/index/dashboard.png",
            softwareVersion: "latest",
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "Qu'est-ce que Databasus et comment sauvegarde-t-il les bases de données MySQL ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus est un outil de sauvegarde auto-hébergé sous licence Apache 2.0 qui utilise mysqldump en interne pour créer des backups MySQL cohérents. Il complète mysqldump avec une interface web moderne, une planification automatisée, l'intégration du stockage cloud, des notifications en temps réel et le chiffrement AES-256-GCM : plus besoin de scripts shell maison.",
                },
              },
              {
                "@type": "Question",
                name: "Databasus est-il une alternative ou un remplacement de mysqldump ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus étend mysqldump plutôt que de le remplacer. En interne, il exécute mysqldump avec des paramètres optimisés (--single-transaction, --routines, --triggers, --events), puis gère la compression, le chiffrement et l'envoi vers le stockage configuré. Vous obtenez toute la fiabilité de mysqldump plus la planification, le stockage cloud, les notifications et les fonctions d'équipe.",
                },
              },
              {
                "@type": "Question",
                name: "Quelles versions de MySQL Databasus prend-il en charge ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus prend en charge MySQL 5.7, 8.0, 8.4, 9 et 26. Pour MySQL 8.0 et plus récent, Databasus utilise la compression réseau zstd pour des transferts plus rapides. Pour MySQL 5.7, la compression legacy est utilisée automatiquement. MySQL 5.7 est pris en charge uniquement sur x86-64.",
                },
              },
              {
                "@type": "Question",
                name: "Quelles versions de MariaDB Databasus prend-il en charge ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus prend en charge MariaDB 5.5, 10, 11, 12 et 13. Pour les backups MariaDB, Databasus utilise l'outil natif mariadb-dump au lieu de mysqldump, ce qui garantit une compatibilité totale avec les fonctionnalités et optimisations propres à MariaDB.",
                },
              },
              {
                "@type": "Question",
                name: "Puis-je sauvegarder des bases MySQL hébergées sur AWS RDS, Google Cloud SQL ou Azure ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Oui, Databasus fonctionne parfaitement avec les bases MySQL hébergées dans le cloud, y compris AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL et PlanetScale. Comme Databasus utilise des backups logiques via mysqldump, il ne nécessite que des identifiants de connexion MySQL standard, sans permissions cloud particulières.",
                },
              },
              {
                "@type": "Question",
                name: "Comment Databasus sécurise-t-il les identifiants et les backups MySQL ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus applique une sécurité à plusieurs niveaux : (1) Tous les mots de passe et identifiants MySQL sont chiffrés avec AES-256-GCM avant stockage ; (2) Chaque fichier de backup est chiffré avec une clé unique dérivée de la clé maître, de l'ID du backup et d'un sel aléatoire ; (3) Les identifiants sont transmis via des fichiers .my.cnf temporaires avec permissions 0600, jamais exposés dans la ligne de commande ou les logs.",
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        lang="fr"
        sectionLinks={[
          { href: "#features", label: "Fonctionnalités" },
          { href: "#mysql-backup", label: "MySQL" },
          { href: "#mariadb-backup", label: "MariaDB" },
        ]}
      />

      {/* MAIN SECTION */}
      <main className="relative overflow-hidden pt-[60px] md:pt-[68px]">
        <div className="relative mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px] px-4 md:px-6 lg:px-0 pt-12 md:pt-[100px] pb-12 md:pb-[100px]">
          {/* Background ellipse */}
          <div className="relative">
            <div className="absolute left-1/2 -translate-x-1/2 -translate-y-1/4 w-[400px] h-[400px] md:w-[900px] md:h-[900px] bg-[#155dfc]/4 top-0 rounded-full blur-3xl -z-10" />
          </div>

          {/* Content */}
          <div className="text-center mb-8 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Databasus</span>
            </div>

            <h1 className="text-2xl sm:text-4xl sm:max-w-[400px] md:text-4xl leading-tight font-bold mb-4 md:mb-6 mx-auto md:max-w-[500px]">
              Outil de sauvegarde MySQL avec prise en charge de MariaDB
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus est un outil gratuit, open source et auto-hébergé pour
              sauvegarder les bases de données MySQL et MariaDB. Planifiez des
              dumps automatiques, stockez-les dans S3, Azure Blob Storage ou en local.
              Recevez des notifications via Slack, Discord ou Telegram quand les
              backups se terminent
            </p>

            <div className="flex flex-col sm:flex-row sm:flex-wrap items-center justify-center gap-2 sm:gap-2 max-w-[400px] mx-auto pb-0 sm:pb-[50px] lg:pb-0">
              <a
                href="#installation"
                className="w-full sm:flex-1 inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-3"
              >
                Auto-hébergement via Docker
              </a>

              <a
                href="https://github.com/databasus/databasus"
                target="_blank"
                rel="noopener noreferrer"
                className="w-full sm:flex-1 inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 rounded-lg font-medium border border-[#ffffff20] bg-[#0C0E13] hover:opacity-70 transition-opacity order-4 sm:order-4"
              >
                <svg
                  aria-hidden={true}
                  width="24"
                  height="24"
                  viewBox="0 0 20 20"
                  fill="none"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <g clipPath="url(#clip0_1_2459)">
                    <path
                      fillRule="evenodd"
                      clipRule="evenodd"
                      d="M9.9702 0C4.45694 0 0 4.4898 0 10.0443C0 14.4843 2.85571 18.2427 6.81735 19.5729C7.31265 19.6729 7.49408 19.3567 7.49408 19.0908C7.49408 18.858 7.47775 18.0598 7.47775 17.2282C4.70429 17.8269 4.12673 16.0308 4.12673 16.0308C3.68102 14.8667 3.02061 14.5676 3.02061 14.5676C2.11286 13.9522 3.08673 13.9522 3.08673 13.9522C4.09367 14.0188 4.62204 14.9833 4.62204 14.9833C5.51327 16.5131 6.94939 16.0808 7.52714 15.8147C7.60959 15.1661 7.87388 14.7171 8.15449 14.4678C5.94245 14.2349 3.6151 13.3702 3.6151 9.51204C3.6151 8.41449 4.01102 7.51653 4.63837 6.81816C4.53939 6.56878 4.19265 5.53755 4.73755 4.15735C4.73755 4.15735 5.57939 3.89122 7.47755 5.18837C8.29022 4.9685 9.12832 4.85666 9.9702 4.85571C10.812 4.85571 11.6702 4.97225 12.4627 5.18837C14.361 3.89122 15.2029 4.15735 15.2029 4.15735C15.7478 5.53755 15.4008 6.56878 15.3018 6.81816C15.9457 7.51653 16.3253 8.41449 16.3253 9.51204C16.3253 13.3702 13.998 14.2182 11.7694 14.4678C12.1327 14.7837 12.4461 15.3822 12.4461 16.3302C12.4461 17.6771 12.4298 18.7582 12.4298 19.0906C12.4298 19.3567 12.6114 19.6729 13.1065 19.5731C17.0682 18.2424 19.9239 14.4843 19.9239 10.0443C19.9402 4.4898 15.4669 0 9.9702 0Z"
                      fill="white"
                    />
                  </g>
                  <defs>
                    <clipPath id="clip0_1_2459">
                      <rect width="20" height="20" fill="white" />
                    </clipPath>
                  </defs>
                </svg>

                <span>GitHub</span>
              </a>
            </div>
          </div>

          {/* Dashboard Screenshot */}
          <div className="relative mx-auto max-w-[1200px]">
            <div>
              <img
                src="/images/index/dashboard.svg"
                alt="Interface du tableau de bord Databasus pour la gestion des sauvegardes MySQL"
                width={980}
                height={620}
                className="w-full h-auto"
                loading="eager"
                fetchPriority="high"
              />
            </div>
          </div>
        </div>
      </main>

      {/* FEATURES OVERVIEW SECTION */}
      <section id="features" className="pb-12 md:pb-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Aperçu</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Fonctionnalités pour la sauvegarde MySQL
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus complète mysqldump avec des fonctions d&apos;entreprise
              : planification automatisée, intégration du stockage cloud,
              notifications en temps réel et chiffrement AES-256-GCM. Idéal pour
              les développeurs, les équipes DevOps et les organisations qui
              gèrent des bases MySQL
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          {/* Feature Cards Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 border border-[#ffffff20] rounded-xl">
            {/* Card 1: Scheduled backups */}
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                1
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Dumps MySQL planifiés
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Configuration des sauvegardes MySQL planifiées"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Planifiez mysqldump aux heures creuses, quand la charge de la
                base est faible. Choisissez un intervalle horaire, quotidien,
                hebdomadaire ou mensuel, ou utilisez des expressions cron pour
                un contrôle précis
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Supervision de l&apos;état MySQL
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Contrôles d'état MySQL"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Surveillez la disponibilité de la connexion MySQL avec des
                contrôles d&apos;état configurables. Soyez averti quand votre
                base devient injoignable ou se rétablit
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                Définissez les intervalles de vérification (chaque minute,
                toutes les 5 minutes, etc.) et le nombre d&apos;échecs avant de
                marquer la base comme indisponible
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Stockez vos dumps MySQL où vous voulez
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Conservez vos fichiers de backup MySQL en local, dans un
                stockage compatible S3, Azure Blob Storage, FTP, un NAS ou
                d&apos;autres destinations. Vos données restent sous votre
                contrôle.{" "}
                <a
                  href="/fr/storages"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Voir tout →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-destinations.svg"
                  alt="Destinations de stockage des sauvegardes MySQL"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 4: Notifications */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                4
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Notifications de backup
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Recevez des alertes quand les backups MySQL se terminent ou
                échouent. Envoyez les notifications dans le chat de votre équipe
                DevOps ou vos canaux personnels.{" "}
                <a
                  href="/fr/notifiers"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Voir tout →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-notifications.svg"
                  alt="Notifications de sauvegarde MySQL"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 5: Self hosted via Docker */}
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                5
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Auto-hébergé via Docker
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Exécutez Databasus sur votre propre infrastructure. Tous les
                identifiants MySQL et les données de backup restent sur des
                serveurs que vous contrôlez. Déployez en 2 minutes environ via
                script, Docker ou Kubernetes
              </p>

              <div className="flex">
                <img
                  src="/images/index/feature-deploy.svg"
                  alt="Déploiement Docker"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 6: Open source and free */}
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                6
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Open source et gratuit
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Databasus est entièrement open source sous licence Apache 2.0.
                Inspectez chaque ligne de code et forkez-le. Gratuit
                pour un usage personnel comme en entreprise
              </p>
              <div>
                <img
                  src="/images/index/feature-github.svg"
                  alt="GitHub open source"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 7: Many MySQL versions - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Many MySQL versions */}
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Versions de MySQL et MariaDB
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  MySQL 5.7, 8.0, 8.4, 9 et 26 sont pris en charge. MariaDB 5.5, 10, 11,
                  12 et 13 sont pris en charge avec l&apos;outil natif mariadb-dump
                  pour une compatibilité totale
                </p>

                <div>
                  <img
                    src="/images/index/database-mysql.svg"
                    alt="Versions MySQL"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              {/* Card 10: Security - Only visible on desktop, merged with card 7 */}
              <div className="hidden lg:block p-5 md:p-6">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  10
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Sécurité
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Les identifiants MySQL sont chiffrés avec AES-256-GCM avant
                  stockage. Chaque fichier de backup est chiffré avec une clé
                  unique. Les mots de passe sont transmis via des fichiers de
                  configuration temporaires, jamais exposés dans la ligne de
                  commande.{" "}
                  <a
                    href="/fr/security"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    En savoir plus →
                  </a>
                </p>

                <div>
                  <img
                    src="/images/index/feature-encryption.svg"
                    alt="Sécurité des sauvegardes MySQL"
                    loading="lazy"
                  />
                </div>
              </div>
            </div>

            {/* Card 8: Access management */}
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20]">
                  8
                </div>
              </div>

              <div className="flex flex-wrap items-center mb-4 md:mb-5">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold">
                  Gestion des accès
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  pour les équipes
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-access-management.svg"
                  alt="Gestion des accès aux sauvegardes MySQL"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Contrôlez qui peut consulter ou gérer les bases MySQL. Créez des
                espaces de travail pour différents projets. Attribuez les rôles
                lecteur, éditeur ou administrateur.{" "}
                <a
                  href="/fr/access-management#settings"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  En savoir plus →
                </a>
              </p>
            </div>

            {/* Card 9: Audit logs */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20]">
                  9
                </div>
              </div>

              <div className="flex flex-wrap items-center mb-4 md:mb-5">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold">
                  Journaux d&apos;audit
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  pour les équipes
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-audit-logs.svg"
                  alt="Journaux d'audit des sauvegardes MySQL"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Suivez toutes les activités : téléchargements de backups,
                changements de planification, mises à jour de configuration.
                Sachez qui a fait quoi et quand, pour la conformité et la
                traçabilité.{" "}
                <a
                  href="/fr/access-management#audit-logs"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  En savoir plus →
                </a>
              </p>
            </div>

            {/* Card 10: Security - Mobile/Tablet only */}
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1 lg:hidden">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                10
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Sécurité
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Les identifiants MySQL sont chiffrés avec AES-256-GCM avant
                stockage. Chaque fichier de backup est chiffré avec une clé
                unique. Les mots de passe sont transmis via des fichiers de
                configuration temporaires, jamais exposés dans la ligne de
                commande.{" "}
                <a
                  href="/fr/security"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  En savoir plus →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-encryption.svg"
                  alt="Sécurité des sauvegardes MySQL"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 11: Suitable for clouds */}
            <div className="col-span-1 md:col-span-2 lg:col-span-2 p-5 md:p-6 flex flex-col md:flex-row gap-4 md:gap-6">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20] shrink-0">
                11
              </div>

              <div>
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Compatible avec les bases MySQL cloud
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus se connecte aux bases MySQL hébergées dans le cloud,
                  y compris AWS RDS for MySQL, Google Cloud SQL, Azure Database
                  for MySQL et PlanetScale. Comme il utilise des backups
                  logiques via mysqldump, seuls des identifiants de connexion
                  standard sont nécessaires : aucune permission cloud
                  particulière ni accès au système de fichiers
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* MYSQL BACKUP SECTION */}
      <section
        id="mysql-backup"
        className="py-12 md:py-20 px-4 md:px-6 lg:px-0"
      >
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            {/* Left side: Info */}
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Basé sur mysqldump</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Sauvegarde MySQL
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Databasus utilise <strong>mysqldump</strong> en interne,
                  l&apos;utilitaire officiel de sauvegarde MySQL. Quand vous
                  lancez un backup, Databasus exécute mysqldump avec des
                  paramètres optimisés :
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --single-transaction
                    </code>{" "}
                    pour des instantanés cohérents sans verrouiller les tables
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --routines
                    </code>{" "}
                    pour inclure les procédures stockées et les fonctions
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --triggers
                    </code>{" "}
                    pour inclure les triggers de la base
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --events
                    </code>{" "}
                    pour inclure les événements planifiés
                  </li>
                </ul>

                <p className="text-gray-400">
                  Pour MySQL 8.0 et plus récent, Databasus active la{" "}
                  <strong>compression réseau zstd</strong> pour des transferts
                  plus rapides sur les connexions lentes. MySQL 5.7 utilise
                  automatiquement la compression legacy.
                </p>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Versions MySQL prises en charge :
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 5.7
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 8.0
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 8.4
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 9
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 26
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Right side: Image */}
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mysql.svg"
                  alt="Base de données MySQL"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Sauvegarde MySQL officielle via mysqldump avec compression,
                  chiffrement et stockage cloud
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <div className="border-b border-[#ffffff20] max-w-[calc(100%-2rem)] md:max-w-[calc(100%-3rem)] lg:max-w-[1000px] 2xl:max-w-[1200px] mx-auto" />

      {/* MARIADB BACKUP SECTION */}
      <section
        id="mariadb-backup"
        className="py-12 md:py-20 px-4 md:px-6 lg:px-0"
      >
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row-reverse gap-8 lg:gap-16">
            {/* Right side: Info */}
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">
                  Outil natif mariadb-dump
                </span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Sauvegarde MariaDB
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Pour les bases MariaDB, Databasus utilise l&apos;outil natif{" "}
                  <strong>mariadb-dump</strong> au lieu de mysqldump. Cela
                  garantit une compatibilité totale avec les fonctionnalités et
                  optimisations propres à MariaDB.
                </p>

                <p className="text-gray-400">
                  mariadb-dump est exécuté avec les mêmes paramètres optimisés :
                  mode single-transaction pour des backups cohérents, routines,
                  triggers et événements inclus par défaut.
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>Compatibilité totale avec les fonctionnalités MariaDB</li>
                  <li>Prise en charge des connexions SSL/TLS</li>
                  <li>Compression réseau pour des transferts plus rapides</li>
                  <li>
                    Mêmes options de chiffrement et de stockage que pour MySQL
                  </li>
                </ul>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Versions MariaDB prises en charge :
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 5.5
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 10
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 11
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 12
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 13
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Left side: Image */}
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mariadb.svg"
                  alt="Base de données MariaDB"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Sauvegarde MariaDB native via mariadb-dump avec compatibilité
                  totale des fonctionnalités
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* INSTALLATION SECTION */}
      <section id="installation" className="px-4 md:px-6 lg:px-0">
        <div className="max-w-[1000px] 2xl:max-w-[1200px] mx-auto border border-[#ffffff20] rounded-xl py-10 md:py-20 px-4 md:px-6">
          <div className="max-w-[1100px] mx-auto">
            <div className="text-center mb-8 md:mb-10">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Premiers pas</span>
              </div>

              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
                Comment installer ?
              </h2>

              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                Databasus propose plusieurs méthodes d&apos;installation.
                Déployez sur votre VPS, votre machine locale ou un cluster
                Kubernetes en 2 minutes environ. La même installation couvre les
                sauvegardes MySQL, MariaDB, PostgreSQL et MongoDB
              </p>
            </div>

            <InstallationComponent lang="fr" />
          </div>
        </div>
      </section>

      {/* FAQ SECTION */}
      <section id="faq" className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-8 md:mb-12">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">FAQ</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Questions sur la sauvegarde MySQL
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Questions fréquentes sur la sauvegarde des bases MySQL et MariaDB
              avec Databasus. Pour toute autre question, rejoignez notre
              communauté sur Telegram
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="Qu'est-ce que Databasus et comment sauvegarde-t-il les bases de données MySQL ?"
              answer="Databasus est un outil de sauvegarde auto-hébergé sous licence Apache 2.0 qui utilise mysqldump en interne pour créer des backups MySQL cohérents. Il complète mysqldump avec une interface web moderne, une planification automatisée, l'intégration du stockage cloud (S3, Azure Blob Storage, FTP), des notifications en temps réel (Slack, Discord, Telegram) et le chiffrement AES-256-GCM : plus besoin de scripts shell maison ni de tâches cron."
            />
            <FaqItem
              number="2"
              question="Databasus est-il une alternative ou un remplacement de mysqldump ?"
              answer="Databasus étend mysqldump plutôt que de le remplacer. En interne, il exécute mysqldump avec des paramètres optimisés (--single-transaction, --routines, --triggers, --events), puis gère la compression, le chiffrement et l'envoi vers le stockage configuré. Vous obtenez toute la fiabilité de mysqldump plus la planification, le stockage cloud, les notifications et les fonctions de collaboration en équipe via une interface web."
            />
            <FaqItem
              number="3"
              question="Quelles versions de MySQL Databasus prend-il en charge ?"
              answer="Databasus prend en charge MySQL 5.7, 8.0, 8.4, 9 et 26. Pour MySQL 8.0 et plus récent, Databasus utilise la compression réseau zstd pour des transferts plus rapides sur les connexions lentes. Pour MySQL 5.7, la compression legacy est utilisée automatiquement. MySQL 5.7 est pris en charge uniquement sur x86-64. Toutes les versions bénéficient des mêmes fonctionnalités : planification, chiffrement, stockage cloud et notifications."
            />
            <FaqItem
              number="4"
              question="Quelles versions de MariaDB Databasus prend-il en charge ?"
              answer="Databasus prend en charge MariaDB 5.5, 10, 11, 12 et 13. Pour les backups MariaDB, Databasus utilise l'outil natif mariadb-dump au lieu de mysqldump, ce qui garantit une compatibilité totale avec les fonctionnalités et optimisations propres à MariaDB. La même interface web, la planification, le stockage et les notifications fonctionnent pour MySQL comme pour MariaDB."
            />
            <FaqItem
              number="5"
              question="Puis-je sauvegarder des bases MySQL hébergées sur AWS RDS, Google Cloud SQL ou Azure ?"
              answer="Oui, Databasus fonctionne parfaitement avec les bases MySQL hébergées dans le cloud, y compris AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL et PlanetScale. Comme Databasus utilise des backups logiques via mysqldump, il ne nécessite que des identifiants de connexion MySQL standard (hôte, port, nom d'utilisateur, mot de passe) : aucune permission cloud particulière, aucun rôle IAM ni accès au système de fichiers."
            />
            <FaqItem
              number="6"
              question="Comment fonctionne la planification des sauvegardes MySQL dans Databasus ?"
              answer="Vous pouvez planifier les backups MySQL à intervalle horaire, quotidien, hebdomadaire ou mensuel, ou utiliser des expressions cron pour un contrôle précis. Choisissez l'heure exacte (par exemple 3 h du matin, quand le trafic est faible) et les jours d'exécution. Databasus s'occupe du reste : connexion à MySQL, exécution de mysqldump, compression du résultat, chiffrement et envoi vers votre destination de stockage."
            />
            <FaqItem
              number="7"
              question="Où mes sauvegardes MySQL sont-elles stockées ?"
              answer="Vous choisissez où stocker vos fichiers de backup MySQL : en local sur votre serveur, dans un stockage compatible S3 (AWS S3, MinIO, Backblaze B2), Azure Blob Storage, Cloudflare R2, FTP, un NAS via SFTP. Chaque base de données peut avoir sa propre destination de stockage. Vos fichiers de backup restent sous votre contrôle."
            />
            <FaqItem
              number="8"
              question="Comment Databasus sécurise-t-il les identifiants et les backups MySQL ?"
              answer={
                <>
                  Databasus applique une sécurité à plusieurs niveaux :
                  <br />
                  <br />
                  <strong>1. Chiffrement des identifiants :</strong> tous les
                  mots de passe, jetons et chaînes de connexion MySQL sont
                  chiffrés avec AES-256-GCM avant stockage. La clé de
                  chiffrement est stockée séparément de la base de données.
                  <br />
                  <br />
                  <strong>2. Chiffrement des backups :</strong> chaque fichier
                  de backup est chiffré avec une clé unique dérivée de la clé
                  maître, de l&apos;ID du backup et d&apos;un sel aléatoire.
                  <br />
                  <br />
                  <strong>3. Gestion sécurisée des mots de passe :</strong> les
                  mots de passe MySQL sont transmis via des fichiers .my.cnf
                  temporaires avec permissions 0600, jamais exposés dans les
                  arguments de ligne de commande ni dans les logs.
                </>
              }
            />
            <FaqItem
              number="9"
              question="Puis-je restaurer des backups MySQL sur un autre serveur ou un autre fournisseur cloud ?"
              answer="Oui, comme Databasus crée des backups logiques mysqldump standard, vous pouvez les restaurer sur n'importe quel serveur MySQL : autre version, autre fournisseur cloud ou machine de développement locale. Téléchargez le backup depuis Databasus (déchiffré automatiquement), puis utilisez la commande mysql pour le restaurer. Databasus affiche la commande de restauration exacte pour chaque backup."
            />
            <FaqItem
              number="10"
              question="Databasus prend-il en charge la sauvegarde des réplicas MySQL ?"
              answer="Oui, vous pouvez sauvegarder des serveurs réplicas (slaves) MySQL. Databasus utilise --single-transaction, qui crée un instantané cohérent sans arrêter la réplication. C'est utile pour décharger votre serveur principal du travail de sauvegarde. Il suffit de pointer Databasus vers les paramètres de connexion de votre réplica."
            />
            <FaqItem
              number="11"
              question="En quoi Databasus diffère-t-il de MySQL Enterprise Backup ou Percona XtraBackup ?"
              answer={
                <>
                  MySQL Enterprise Backup et Percona XtraBackup créent des
                  backups physiques (binaires) qui nécessitent un accès direct
                  au système de fichiers du répertoire de données MySQL. Ils
                  sont plus rapides pour les très grandes bases, mais :
                  <br />
                  <br />
                  • Ils ne peuvent pas sauvegarder les bases MySQL cloud (RDS,
                  Cloud SQL, Azure)
                  <br />
                  • Ils doivent être installés sur le serveur de base de données
                  lui-même
                  <br />
                  • Leur configuration et leur restauration sont plus complexes
                  <br />
                  <br />
                  Databasus utilise des backups logiques (mysqldump) qui
                  fonctionnent avec tout serveur MySQL auquel vous pouvez vous
                  connecter, y compris tous les fournisseurs cloud. Pour la
                  plupart des bases de moins de 100 Go, les backups logiques
                  sont pratiques et bien plus simples à gérer.
                </>
              }
            />
            <FaqItem
              number="12"
              question="Puis-je sauvegarder à la fois des bases MySQL et PostgreSQL avec la même instance Databasus ?"
              answer="Oui, Databasus prend en charge MySQL, MariaDB, PostgreSQL et MongoDB dans une seule installation. Vous pouvez gérer les backups de toutes vos bases depuis une seule interface web, avec des planifications, destinations de stockage et canaux de notification différents pour chacune. C'est utile pour les équipes qui gèrent une infrastructure de bases de données variée."
            />
          </div>
        </div>
      </section>

      <LandingFooterComponent lang="fr" />
    </div>
  );
}

function FaqItem({
  number,
  question,
  answer,
}: {
  number: string;
  question: string;
  answer: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-[#ffffff20] p-4 md:p-6">
      <div className="flex items-center justify-center w-6 h-6 rounded border border-[#ffffff20] text-sm font-semibold mb-3 md:mb-4">
        {number}
      </div>

      <h3 className="text-base md:text-lg font-bold mb-2 md:mb-3">
        {question}
      </h3>

      <div className="text-gray-400 text-sm md:text-base">{answer}</div>
    </div>
  );
}
