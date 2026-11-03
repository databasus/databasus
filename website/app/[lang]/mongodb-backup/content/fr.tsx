import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "Sauvegarde MongoDB",
  description:
    "Outil gratuit et open source pour les backups planifiés de MongoDB. Automatisez mongodump avec une interface web, stockez les archives dans S3, Google Drive ou en local. Notifications via Slack, Discord, Telegram. Chiffrement AES-256 des données BSON.",
  keywords:
    "sauvegarde MongoDB, alternative mongodump, automatisation sauvegarde MongoDB, outil de sauvegarde MongoDB, sauvegarde MongoDB planifiée, sauvegarde MongoDB cloud, sauvegarde MongoDB S3, sauvegarde MongoDB Docker, chiffrement sauvegarde MongoDB, sauvegarde MongoDB Atlas, sauvegarde replica set, sauvegarde base documentaire, sauvegarde BSON, sauvegarde NoSQL",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("fr", "mongodb-backup"),
    languages: getLanguageAlternates("mongodb-backup"),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("fr", "mongodb-backup"),
    title: "Sauvegarde MongoDB",
    description:
      "Outil gratuit et open source pour les backups planifiés de MongoDB. Automatisez mongodump avec interface web, stockage cloud, notifications et chiffrement.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Interface du tableau de bord Databasus pour la gestion des sauvegardes MongoDB",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary_large_image",
    title: "Sauvegarde MongoDB",
    description:
      "Outil gratuit et open source pour les backups planifiés de MongoDB. Automatisez mongodump avec interface web, stockage cloud, notifications et chiffrement.",
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

export default function MongodbBackupPage() {
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
              "Outil gratuit et open source pour les backups planifiés de MongoDB. Automatisez mongodump avec interface web, stockage cloud, notifications et chiffrement.",
            url: "https://databasus.com/fr/mongodb-backup/",
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
              "Sauvegardes MongoDB planifiées via mongodump",
              "Plusieurs destinations de stockage (S3, Google Drive, Dropbox, SFTP, rclone, etc.)",
              "Notifications en temps réel (Slack, Telegram, Discord, Webhook, email, etc.)",
              "Supervision de l'état des connexions MongoDB",
              "Auto-hébergé via Docker",
              "Open source et gratuit",
              "Prise en charge de MongoDB 4, 5, 6, 7, 8 et 9",
              "Compression gzip des archives BSON",
              "Chiffrement AES-256-GCM des fichiers de backup",
              "Prise en charge de MongoDB Atlas et des replica sets",
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
                name: "Qu'est-ce que Databasus et comment sauvegarde-t-il les bases de données MongoDB ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus est un outil de sauvegarde auto-hébergé sous licence Apache 2.0 qui utilise mongodump en interne pour créer des backups MongoDB cohérents. Il complète mongodump avec une interface web moderne, une planification automatisée, l'intégration du stockage cloud, des notifications en temps réel et le chiffrement AES-256-GCM : plus besoin de scripts shell maison.",
                },
              },
              {
                "@type": "Question",
                name: "Databasus prend-il en charge les replica sets MongoDB ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Oui, Databasus prend entièrement en charge les replica sets MongoDB. Vous pouvez vous connecter à n'importe quel membre d'un replica set en utilisant le format d'URI de connexion MongoDB standard. Databasus lit depuis le nœud indiqué, ce qui vous permet de sauvegarder depuis des nœuds secondaires pour réduire la charge sur votre primaire.",
                },
              },
              {
                "@type": "Question",
                name: "Puis-je sauvegarder des bases MongoDB Atlas avec Databasus ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Oui, Databasus fonctionne parfaitement avec MongoDB Atlas. Comme Databasus utilise des backups logiques via mongodump, il ne nécessite que des identifiants de connexion MongoDB standard, sans permissions ni rôles Atlas particuliers. Fournissez simplement votre chaîne de connexion Atlas et Databasus s'occupe du reste.",
                },
              },
              {
                "@type": "Question",
                name: "Quelles versions de MongoDB Databasus prend-il en charge ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus prend en charge les versions 4, 5, 6, 7, 8 et 9 de MongoDB. Tous les backups utilisent l'outil natif mongodump avec les options --archive et --gzip pour produire des archives BSON compressées et efficaces, restaurables avec mongorestore.",
                },
              },
              {
                "@type": "Question",
                name: "Comment Databasus sécurise-t-il les identifiants et les backups MongoDB ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus applique une sécurité à plusieurs niveaux : (1) Tous les mots de passe et chaînes de connexion MongoDB sont chiffrés avec AES-256-GCM avant stockage ; (2) Chaque fichier de backup est chiffré avec une clé unique dérivée de la clé maître, de l'ID du backup et d'un sel aléatoire ; (3) Les URI de connexion sont transmises de manière sécurisée à mongodump, jamais exposées dans les logs ni la sortie de la ligne de commande.",
                },
              },
              {
                "@type": "Question",
                name: "Databasus prend-il en charge les sauvegardes MongoDB incrémentales ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus se concentre sur les backups logiques complets via mongodump plutôt que sur les backups incrémentaux. Pour la plupart des cas d'usage, des backups complets planifiés (horaires, quotidiens, hebdomadaires) offrent des points de restauration suffisants. MongoDB Atlas propose déjà une restauration native à un instant donné, et les backups incrémentaux externes ne peuvent pas être facilement restaurés sur des clusters Atlas.",
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
              Outil de sauvegarde MongoDB
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus est un outil gratuit, open source et auto-hébergé pour
              sauvegarder les bases de données documentaires MongoDB.
              Automatisez mongodump avec des planifications, stockez les
              archives BSON dans S3, Google Drive ou en local. Soyez averti via
              Slack, Discord ou Telegram quand les backups se terminent
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
                alt="Interface du tableau de bord Databasus pour la gestion des sauvegardes MongoDB"
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
              Fonctionnalités pour la sauvegarde MongoDB
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus complète mongodump avec des fonctions d&apos;entreprise
              : planification automatisée, intégration du stockage cloud,
              notifications en temps réel et chiffrement AES-256-GCM. Idéal pour
              les développeurs et les équipes DevOps qui gèrent des bases
              documentaires et des collections MongoDB
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
                Dumps MongoDB planifiés
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Configuration des sauvegardes MongoDB planifiées"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Planifiez mongodump aux heures creuses, quand la charge de votre
                application est faible. Choisissez un intervalle horaire,
                quotidien, hebdomadaire ou mensuel, ou utilisez des expressions
                cron pour un contrôle précis des horaires
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Supervision de l&apos;état MongoDB
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Contrôles d'état MongoDB"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Surveillez la disponibilité de la connexion MongoDB avec des
                contrôles d&apos;état configurables. Soyez averti quand votre
                base ou votre replica set devient injoignable
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
                Stockez vos archives BSON où vous voulez
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Conservez vos archives de backup MongoDB en local, dans un
                stockage compatible S3, Google Drive, Dropbox, un NAS ou
                d&apos;autres destinations. Vos données documentaires restent
                sous votre contrôle.{" "}
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
                  alt="Destinations de stockage des sauvegardes MongoDB"
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
                Recevez des alertes quand les backups MongoDB se terminent ou
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
                  alt="Notifications de sauvegarde MongoDB"
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
                Exécutez Databasus sur votre propre infrastructure. Toutes les
                chaînes de connexion MongoDB et les données de backup restent
                sur des serveurs que vous contrôlez. Déployez en 2 minutes
                environ via script, Docker ou Kubernetes
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

            {/* Card 7: Many MongoDB versions - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Many MongoDB versions */}
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Versions MongoDB prises en charge
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  MongoDB 4, 5, 6, 7, 8 et 9 sont pris en charge. Databasus utilise
                  l&apos;outil natif mongodump pour chaque version afin de
                  garantir une compatibilité totale avec votre base documentaire
                </p>

                <div>
                  <img
                    src="/images/index/database-mongodb.svg"
                    alt="Versions MongoDB"
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
                  Les chaînes de connexion MongoDB sont chiffrées avec
                  AES-256-GCM avant stockage. Chaque archive BSON est chiffrée
                  avec une clé unique. Les identifiants sont transmis de manière
                  sécurisée à mongodump, jamais exposés dans les logs.{" "}
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
                    alt="Sécurité des sauvegardes MongoDB"
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
                  alt="Gestion des accès aux sauvegardes MongoDB"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Contrôlez qui peut consulter ou gérer les bases MongoDB. Créez
                des espaces de travail pour différents projets. Attribuez les
                rôles lecteur, éditeur ou administrateur.{" "}
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
                  alt="Journaux d'audit des sauvegardes MongoDB"
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
                Les chaînes de connexion MongoDB sont chiffrées avec AES-256-GCM
                avant stockage. Chaque archive BSON est chiffrée avec une clé
                unique. Les identifiants sont transmis de manière sécurisée à
                mongodump, jamais exposés dans les logs.{" "}
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
                  alt="Sécurité des sauvegardes MongoDB"
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
                  Compatible avec MongoDB Atlas et l&apos;auto-hébergement
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus se connecte aux bases MongoDB hébergées dans le
                  cloud, y compris MongoDB Atlas, AWS DocumentDB et les
                  déploiements auto-hébergés. Comme il utilise des backups
                  logiques via mongodump, seuls des identifiants de connexion
                  standard sont nécessaires : aucune permission cloud
                  particulière ni accès au système de fichiers
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* MONGODUMP SECTION */}
      <section id="mongodump" className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            {/* Left side: Info */}
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Basé sur mongodump</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Comment fonctionne la sauvegarde MongoDB
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Databasus utilise <strong>mongodump</strong> en interne,
                  l&apos;utilitaire officiel de sauvegarde MongoDB. Quand vous
                  lancez un backup, Databasus exécute mongodump avec des
                  paramètres optimisés :
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --archive
                    </code>{" "}
                    pour une sortie BSON en un seul fichier au lieu d&apos;une
                    arborescence de répertoires
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --gzip
                    </code>{" "}
                    pour des archives compressées qui réduisent le volume de
                    stockage et de transfert
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --db
                    </code>{" "}
                    pour sauvegarder des bases précises de votre instance
                    MongoDB
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --uri
                    </code>{" "}
                    pour une gestion sécurisée de la chaîne de connexion avec
                    authentification
                  </li>
                </ul>

                <p className="text-gray-400">
                  Le flux de sauvegarde est envoyé directement vers votre
                  stockage configuré, avec un chiffrement AES-256-GCM optionnel
                  avant écriture. Cette approche minimise les entrées-sorties
                  disque et fonctionne efficacement avec les grandes
                  collections.
                </p>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Versions MongoDB prises en charge :
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 4
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 5
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 6
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 7
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 8
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 9
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Right side: Image */}
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mongodb.svg"
                  alt="Base de données MongoDB"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Sauvegarde MongoDB officielle via mongodump avec compression
                  gzip, chiffrement et stockage cloud
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
                sauvegardes MongoDB, PostgreSQL, MySQL et MariaDB
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
              Questions sur la sauvegarde MongoDB
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Questions fréquentes sur la sauvegarde des bases documentaires
              MongoDB avec Databasus. Pour toute autre question, rejoignez notre
              communauté sur Telegram
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="Qu'est-ce que Databasus et comment sauvegarde-t-il les bases de données MongoDB ?"
              answer="Databasus est un outil de sauvegarde auto-hébergé sous licence Apache 2.0 qui utilise mongodump en interne pour créer des backups MongoDB cohérents. Il complète mongodump avec une interface web moderne, une planification automatisée, l'intégration du stockage cloud (S3, Google Drive, Dropbox), des notifications en temps réel (Slack, Discord, Telegram) et le chiffrement AES-256-GCM : plus besoin de scripts shell maison ni de tâches cron."
            />
            <FaqItem
              number="2"
              question="Databasus prend-il en charge les replica sets MongoDB ?"
              answer="Oui, Databasus prend entièrement en charge les replica sets MongoDB. Vous pouvez vous connecter à n'importe quel membre d'un replica set en utilisant le format d'URI de connexion MongoDB standard avec les options de replica set. Databasus lit depuis le nœud indiqué, ce qui vous permet de sauvegarder depuis des nœuds secondaires pour réduire la charge sur votre primaire. C'est particulièrement utile en production, quand vous voulez éviter d'impacter les performances du nœud primaire."
            />
            <FaqItem
              number="3"
              question="Puis-je sauvegarder des bases MongoDB Atlas avec Databasus ?"
              answer="Oui, Databasus fonctionne parfaitement avec MongoDB Atlas. Comme Databasus utilise des backups logiques via mongodump, il ne nécessite que des identifiants de connexion MongoDB standard : aucune permission Atlas particulière, aucun rôle administratif, et pas d'autre liste blanche d'IP que celle de votre serveur Databasus. Fournissez simplement votre chaîne de connexion Atlas (disponible dans le tableau de bord Atlas) et Databasus s'occupe du reste."
            />
            <FaqItem
              number="4"
              question="Quelles versions de MongoDB Databasus prend-il en charge ?"
              answer="Databasus prend en charge les versions 4, 5, 6, 7, 8 et 9 de MongoDB. Tous les backups utilisent l'outil natif mongodump avec les options --archive et --gzip pour produire des archives BSON compressées et efficaces. Les archives peuvent être restaurées avec mongorestore vers toute version MongoDB compatible, ce qui simplifie les migrations entre versions."
            />
            <FaqItem
              number="5"
              question="Comment Databasus gère-t-il les grandes collections MongoDB ?"
              answer="Databasus envoie la sortie de mongodump directement vers votre destination de stockage, en chiffrant éventuellement le flux en transit. Cette approche évite d'écrire des fichiers temporaires sur disque, ce qui la rend efficace pour les bases avec de grandes collections. L'option --archive crée un seul fichier compressé au lieu d'une arborescence de répertoires, ce qui réduit les entrées-sorties et simplifie la gestion du stockage."
            />
            <FaqItem
              number="6"
              question="Puis-je sauvegarder des clusters MongoDB shardés avec Databasus ?"
              answer={
                <>
                  Databasus se concentre actuellement sur la sauvegarde de bases
                  MongoDB individuelles plutôt que sur des sauvegardes
                  coordonnées de clusters shardés.
                  <br />
                  <br />
                  Pour les clusters shardés, vous pouvez :
                  <br />
                  <br />
                  • Sauvegarder chaque shard individuellement en vous connectant
                  aux replica sets des shards
                  <br />
                  • Sauvegarder via un routeur mongos (cela peut toutefois
                  affecter les performances)
                  <br />
                  <br />
                  Pour les clusters shardés en production, envisagez les
                  sauvegardes natives de MongoDB Atlas ou mongodump avec --oplog
                  pour une cohérence à un instant donné entre les shards.
                </>
              }
            />
            <FaqItem
              number="7"
              question="Comment Databasus sécurise-t-il les identifiants et les backups MongoDB ?"
              answer={
                <>
                  Databasus applique une sécurité à plusieurs niveaux :
                  <br />
                  <br />
                  <strong>1. Chiffrement des identifiants :</strong> toutes les
                  URI de connexion, les mots de passe et les informations
                  d&apos;authentification MongoDB sont chiffrés avec AES-256-GCM
                  avant stockage.
                  <br />
                  <br />
                  <strong>2. Chiffrement des backups :</strong> chaque archive
                  BSON est chiffrée avec une clé unique dérivée de la clé
                  maître, de l&apos;ID du backup et d&apos;un sel aléatoire.
                  <br />
                  <br />
                  <strong>3. Gestion sécurisée des identifiants :</strong> les
                  URI de connexion sont transmises directement à mongodump via
                  des paramètres sécurisés, jamais exposées dans les logs ni la
                  liste des processus.
                </>
              }
            />
            <FaqItem
              number="8"
              question="Databasus prend-il en charge les sauvegardes MongoDB incrémentales ou le suivi de l'oplog ?"
              answer="Databasus se concentre sur les backups logiques complets via mongodump plutôt que sur les backups incrémentaux ou la restauration à un instant donné basée sur l'oplog. Pour la plupart des cas d'usage, des backups complets planifiés (horaires, quotidiens, hebdomadaires) offrent des points de restauration suffisants sans la complexité de la gestion de l'oplog. MongoDB Atlas propose déjà des sauvegardes continues natives avec restauration à un instant donné, et les backups incrémentaux externes ne peuvent pas être facilement restaurés sur des clusters Atlas."
            />
            <FaqItem
              number="9"
              question="Puis-je restaurer des backups MongoDB vers une autre version ou un autre cluster ?"
              answer="Oui, comme Databasus crée des archives mongodump standard au format BSON, vous pouvez les restaurer sur tout serveur MongoDB compatible : autre version, autre fournisseur cloud ou machine de développement locale. Téléchargez le backup depuis Databasus (déchiffré automatiquement), puis utilisez mongorestore avec les options --archive et --gzip. Databasus affiche la commande de restauration exacte pour chaque backup."
            />
            <FaqItem
              number="10"
              question="Comment fonctionne la compression mongodump dans Databasus ?"
              answer="Databasus utilise l'option intégrée --gzip de mongodump, qui compresse les données BSON pendant le dump. Cela réduit généralement la taille de l'archive de 60 à 80 % par rapport au BSON non compressé. La compression a lieu dans le flux mongodump avant le chiffrement optionnel, donc les archives compressées et chiffrées restent efficaces. La décompression est automatique lors de l'utilisation de mongorestore avec l'option --gzip."
            />
            <FaqItem
              number="11"
              question="Puis-je sauvegarder des collections MongoDB précises au lieu de bases entières ?"
              answer="Actuellement, Databasus sauvegarde des bases MongoDB entières plutôt que des collections individuelles. Cela garantit des backups complets et cohérents, incluant toutes les collections, les index et les métadonnées. Si vous avez besoin de sauvegardes au niveau des collections, vous pouvez créer des bases séparées pour vos différents domaines de données, chacune avec sa propre planification de sauvegarde dans Databasus."
            />
            <FaqItem
              number="12"
              question="Databasus fonctionne-t-il avec MongoDB exécuté dans Docker ou Kubernetes ?"
              answer="Oui, Databasus se connecte à MongoDB via le réseau avec des URI de connexion standard, il fonctionne donc avec MongoDB quel que soit son déploiement : conteneurs Docker, pods Kubernetes, VM ou serveurs physiques. Assurez simplement la connectivité réseau entre Databasus et votre instance MongoDB. Pour les déploiements Kubernetes, vous pouvez utiliser les noms DNS des services internes ou les points d'accès d'un load balancer externe."
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
