import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Installation - Documentation Databasus",
  description:
    "Découvrez comment installer Databasus : script automatisé, Docker run, Docker Compose, Helm pour Kubernetes ou reverse proxy Caddy. Installation simple, sans configuration, de votre système auto-hébergé de sauvegarde PostgreSQL.",
  keywords: [
    "installation Databasus",
    "installation Docker",
    "configuration sauvegarde PostgreSQL",
    "backup auto-hébergé",
    "Docker Compose",
    "installation sauvegarde base de données",
    "configuration pg_dump",
    "Kubernetes",
    "chart Helm",
    "déploiement K8s",
    "reverse proxy Caddy",
    "configuration HTTPS",
    "health check",
    "monitoring",
    "liveness probe",
  ],
  openGraph: {
    title: "Installation - Documentation Databasus",
    description:
      "Découvrez comment installer Databasus : script automatisé, Docker run, Docker Compose, Helm pour Kubernetes ou reverse proxy Caddy. Installation simple, sans configuration, de votre système auto-hébergé de sauvegarde PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("fr", "installation"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Installation - Documentation Databasus",
    description:
      "Découvrez comment installer Databasus : script automatisé, Docker run, Docker Compose, Helm pour Kubernetes ou reverse proxy Caddy. Installation simple, sans configuration, de votre système auto-hébergé de sauvegarde PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "installation"),
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
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Installation - Documentation Databasus",
            description:
              "Découvrez comment installer Databasus : script automatisé, Docker run, Docker Compose, Helm pour Kubernetes ou reverse proxy Caddy. Installation simple, sans configuration, de votre système auto-hébergé de sauvegarde PostgreSQL.",
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
            name: "Comment installer Databasus",
            description:
              "Guide pas à pas pour installer Databasus, l'outil de sauvegarde PostgreSQL",
            step: [
              {
                "@type": "HowToStep",
                name: "Script d'installation automatisé",
                text: "Exécutez le script d'installation automatisé : il installe Docker et configure Databasus avec le démarrage automatique.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Exécutez la commande curl pour télécharger et lancer le script d'installation",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Docker Run",
                text: "Utilisez la commande docker run pour démarrer rapidement le conteneur Databasus avec persistance des données.",
              },
              {
                "@type": "HowToStep",
                name: "Docker Compose",
                text: "Créez un fichier docker-compose.yml et gérez le déploiement avec Docker Compose.",
              },
              {
                "@type": "HowToStep",
                name: "Kubernetes avec Helm",
                text: "Utilisez le chart Helm officiel pour déployer Databasus sur Kubernetes avec StatefulSet, stockage persistant et ingress optionnel.",
              },
              {
                "@type": "HowToStep",
                name: "Exécution derrière le reverse proxy Caddy",
                text: "Utilisez Docker Compose avec Caddy pour les déploiements en production avec des certificats HTTPS automatiques.",
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
              <h1 id="installation">Installation</h1>

              <p className="text-lg text-gray-400">
                Vous pouvez installer Databasus de plusieurs façons : script
                automatisé (recommandé), simple docker run, Docker Compose, Helm
                pour Kubernetes ou Docker Compose avec Caddy pour les
                déploiements en production.
              </p>

              <h2 id="system-requirements">Configuration requise</h2>

              <p>
                Databasus a besoin des ressources système minimales suivantes
                pour fonctionner correctement :
              </p>

              <ul>
                <li>
                  <strong>CPU</strong> : au moins 1 cœur
                </li>
                <li>
                  <strong>RAM</strong> : minimum 500 Mo
                </li>
                <li>
                  <strong>Stockage</strong> : 5 Go pour l&apos;installation,
                  plus l&apos;espace nécessaire pour vos backups
                </li>
                <li>
                  <strong>Docker</strong> : Docker Engine 20.10+ et Docker
                  Compose v2.0+
                </li>
              </ul>

              <h2 id="option-1-automated-script">
                Option 1 : script d&apos;installation (recommandé, Linux
                uniquement)
              </h2>

              <p>Le script d&apos;installation va :</p>

              <ul>
                <li>
                  ✅ Installer Docker avec Docker Compose (s&apos;ils ne sont
                  pas déjà installés)
                </li>
                <li>✅ Configurer Databasus</li>
                <li>
                  ✅ Activer le démarrage automatique au redémarrage du système
                </li>
              </ul>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{installScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={installScript} />
                </div>
              </div>

              <p>
                Dans ce cas, Databasus sera installé dans le répertoire{" "}
                <code>/opt/databasus</code>.
              </p>

              <h2 id="option-2-docker-run">Option 2 : simple docker run</h2>

              <p>La façon la plus simple de lancer Databasus :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRun}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={dockerRun} />
                </div>
              </div>

              <p>Cette seule commande va :</p>

              <ul>
                <li>✅ Démarrer Databasus</li>
                <li>
                  ✅ Stocker toutes les données dans le répertoire{" "}
                  <code>./databasus-data</code>
                </li>
                <li>
                  ✅ Redémarrer automatiquement après un redémarrage du système
                </li>
              </ul>

              <h2 id="option-3-docker-compose">
                Option 3 : configuration Docker Compose
              </h2>

              <p>
                Créez un fichier <code>docker-compose.yml</code> avec la
                configuration suivante :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerCompose}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={dockerCompose} />
                </div>
              </div>

              <p>Puis exécutez :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text="docker compose up -d" />
                </div>
              </div>

              <p>
                Gardez à l&apos;esprit que le démarrage peut prendre
                jusqu&apos;à ~2 minutes.
              </p>

              <h2 id="option-4-helm">Option 4 : Kubernetes avec Helm</h2>

              <p>
                Pour les déploiements Kubernetes, installez directement depuis
                le registre OCI. Choisissez la méthode d&apos;accès adaptée à
                votre environnement.
              </p>

              <h3 id="helm-clusterip">
                Avec ClusterIP + port-forward (développement)
              </h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallClusterIP}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={helmInstallClusterIP} />
                </div>
              </div>

              <p>Accès via port-forward :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmPortForward}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={helmPortForward} />
                </div>
              </div>

              <h3 id="helm-loadbalancer">
                Avec LoadBalancer (environnements cloud)
              </h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallLoadBalancer}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={helmInstallLoadBalancer} />
                </div>
              </div>

              <p>Récupérez l&apos;IP externe et accédez à Databasus :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmGetSvc}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={helmGetSvc} />
                </div>
              </div>

              <h3 id="helm-ingress">Avec Ingress (accès par nom de domaine)</h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallIngress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={helmInstallIngress} />
                </div>
              </div>

              <p>
                Pour les autres options (NodePort, TLS, HTTPRoute pour Gateway
                API), consultez la{" "}
                <a
                  href="https://github.com/databasus/databasus/tree/main/deploy/helm"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  documentation du chart Helm
                </a>
                .
              </p>

              <h2 id="caddy-reverse-proxy">
                Exécution derrière le reverse proxy Caddy
              </h2>

              <p>
                Pour les déploiements en production, vous pouvez utiliser{" "}
                <a
                  href="https://caddyserver.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Caddy
                </a>{" "}
                comme reverse proxy pour obtenir des certificats HTTPS
                automatiques et un accès sécurisé à Databasus. Voici une
                configuration Docker Compose complète avec Caddy.
              </p>

              <h3 id="caddy-docker-compose">Docker Compose avec Caddy</h3>

              <p>
                Créez un fichier <code>docker-compose.yml</code> :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeCaddy}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={dockerComposeCaddy} />
                </div>
              </div>

              <p>
                Créez un <code>Caddyfile</code> dans le même répertoire :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{caddyfile}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={caddyfile} />
                </div>
              </div>

              <p>Puis démarrez les services :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text="docker compose up -d" />
                </div>
              </div>

              <p>Cette configuration fournit :</p>

              <ul>
                <li>
                  ✅ HTTPS automatique avec les certificats Let&apos;s Encrypt
                </li>
                <li>✅ Redirection de HTTP vers HTTPS</li>
                <li>✅ Reverse proxy vers Databasus</li>
                <li>
                  ✅ Persistance des données pour Caddy comme pour Databasus
                </li>
              </ul>

              <p>
                Remplacez <code>backup.example.com</code> par votre propre
                domaine. Assurez-vous que le DNS de votre domaine pointe vers
                l&apos;adresse IP de votre serveur avant de démarrer les
                services.
              </p>

              <h2 id="getting-started">Premiers pas</h2>

              <p>Après l&apos;installation :</p>

              <ol>
                <li>
                  <strong>Lancez et ouvrez Databasus</strong> : démarrez
                  Databasus et rendez-vous sur{" "}
                  <code>http://localhost:4005</code>
                </li>
                <li>
                  <strong>Créez votre première tâche de sauvegarde</strong> :
                  cliquez sur &quot;New Backup&quot; et configurez la connexion
                  à votre base PostgreSQL
                </li>
                <li>
                  <strong>Configurez la planification</strong> : définissez la
                  fréquence des backups (horaire, quotidienne, hebdomadaire,
                  mensuelle ou cron)
                </li>
                <li>
                  <strong>Choisissez la destination de stockage</strong> :
                  sélectionnez où stocker vos backups (local, S3, Azure Blob Storage,
                  etc.)
                </li>
                <li>
                  <strong>Configurez les notifications</strong> : ajoutez des
                  canaux de notification (Slack, Telegram, Discord) pour être
                  alerté du statut des backups
                </li>
                <li>
                  <strong>Lancez la sauvegarde</strong> : enregistrez votre
                  configuration et regardez votre premier backup s&apos;exécuter
                  !
                </li>
              </ol>

              <h2 id="health-checks">Health checks</h2>

              <h3 id="docker-health-check">Health check Docker</h3>

              <p>
                Un health check intégré est activé automatiquement pour{" "}
                <code>docker run</code> et Docker Compose. Le conteneur passe à
                l&apos;état <code>healthy</code> dès que Databasus répond aux
                requêtes (après un court délai de démarrage). Il vérifie
                uniquement que l&apos;application répond, donc le conteneur
                n&apos;est pas redémarré pour des conditions non critiques comme
                un espace disque faible.
              </p>

              <h3 id="monitoring-endpoint">Endpoint de monitoring / statut</h3>

              <p>
                Pour le monitoring de disponibilité et les tableaux de bord de
                statut :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{healthEndpoint}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={healthEndpoint} />
                </div>
              </div>

              <p>
                Renvoie <code>200</code> quand tout va bien, ou <code>503</code>{" "}
                avec une raison quand quelque chose demande de l&apos;attention
                : base de données interne, occupation disque (au-delà de 95 %),
                outils clients de bases de données, tâches de fond et vivacité
                de l&apos;agent de vérification.
                L&apos;endpoint est accessible sans authentification, et CORS
                est ouvert pour les outils de monitoring qui tournent dans le
                navigateur.
              </p>

              <p>
                <strong>⚠️ Important :</strong> utilisez cet endpoint pour le
                monitoring et les alertes, pas comme liveness probe de conteneur
                ou de Kubernetes : il renvoie <code>503</code> pour des états
                dégradés mais fonctionnels (par exemple un disque presque
                plein), ce qui redémarrerait un conteneur en état de marche.
              </p>

              <h3 id="kubernetes-health-check">Kubernetes</h3>

              <p>
                Utilisez une liveness/readiness probe qui exécute{" "}
                <code>databasus healthcheck</code> ; gardez l&apos;endpoint{" "}
                <a
                  href="#monitoring-endpoint"
                  className="text-blue-400 hover:text-blue-300"
                >
                  /api/v1/system/health
                </a>{" "}
                pour le monitoring externe.
              </p>

              <h2 id="how-to-update">Comment mettre à jour Databasus ?</h2>

              <h3 id="update-docker">Mettre à jour une installation Docker</h3>

              <p>
                Pour mettre à jour Databasus exécuté via Docker, vous devez
                l&apos;arrêter, nettoyer le cache Docker et redémarrer le
                conteneur.
              </p>

              <ol>
                <li>
                  Allez dans le répertoire où Databasus est installé
                  (généralement <code>/opt/databasus</code>)
                </li>
                <li>
                  Arrêtez le conteneur : <code>docker compose stop</code>
                </li>
                <li>
                  Nettoyez le cache Docker : <code>docker system prune -a</code>
                </li>
                <li>
                  Redémarrez le conteneur : <code>docker compose up -d</code>
                </li>
              </ol>

              <p>
                Cela récupérera la dernière version de Databasus depuis Docker
                Hub (si vous n&apos;avez pas figé la version dans le fichier{" "}
                <code>docker-compose.yml</code>).
              </p>

              <h3 id="update-helm">Mettre à jour une installation Helm</h3>

              <p>
                Pour mettre à jour Databasus exécuté sur Kubernetes via Helm,
                utilisez la commande upgrade :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmUpgrade}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text={helmUpgrade} />
                </div>
              </div>

              <p>
                Si vous avez des valeurs personnalisées, ajoutez{" "}
                <code>-f values.yaml</code> ou utilisez des options{" "}
                <code>--set</code> pour préserver votre configuration. Helm
                effectuera une mise à jour progressive vers la nouvelle version.
              </p>

              <h2 id="postgresus-migration">Migrer depuis Postgresus</h2>

              <p>
                Databasus est le nouveau nom de Postgresus. Si vous utilisez
                actuellement Postgresus, vous pouvez continuer à l&apos;utiliser
                ou migrer vers Databasus.
              </p>

              <p>
                <strong>Important :</strong> renommer simplement l&apos;image
                Docker ne suffit pas, car Postgresus et Databasus utilisent des
                dossiers de données et des noms de base internes différents.
              </p>

              <p>Pour migrer :</p>

              <ol>
                <li>
                  Arrêtez votre conteneur Postgresus :{" "}
                  <code>docker compose stop</code>
                </li>
                <li>
                  Installez Databasus avec l&apos;une des méthodes ci-dessus
                  (utilisez un autre chemin de volume,{" "}
                  <code>./databasus-data</code>)
                </li>
                <li>
                  Recréez manuellement vos bases de données, stockages et
                  notificateurs dans Databasus
                </li>
              </ol>

              <p>
                Pendant la migration, Postgresus et Databasus peuvent tourner
                côte à côte avec des ports et des chemins de volume différents.
              </p>

              <h2 id="troubleshooting">Dépannage</h2>

              <h3 id="container-wont-start">Le conteneur ne démarre pas</h3>

              <p>Si le conteneur ne démarre pas, consultez les logs :</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker logs databasus</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="fr" text="docker logs databasus" />
                </div>
              </div>

              <h3 id="port-already-in-use">Port déjà utilisé</h3>

              <p>
                Si le port 4005 est déjà utilisé, vous pouvez le changer dans
                votre docker-compose.yml :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>
                    ports:
                    {"\n  "}- &quot;8080:4005&quot; # Change 8080 to any
                    available port
                  </code>
                </pre>
              </div>

              <h3 id="permission-denied">Erreurs de permissions</h3>

              <p>
                Si vous rencontrez des problèmes de permissions sur le
                répertoire de données :
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>
                    sudo chown -R $USER:$USER ./databasus-data
                    {"\n"}
                    chmod -R 755 ./databasus-data
                  </code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton
                    lang="fr"
                    text={`sudo chown -R $USER:$USER ./databasus-data\nchmod -R 755 ./databasus-data`}
                  />
                </div>
              </div>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
