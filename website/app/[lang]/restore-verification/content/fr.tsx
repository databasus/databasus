import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Vérification de restauration de sauvegarde - Documentation Databasus",
  description:
    "Prouvez que vos sauvegardes de bases de données sont réellement restaurables. Databasus récupère le dernier backup, le restaure dans un conteneur de base jetable, compare la base restaurée à la source et rapporte le nombre de lignes par table à chaque exécution.",
  keywords: [
    "vérification de restauration",
    "restauration de base de données",
    "vérification de sauvegarde",
    "reprise après sinistre",
    "test de sauvegarde de base de données",
    "agent de vérification Databasus",
    "intégrité des sauvegardes",
    "test de restauration automatisé",
  ],
  openGraph: {
    title:
      "Vérification de restauration de sauvegarde - Documentation Databasus",
    description:
      "Prouvez que vos sauvegardes de bases de données sont réellement restaurables. Databasus récupère le dernier backup, le restaure dans un conteneur de base jetable, compare la base restaurée à la source et rapporte le nombre de lignes par table à chaque exécution.",
    type: "article",
    url: getLocalizedUrl("fr", "restore-verification"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title:
      "Vérification de restauration de sauvegarde - Documentation Databasus",
    description:
      "Prouvez que vos sauvegardes de bases de données sont réellement restaurables. Databasus récupère le dernier backup, le restaure dans un conteneur de base jetable, compare la base restaurée à la source et rapporte le nombre de lignes par table à chaque exécution.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "restore-verification"),
    languages: getLanguageAlternates("restore-verification"),
  },
  robots: "index, follow",
};

export default function RestoreVerificationPage() {
  const downloadAgent = `curl -L -o verification-agent "https://your-databasus-host/api/v1/system/verification-agent?arch=amd64" \\
  && chmod +x verification-agent`;

  const startAgent = `./verification-agent start \\
  --databasus-host=https://your-databasus-host \\
  --agent-id=<AGENT_ID> \\
  --token=<TOKEN> \\
  --max-cpu=2 \\
  --max-ram-mb=2048 \\
  --max-disk-gb=20 \\
  --max-concurrent-jobs=1`;

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
              "Vérification de restauration de sauvegarde - Documentation Databasus",
            description:
              "Prouvez que vos sauvegardes de bases de données sont réellement restaurables. Databasus récupère le dernier backup, le restaure dans un conteneur de base jetable, compare la base restaurée à la source et rapporte le nombre de lignes par table à chaque exécution.",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Comment configurer la vérification de restauration dans Databasus",
            description:
              "Guide pas à pas pour enregistrer un agent de vérification, le lancer sur votre serveur et configurer des vérifications de restauration planifiées.",
            step: [
              {
                "@type": "HowToStep",
                name: "Créer un agent de vérification dans l'interface",
                text: "Allez dans Settings → Verification agents, cliquez sur Create verification agent, nommez-le, puis copiez le jeton et l'ID de l'agent depuis la boîte de dialogue.",
              },
              {
                "@type": "HowToStep",
                name: "Télécharger le binaire de l'agent",
                text: "Exécutez la commande curl sur l'hôte qui exécutera les vérifications, en choisissant amd64 ou arm64 selon votre architecture.",
              },
              {
                "@type": "HowToStep",
                name: "Lancer l'agent",
                text: "Démarrez l'agent avec --agent-id, --token et les budgets de ressources (--max-cpu, --max-ram-mb, --max-disk-gb, --max-concurrent-jobs).",
              },
              {
                "@type": "HowToStep",
                name: "Planifier les vérifications",
                text: "Ouvrez les paramètres de vérification de la base, activez Scheduled verification et choisissez un intervalle (After backup, Hourly, Daily, Weekly, Monthly ou Cron).",
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
              <h1 id="restore-verification">
                Vérification de restauration de sauvegarde
              </h1>

              <p className="text-lg text-gray-400">
                Un backup qui se termine sans erreur n&apos;est pas la même
                chose qu&apos;un backup que vous pouvez réellement restaurer. La
                seule vraie preuve, c&apos;est de le restaurer. Databasus le
                fait pour vous, selon un planning :
              </p>

              <ul>
                <li>il prend le dernier backup</li>
                <li>
                  il le restaure dans un conteneur de base de données jetable
                </li>
                <li>il compare la base restaurée à la source</li>
                <li>il détruit le conteneur</li>
                <li>il rapporte le résultat</li>
              </ul>

              <img
                src="/images/restore-verification/verified-backups.png"
                alt="Onglet des sauvegardes vérifiées"
                loading="lazy"
              />

              <img
                className="mt-5"
                src="/images/restore-verification/verifications.png"
                alt="Onglet des vérifications"
                loading="lazy"
              />

              <h2 id="what-is-verification-agent">
                Qu&apos;est-ce qu&apos;un agent de vérification ?
              </h2>

              <p>
                L&apos;agent de vérification est un petit binaire Go que vous
                lancez sur une machine que vous contrôlez : tout ce qui a du
                CPU, de la RAM et du disque disponibles convient. L&apos;agent
                s&apos;enregistre auprès de Databasus, récupère les tâches de
                vérification depuis une file d&apos;attente, les exécute
                localement et renvoie les résultats.
              </p>

              <h3 id="what-you-need">Ce dont vous avez besoin</h3>

              <ul>
                <li>
                  Un hôte avec un accès HTTPS sortant vers votre URL Databasus.
                </li>
                <li>
                  Docker disponible sur cet hôte : l&apos;agent démarre pour
                  chaque tâche des conteneurs de base de données éphémères de la
                  version majeure correspondante.
                </li>
                <li>
                  De la capacité disque pour chaque tâche de vérification
                  couvrant la <strong>taille du fichier de backup</strong>, la{" "}
                  <strong>taille brute de la base</strong>, une réserve et les{" "}
                  <strong>plus gros index</strong> construits en même temps.
                </li>
                <li>
                  Au moins 1 cœur CPU et 512 Mo de RAM disponibles par tâche
                  simultanée.
                </li>
              </ul>

              <h3 id="why-not-just-checksums">
                Pourquoi pas de simples checksums ?
              </h3>

              <p>
                Les checksums et les codes de sortie détectent certains modes de
                défaillance mais en ratent d&apos;autres complètement :
              </p>

              <ul>
                <li>
                  <strong>Les checksums</strong> détectent la corruption de bits
                  du fichier d&apos;archive, mais ne disent rien sur le fait que
                  le dump lui-même soit complet ou sémantiquement valide.
                </li>
                <li>
                  <strong>Le code de sortie du dump</strong> dit que la commande
                  de dump s&apos;est exécutée. Il ne détecte pas un rôle sans
                  permission de lecture sur certains objets, une extension
                  manquante sur la source ou une incohérence de tablespace,
                  autant de causes d&apos;objets silencieusement ignorés ou
                  tronqués.
                </li>
                <li>
                  <strong>La vérification de restauration</strong> fait
                  réellement passer l&apos;archive par l&apos;outil de
                  restauration natif de la base et compte les lignes par table.
                  C&apos;est le seul contrôle qui détecte tout ce qui précède :
                  si un backup ne se restaure pas, vous le découvrez avant
                  d&apos;en avoir besoin, pas pendant un sinistre.
                </li>
              </ul>

              <h2 id="configuration">Configuration</h2>

              <h3 id="create-on-ui">Créer un agent dans l&apos;interface</h3>

              <p>
                Ouvrez <strong>Settings → Verification agents</strong> et
                cliquez sur <strong>Create verification agent</strong>.
                Choisissez un nom parlant comme <code>staging-verifier</code> ou{" "}
                <code>eu-west-host-1</code>. La boîte de dialogue suivante
                affiche le <strong>jeton</strong> et l&apos;<strong>ID</strong>{" "}
                de l&apos;agent.
              </p>

              <p>
                Le jeton n&apos;est affiché qu&apos;
                <strong>une seule fois</strong> : copiez-le avant de fermer la
                boîte de dialogue. Si vous le perdez plus tard, utilisez
                l&apos;action <strong>Rotate token</strong> sur la ligne de
                l&apos;agent pour en émettre un nouveau ; l&apos;ancien jeton
                cesse de fonctionner au heartbeat suivant de l&apos;agent. La
                boîte de dialogue qui suit montre les commandes
                d&apos;installation pour l&apos;architecture de votre serveur,
                les mêmes commandes que celles décrites ci-dessous.
              </p>

              <h3 id="launch">Lancer l&apos;agent sur votre serveur</h3>

              <p>
                Connectez-vous en SSH à la machine qui exécutera les
                vérifications. Téléchargez d&apos;abord le binaire de
                l&apos;agent. Remplacez <code>https://your-databasus-host</code>{" "}
                par votre propre URL Databasus, et remplacez <code>amd64</code>{" "}
                par <code>arm64</code> si votre serveur est en ARM :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={downloadAgent} />
                </div>
              </div>

              <p>
                Puis lancez l&apos;agent. L&apos;ID de l&apos;agent et le jeton
                proviennent de la boîte de dialogue de l&apos;étape précédente :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={startAgent} />
                </div>
              </div>

              <p>
                <code>start</code> lance l&apos;agent en démon et écrit ses
                options dans <code>databasus-verification.json</code> dans le
                répertoire de travail, si bien que les redémarrages suivants
                peuvent utiliser <code>./verification-agent start</code> sans
                aucune option. Les logs sont écrits dans{" "}
                <code>databasus-verification.log</code> à côté du binaire.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] mb-3 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    L&apos;hôte Databasus doit être en <code>https://</code>. Le
                    HTTP simple n&apos;est autorisé qu&apos;avec{" "}
                    <code>--allow-insecure-http</code>, et il est destiné aux
                    tests locaux. N&apos;exposez jamais un agent de production
                    en HTTP non chiffré.
                  </p>
                </div>
              </div>

              <p>
                Les quatre options <code>--max-*</code> sont des{" "}
                <strong>budgets</strong>, pas des allocations par tâche.
                L&apos;agent les rapporte à Databasus à chaque heartbeat, et
                Databasus les répartit entre les tâches simultanées que vous
                autorisez. Avec{" "}
                <code>
                  --max-cpu=2 --max-ram-mb=2048 --max-concurrent-jobs=1
                </code>{" "}
                l&apos;unique tâche obtient les 2 CPU et les 2 Go de RAM. Avec{" "}
                <code>--max-concurrent-jobs=2</code>, chaque tâche obtient 1 CPU
                et 1 Go. Le plancher est de 1 CPU et 512 Mo par tâche : si votre
                budget ne peut pas le satisfaire, l&apos;agent annonce une
                concurrence plus basse. Le budget disque est le plus facile à
                rater : chaque tâche a besoin d&apos;espace pour la{" "}
                <strong>taille du fichier de backup</strong>, la{" "}
                <strong>taille brute de la base</strong>, une{" "}
                <strong>réserve d&apos;environ 1,5 Go</strong> et ses{" "}
                <strong>plus gros index</strong>, un par tâche de restauration
                parallèle, car chaque index en cours de construction occupe sur le
                disque un tri d&apos;à peu près sa propre taille. Databasus réduit
                le nombre de tâches de restauration parallèles jusqu&apos;à ce que
                le tout tienne dans le disque libre de l&apos;agent. Un{" "}
                <code>--max-cpu</code> plus élevé demande donc plus de disque, et
                un <code>--max-disk-gb</code> plus grand permet de restaurer plus
                vite une grosse base.
              </p>

              <h3 id="manage">Gérer l&apos;agent</h3>

              <p>Le même binaire fournit quatre sous-commandes :</p>

              <ul>
                <li>
                  <code>./verification-agent status</code> — indique si le démon
                  tourne et sur quelles tâches il travaille actuellement.
                </li>
                <li>
                  <code>./verification-agent stop</code> — arrête le démon. Les
                  vérifications en cours sont rapportées à Databasus comme
                  échouées et remises en file d&apos;attente.
                </li>
                <li>
                  <code>./verification-agent start</code> — relance le démon.
                  Les options sont mémorisées depuis le premier lancement ;
                  passez <code>--token=&lt;NEW&gt;</code> après une rotation
                  pour mettre à jour le jeton enregistré.
                </li>
                <li>
                  <code>./verification-agent run</code> — exécute au premier
                  plan plutôt qu&apos;en démon. Utilisez cette commande pour
                  encapsuler l&apos;agent dans une unité systemd ou un conteneur
                  Docker : ces superviseurs s&apos;attendent à ce que le
                  processus reste au premier plan.
                </li>
              </ul>

              <p>
                La page Settings affiche trois actions sur chaque ligne
                d&apos;agent : revoir les commandes d&apos;installation (sans
                révéler le jeton), effectuer une rotation du jeton et supprimer
                l&apos;agent. La suppression est sûre : les vérifications
                actuellement assignées à cet agent retournent dans la file et
                sont reprises par un autre agent s&apos;il y en a un de
                disponible.
              </p>

              <h2 id="schedules-and-notifications">
                Planification et notifications
              </h2>

              <p>
                La vérification de restauration se configure par base de
                données. Ouvrez les paramètres de vérification de la base,
                activez <strong>Scheduled verification</strong>, puis choisissez
                un intervalle.
              </p>

              <h3 id="interval-options">Options d&apos;intervalle</h3>

              <ul>
                <li>
                  <strong>After backup</strong> — la garantie la plus forte :
                  chaque backup réussi est vérifié dès qu&apos;il se termine.
                </li>
                <li>
                  <strong>Hourly, daily, weekly, monthly</strong> — choisissez
                  une cadence et une heure de la journée.
                </li>
                <li>
                  <strong>Cron</strong> — une expression cron en UTC pour tout
                  ce que les préréglages ne couvrent pas. Exemples :{" "}
                  <code>0 4 * * 0</code> (chaque dimanche à 4 h 00 UTC) et{" "}
                  <code>0 */6 * * *</code> (toutes les six heures).
                </li>
              </ul>

              <h3 id="how-the-queue-works">
                Comment la file gère &quot;After backup&quot;
              </h3>

              <p>
                Une vérification est généralement plus lente que le backup qui
                l&apos;a produite, donc si les backups arrivent plus vite que
                les vérifications ne se terminent, la file grandirait sans fin.
                Databasus l&apos;évite en{" "}
                <strong>
                  annulant toute vérification en attente pour la même base dès
                  qu&apos;un nouveau backup arrive
                </strong>{" "}
                : seul le backup le plus récent attend son tour. Ce compromis
                est voulu : mieux vaut sauter la vérification d&apos;un backup
                obsolète que passer des heures à vérifier quelque chose que vous
                ne restaureriez de toute façon jamais.
              </p>

              <h3 id="manual-runs">Exécutions manuelles</h3>

              <p>
                Vous pouvez aussi déclencher une vérification ponctuelle depuis
                l&apos;onglet <strong>Restore verifications</strong> de la base,
                sans modifier la planification. Utile pour contrôler un backup
                précis ou tester un nouvel agent de bout en bout avant de lui
                confier la charge planifiée.
              </p>

              <h3 id="notifications">Notifications</h3>

              <p>
                Les succès et les échecs peuvent être envoyés via n&apos;importe
                quel notificateur déjà relié à la base. Les deux cases à cocher,{" "}
                <strong>Verification success</strong> et{" "}
                <strong>Verification failed</strong>, sont indépendantes. La
                plupart des équipes n&apos;activent que celle des échecs pour ne
                pas crouler sous les notifications. Consultez la{" "}
                <a
                  href="/fr/notifiers"
                  className="text-blue-400 hover:text-blue-300"
                >
                  documentation des notificateurs
                </a>{" "}
                pour connecter Slack, Microsoft Teams, Discord, l&apos;email et
                d&apos;autres.
              </p>

              <h3 id="results">Lire les résultats</h3>

              <p>
                Chaque tentative de vérification apparaît comme une ligne dans
                l&apos;onglet <strong>Restore verifications</strong> de la base.
                Le statut est <strong>Pending</strong>, <strong>Running</strong>
                , <strong>Successful</strong>, <strong>Failed</strong> ou{" "}
                <strong>Canceled</strong>. Cliquer sur une ligne ouvre un
                panneau avec la chronologie complète, le code de sortie de la
                restauration, la taille de la base restaurée, le nombre de
                schémas et de tables, et le détail du nombre de lignes par
                table. Les exécutions échouées affichent le message
                d&apos;erreur en haut du panneau.
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
