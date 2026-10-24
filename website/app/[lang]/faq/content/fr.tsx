import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "FAQ - Questions fréquentes | Databasus",
  description:
    "Questions fréquentes sur Databasus, outil de sauvegarde PostgreSQL avec prise en charge de MySQL, MariaDB et MongoDB. Apprenez à sauvegarder des bases localhost, comprendre les formats de backup, les méthodes de compression et plus encore.",
  keywords: [
    "FAQ Databasus",
    "questions sauvegarde PostgreSQL",
    "sauvegarde base de données localhost",
    "formats de sauvegarde",
    "compression pg_dump",
    "compression zstd",
    "aide sauvegarde PostgreSQL",
    "guide de sauvegarde de bases de données",
  ],
  openGraph: {
    title: "FAQ - Questions fréquentes | Databasus",
    description:
      "Questions fréquentes sur Databasus, outil de sauvegarde PostgreSQL avec prise en charge de MySQL, MariaDB et MongoDB. Apprenez à sauvegarder des bases localhost, comprendre les formats de backup, les méthodes de compression et plus encore.",
    type: "article",
    url: getLocalizedUrl("fr", "faq"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "FAQ - Questions fréquentes | Databasus",
    description:
      "Questions fréquentes sur Databasus, outil de sauvegarde PostgreSQL avec prise en charge de MySQL, MariaDB et MongoDB. Apprenez à sauvegarder des bases localhost, comprendre les formats de backup, les méthodes de compression et plus encore.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "faq"),
    languages: getLanguageAlternates("faq"),
  },
  robots: "index, follow",
};

export default function FAQPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "Pourquoi Databasus n'utilise-t-il pas le format de dump SQL brut pour les sauvegardes logiques PostgreSQL ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Pour les sauvegardes logiques, Databasus utilise le format custom de pg_dump avec compression zstd, car des tests approfondis ont montré que c'est le plus efficace en vitesse de sauvegarde et de restauration. Le format custom avec compression zstd niveau 5 offre le meilleur équilibre entre vitesse de création du backup, vitesse de restauration et taille des fichiers.",
                },
              },
              {
                "@type": "Question",
                name: "Où Databasus est-il installé ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus est installé dans /opt/databasus/",
                },
              },
              {
                "@type": "Question",
                name: "Comment fonctionnent les sauvegardes physiques et PITR (Point-in-Time Recovery) ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus exécute les sauvegardes physiques à distance depuis son propre hôte, en se connectant à votre PostgreSQL via le protocole de réplication standard, donc rien n'est à installer sur le serveur de base de données. Les bases dans des réseaux fermés peuvent être atteintes par un tunnel SSH. Les sauvegardes physiques utilisent la pile native de PostgreSQL 17 : sauvegardes complètes via pg_basebackup, incrémentales par blocs via pg_basebackup --incremental pilotées par les résumés WAL côté serveur (summarize_wal = on), et streaming WAL continu via pg_receivewal. Les sauvegardes physiques exigent PostgreSQL 17 ou plus récent ; sur les versions antérieures, on utilise les sauvegardes logiques pg_dump. Pour restaurer à un instant donné, pg_combinebackup reconstruit un répertoire de données exécutable à partir de la sauvegarde complète et de sa chaîne d'incrémentales, puis PostgreSQL rejoue les WAL jusqu'à l'instant cible choisi, avec une restauration possible à n'importe quelle seconde entre deux sauvegardes. L'interface Databasus donne des instructions pas à pas pour restaurer vers un hôte ou une base Docker, soit avec un script prêt à l'emploi qui fait de la restauration une seule commande, soit en téléchargeant les backups et en reconstruisant vous-même la chaîne de parties complètes, incrémentales et WAL. Les incrémentales et les WAL sont optionnels : vous pouvez ne faire qu'une sauvegarde complète, et les WAL ne sont pas obligatoires. Nous utilisons les sauvegardes natives de PostgreSQL 17 parce qu'elles réutilisent la mécanique de sauvegarde éprouvée de PostgreSQL au lieu de la réinventer, fonctionnent avec des bases distantes y compris les services managés comme RDS et Cloud SQL, et offrent une perte de données quasi nulle.",
                },
              },
              {
                "@type": "Question",
                name: "Pourquoi Databasus a-t-il abandonné les sauvegardes par agent ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Une version antérieure de Databasus fournissait un agent de sauvegarde : un binaire qui tournait sur l'hôte de la base pour streamer les WAL et créer des sauvegardes physiques en local. Cette première implémentation s'est révélée être une erreur et a été supprimée. C'était une implémentation naïve qui ne faisait que copier les WAL par-dessus des sauvegardes complètes, avec un RTO long à la clé. Les utilisateurs devaient configurer à la fois Databasus et un agent séparé, alors que tout faire à distance depuis un seul endroit est bien plus simple. Comme l'agent vivait en dehors du système principal, il était difficile de couvrir tous les cas de test. Un agent ne résout en réalité qu'un seul problème : atteindre une base inaccessible de l'extérieur. Pour 99 % des utilisateurs, c'est déjà réglé en exécutant Databasus dans le réseau privé ou en se connectant via SSH. L'agent réinventait donc la roue et compliquait inutilement un problème simple. Il ne pouvait pas non plus tourner sur les bases managées comme RDS et Cloud SQL, qui interdisent les installations sur l'hôte mais exposent déjà le protocole de réplication : un chemin distant était donc nécessaire de toute façon. Il apportait enfin de nombreux cas limites : connexions interrompues, mises à jour de l'agent, collecte des journaux d'un processus séparé. Moins un système a de pièces mobiles, plus il est fiable au quotidien. Les sauvegardes physiques s'exécutent maintenant à distance depuis l'hôte Databasus. Les backups existants restent en sécurité : si vous mettez à jour depuis une version qui a encore des sauvegardes par agent, Databasus ne le fera pas silencieusement. Il vous avertit du changement et vous laisse soit rester sur la version prise en charge 3.42.0, soit supprimer vous-même les anciennes sauvegardes par agent avant la mise à jour. L'implémentation par agent reste disponible jusqu'à la version 3.42.0 et continuera de fonctionner pendant longtemps.",
                },
              },
              {
                "@type": "Question",
                name: "Comment sauvegarder Databasus lui-même ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Pour sauvegarder Databasus, allez dans /opt/databasus (ou le dossier où vous l'avez installé), puis dans le répertoire databasus-data. Vous devez sauvegarder le fichier secret.key (clé de chiffrement des identifiants) et le dossier /pgdata (base interne contenant les configurations et les métadonnées des backups). Il y a deux scénarios de récupération : 1) vous pouvez récupérer les backups de bases avec le seul secret.key, sans l'interface Databasus (voir le guide de récupération manuelle) ; 2) pour restaurer l'interface Databasus avec toutes les configurations et l'historique, il faut à la fois secret.key et le dossier /pgdata. Pour restaurer, recréez cette structure de dossiers sur un autre serveur.",
                },
              },
              {
                "@type": "Question",
                name: "Comment Databasus est-il soutenu par les programmes open source d'Anthropic et d'OpenAI ?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "En mars 2026, Databasus a été accepté à la fois dans Claude for Open Source d'Anthropic et Codex for Open Source d'OpenAI. Être soutenu par ces programmes est un signal de fiabilité : le projet a été évalué de manière indépendante et reconnu par des leaders du secteur comme une infrastructure open source critique méritant d'être soutenue.",
                },
              },
            ],
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
              <h1 id="faq">Questions fréquentes</h1>

              <p className="text-lg text-gray-400">
                Trouvez des réponses aux questions les plus courantes sur
                Databasus : installation, configuration et stratégies de
                sauvegarde.
              </p>

              <h2 id="why-no-raw-sql-dump">
                Pourquoi Databasus n&apos;utilise-t-il pas le format de dump SQL
                brut pour les sauvegardes logiques PostgreSQL ?
              </h2>

              <p>
                Pour les sauvegardes logiques, Databasus utilise le{" "}
                <strong>format custom</strong> de <code>pg_dump</code> avec{" "}
                <strong>compression zstd au niveau 5</strong> au lieu du format
                SQL brut, parce qu&apos;il offre l&apos;équilibre le plus
                efficace entre :
              </p>

              <ul>
                <li>La vitesse de création du backup</li>
                <li>La vitesse de restauration</li>
                <li>
                  La taille des fichiers (jusqu&apos;à 20 fois plus petits que
                  le format SQL brut)
                </li>
              </ul>

              <p>
                Cette décision a été prise après des tests et des benchmarks
                approfondis de différents formats de sauvegarde PostgreSQL et
                méthodes de compression. Vous pouvez en lire plus sur ces tests
                ici :{" "}
                <a
                  href="https://dev.to/rostislav_dugin/postgresql-backups-comparing-pgdump-speed-in-different-formats-and-with-different-compression-4pmd"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  PostgreSQL backups: comparing pg_dump speed in different
                  formats and with different compression
                </a>
                .
              </p>

              <p>
                Databasus n&apos;inclura pas le format de dump SQL brut, car :
              </p>

              <ul>
                <li>la variété superflue nuit à l&apos;expérience ;</li>
                <li>elle rend le code plus difficile à maintenir ;</li>
                <li>le format actuel convient à 99 % des cas</li>
              </ul>

              <h2 id="installation-directory">
                Où Databasus est-il installé via le script .sh ?
              </h2>

              <p>
                Databasus est installé dans le répertoire{" "}
                <code>/opt/databasus/</code>.
              </p>

              <h2 id="pitr">
                Comment fonctionnent les sauvegardes physiques et PITR
                (Point-in-Time Recovery) ?
              </h2>

              <p>
                Databasus exécute les sauvegardes physiques{" "}
                <strong>à distance, depuis son propre hôte</strong>, en se
                connectant à votre PostgreSQL via le{" "}
                <strong>protocole de réplication</strong> standard, donc rien
                n&apos;est à installer sur le serveur de base de données. Si la
                base se trouve dans un réseau fermé, Databasus peut
                l&apos;atteindre par un tunnel SSH vers un hôte interne ou un
                bastion, si bien que la base n&apos;a jamais besoin d&apos;être
                exposée publiquement.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>Pourquoi c&apos;est possible maintenant :</strong>{" "}
                    pendant des années, des outils comme pgBackRest et WAL-G ont
                    dû construire leurs propres moteurs de sauvegardes
                    incrémentales par blocs, faute de moteur natif dans
                    PostgreSQL. Cela a changé avec PostgreSQL 17, où la
                    fonctionnalité a été développée par{" "}
                    <strong>Robert Haas</strong> avec l&apos;aide de{" "}
                    <strong>David Steele</strong>, l&apos;auteur de pgBackRest.
                    PostgreSQL fournit désormais nativement des sauvegardes
                    incrémentales par blocs côté serveur (
                    <code>pg_basebackup --incremental</code> et{" "}
                    <code>summarize_wal</code>), et Databasus s&apos;appuie
                    dessus au lieu d&apos;en réinventer un.
                  </p>
                </div>
              </div>

              <p>
                <strong>Comment fonctionnent les sauvegardes :</strong>
              </p>

              <ul>
                <li>
                  Les sauvegardes complètes sont créées avec{" "}
                  <code>pg_basebackup</code>, streamées directement vers
                  Databasus
                </li>
                <li>
                  Les incrémentales par blocs utilisent{" "}
                  <code>pg_basebackup --incremental</code> : les résumés WAL
                  côté serveur de PostgreSQL 17 (<code>summarize_wal = on</code>
                  ) suivent les changements pour ne transférer que les blocs
                  modifiés
                </li>
                <li>
                  Les WAL sont streamés en continu via{" "}
                  <code>pg_receivewal</code> pour garder la chaîne de
                  récupération complète entre les sauvegardes
                </li>
                <li>
                  Les sauvegardes physiques exigent{" "}
                  <strong>PostgreSQL 17 ou plus récent</strong> ; sur les
                  versions antérieures, vous utilisez les sauvegardes logiques{" "}
                  <code>pg_dump</code>
                </li>
              </ul>

              <p>
                <strong>Comment fonctionne la restauration :</strong>
              </p>

              <ul>
                <li>
                  <code>pg_combinebackup</code> reconstruit un répertoire de
                  données exécutable à partir de la sauvegarde complète et de sa
                  chaîne d&apos;incrémentales
                </li>
                <li>
                  PostgreSQL rejoue ensuite les WAL jusqu&apos;à l&apos;instant
                  cible que vous choisissez, avec une restauration possible à
                  n&apos;importe quelle seconde entre deux sauvegardes
                </li>
                <li>
                  Une fois PostgreSQL démarré, il termine la récupération, se
                  promeut en primaire et reprend un fonctionnement normal
                </li>
              </ul>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>
                      Vous n&apos;avez pas à faire tout cela à la main.
                    </strong>{" "}
                    L&apos;interface Databasus vous donne des instructions pas à
                    pas pour restaurer vers un hôte ou une base Docker, soit
                    avec un script prêt à l&apos;emploi, soit en téléchargeant
                    les backups manuellement. Nous avons préparé le script pour
                    qu&apos;une restauration tienne en une seule commande, mais
                    vous pouvez aussi reconstruire vous-même la chaîne de
                    parties complètes, incrémentales et WAL si vous préférez.
                    Les incrémentales et les WAL sont d&apos;ailleurs optionnels
                    : vous pouvez ne faire qu&apos;une sauvegarde complète, sans
                    incrémentales, et les WAL ne sont pas obligatoires.
                  </p>
                </div>
              </div>

              <p>
                <strong>
                  Pourquoi nous utilisons les sauvegardes natives de PG 17 :
                </strong>
              </p>

              <ul>
                <li>
                  Elles réutilisent la mécanique de sauvegarde de PostgreSQL au
                  lieu de la réinventer, vous profitez donc d&apos;internes
                  éprouvés, avec des milliers de tests et de cas limites
                  derrière eux
                </li>
                <li>
                  Elles fonctionnent avec des bases distantes, y compris les
                  services managés comme Amazon RDS et Google Cloud SQL qui
                  exposent le protocole de réplication mais interdisent
                  d&apos;installer des logiciels sur l&apos;hôte
                </li>
                <li>
                  Elles offrent une perte de données quasi nulle, avec une
                  restauration possible à n&apos;importe quelle seconde entre
                  deux sauvegardes
                </li>
              </ul>

              <h2 id="why-no-agent">
                Pourquoi Databasus a-t-il abandonné les sauvegardes par agent ?
              </h2>

              <p>
                Une version antérieure de Databasus fournissait un{" "}
                <strong>agent</strong> de sauvegarde : un binaire qui tournait
                sur l&apos;hôte de la base pour streamer les WAL et créer des
                sauvegardes physiques en local. Cette première implémentation
                s&apos;est révélée être une erreur, et nous l&apos;avons
                supprimée. Les sauvegardes physiques s&apos;exécutent maintenant
                à distance depuis l&apos;hôte Databasus, comme décrit ci-dessus.
              </p>

              <p>
                <strong>
                  Pourquoi l&apos;agent était la mauvaise approche :
                </strong>
              </p>

              <ul>
                <li>
                  C&apos;était une implémentation naïve qui ne faisait que
                  copier les WAL par-dessus des sauvegardes complètes, avec un
                  RTO long à la clé
                </li>
                <li>
                  Les utilisateurs devaient configurer à la fois Databasus et un
                  agent séparé, alors que tout faire à distance depuis un seul
                  endroit est bien plus simple
                </li>
                <li>
                  Comme l&apos;agent vivait en dehors du système principal, il
                  était difficile de couvrir tous les cas de test
                </li>
                <li>
                  Il n&apos;y a en réalité qu&apos;un seul problème qu&apos;un
                  agent résout : atteindre une base inaccessible de
                  l&apos;extérieur. Pour 99 % des utilisateurs, c&apos;est déjà
                  réglé en exécutant Databasus dans le réseau privé ou en se
                  connectant via SSH ; l&apos;agent réinventait donc la roue et
                  rendait un problème simple bien plus compliqué que nécessaire
                </li>
                <li>
                  Il ne pouvait pas tourner sur les bases managées comme RDS et
                  Cloud SQL, qui interdisent les installations sur l&apos;hôte
                  mais exposent déjà le protocole de réplication : un chemin
                  distant était donc nécessaire de toute façon
                </li>
                <li>
                  Il s&apos;accompagnait aussi de nombreux cas limites. Les
                  connexions interrompues, la gestion des mises à jour de
                  l&apos;agent et la collecte des journaux d&apos;un processus
                  séparé étaient toutes pénibles, et moins un système a de
                  pièces mobiles, plus il est fiable au quotidien
                </li>
              </ul>

              <p>
                <strong>
                  Nous avons veillé à ce que les backups existants restent en
                  sécurité.
                </strong>{" "}
                Si vous mettez à jour depuis une version qui a encore des
                sauvegardes par agent, Databasus ne le fera pas silencieusement
                : il vous avertit du changement et vous laisse soit rester sur
                la <strong>version 3.42.0</strong> prise en charge, soit
                supprimer vous-même les anciennes sauvegardes par agent avant la
                mise à jour. L&apos;implémentation par agent reste disponible
                jusqu&apos;à la version 3.42.0 et continuera de fonctionner
                pendant longtemps, donc rien ne casse.
              </p>

              <h2 id="backup-databasus">
                Comment sauvegarder Databasus lui-même ?
              </h2>

              <p>
                Si vous voulez sauvegarder votre instance Databasus (avec toutes
                les configurations, bases de données et identifiants), suivez
                ces étapes :
              </p>

              <ol>
                <li>
                  Allez dans <code>/opt/databasus</code> (ou le dossier où vous
                  avez installé Databasus)
                </li>
                <li>
                  Rendez-vous dans le répertoire <code>databasus-data</code>
                </li>
              </ol>

              <p>
                <strong>Vous devez sauvegarder :</strong>
              </p>

              <ul>
                <li>
                  <code>secret.key</code> : la clé de chiffrement de vos
                  identifiants
                </li>
                <li>
                  <code>/pgdata</code> : la base PostgreSQL interne de Databasus
                  qui contient toutes vos configurations et les métadonnées des
                  backups
                </li>
              </ul>

              <p>
                Si vous utilisez le stockage local pour vos backups, vous pouvez
                aussi sauvegarder le dossier <code>backups</code>.
              </p>

              <p>
                <strong>Important :</strong> il existe deux scénarios de
                récupération différents :
              </p>

              <ul>
                <li>
                  <strong>
                    Récupérer les backups sans l&apos;interface Databasus :
                  </strong>{" "}
                  vous pouvez récupérer vos backups de bases avec le seul
                  fichier <code>secret.key</code>, sans avoir besoin de
                  Databasus ni de ses données internes. Consultez le{" "}
                  <a href="/fr/how-to-recover-without-databasus">
                    guide de récupération manuelle
                  </a>{" "}
                  pour des instructions détaillées.
                </li>
                <li>
                  <strong>
                    Restaurer l&apos;interface Databasus et toutes les
                    configurations :
                  </strong>{" "}
                  pour restaurer l&apos;interface Databasus avec toutes vos
                  configurations, sauvegardes planifiées et historique, vous
                  devez sauvegarder à la fois <code>secret.key</code> et le
                  dossier <code>/pgdata</code> (qui contient les métadonnées de
                  chiffrement et toutes les configurations Databasus).
                </li>
              </ul>

              <p>
                <strong>Pour restaurer Databasus sur un autre serveur :</strong>{" "}
                recréez simplement la structure du dossier{" "}
                <code>databasus-data</code> avec les fichiers sauvegardés et
                démarrez Databasus.
              </p>

              <h2 id="oss-programs">
                Comment Databasus est-il soutenu par les programmes open source
                d&apos;Anthropic et d&apos;OpenAI ?
              </h2>

              <p>
                En mars 2026, Databasus a été accepté à la fois dans{" "}
                <strong>
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Claude for Open Source
                  </a>
                </strong>{" "}
                d&apos;Anthropic et{" "}
                <strong>
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Codex for Open Source
                  </a>
                </strong>{" "}
                d&apos;OpenAI. Il est très précieux pour nous que le projet ait
                été reconnu comme un logiciel open source important pour le
                secteur par deux des plus grandes entreprises d&apos;IA au
                monde, d&apos;autant plus que les critères d&apos;éligibilité
                des deux programmes sont exigeants.
              </p>

              <p>
                Qu&apos;est-ce que cela signifie pour les utilisateurs ?
                C&apos;est une confirmation de fiabilité supplémentaire : le
                projet a été évalué de manière indépendante et reconnu par des
                leaders du secteur comme une infrastructure critique méritant
                d&apos;être soutenue. Nous en tirons une qualité de code encore
                plus élevée, des revues de sécurité plus rapides et un
                développement actif continu grâce à l&apos;accès illimité aux
                dernières IA.
              </p>

              <img
                src="/images/faq/anthropic-email.png"
                alt="Databasus accepté dans le programme Claude for Open Source d'Anthropic"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <img
                src="/images/faq/openai-email.png"
                alt="Databasus accepté dans le programme Codex for Open Source d'OpenAI"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
