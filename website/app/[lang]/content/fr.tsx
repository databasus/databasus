import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl, getLocalizedHref } from "@/app/i18n";
import HeroStatsComponent from "@/app/components/HeroStatsComponent";
import InstallationComponent from "@/app/components/InstallationComponent";
import LiteYouTubeEmbed from "@/app/components/LiteYouTubeEmbed";

export const metadata: Metadata = {
  title: "Sauvegarde PostgreSQL | Databasus",
  description:
    "Outil gratuit et open source pour les sauvegardes planifiées de PostgreSQL (avec prise en charge de MySQL et MongoDB). Stockez-les en local et dans le cloud. Notifications vers Slack, Discord, Telegram, e-mail, webhook, etc.",
  keywords:
    "PostgreSQL, sauvegarde, backup, supervision, base de données, sauvegardes planifiées, Docker, auto-hébergé, open source, S3, Azure Blob Storage, notifications Slack, Discord, DevOps, supervision de bases de données, pg_dump, restauration de base de données, chiffrement, AES-256, chiffrement des sauvegardes",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("fr", ""),
    languages: getLanguageAlternates(""),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("fr", ""),
    title: "Sauvegarde PostgreSQL | Databasus",
    description:
      "Outil gratuit et open source pour les sauvegardes planifiées de PostgreSQL (avec prise en charge de MySQL et MongoDB). Stockez-les en local et dans le cloud. Notifications vers Slack, Discord, Telegram, e-mail, webhook, etc.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Interface du tableau de bord Databasus pour la gestion des sauvegardes",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary_large_image",
    title: "Sauvegarde PostgreSQL | Databasus",
    description:
      "Outil gratuit et open source pour les sauvegardes planifiées de PostgreSQL (avec prise en charge de MySQL et MongoDB). Stockez-les en local et dans le cloud. Notifications vers Slack, Discord, Telegram, e-mail, webhook, etc.",
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

export default function Index() {
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
              "Outil gratuit et open source pour les sauvegardes planifiées de PostgreSQL (avec prise en charge de MySQL et MongoDB). Stockez-les en local et dans le cloud. Notifications vers Slack, Discord, Telegram, e-mail, webhook, etc.",
            url: "https://databasus.com",
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
              "Sauvegardes planifiées de PostgreSQL",
              "Plusieurs destinations de stockage (S3, Azure Blob Storage, FTP, SFTP, NAS, etc.)",
              "Notifications en temps réel (Slack, Telegram, Discord, webhook, e-mail, etc.)",
              "Supervision de l'état des bases de données",
              "Auto-hébergé via Docker",
              "Open source et gratuit",
              "Prise en charge de PostgreSQL 12-18",
              "Compression des sauvegardes et chiffrement AES-256-GCM",
              "Prise en charge de PostgreSQL, MySQL, MariaDB et MongoDB",
              "Politiques de rétention : période, nombre, GFS et limites de taille",
              "Point-in-Time Recovery (PITR) avec archivage WAL",
              "Vérification de restauration : tests de restauration automatisés dans de vrais conteneurs Docker de base de données",
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
            "@type": "Organization",
            name: "Databasus",
            url: "https://databasus.com/",
            alternateName: ["databasus", "Databasus"],
            logo: "https://databasus.com/logo.svg",
            sameAs: ["https://github.com/databasus/databasus"],
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "WebSite",
            name: "Databasus",
            alternateName: ["databasus", "Databasus"],
            url: "https://databasus.com/",
            description: "Outil de sauvegarde PostgreSQL",
            publisher: { "@type": "Organization", name: "Databasus" },
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
                name: "Qu'est-ce que Databasus et pourquoi l'utiliser plutôt que des scripts écrits à la main ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus est un service auto-hébergé sous licence Apache 2.0 qui sauvegarde PostgreSQL, de la v13 à la v18. Il se distingue des scripts shell par son interface qui permet de planifier les tâches, de compresser et de stocker les archives vers plusieurs destinations (disque local, S3, Azure Blob Storage, FTP, SFTP, NAS, etc.), de configurer des politiques de rétention pour supprimer automatiquement les anciennes sauvegardes et de notifier votre équipe lorsque les tâches se terminent ou échouent, le tout sans code écrit à la main",
                },
              },
              {
                "@type": "Question",
                name: "Comment installer Databasus le plus rapidement possible ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Le moyen le plus direct est de lancer l'installateur cURL en une ligne. Il récupère l'image Docker actuelle et démarre un conteneur PostgreSQL unique. Il crée ensuite un docker-compose.yml et lance le service afin qu'il redémarre automatiquement après un reboot. Le temps total est en général inférieur à deux minutes sur un VPS classique.",
                },
              },
              {
                "@type": "Question",
                name: "Comment fonctionne la vérification de restauration ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus exécute un petit agent de vérification sur un hôte que vous contrôlez. À chaque exécution planifiée, l'agent télécharge la dernière sauvegarde. Il la restaure dans un conteneur de base de données jetable. Il vérifie ensuite la cohérence de la base restaurée par rapport à la source. Le résultat est remonté, avec le code de sortie de la restauration et le nombre de lignes par table. Les planifications acceptent : après chaque sauvegarde, toutes les heures, chaque jour, chaque semaine, chaque mois ou une expression cron en UTC. Les échecs peuvent être envoyés via n'importe quel canal de notification relié à la base : Slack, Teams, Discord, e-mail et autres.",
                },
              },
              {
                "@type": "Question",
                name: "Où sont stockées mes sauvegardes et combien d'espace occuperont-elles ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Les archives peuvent être enregistrées sur des volumes locaux, des buckets compatibles S3, Azure Blob Storage, FTP et d'autres destinations cloud. Databasus applique une compression équilibrée, qui réduit en général la taille du dump de 4 à 8 fois avec seulement environ 20 % de surcoût d'exécution, ce qui économise du stockage et de la bande passante.",
                },
              },
              {
                "@type": "Question",
                name: "Comment saurai-je qu'une sauvegarde a réussi, ou pire, échoué ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus peut vous notifier en temps réel par e-mail, Slack, Telegram, webhooks, Mattermost, Discord et plus encore. Vous choisissez les canaux à alerter afin que votre équipe DevOps soit informée des succès et des échecs, ce qui facilite les procédures de reprise et les audits de conformité.",
                },
              },
              {
                "@type": "Question",
                name: "Comment Databasus assure-t-il la sécurité ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus applique la sécurité à trois niveaux : (1) Chiffrement des données sensibles — tous les mots de passe, jetons et identifiants sont chiffrés avec AES-256-GCM et stockés séparément de la base de données ; (2) Chiffrement des sauvegardes — chaque fichier de sauvegarde est chiffré avec une clé unique dérivée d'une clé maître, de l'ID de la sauvegarde et d'un sel aléatoire, ce qui rend les sauvegardes inutilisables sans votre clé de chiffrement même si quelqu'un accède au stockage ; (3) Accès en lecture seule à la base — Databasus ne requiert que des permissions SELECT et effectue des vérifications complètes pour s'assurer qu'aucun privilège d'écriture n'existe, ce qui empêche toute corruption de données même si l'outil est compromis. La connexion elle-même peut exiger un second facteur : une fois activé, un mot de passe correct est suivi d'un code à six chiffres envoyé à l'adresse du compte, tandis que la connexion par Google ou GitHub continue de s'appuyer sur les vérifications de ces fournisseurs. Avant chaque version, des contrôles automatiques analysent le code, les dépendances, les secrets et les images de conteneurs. Les tests vérifient des cycles complets de sauvegarde et de restauration dans de vrais conteneurs PostgreSQL, MySQL, MariaDB et MongoDB. Les contrôles obligatoires doivent réussir avant la publication. Des audits réguliers de Codex Security d’OpenAI les complètent et sont distincts du processus de publication. Toutes les opérations tournent dans des conteneurs que vous contrôlez sur des serveurs qui vous appartiennent, et comme le projet est open source, votre équipe de sécurité peut auditer chaque ligne de code avant le déploiement.",
                },
              },
              {
                "@type": "Question",
                name: "Databasus est-il soutenu par les programmes OSS d'Anthropic et d'OpenAI ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Oui, en mars 2026 Databasus a été accepté à la fois dans Claude for Open Source d'Anthropic et dans Codex for Open Source d'OpenAI. Le projet a été évalué de manière indépendante et reconnu par les leaders du secteur comme une infrastructure open source critique qui mérite d'être soutenue.",
                },
              },
              {
                "@type": "Question",
                name: "En quoi Databasus diffère-t-il de PgBackRest, Barman ou pg_dump ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus privilégie la simplicité : il fournit une interface web moderne pour gérer les sauvegardes de nombreuses bases à la fois, avec planification intégrée, compression, plusieurs destinations de stockage, supervision de l'état et notifications en temps réel. En même temps, contrairement à pgBackRest et WAL-G, Databasus réalise les sauvegardes physiques, incrémentales et WAL au-dessus de l'approche native de PostgreSQL 17, il ne réinvente donc pas son propre moteur de sauvegarde. Il se connecte à vos bases à distance et atteint les réseaux fermés via un tunnel SSH vers le serveur ou un bastion, si bien que les bases non exposées publiquement peuvent quand même être sauvegardées et gérées depuis un seul tableau de bord.",
                },
              },
              {
                "@type": "Question",
                name: "Quelles bases de données sont prises en charge par Databasus ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus prend en charge PostgreSQL, MySQL, MariaDB et MongoDB. Il a cependant été créé à l'origine pour PostgreSQL et reste centré sur celui-ci, avec une prise en charge complète et une efficacité maximale pour les sauvegardes PostgreSQL. PostgreSQL garde la priorité, avec les fonctionnalités les plus abouties et un développement continu. Par exemple, Databasus offre un support natif des sauvegardes physiques et WAL pour la reprise après sinistre de PostgreSQL. Databasus est donc avant tout un outil de sauvegarde PostgreSQL ; les autres bases viennent en complément.",
                },
              },
              {
                "@type": "Question",
                name: "Quel est le niveau d'adoption de Databasus ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus est aujourd'hui l'outil open source de sauvegarde PostgreSQL le plus largement adopté. Au 17 juin 2026, il a été téléchargé plus de 1 800 000 fois sur Docker par des DBA, des ingénieurs DevOps, des développeurs et des équipes du monde entier. Avec plus de 8 500 étoiles GitHub, il dépasse pgBackRest (~4 200 étoiles, disponible depuis 2014) et WAL-G (~4 100 étoiles, disponible depuis 2017). Databasus a été lancé en 2025 et les a dépassés tous les deux dès sa première année.",
                },
              },
              {
                "@type": "Question",
                name: "Quels types de sauvegarde Databasus prend-il en charge ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus prend en charge les sauvegardes physiques, complètes, incrémentales, WAL et logiques. Les sauvegardes physiques sont une copie au niveau fichier de l'ensemble du cluster de base de données : plus rapides à sauvegarder et à restaurer pour les gros volumes que les dumps logiques, et construites sur le mécanisme de sauvegarde natif de PostgreSQL 17, nous nous appuyons donc sur l'outillage éprouvé de PostgreSQL au lieu de le réinventer. Les sauvegardes complètes sont une copie intégrale et autonome du cluster, la base de départ de chaque chaîne de sauvegardes. Les sauvegardes incrémentales ne stockent que ce qui a changé depuis la sauvegarde précédente, elles restent donc petites et rapides. Le streaming WAL capture en continu le flux d'écriture de la base, ce qui permet la récupération à un instant donné (PITR) pour la reprise après sinistre et une perte de données quasi nulle. Les sauvegardes logiques sont un dump natif de la base dans son format binaire propre au moteur, compressé et envoyé en flux directement vers le stockage sans fichiers intermédiaires. Toutes ces sauvegardes peuvent passer par un tunnel SSH si vous exigez des connexions non publiques, la base n'a donc jamais à être exposée publiquement. Le tunnel SSH est intégré.",
                },
              },
              {
                "@type": "Question",
                name: "J'ai oublié l'e-mail ou le mot de passe de l'administrateur",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Le premier compte créé sur l'instance est celui qui l'administre. Si vous ne savez plus quelle est cette adresse, exécutez docker exec -it databasus ./main --list-admins sur le serveur où Databasus tourne : la commande nomme chaque compte administrateur avec son adresse e-mail, son nom affiché, sa date de création et son état d'activité, et n'affiche aucun mot de passe. Pour définir un nouveau mot de passe pour ce compte, exécutez docker exec -it databasus ./main --new-password=\"YourNewSecurePassword123\" --email=\"owner@example.com\". Une instance créée avant que le premier compte ne l'administre porte encore l'adresse provisoire admin tant que son propriétaire ne l'a pas remplacée. Si l'instance exige en plus un code à la connexion et que le serveur de messagerie ne le délivre plus, exécutez docker exec -it databasus ./main --disable-2fa pour qu'elle cesse de le demander. Les trois commandes sont documentées sur la page mot de passe.",
                },
              },
              {
                "@type": "Question",
                name: "Comment l’IA est-elle utilisée dans le développement de Databasus ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "L’IA nous aide à développer Databasus dans un cadre d’ingénierie défini. Les personnes définissent l’orientation et prennent les décisions finales. Une personne lit chaque ligne de code avant l’intégration d’un changement. Des agents indépendants examinent le fonctionnement, l’architecture, les tests et la documentation dans des sessions séparées en lecture seule. Une analyse statique stricte et des tests sur de vraies bases de données complètent leurs revues. Les contrôles CI obligatoires doivent réussir avant la publication.",
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        lang="fr"
        sectionLinks={[
          { href: "#how-to-use", label: "Comment l'utiliser" },
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

            <h1 className="text-2xl sm:text-4xl sm:max-w-[300px] md:text-4xl leading-tight font-bold mb-4 md:mb-6 mx-auto md:max-w-[650px]">
              Sauvegarde PostgreSQL
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus est un outil gratuit, open source et auto-hébergé pour
              faire des backups de PostgreSQL. Créez des sauvegardes vers
              différents stockages (S3, Azure Blob Storage, FTP, etc.) avec des
              notifications sur leur progression (Slack, Discord, Telegram,
              etc.). Avec un focus sur le Point-in-Time Recovery{" "}
              <span className="underline decoration-2 underline-offset-2 decoration-blue-600">
                à faible RPO/RTO
              </span>
            </p>

            <div>
              <div className="flex flex-col items-center justify-center gap-2 max-w-[370px] sm:max-w-[340px] mx-auto">
                <a
                  href="#installation"
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-1"
                >
                  Auto-héberger via Docker
                </a>

                <a
                  href="https://github.com/databasus/databasus"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 rounded-lg font-medium border border-[#ffffff20] bg-[#0C0E13] hover:opacity-70 transition-opacity order-2 sm:order-2"
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

              <div className="mt-2 max-w-[370px] sm:max-w-[340px] mx-auto pb-0 sm:pb-[50px] lg:pb-0">
                <HeroStatsComponent />
              </div>
            </div>
          </div>

          {/* Dashboard Screenshot */}
          <div className="relative mx-auto max-w-[1200px]">
            <div>
              <img
                src="/images/index/dashboard.svg"
                alt="Interface du tableau de bord Databasus"
                width={980}
                height={620}
                className="w-full h-auto"
                loading="eager"
                fetchPriority="high"
              />
            </div>
          </div>

          <div className="mt-10 md:mt-15 mb-12 md:mb-20 flex justify-center px-4 md:px-0">
            <div className="flex flex-col md:flex-row items-center">
              <img
                className="h-[45px] md:h-[55px]"
                src="/images/index/ais.svg"
                alt="Soutenu par les programmes OSS d'Anthropic et d'OpenAI"
              />

              <div className="flex justify-center text-base md:text-xl mt-4 md:mt-0 md:ml-10">
                <div className="max-w-[370px] text-gray-400 text-center md:text-left">
                  Soutenu par les programmes open source d&apos;Anthropic et
                  d&apos;OpenAI.{" "}
                  <a
                    href="/fr/faq#oss-programs"
                    target="_blank"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    En&nbsp;savoir&nbsp;plus&nbsp;→
                  </a>
                </div>
              </div>
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
              Fonctionnalités
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus fournit tout ce dont vous avez besoin pour gérer des
              backups fiables en production. De la planification automatisée au
              chiffrement des sauvegardes. Il convient aussi bien aux
              développeurs indépendants avec des projets personnels qu&apos;aux
              équipes DevOps et aux entreprises
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
                Sauvegardes planifiées
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Sauvegardes planifiées"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Un backup doit être exécuté à heure fixe et de manière
                régulière. Nous proposons donc de nombreuses options : toutes
                les heures, chaque jour, chaque semaine, chaque mois, cron, etc.
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Contrôles d&apos;état configurables
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Contrôles d'état"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Chaque minute (ou tout autre intervalle), le système interroge
                votre base de données et vous montre l&apos;historique des
                tentatives
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                La base peut être considérée comme indisponible après 3
                tentatives échouées, par exemple. Dès qu&apos;elle est de
                nouveau en bonne santé, vous recevez aussi une notification
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                De nombreuses destinations de stockage
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Les fichiers sont conservés sur VPS, dans des stockages cloud et
                ailleurs. Vous pouvez choisir le stockage qui vous convient. Les
                fichiers vous appartiennent toujours.{" "}
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
                  alt="Stockage"
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
                Notifications
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Vous pouvez recevoir des notifications de succès ou d&apos;échec
                du processus. C&apos;est utile pour les développeurs comme pour
                les équipes DevOps.{" "}
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
                  alt="Notifications"
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
                Databasus tourne sur votre PC ou votre VPS. Toutes vos données
                vous appartiennent donc et restent sécurisées. Le déploiement
                prend environ 2 minutes via script, Docker ou k8s
              </p>

              <div className="flex">
                <img
                  src="/images/index/feature-deploy.svg"
                  alt="Docker"
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
                Le projet est entièrement open source, gratuit et sous licence
                Apache 2.0. Vous pouvez copier et forker le code librement.
                Ouvert aussi aux entreprises
              </p>
              <div>
                <img
                  src="/images/index/feature-github.svg"
                  alt="GitHub"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 7: Restore verification - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Restore verification */}
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Vérification de restauration
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Un backup qui se termine sans erreur n&apos;est pas la même
                  chose qu&apos;un backup que vous pouvez restaurer. Databasus
                  récupère périodiquement la dernière sauvegarde, la restaure
                  dans un conteneur de base de données jetable et rapporte le
                  résultat.{" "}
                  <a
                    href="/fr/restore-verification"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    En savoir plus →
                  </a>
                </p>

                <div>
                  <img
                    src="/images/index/feature-postgresql.svg"
                    alt="PostgreSQL"
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
                  Un chiffrement de niveau entreprise protège les données
                  sensibles et les sauvegardes. L&apos;accès en lecture seule à
                  la base empêche la corruption de données. Tout fonctionne
                  automatiquement dès l&apos;installation, sans connaissance
                  particulière.{" "}
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
                    alt="Sécurité"
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
                  alt="Gestion des accès"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Donnez aux utilisateurs un accès en consultation ou en gestion
                des bases. Séparez les équipes et les projets. Convient aux
                équipes DevOps et aux développeurs.{" "}
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
                  alt="Journaux d'audit"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Suivez toutes les activités du système grâce à des journaux
                d&apos;audit complets. Vous pouvez consulter l&apos;historique
                des accès et des modifications de chaque utilisateur
                (téléchargements de sauvegardes, changements de planification,
                mises à jour de configuration, etc.).{" "}
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
                Un chiffrement de niveau entreprise protège les données
                sensibles et les sauvegardes. L&apos;accès en lecture seule à la
                base empêche la corruption de données. Tout fonctionne
                automatiquement dès l&apos;installation, sans connaissance
                particulière.{" "}
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
                  alt="Sécurité"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 11: Backup types and modes */}
            <div className="col-span-1 md:col-span-2 lg:col-span-2 p-5 md:p-6 flex flex-col md:flex-row gap-4 md:gap-6">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20] shrink-0">
                11
              </div>

              <div>
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Sauvegardes logiques, physiques, incrémentales et WAL
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus prend en charge les sauvegardes logiques et
                  physiques (complètes et incrémentales) avec streaming WAL pour
                  le Point-in-Time Recovery. Cela rend Databasus adapté à la
                  reprise après sinistre, et il fonctionne aussi bien avec les
                  bases auto-hébergées que cloud : utilisez le mode distant pour
                  les bases gérées dans le cloud ou accessibles publiquement.
                  Les sauvegardes physiques s&apos;appuient sur les sauvegardes
                  natives de PG 17.{" "}
                  <a
                    href="/fr/faq/#pitr"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    En savoir plus →
                  </a>
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* VIDEO SECTION */}
      <section className="pb-12 md:pb-20 px-4 md:px-6 lg:px-0" id="how-to-use">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            {/* Left side: Info */}
            <div className="w-full lg:w-[450px] lg:shrink-0">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Aperçu en 4 minutes</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Comment utiliser Databasus ?
              </h2>

              <p className="text-gray-200 max-w-[450px] leading-relaxed mb-6 md:mb-8 text-sm sm:text-base">
                Découvrez dans cette vidéo comment connecter votre base de
                données, configurer la planification des sauvegardes,
                télécharger et restaurer les sauvegardes, ajouter des membres à
                votre équipe et ce que sont les journaux d&apos;audit des
                utilisateurs
              </p>

              <a
                href="https://rostislav-dugin.com"
                target="_blank"
                className="flex items-center gap-3 md:gap-4 hover:opacity-70 transition-colors"
              >
                <img
                  src="/images/index/rostislav.png"
                  alt="Rostislav Dugin"
                  className="w-10 h-10 md:w-12 md:h-12 rounded-full object-cover"
                  loading="lazy"
                />

                <div>
                  <p className="font-medium text-base md:text-lg">
                    Rostislav Dugin
                  </p>
                  <p className="text-sm text-gray-400">
                    Développeur de Databasus
                  </p>
                </div>
              </a>
            </div>

            {/* Right side: Video */}
            <div className="flex-1 relative">
              <div className="rounded-lg overflow-hidden shadow-lg border border-[#ffffff20]">
                <LiteYouTubeEmbed
                  videoId="1qsAnijJfJE"
                  title="How to use Databasus (overview)?"
                  thumbnailSrc="/images/index/how-to-use-preview.svg"
                />
              </div>
            </div>
          </div>
        </div>
      </section>

      <div className="border-b border-[#ffffff20] max-w-[calc(100%-2rem)] md:max-w-[calc(100%-3rem)] lg:max-w-[1000px] 2xl:max-w-[1200px] mx-auto" />

      {/* Databases section */}
      <section className="pt-12 md:pt-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-10 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Bases de données</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Bases de données prises en charge
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              Databasus prend en charge PostgreSQL, MySQL, MariaDB et MongoDB.
              Vous pouvez toutes les sauvegarder et les restaurer avec le même
              outil. PostgreSQL reste néanmoins la priorité principale
            </p>
          </div>

          {/* Databases list */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 border border-[#ffffff20] rounded-xl">
            {/* PostgreSQL */}
            <div className="border-b md:border-r lg:border-b-0 border-[#ffffff20] p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-postgresql.svg"
                    alt="PostgreSQL"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                PostgreSQL
              </h3>

              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                PostgreSQL est la base de données principale prise en charge par
                Databasus. Toutes les versions de 12 à 18 sont prises en charge
              </p>
            </div>

            {/* MySQL */}
            <div className="border-b lg:border-r lg:border-b-0 border-[#ffffff20] p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-mysql.svg"
                    alt="MySQL"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                MySQL
              </h3>

              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                MySQL est la deuxième base de données la plus populaire au
                monde. Vous pouvez sauvegarder et restaurer vos bases MySQL avec
                la même simplicité
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/fr/mysql-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  En savoir plus →
                </a>
              </div>
            </div>

            {/* MariaDB */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-mariadb.svg"
                    alt="MariaDB"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                MariaDB
              </h3>

              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                MariaDB est pris en charge avec les mêmes fonctionnalités que
                MySQL. Vous pouvez sauvegarder et restaurer vos bases MariaDB en
                toute fluidité
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/fr/mysql-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  En savoir plus →
                </a>
              </div>
            </div>

            {/* MongoDB */}
            <div className="p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-mongodb.svg"
                    alt="MongoDB"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                MongoDB
              </h3>

              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                MongoDB est la base NoSQL la plus populaire. Vous pouvez
                sauvegarder et restaurer vos bases MongoDB avec la même
                interface simple à utiliser
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/fr/mongodb-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  En savoir plus →
                </a>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* PROCESS SECTION */}
      <section className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-10 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Processus</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Comment faire une sauvegarde PostgreSQL ?
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              La priorité de Databasus est la simplicité : c&apos;est
              aujourd&apos;hui l&apos;outil le plus simple au monde pour faire
              des backups de PostgreSQL. Pour créer des sauvegardes, suivez 4
              étapes. Ensuite, vous pourrez restaurer en un clic
            </p>
          </div>

          {/* Steps */}
          <div className="space-y-6 md:space-y-10 max-w-[1000px] mx-auto">
            {/* Step 1 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Étape 1
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Sélectionnez la planification souhaitée
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Vous pouvez choisir le rythme qui vous convient : chaque
                    jour, chaque semaine, chaque mois, à une heure précise
                    (comme 4 h du matin) ou par cycles cron
                  </p>
                  <p>
                    Pour un intervalle hebdomadaire, vous devez indiquer un jour
                    précis ; pour un intervalle mensuel, vous devez aussi
                    indiquer un jour précis
                  </p>
                  <p>
                    Si votre base est volumineuse, nous vous recommandons de
                    choisir un créneau où le trafic diminue
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Étape 1"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 2 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Étape 2
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Renseignez les informations de votre base
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Saisissez les identifiants de votre base PostgreSQL,
                    sélectionnez la version et la base cible. Indiquez aussi si
                    SSL est requis
                  </p>
                  <p>
                    Par défaut, Databasus compresse les sauvegardes à un niveau
                    équilibré pour ne pas ralentir le processus (~20 % plus
                    lent) tout en économisant 4 à 8 fois l&apos;espace (ce qui
                    réduit le trafic réseau)
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-2.svg"
                  alt="Étape 2"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 3 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Étape 3
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Choisissez le stockage des sauvegardes
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Vous pouvez conserver les fichiers de sauvegarde en local,
                    dans S3, Azure Blob Storage, sur un NAS, FTP et d&apos;autres
                    services
                  </p>
                  <p>
                    Gardez à l&apos;esprit que vous devez disposer de
                    suffisamment d&apos;espace sur le stockage
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-3.svg"
                  alt="Étape 3"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 4 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Étape 4
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Choisissez où recevoir les notifications (optionnel)
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Quand un backup réussit ou échoue, Databasus peut vous
                    envoyer une notification. Cela peut être un chat avec les
                    DevOps, vos e-mails ou même le webhook de votre équipe
                  </p>
                  <p>
                    Nous allons prendre en charge la plupart des messageries et
                    plateformes populaires
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-4.svg"
                  alt="Étape 4"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>
          </div>

          {/* CTA Button */}
          <div className="text-center mt-8 md:mt-12">
            <a
              href="#installation"
              className="inline-flex items-center gap-2 px-6 py-3 bg-white text-black rounded-lg text-[15px] font-medium hover:opacity-70 transition-colors"
            >
              Commencer
              <svg
                aria-hidden={true}
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </a>
          </div>
        </div>
      </section>

      {/* INSTALLATION SECTION */}
      <section id="installation" className="px-4 md:px-6 lg:px-0">
        <div className="max-w-[1000px] 2xl:max-w-[1200px] mx-auto border border-[#ffffff20] rounded-xl py-10 md:py-20 px-4 md:px-6">
          <div className="max-w-[1100px] mx-auto">
            <div className="text-center mb-8 md:mb-10">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Commencer</span>
              </div>

              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
                Comment l&apos;installer ?
              </h2>

              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                Databasus prend en charge de nombreux modes d&apos;installation.
                En local comme dans le cloud. Les deux sont extrêmement simples
                et accessibles même à ceux qui n&apos;ont aucune expérience en
                administration ou en DevOps
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
              Questions fréquentes
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              L&apos;objectif de Databasus : rendre la sauvegarde aussi simple
              que possible pour les développeurs seuls (comme pour les DevOps)
              et les équipes. L&apos;interface facilite la création de backups,
              visualise leur progression et permet de tout restaurer en quelques
              clics
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="Qu'est-ce que Databasus et pourquoi l'utiliser plutôt que des scripts écrits à la main ?"
              answer="Databasus est un service auto-hébergé sous licence Apache 2.0 qui sauvegarde les bases de données. Il se distingue des scripts shell par son interface qui permet de planifier les tâches, de compresser et de stocker les archives vers plusieurs destinations (disque local, S3, Azure Blob Storage, NAS, FTP, SFTP, etc.), de configurer des politiques de rétention pour supprimer automatiquement les anciennes sauvegardes et de notifier votre équipe lorsque les tâches se terminent ou échouent, le tout sans code écrit à la main"
            />
            <FaqItem
              number="2"
              question="Comment installer Databasus le plus rapidement possible ?"
              answer="Databasus prend en charge plusieurs méthodes d'installation : script automatisé, Docker, Docker Compose et Kubernetes avec Helm. Le moyen le plus rapide est de lancer l'installateur cURL en une ligne, qui récupère l'image Docker actuelle, crée un docker-compose.yml et démarre le service afin qu'il redémarre automatiquement après un reboot. Pour les environnements Kubernetes, utilisez le chart Helm officiel pour des déploiements prêts pour la production. Le temps total est en général inférieur à deux minutes sur un VPS classique."
            />
            <FaqItem
              number="3"
              question="Comment fonctionne la vérification de restauration ?"
              answer="Databasus exécute un petit agent de vérification sur un hôte que vous contrôlez. À chaque exécution planifiée, l'agent télécharge la dernière sauvegarde. Il la restaure dans un conteneur de base de données jetable. Il vérifie ensuite la cohérence de la base restaurée par rapport à la source. Le résultat est remonté, avec le code de sortie de la restauration et le nombre de lignes par table. Les planifications acceptent : après chaque sauvegarde, toutes les heures, chaque jour, chaque semaine, chaque mois ou une expression cron en UTC. Les échecs peuvent être envoyés via n'importe quel canal de notification relié à la base : Slack, Teams, Discord, e-mail et autres."
            />
            <FaqItem
              number="4"
              question="Comment Databasus assure-t-il la sécurité ?"
              answer={
                <>
                  Databasus applique la sécurité à trois niveaux : (1)
                  Chiffrement des données sensibles : tous les mots de passe,
                  jetons et identifiants sont chiffrés avec AES-256-GCM et
                  stockés séparément de la base de données ; (2) Chiffrement des
                  sauvegardes : chaque fichier de sauvegarde est chiffré avec
                  une clé unique dérivée d&apos;une clé maître, de l&apos;ID de
                  la sauvegarde et d&apos;un sel aléatoire, ce qui rend les
                  sauvegardes inutilisables sans votre clé de chiffrement même
                  si quelqu&apos;un accède au stockage ; (3) Accès en lecture
                  seule à la base : Databasus ne requiert que des permissions
                  SELECT et effectue des vérifications complètes pour
                  s&apos;assurer qu&apos;aucun privilège d&apos;écriture
                  n&apos;existe, ce qui empêche toute corruption de données même
                  si l&apos;outil est compromis.
                  <br />
                  <br />
                  La connexion elle-même peut exiger un second facteur : une
                  fois activé, un mot de passe correct est suivi d&apos;un code
                  à six chiffres envoyé à l&apos;adresse du compte, tandis que
                  la connexion par Google ou GitHub continue de s&apos;appuyer
                  sur les vérifications de ces fournisseurs.
                  <br />
                  <br />
                  Avant chaque version, des contrôles automatiques analysent le code, les
                  dépendances, les secrets et les images de conteneurs. Les tests vérifient
                  des cycles complets de sauvegarde et de restauration dans de vrais
                  conteneurs PostgreSQL, MySQL, MariaDB et MongoDB. Les contrôles obligatoires
                  doivent réussir avant la publication. Des audits réguliers de Codex Security
                  d’OpenAI les complètent et sont distincts du processus de publication.
                  <br />
                  <br />
                  Consultez{" "}
                  <a
                    href={`${getLocalizedHref("fr", "security")}#security-and-reliability-engineering`}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Ingénierie de la sécurité et de la fiabilité
                  </a>{" "}
                  pour le pipeline complet.
                  <br />
                  <br />
                  De plus, tous les journaux peuvent être exportés vers
                  n&apos;importe quel système externe (VictoriaLogs, SigNoz,
                  Graylog, etc.) via le standard OpenTelemetry. Par défaut, les
                  journaux (y compris les journaux d&apos;audit) sont aussi
                  écrits dans des fichiers locaux, les journaux d&apos;audit ne
                  peuvent donc pas être perdus. Consultez la{" "}
                  <a
                    href="/fr/advanced-config#logging"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    configuration avancée
                  </a>{" "}
                  ici.
                </>
              }
            />
            <FaqItem
              number="5"
              question="Comment configurer et lancer ma première tâche de sauvegarde dans Databasus ?"
              answer={
                <>
                  Pour lancer votre toute première sauvegarde Databasus,
                  connectez-vous simplement au tableau de bord, cliquez sur New
                  Backup, sélectionnez un intervalle : toutes les heures, chaque
                  jour, chaque semaine, chaque mois ou cron. Indiquez ensuite
                  l&apos;heure exacte d&apos;exécution (par exemple 02:30 pour
                  les heures creuses).
                  <br />
                  <br />
                  Saisissez ensuite l&apos;hôte de votre PostgreSQL, le port, le
                  nom de la base, les identifiants et la préférence SSL.
                  Choisissez où l&apos;archive doit être envoyée (chemin local,
                  bucket S3, Azure Blob Storage, FTP, etc.). <br />
                  <br />
                  Si besoin, ajoutez des canaux de notification comme
                  l&apos;e-mail, Slack, Telegram ou un webhook et cliquez sur
                  Save. Databasus valide instantanément les informations, lance
                  la planification, exécute la première tâche et envoie le
                  statut en direct. Vous pourrez ainsi restaurer d&apos;un geste
                  une fois la sauvegarde terminée.
                </>
              }
            />
            <FaqItem
              number="6"
              question="Quel est le niveau d'adoption de Databasus ?"
              answer="Databasus est aujourd'hui l'outil open source de sauvegarde PostgreSQL le plus largement adopté. Au 17 juin 2026, il a été téléchargé plus de 1 800 000 fois sur Docker par des DBA, des ingénieurs DevOps, des développeurs et des équipes du monde entier. Avec plus de 8 500 étoiles GitHub, il dépasse pgBackRest (~4 200 étoiles, disponible depuis 2014) et WAL-G (~4 100 étoiles, disponible depuis 2017). Databasus a été lancé en 2025 et les a dépassés tous les deux dès sa première année."
            />
            <FaqItem
              number="7"
              question="En quoi Databasus diffère-t-il de PgBackRest, Barman ou pg_dump ? Où lire les comparaisons ?"
              answer={
                <>
                  Databasus privilégie la simplicité : il fournit une interface
                  web moderne pour gérer les backups de nombreuses bases à la
                  fois, plutôt que des fichiers de configuration complexes et
                  des outils en ligne de commande. Contrairement aux scripts
                  pg_dump bruts, il intègre la planification, la compression,
                  plusieurs destinations de stockage, la supervision de
                  l&apos;état et des notifications en temps réel, le tout géré
                  via une interface web simple.
                  <br />
                  <br />
                  En même temps, contrairement à pgBackRest et WAL-G, Databasus
                  réalise les sauvegardes physiques, incrémentales et WAL
                  au-dessus de l&apos;approche native de PostgreSQL 17, il ne
                  réinvente donc pas son propre moteur de sauvegarde. Il se
                  connecte à vos bases à distance et atteint les réseaux fermés
                  via un tunnel SSH vers le serveur ou un bastion, si bien que
                  les bases non exposées publiquement peuvent quand même être
                  sauvegardées et gérées depuis un seul tableau de bord.{" "}
                  <a
                    href="/fr/faq/#pitr"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Lisez comment les sauvegardes physiques et PITR sont
                    implémentées
                  </a>
                  .
                  <br />
                  <br />
                  Nous avons des pages de comparaison détaillées pour les outils
                  de sauvegarde populaires :{" "}
                  <a
                    href="/fr/pgdump-alternative"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pg_dump
                  </a>
                  ,{" "}
                  <a
                    href="/fr/databasus-vs-pgbackrest"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pgBackRest
                  </a>
                  ,{" "}
                  <a
                    href="/fr/databasus-vs-barman"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs Barman
                  </a>
                  ,{" "}
                  <a
                    href="/fr/databasus-vs-wal-g"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs WAL-G
                  </a>{" "}
                  et{" "}
                  <a
                    href="/fr/databasus-vs-pgbackweb"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pgBackWeb
                  </a>
                  . Chaque comparaison explique les différences clés, les
                  avantages et les inconvénients, et vous aide à choisir
                  l&apos;outil adapté à vos besoins.
                </>
              }
            />
            <FaqItem
              number="8"
              question="Databasus est-il soutenu par les programmes OSS d'Anthropic et d'OpenAI ?"
              answer={
                <>
                  Oui, nous sommes fiers que Databasus ait été reconnu comme un
                  projet open source de valeur par deux des plus grandes
                  entreprises d&apos;IA au monde. En mars 2026, Databasus a été
                  accepté à la fois dans{" "}
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Claude for Open Source
                  </a>{" "}
                  d&apos;Anthropic et{" "}
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Codex for Open Source
                  </a>{" "}
                  d&apos;OpenAI. C&apos;est pour nous une confirmation
                  indépendante de fiabilité : le projet a été évalué et reconnu
                  comme une infrastructure critique qui mérite d&apos;être
                  soutenue.{" "}
                  <a
                    href="/fr/faq#oss-programs"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    En savoir plus →
                  </a>
                </>
              }
            />
            <FaqItem
              number="9"
              question="Databasus est-il une alternative à pg_dump ?"
              answer="Pas exactement. Databasus se concentre sur la reprise après sinistre avec un RTO et un RPO faibles, il est donc plus proche d'une alternative à pgBackRest ou WAL-G : il essaie de rendre la reprise après sinistre aussi simple que pg_dump. Cela dit, pour les sauvegardes logiques, il sert bien d'alternative à pg_dump et utilise pg_dump sous le capot, en y ajoutant une interface web conviviale, la planification automatisée, plusieurs destinations de stockage, des notifications en temps réel, la supervision de l'état et le chiffrement des sauvegardes. Les sauvegardes logiques sont aussi disponibles pour MySQL, MariaDB et MongoDB."
            />
            <FaqItem
              number="10"
              question="Quelles bases de données Databasus prend-il en charge ?"
              answer={
                <>
                  Databasus prend en charge PostgreSQL, MySQL, MariaDB et
                  MongoDB. Il a cependant été créé à l&apos;origine pour
                  PostgreSQL et reste centré sur celui-ci, avec une prise en
                  charge complète et une efficacité maximale pour les
                  sauvegardes PostgreSQL.
                  <br />
                  <br />
                  PostgreSQL garde la priorité, avec les fonctionnalités les
                  plus abouties et un développement continu.
                  <br />
                  <br />
                  Par exemple, Databasus offre un support natif des sauvegardes
                  physiques et WAL pour la reprise après sinistre de PostgreSQL.
                  Databasus est donc avant tout un outil de sauvegarde
                  PostgreSQL ; les autres bases viennent en complément.
                </>
              }
            />
            <FaqItem
              number="11"
              question="Quels types de sauvegarde Databasus prend-il en charge ?"
              answer={
                <>
                  Databasus prend en charge les sauvegardes physiques,
                  complètes, incrémentales, WAL et logiques : il convient donc
                  aussi bien à ceux qui veulent de simples dumps logiques
                  qu&apos;à ceux qui ont besoin d&apos;un outil solide de
                  reprise après sinistre.
                  <ul className="list-disc list-inside mt-3 space-y-2">
                    <li>
                      <strong>Physique</strong> : copie au niveau fichier de
                      l&apos;ensemble du cluster de base de données. Sauvegarde
                      et restauration plus rapides que les dumps logiques pour
                      les gros volumes. Construite sur le mécanisme de
                      sauvegarde natif de PostgreSQL 17, nous nous appuyons donc
                      sur l&apos;outillage éprouvé de PostgreSQL au lieu de le
                      réinventer
                    </li>
                    <li>
                      <strong>Complète</strong> : copie intégrale et autonome du
                      cluster, la base de départ de chaque chaîne de sauvegardes
                    </li>
                    <li>
                      <strong>Incrémentale</strong> : ne stocke que ce qui a
                      changé depuis la sauvegarde précédente, les sauvegardes
                      restent donc petites et rapides
                    </li>
                    <li>
                      <strong>Streaming WAL</strong> : capture en continu le
                      flux d&apos;écriture de la base, ce qui permet la
                      récupération à un instant donné (PITR). Conçu pour la
                      reprise après sinistre et une perte de données quasi nulle
                    </li>
                    <li>
                      <strong>Logique</strong> : dump natif de la base dans son
                      format binaire propre au moteur. Compressé et envoyé en
                      flux directement vers le stockage sans fichiers
                      intermédiaires
                    </li>
                  </ul>
                  <br />
                  Les sauvegardes physiques, incrémentales et WAL reposent sur
                  le mécanisme natif de PostgreSQL 17, elles nécessitent donc
                  PostgreSQL 17 ou plus récent ; sur les versions antérieures,
                  seules les sauvegardes logiques sont disponibles. C&apos;est
                  voulu : la plupart des bases en production tournent déjà sur
                  PostgreSQL 17 ou plus, et d&apos;ici environ deux ans les
                  versions plus anciennes atteindront leur fin de vie. Databasus
                  vise à devenir l&apos;outil de sauvegarde standard pour les
                  bases à partir de PostgreSQL 17.
                  <br />
                  <br />
                  Toutes ces sauvegardes peuvent passer par un tunnel SSH si
                  vous exigez des connexions non publiques, la base n&apos;a
                  donc jamais à être exposée publiquement. Le tunnel SSH est
                  intégré.
                </>
              }
            />
            <FaqItem
              number="12"
              question="Comment l’IA est-elle utilisée dans le développement de Databasus ?"
              answer={
                <>
                  <p>
                    L’IA nous aide à développer Databasus dans un cadre d’ingénierie défini. Les personnes
                    définissent l’orientation et prennent les décisions finales.{" "}
                    <strong>Une personne lit chaque ligne de code avant l’intégration d’un changement.</strong>
                  </p>
                  <p className="mt-3">
                    Des agents indépendants examinent le fonctionnement, l’architecture, les tests et la
                    documentation dans des sessions séparées en lecture seule. Une analyse statique stricte et des
                    tests sur de vraies bases de données complètent leurs revues. Les contrôles CI obligatoires
                    doivent réussir avant la publication.{" "}
                    <a
                      href={`${getLocalizedHref("fr", "faq")}#ai-usage`}
                      className="text-blue-400 hover:text-blue-600"
                    >
                      Découvrez le processus de développement
                    </a>
                    .
                  </p>
                </>
              }
            />
            <FaqItem
              number="13"
              question="Comment rejoindre la communauté Databasus ?"
              answer={
                <>
                  Vous pouvez rejoindre notre grande communauté de développeurs,
                  de DBA et d&apos;ingénieurs DevOps sur{" "}
                  <a
                    href="https://t.me/databasus_community"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    t.me/databasus_community
                  </a>
                  . La communauté est un excellent endroit pour poser des
                  questions, partager des expériences, obtenir de l&apos;aide
                  sur la configuration et rester informé des dernières
                  fonctionnalités et versions.
                </>
              }
            />
            <FaqItem
              number="14"
              question="Quel est le niveau d'adoption de Databasus ?"
              answer={
                <>
                  Databasus compte plus de 1,8 million de téléchargements Docker
                  et 8,5 k étoiles GitHub. En comparaison, pgBackRest et WAL-G
                  tournent autour de 4,2 k étoiles chacun et Barman autour de
                  3,1 k, ce qui fait de Databasus l&apos;outil de sauvegarde de
                  bases de données le plus populaire sur GitHub.
                  <br />
                  <br />
                  Il a été accepté dans les programmes open source
                  d&apos;Anthropic et d&apos;OpenAI en tant que projet important
                  et critique. Aujourd&apos;hui, Databasus est utilisé par des
                  entreprises, des équipes et des ingénieurs DevOps, avec le
                  soutien d&apos;une communauté vaste et active.
                  <br />
                  <br />
                  Databasus est développé et utilisé depuis 2023, et open source
                  avec un usage répandu depuis début 2025. Il est en usage réel
                  en production depuis un moment, il a donc été éprouvé sur de
                  nombreux cas limites. Point crucial : Databasus n&apos;invente
                  pas de méthodes maison pour sauvegarder vos données, il
                  s&apos;appuie sur l&apos;implémentation native et testée de
                  PostgreSQL au lieu de construire ses propres contournements
                  pour les cas limites.
                  <br />
                  <br />
                  Notre objectif est de devenir l&apos;outil de sauvegarde
                  standard pour PostgreSQL à partir de la version 17. Databasus
                  est le premier outil de sauvegarde construit sur le protocole
                  de sauvegarde natif, efficace et désormais standard de
                  PostgreSQL au lieu d&apos;écrire ses propres implémentations.
                </>
              }
            />
            <FaqItem
              number="15"
              question="J'ai oublié l'e-mail ou le mot de passe de l'administrateur"
              answer={
                <>
                  Le premier compte créé sur l&apos;instance est celui qui
                  l&apos;administre. Si vous ne savez plus quelle est cette
                  adresse, exécutez docker exec -it databasus ./main
                  --list-admins sur le serveur où Databasus tourne : la commande
                  nomme chaque compte administrateur avec son adresse e-mail,
                  son nom affiché, sa date de création et son état
                  d&apos;activité, et n&apos;affiche aucun mot de passe. Pour
                  définir un nouveau mot de passe pour ce compte, exécutez
                  docker exec -it databasus ./main
                  --new-password=&quot;YourNewSecurePassword123&quot;
                  --email=&quot;owner@example.com&quot;. Une instance créée
                  avant que le premier compte ne l&apos;administre porte encore
                  l&apos;adresse provisoire admin tant que son propriétaire ne
                  l&apos;a pas remplacée. Si l&apos;instance exige en plus un
                  code à la connexion et que le serveur de messagerie ne le
                  délivre plus, exécutez docker exec -it databasus ./main
                  --disable-2fa pour qu&apos;elle cesse de le demander. Les
                  trois commandes sont documentées sur la
                  <a
                    href="/fr/password"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    page mot de passe
                  </a>
                  .
                </>
              }
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
