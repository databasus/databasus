import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Databasus vs PgBackWeb : comparaison d'outils de sauvegarde PostgreSQL",
  description:
    "Comparez les outils de sauvegarde PostgreSQL Databasus et PgBackWeb : fonctionnalités, sécurité, travail en équipe, options de stockage, notifications et facilité d'utilisation.",
  keywords: [
    "Databasus vs PgBackWeb",
    "comparaison sauvegarde PostgreSQL",
    "alternative à PgBackWeb",
    "outils de sauvegarde PostgreSQL",
    "comparaison de sauvegarde de bases de données",
    "interface graphique pg_dump",
    "backup auto-hébergé",
    "sécurité des sauvegardes PostgreSQL",
  ],
  openGraph: {
    title:
      "Databasus vs PgBackWeb : comparaison d'outils de sauvegarde PostgreSQL",
    description:
      "Comparez les outils de sauvegarde PostgreSQL Databasus et PgBackWeb : fonctionnalités, sécurité, travail en équipe, options de stockage, notifications et facilité d'utilisation.",
    type: "article",
    url: getLocalizedUrl("fr", "databasus-vs-pgbackweb"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs PgBackWeb : comparaison d'outils de sauvegarde PostgreSQL",
    description:
      "Comparez les outils de sauvegarde PostgreSQL Databasus et PgBackWeb : fonctionnalités, sécurité, travail en équipe, options de stockage, notifications et facilité d'utilisation.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "databasus-vs-pgbackweb"),
    languages: getLanguageAlternates("databasus-vs-pgbackweb"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackWebPage() {
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
              "Databasus vs PgBackWeb : comparaison d'outils de sauvegarde PostgreSQL",
            description:
              "Une comparaison détaillée des outils de sauvegarde PostgreSQL Databasus et PgBackWeb : fonctionnalités, sécurité, travail en équipe, options de stockage et facilité d'utilisation.",
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
              <h1 id="databasus-vs-pgbackweb">Databasus vs PgBackWeb</h1>

              <p className="text-lg text-gray-400">
                Databasus et PgBackWeb sont deux outils open source conçus pour
                simplifier la gestion des sauvegardes PostgreSQL via une
                interface web. S&apos;ils partagent le même objectif, rendre les
                backups plus accessibles, ils diffèrent nettement en matière de
                fonctionnalités, de sécurité, de travail en équipe et de
                facilité d&apos;utilisation.
              </p>

              <h2 id="quick-comparison">Comparaison rapide</h2>

              <p>
                Voici un aperçu rapide des principales différences entre
                Databasus et PgBackWeb :
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Fonctionnalité</th>
                    <th>Databasus</th>
                    <th>PgBackWeb</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Licence</td>
                    <td data-label="Databasus">Apache 2.0</td>
                    <td data-label="PgBackWeb">AGPL-3.0</td>
                  </tr>
                  <tr>
                    <td>Gestion des sauvegardes</td>
                    <td data-label="Databasus">✅ Plusieurs bases</td>
                    <td data-label="PgBackWeb">✅ Plusieurs bases</td>
                  </tr>
                  <tr>
                    <td>Support d&apos;autres bases de données</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="PgBackWeb">❌ PostgreSQL uniquement</td>
                  </tr>
                  <tr>
                    <td>Options de stockage</td>
                    <td data-label="Databasus">
                      Local, S3, Cloudflare R2, Azure, NAS,
                      FTP
                    </td>
                    <td data-label="PgBackWeb">
                      Local et compatible S3 uniquement
                    </td>
                  </tr>
                  <tr>
                    <td>Notifications</td>
                    <td data-label="Databasus">
                      Slack, Discord, Telegram, Teams, e-mail, webhooks
                    </td>
                    <td data-label="PgBackWeb">Webhooks uniquement</td>
                  </tr>
                  <tr>
                    <td>Sécurité</td>
                    <td data-label="Databasus">
                      ✅ AES-256-GCM, clés uniques par backup, lecture seule
                      imposée
                    </td>
                    <td data-label="PgBackWeb">✅ Chiffrement PGP</td>
                  </tr>
                  <tr>
                    <td>Fonctionnalités d&apos;équipe</td>
                    <td data-label="Databasus">
                      ✅ Espaces de travail, accès par rôles, journaux
                      d&apos;audit
                    </td>
                    <td data-label="PgBackWeb">❌ Non disponible</td>
                  </tr>
                  <tr>
                    <td>Supervision de la santé</td>
                    <td data-label="Databasus">✅ Intégrée</td>
                    <td data-label="PgBackWeb">❌ Non disponible</td>
                  </tr>
                  <tr>
                    <td>Installation</td>
                    <td data-label="Databasus">
                      Script en une ligne, Docker ou Helm
                    </td>
                    <td data-label="PgBackWeb">
                      Configuration Docker manuelle
                    </td>
                  </tr>
                  <tr>
                    <td>Sauvegardes physiques</td>
                    <td data-label="Databasus">✅ Oui</td>
                    <td data-label="PgBackWeb">❌ Non disponible</td>
                  </tr>
                  <tr>
                    <td>Sauvegardes incrémentales</td>
                    <td data-label="Databasus">✅ Au niveau bloc (PG 17+)</td>
                    <td data-label="PgBackWeb">❌ Non disponible</td>
                  </tr>
                  <tr>
                    <td>Archivage WAL</td>
                    <td data-label="Databasus">✅ Streaming continu</td>
                    <td data-label="PgBackWeb">❌ Non disponible</td>
                  </tr>
                  <tr>
                    <td>Récupération à un instant donné (PITR)</td>
                    <td data-label="Databasus">✅ Oui</td>
                    <td data-label="PgBackWeb">❌ Non disponible</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="backup-features">Fonctionnalités de sauvegarde</h2>

              <p>
                Les deux outils prennent en charge les sauvegardes planifiées
                avec des horaires flexibles :
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong> : prend en charge des
                  planifications horaires, quotidiennes, hebdomadaires,
                  mensuelles ou cron, avec un horaire précis (par exemple 4 h du
                  matin). Il applique une{" "}
                  <strong>compression équilibrée avec zstd (niveau 5)</strong> :
                  les backups en ressortent 4 à 8 fois plus petits, pour
                  seulement ~20 % de temps d&apos;exécution en plus. C&apos;est
                  nettement plus efficace que gzip.
                </li>
                <li>
                  <strong>PgBackWeb</strong> : prend en charge la planification
                  cron pour l&apos;exécution des sauvegardes. Utilise la
                  compression gzip, plus lente et moins efficace que zstd.
                </li>
              </ul>

              <p>
                Au-delà des sauvegardes logiques, Databasus prend aussi en
                charge les sauvegardes physiques, incrémentales et WAL. Elles
                reposent sur la pile de sauvegarde native de PostgreSQL 17 et
                s&apos;exécutent à distance : rien n&apos;est installé sur le
                serveur de base de données et les réseaux fermés sont
                accessibles via un tunnel SSH. Vous obtenez ainsi des backups
                incrémentaux au niveau bloc, un streaming WAL continu et la
                récupération à un instant donné pour une reprise après sinistre
                avec une perte de données quasi nulle, en restaurant à la
                seconde près entre deux sauvegardes. PgBackWeb n&apos;offre rien
                de tout cela.
              </p>

              <h2 id="storage-options">Options de stockage</h2>

              <p>
                La flexibilité du stockage est essentielle pour une stratégie de
                sauvegarde. Voici comment les deux outils se comparent :
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong> : prend en charge un large éventail
                  de destinations de stockage :
                  <ul>
                    <li>Stockage local</li>
                    <li>Amazon S3 et services compatibles S3</li>
                    <li>Cloudflare R2</li>
                    <li>Azure Blob Storage</li>
                    <li>NAS (stockage en réseau)</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong> : limité au stockage local et au
                  stockage compatible S3 uniquement.
                </li>
              </ul>

              <p>
                <a
                  href="/fr/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Voir toutes les options de stockage de Databasus →
                </a>
              </p>

              <h2 id="security">Sécurité</h2>

              <p>
                La sécurité est un aspect critique de la gestion des
                sauvegardes. Databasus met en œuvre une sécurité de niveau
                entreprise sur trois niveaux :
              </p>

              <h3 id="security-databasus">Modèle de sécurité de Databasus</h3>

              <ol>
                <li>
                  <strong>Chiffrement des données sensibles</strong> : tous les
                  mots de passe, jetons et identifiants sont chiffrés en
                  AES-256-GCM. La clé de chiffrement est stockée séparément de
                  la base de données : même si la base est compromise, les
                  données sensibles restent protégées.
                </li>
                <li>
                  <strong>Chiffrement des sauvegardes</strong> : chaque fichier
                  de backup est chiffré avec une clé unique dérivée de la clé
                  maîtresse, de l&apos;identifiant de la sauvegarde et d&apos;un
                  sel aléatoire. Même si quelqu&apos;un accède à votre stockage
                  cloud, il ne peut pas lire les sauvegardes sans votre clé de
                  chiffrement.
                </li>
                <li>
                  <strong>Accès en lecture seule à la base</strong> : Databasus
                  impose l&apos;accès en lecture seule en vérifiant les
                  permissions au niveau du rôle, de la base et des tables. Il
                  n&apos;exige que des permissions SELECT et vous avertit si des
                  privilèges d&apos;écriture sont détectés. Cela empêche toute
                  corruption des données même si Databasus est compromis.
                </li>
              </ol>

              <h3 id="security-pgbackweb">Modèle de sécurité de PgBackWeb</h3>

              <ul>
                <li>
                  <strong>Chiffrement PGP</strong> : PgBackWeb propose le
                  chiffrement PGP des fichiers de sauvegarde.
                </li>
                <li>
                  <strong>Pas de lecture seule imposée</strong> : PgBackWeb
                  n&apos;impose ni ne vérifie l&apos;accès en lecture seule à la
                  base : les backups peuvent donc être créés avec des
                  utilisateurs disposant de droits d&apos;écriture.
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

              <h2 id="notifications">Notifications</h2>

              <p>
                Être informé de l&apos;état des sauvegardes est essentiel pour
                une exploitation fiable :
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong> : fournit des notifications en
                  temps réel via plusieurs canaux :
                  <ul>
                    <li>Slack</li>
                    <li>Discord</li>
                    <li>Telegram</li>
                    <li>Microsoft Teams</li>
                    <li>E-mail</li>
                    <li>Webhooks</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong> : ne prend en charge que les
                  webhooks. Pour recevoir des alertes via Slack, Telegram ou
                  d&apos;autres plateformes, vous devez mettre en place des
                  services ou intermédiaires supplémentaires.
                </li>
              </ul>

              <p>
                <a
                  href="/fr/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Voir tous les canaux de notification de Databasus →
                </a>
              </p>

              <h2 id="team-features">Fonctionnalités d&apos;équipe</h2>

              <p>
                Pour les organisations et les équipes DevOps, les
                fonctionnalités de collaboration sont essentielles. C&apos;est
                là que Databasus surpasse nettement PgBackWeb :
              </p>

              <h3 id="team-databasus">Capacités d&apos;équipe de Databasus</h3>

              <ul>
                <li>
                  <strong>Espaces de travail</strong> : regroupez bases de
                  données, notificateurs et stockages par projet ou par équipe.
                  Les utilisateurs ne voient que les espaces auxquels ils sont
                  invités.
                </li>
                <li>
                  <strong>Contrôle d&apos;accès par rôles</strong> : des niveaux
                  de permission pour contrôler ce que chaque membre peut faire
                  dans les espaces de travail.
                </li>
                <li>
                  <strong>Journaux d&apos;audit</strong> : tracez toutes les
                  activités du système et les modifications faites par les
                  utilisateurs. Essentiel pour la conformité et la
                  responsabilisation de l&apos;équipe.
                </li>
              </ul>

              <h3 id="team-pgbackweb">Capacités d&apos;équipe de PgBackWeb</h3>

              <p>
                PgBackWeb n&apos;a ni gestion des utilisateurs, ni espaces de
                travail, ni journaux d&apos;audit intégrés. Il est
                principalement conçu pour un usage mono-utilisateur.
              </p>

              <p>
                <a
                  href="/fr/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  En savoir plus sur la gestion des accès dans Databasus →
                </a>
              </p>

              <h2 id="ease-of-use">Facilité d&apos;utilisation</h2>

              <p>
                <strong>
                  Databasus est conçu pour être nettement plus simple à utiliser
                </strong>{" "}
                que PgBackWeb, avec un accent mis sur une expérience intuitive
                et un temps de mise en route minimal :
              </p>

              <h3 id="ease-databasus">Expérience utilisateur de Databasus</h3>

              <ul>
                <li>
                  <strong>Installation facile</strong> : utilisez Docker
                  directement ou lancez un script en une ligne qui installe
                  Docker (si nécessaire), met en place Databasus et configure le
                  démarrage automatique. Durée totale : ~2 minutes.
                </li>
                <li>
                  <strong>Interface web intuitive</strong> : une interface
                  soignée qui vous guide pas à pas dans la configuration des
                  sauvegardes. Aucune expertise PostgreSQL requise.
                </li>
                <li>
                  <strong>Thèmes sombre et clair</strong> : choisissez
                  l&apos;apparence qui convient à votre façon de travailler.
                </li>
                <li>
                  <strong>Adapté au mobile</strong> : consultez vos sauvegardes
                  où que vous soyez, sur n&apos;importe quel appareil.
                </li>
                <li>
                  <strong>Supervision de la santé intégrée</strong> : contrôles
                  de santé configurables avec graphiques de disponibilité.
                </li>
                <li>
                  <strong>Restauration en un clic</strong> : téléchargez et
                  restaurez n&apos;importe quelle sauvegarde en un seul clic.
                </li>
              </ul>

              <h3 id="ease-pgbackweb">Expérience utilisateur de PgBackWeb</h3>

              <ul>
                <li>
                  <strong>Configuration Docker manuelle</strong> : nécessite de
                  configurer des variables d&apos;environnement et de mettre en
                  place une base PostgreSQL externe pour stocker la
                  configuration.
                </li>
                <li>
                  <strong>Interface web basique</strong> : fonctionnelle mais
                  moins aboutie que celle de Databasus. Thème sombre disponible.
                </li>
                <li>
                  <strong>Pas de supervision de la santé</strong> : la
                  surveillance de la disponibilité des bases doit être mise en
                  place séparément.
                </li>
              </ul>

              <h2 id="installation">Installation et déploiement</h2>

              <h3 id="install-databasus">Installer Databasus</h3>

              <p>
                Databasus propose trois méthodes d&apos;installation, le script
                automatisé étant la plus rapide :
              </p>

              <ul>
                <li>
                  <strong>Script automatisé (recommandé)</strong> : une commande
                  cURL en une ligne qui installe Docker, met en place Databasus
                  et configure le démarrage automatique.
                </li>
                <li>
                  <strong>Docker run</strong> : une seule commande pour lancer
                  Databasus avec PostgreSQL embarqué.
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
                  Consulter le guide d&apos;installation de Databasus →
                </a>
              </p>

              <h3 id="install-pgbackweb">Installer PgBackWeb</h3>

              <p>
                PgBackWeb exige Docker et la configuration manuelle de variables
                d&apos;environnement. Vous devez aussi mettre en place une base
                PostgreSQL externe pour stocker la configuration de PgBackWeb.
              </p>

              <h2 id="licensing">Licences</h2>

              <p>
                Le modèle de licence peut fortement influencer la façon dont
                vous pouvez utiliser et modifier le logiciel :
              </p>

              <ul>
                <li>
                  <strong>Databasus (Apache 2.0)</strong> : licence permissive
                  qui autorise sans restriction l&apos;usage commercial, la
                  modification et la distribution. Vous pouvez utiliser
                  Databasus dans des projets propriétaires sans souci de
                  licence.
                </li>
                <li>
                  <strong>PgBackWeb (AGPL-3.0)</strong> : licence copyleft qui
                  exige que toute œuvre dérivée ou modification soit également
                  open source sous AGPL-3.0. Si vous modifiez PgBackWeb et le
                  proposez comme service, vous devez publier vos modifications.
                </li>
              </ul>

              <h2 id="conclusion">Conclusion</h2>

              <p>
                Databasus et PgBackWeb sont deux outils de sauvegarde PostgreSQL
                capables, mais ils répondent à des besoins différents :
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Choisissez Databasus si vous avez besoin de :
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Sécurité de niveau entreprise avec une protection à 3
                    niveaux
                  </li>
                  <li>
                    Collaboration en équipe avec espaces de travail et journaux
                    d&apos;audit
                  </li>
                  <li>
                    Plusieurs destinations de stockage (FTP, SFTP, Azure,
                    etc.)
                  </li>
                  <li>
                    Notifications intégrées vers Slack, Teams, Telegram, etc.
                  </li>
                  <li>
                    Installation rapide via un script en une ligne ou Docker
                  </li>
                  <li>
                    Interface moderne et intuitive avec une courbe
                    d&apos;apprentissage minimale
                  </li>
                  <li>
                    Licence permissive Apache 2.0 pour un usage commercial
                  </li>
                  <li>
                    Sauvegardes physiques, incrémentales, archivage WAL et PITR
                    pour la reprise après sinistre
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>Choisissez PgBackWeb si :</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Il vous faut une solution de backup simple pour un seul
                    utilisateur
                  </li>
                  <li>Le stockage local ou compatible S3 vous suffit</li>
                  <li>Des notifications par webhooks suffisent</li>
                  <li>La licence AGPL-3.0 convient à votre cas d&apos;usage</li>
                </ul>
              </div>

              <p>
                Pour la plupart des utilisateurs, en particulier les équipes et
                organisations qui exigent une sécurité solide, plusieurs options
                de stockage et des canaux de notification complets,{" "}
                <strong>Databasus est le choix recommandé</strong>.
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
