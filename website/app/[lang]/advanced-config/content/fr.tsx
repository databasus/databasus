import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Configuration avancée - Documentation Databasus",
  description:
    "Variables d'environnement optionnelles pour l'auto-hébergement de Databasus : connexion Google et GitHub, e-mail SMTP, captcha Cloudflare Turnstile, télémétrie, export des journaux OpenTelemetry et script d'analytique personnalisé. Inutiles pour une installation par défaut.",
  keywords: [
    "variables d'environnement Databasus",
    "configuration avancée Databasus",
    "configuration auto-hébergée",
    "OAuth GitHub",
    "OAuth Google",
    "configuration e-mail SMTP",
    "Cloudflare Turnstile",
    "variables d'environnement Docker",
    "journaux OpenTelemetry",
  ],
  openGraph: {
    title: "Configuration avancée - Documentation Databasus",
    description:
      "Variables d'environnement optionnelles pour l'auto-hébergement de Databasus : connexion Google et GitHub, e-mail SMTP, captcha Cloudflare Turnstile, télémétrie, export des journaux OpenTelemetry et script d'analytique personnalisé. Inutiles pour une installation par défaut.",
    type: "article",
    url: getLocalizedUrl("fr", "advanced-config"),
    locale: OG_LOCALES.fr,
  },
  twitter: {
    card: "summary",
    title: "Configuration avancée - Documentation Databasus",
    description:
      "Variables d'environnement optionnelles pour l'auto-hébergement de Databasus : connexion Google et GitHub, e-mail SMTP, captcha Cloudflare Turnstile, télémétrie, export des journaux OpenTelemetry et script d'analytique personnalisé. Inutiles pour une installation par défaut.",
  },
  alternates: {
    canonical: getLocalizedUrl("fr", "advanced-config"),
    languages: getLanguageAlternates("advanced-config"),
  },
  robots: "index, follow",
};

export default function AdvancedConfigPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Configuration avancée - Documentation Databasus",
            description:
              "Variables d'environnement optionnelles pour l'auto-hébergement de Databasus : connexion Google et GitHub, e-mail SMTP, captcha Cloudflare Turnstile, télémétrie, export des journaux OpenTelemetry et script d'analytique personnalisé. Inutiles pour une installation par défaut.",
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
              <h1 id="advanced-config">Configuration avancée</h1>

              <p className="text-lg text-gray-400">
                Databasus fonctionne d&apos;emblée avec des valeurs par défaut
                raisonnables : une installation standard en un seul conteneur ne
                demande aucune configuration. Chaque variable de cette page est{" "}
                <strong>optionnelle</strong> et inutile dans 99 % des
                déploiements en production.
              </p>

              <h2 id="oauth">OAuth</h2>

              <p>
                Par défaut, Databasus utilise la connexion par e-mail et mot de
                passe. Vous pouvez en plus permettre la connexion avec un compte
                Google ou GitHub. Le bouton d&apos;un fournisseur apparaît dès
                que son client ID est défini, mais la connexion n&apos;aboutit
                que lorsque <strong>les deux</strong> valeurs, client ID et
                client secret, sont présentes.
              </p>

              <p>
                Quand vous enregistrez l&apos;application OAuth, définissez son
                URL de redirection (callback) sur{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>. À cause
                de cette redirection, la connexion OAuth exige que votre
                instance soit servie en HTTPS sur un domaine public (voir la
                note ci-dessous).
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                      HTTPS est requis pour la connexion et l&apos;e-mail.
                    </strong>{" "}
                    La connexion OAuth et l&apos;e-mail exigent tous deux que
                    votre instance soit joignable en HTTPS sur un domaine public
                    : les fournisseurs OAuth redirigent le navigateur vers{" "}
                    <code>https://&lt;your-domain&gt;/auth/callback</code>, et
                    les liens contenus dans les e-mails doivent s&apos;ouvrir
                    chez leurs destinataires. Une instance limitée à localhost
                    ou en HTTP simple ne peut pas utiliser ces fonctionnalités.
                    Le moyen le plus simple d&apos;obtenir HTTPS est la
                    configuration{" "}
                    <a
                      href="/fr/installation/#caddy-reverse-proxy"
                      className="text-blue-400 hover:text-blue-300"
                    >
                      reverse proxy Caddy
                    </a>
                    .
                  </p>
                </div>
              </div>

              <h3 id="oauth-google">Google</h3>

              <p>
                Créez un client OAuth dans la{" "}
                <a
                  href="https://console.cloud.google.com/apis/credentials"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Google Cloud Console
                </a>{" "}
                (APIs &amp; Services → Credentials → Create credentials → OAuth
                client ID, type d&apos;application <em>Web application</em>) et
                ajoutez <code>https://&lt;your-domain&gt;/auth/callback</code>{" "}
                comme URI de redirection autorisée.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      Client ID de votre client OAuth Google. Le définir affiche
                      le bouton &quot;Sign in with Google&quot;.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      Client secret de votre client OAuth Google. Requis avec le
                      client ID pour que la connexion fonctionne.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h3 id="oauth-github">GitHub</h3>

              <p>
                Créez une application OAuth dans les{" "}
                <a
                  href="https://github.com/settings/developers"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  paramètres développeur GitHub
                </a>{" "}
                (Settings → Developer settings → OAuth Apps → New OAuth App) et
                définissez l&apos;URL de callback d&apos;autorisation sur{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      Client ID de votre application OAuth GitHub. Le définir
                      affiche le bouton &quot;Sign in with GitHub&quot;.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      Client secret de votre application OAuth GitHub. Requis
                      avec le client ID pour que la connexion fonctionne.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="email-smtp">E-mail (SMTP)</h2>

              <p>
                Connectez un serveur de messagerie pour que Databasus puisse envoyer les invitations aux espaces de travail, les codes de réinitialisation du mot de passe et les codes de connexion. Le serveur de messagerie est considéré comme configuré <strong>dès que <code>SMTP_HOST</code> est défini</strong>. Sans lui, l&apos;écran des paramètres indique que le serveur de messagerie n&apos;est pas configuré, et l&apos;écran de connexion masque le lien de réinitialisation du mot de passe.
              </p>

              <p>
                Il s&apos;agit du serveur de messagerie de l&apos;instance. Les canaux de notification par e-mail, qui signalent les sauvegardes, ont leurs propres paramètres SMTP dans le formulaire de chaque canal : un canal de notification qui délivre ses e-mails ne dit donc rien de ce serveur. Pour vérifier le serveur de messagerie de l&apos;instance, ouvrez <strong>Databasus settings → Mail server</strong> et cliquez sur <strong>Send test email</strong>. Le message part vers votre propre adresse et, si la livraison échoue, vous voyez la réponse du serveur de messagerie.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>SMTP_HOST</code>
                    </td>
                    <td data-label="Description">
                      Nom d&apos;hôte du serveur de messagerie (par ex. <code>smtp.gmail.com</code>). Le définir active l&apos;e-mail de l&apos;instance.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PORT</code>
                    </td>
                    <td data-label="Description">
                      Port du serveur de messagerie (par ex. <code>587</code>). Doit être un entier positif quand <code>SMTP_HOST</code> est défini.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_SECURITY</code>
                    </td>
                    <td data-label="Description">
                      Sécurité de la connexion : <code>tls</code>, <code>starttls</code> ou <code>none</code>, décrits plus bas. Vaut <code>tls</code> par défaut sur le port 465 et <code>starttls</code> sur tous les autres. Avec toute autre valeur, l&apos;instance refuse de démarrer.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_USER</code>
                    </td>
                    <td data-label="Description">
                      Nom d&apos;utilisateur pour l&apos;authentification SMTP.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PASSWORD</code>
                    </td>
                    <td data-label="Description">
                      Mot de passe pour l&apos;authentification SMTP. Pour Gmail, utilisez un App Password, pas le mot de passe de votre compte.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_FROM</code>
                    </td>
                    <td data-label="Description">
                      Adresse de l&apos;expéditeur, seule (<code>noreply@example.com</code>) ou avec un nom (<code>Acme Backups &lt;noreply@example.com&gt;</code>). Sans nom, les destinataires voient <code>Databasus</code>. Si elle n&apos;est pas définie, <code>SMTP_USER</code> est utilisé s&apos;il s&apos;agit d&apos;une adresse e-mail, sinon <code>noreply@</code> suivi de <code>SMTP_HOST</code> : un nom d&apos;utilisateur comme <code>apikey</code> ne devient donc jamais l&apos;expéditeur. Si la valeur ne peut pas être lue comme une adresse, l&apos;instance refuse de démarrer.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_HELO_NAME</code>
                    </td>
                    <td data-label="Description">
                      Le nom sous lequel l&apos;instance se présente au serveur de messagerie. Par défaut, l&apos;hôte de <code>DATABASUS_URL</code>, ou à défaut le nom de la machine. Doit être un nom d&apos;hôte ou un littéral d&apos;adresse comme <code>[192.0.2.1]</code> ou <code>[IPv6:2001:db8::1]</code>, sinon l&apos;instance refuse de démarrer.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_INSECURE_SKIP_VERIFY</code>
                    </td>
                    <td data-label="Description">
                      Mettez <code>true</code> pour ignorer la vérification du certificat TLS dans les modes <code>tls</code> et <code>starttls</code>. Vaut <code>false</code> par défaut. À réserver aux serveurs avec un certificat auto-signé sur un réseau de confiance : cette option désactive la protection contre les attaques de l&apos;homme du milieu.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>DATABASUS_URL</code>
                    </td>
                    <td data-label="Description">
                      URL de base publique de votre instance (par ex. <code>https://backup.example.com</code>). Facultative : elle ajoute un lien aux e-mails d&apos;invitation et fournit le nom annoncé par défaut. L&apos;e-mail fonctionne sans elle.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                <code>SMTP_SECURITY</code> choisit l&apos;un des trois modes. Les canaux de notification par e-mail proposent le même choix dans <strong>Advanced settings</strong>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Mode</th>
                    <th>Comportement</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>tls</code>
                    </td>
                    <td data-label="Comportement">
                      La connexion est chiffrée dès le premier octet. Habituel sur le port 465.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>starttls</code>
                    </td>
                    <td data-label="Comportement">
                      La connexion démarre en clair puis passe en chiffré avant l&apos;envoi du mot de passe ou de tout message. Si le serveur ne le propose pas, la livraison échoue au lieu de continuer sans chiffrement. Habituel sur le port 587.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>none</code>
                    </td>
                    <td data-label="Comportement">
                      La connexion n&apos;est jamais chiffrée. À utiliser pour un relais qui ne propose pas STARTTLS, comme un Postfix local sur le port 25.
                    </td>
                  </tr>
                </tbody>
              </table>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-red-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-red-500 mt-0.5 shrink-0"
                >
                  <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                  <path d="M12 9v4M12 17h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <code>none</code> envoie le mot de passe SMTP et chaque message en clair, et toute personne placée entre Databasus et le relais peut les lire. Ne l&apos;utilisez qu&apos;avec un relais sur le même hôte ou sur un réseau privé de confiance.
                  </p>
                </div>
              </div>

              <h2 id="signup-captcha">
                Captcha à l&apos;inscription (Cloudflare Turnstile)
              </h2>

              <p>
                Si votre instance est joignable depuis l&apos;internet public,
                vous pouvez placer un défi{" "}
                <a
                  href="https://www.cloudflare.com/products/turnstile/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Cloudflare Turnstile
                </a>{" "}
                sur les formulaires d&apos;inscription et de connexion pour
                tenir les bots à l&apos;écart. Les deux clés viennent du tableau
                de bord Turnstile, et le défi ne s&apos;active que lorsque les
                deux sont définies.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    Pour bloquer complètement les inscriptions externes au lieu
                    de simplement les filtrer, vous n&apos;avez pas besoin de
                    captcha : ouvrez{" "}
                    <strong>Databasus settings → Allow sign up</strong> dans
                    l&apos;interface et désactivez l&apos;option. Cela ferme
                    entièrement le formulaire d&apos;inscription.
                  </p>
                </div>
              </div>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SITE_KEY</code>
                    </td>
                    <td data-label="Description">
                      Clé de site Turnstile publique, utilisée pour afficher le
                      widget dans le navigateur.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SECRET_KEY</code>
                    </td>
                    <td data-label="Description">
                      Clé Turnstile secrète, utilisée par le backend pour
                      valider les réponses au défi.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="docker-storage-permissions">Permissions du stockage Docker</h2>

              <p>
                Databasus prend normalement les ID numériques de l&apos;utilisateur
                et du groupe dans le répertoire pgdata existant, puis dans le
                mount des sauvegardes ou la racine des données. À défaut, il
                utilise <code>999</code>. Définissez <code>PUID</code> ou{" "}
                <code>PGID</code> uniquement si ce choix automatique ne convient
                pas à votre bind mount, partage CIFS ou export NFS. Les valeurs
                doivent être des entiers décimaux de <code>1</code> à{" "}
                <code>4294967294</code>.
              </p>

              <table>
                <thead><tr><th>Variable</th><th>Valeur automatique</th><th>Compte</th></tr></thead>
                <tbody>
                  <tr><td><code>PUID</code></td><td>Propriétaire du mount ou <code>999</code></td><td>Utilisateur <code>databasus</code></td></tr>
                  <tr><td><code>PGID</code></td><td>Groupe du mount ou <code>999</code></td><td>Groupe principal <code>databasus</code></td></tr>
                </tbody>
              </table>

              <p>
                Databasus et PostgreSQL utilisent le même compte du
                système d&apos;exploitation sans privilèges root, nommé{" "}
                <code>databasus</code>.
              </p>

              <p>
                L&apos;entrypoint démarre comme root dans le conteneur pour choisir
                les ID et tenter <code>chown</code> et <code>chmod</code>. Il
                vérifie ensuite tout le cycle des fichiers sous le compte{" "}
                <code>databasus</code>. Un mount peut fonctionner même s&apos;il
                refuse les changements de propriétaire ou de mode. Une valeur
                Docker <code>user:</code> arbitraire n&apos;est pas prise en charge.
              </p>

              <pre><code>{`ERROR: Databasus cannot write to local storage paths /databasus-data/temp and /databasus-data/backups as UID 999 and GID 999.
Required operation: save a file through local storage.
Set PUID and PGID or fix the mounted directory permissions: https://databasus.com/advanced-config/#docker-storage-permissions
Details: permission denied`}</code></pre>

              <p>
                Les quatre anciennes variables propres aux différents services
                ont été supprimées volontairement. Retirez-les de votre
                configuration lors de la mise à jour. Si les ID automatiques ne
                peuvent pas accéder à un mount, le démarrage s&apos;arrête et
                indique l&apos;opération en échec ainsi que le lien vers cette page.
              </p>

              <h2 id="telemetry">Télémétrie</h2>

              <p>
                Databasus envoie par défaut une télémétrie d&apos;usage anonyme
                et non identifiante. Elle ne contient aucune donnée personnelle
                et nous aide à comprendre comment le projet est utilisé. Vous
                pouvez lire exactement ce qui est collecté dans la{" "}
                <a
                  href="/privacy"
                  className="text-blue-400 hover:text-blue-300"
                >
                  politique de confidentialité
                </a>
                , et vous pouvez la désactiver complètement.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Défaut</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>IS_DISABLE_ANONYMOUS_TELEMETRY</code>
                    </td>
                    <td data-label="Défaut">
                      <code>false</code>
                    </td>
                    <td data-label="Description">
                      Mettez <code>true</code> pour désactiver la télémétrie
                      d&apos;usage anonyme.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="logging">Journalisation</h2>

              <p>
                Databasus écrit ses journaux sur stdout et les duplique en JSON
                dans <code>databasus.log</code> sur le volume de données.
                Définissez <code>OPEN_TELEMETRY_URL</code> et il les exporte
                aussi via Open Telemetry vers un backend comme VictoriaLogs,
                Graylog, SigNoz, Grafana Loki, Datadog ou Honeycomb, ou vers un
                OpenTelemetry Collector, qui est lui-même un récepteur OTLP.
              </p>

              <ul>
                <li>
                  <strong>Le transport</strong> suit le schéma de l&apos;URL.{" "}
                  <code>http://</code> et <code>https://</code> envoient en
                  OTLP/HTTP et utilisent l&apos;URL telle quelle, chemin compris
                  ; <code>grpc://</code> et <code>grpcs://</code> envoient en
                  OTLP/gRPC et n&apos;utilisent que l&apos;hôte et le port.
                </li>
                <li>
                  <strong>L&apos;authentification</strong> se met dans{" "}
                  <code>OPEN_TELEMETRY_HEADERS</code> ou dans l&apos;URL sous la
                  forme <code>user:password@host</code>.
                </li>
                <li>
                  <strong>Les secrets</strong> (mots de passe, jetons,
                  identifiants) dans les URL sont caviardés avant qu&apos;un
                  enregistrement ne quitte le processus.
                </li>
                <li>
                  <strong>Les entrées d&apos;audit</strong> partent avec les
                  journaux applicatifs, marquées <code>log_type=audit</code>, et
                  ignorent <code>LOG_LEVEL</code> : élever le niveau ne fait
                  donc jamais disparaître la piste d&apos;audit.
                </li>
              </ul>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Défaut</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_URL</code>
                    </td>
                    <td data-label="Défaut">—</td>
                    <td data-label="Description">
                      URL complète du point de terminaison OTLP, chemin compris.
                      Laissez vide pour garder les journaux dans le conteneur.
                      Une chaîne de requête, un hôte manquant ou un schéma
                      inconnu arrête le conteneur au démarrage plutôt que
                      d&apos;exporter dans le vide.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </td>
                    <td data-label="Défaut">—</td>
                    <td data-label="Description">
                      Paires <code>key=value</code> séparées par des virgules,
                      envoyées avec chaque export, en général une clé API. Les
                      valeurs sont décodées en pourcent, conformément au format
                      standard <code>OTEL_EXPORTER_OTLP_HEADERS</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_LEVEL</code>
                    </td>
                    <td data-label="Défaut">
                      <code>info</code>
                    </td>
                    <td data-label="Description">
                      L&apos;une des valeurs <code>debug</code>,{" "}
                      <code>info</code>, <code>warn</code> ou <code>error</code>
                      . Une valeur non reconnue retombe sur <code>info</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_FILE_IS_ENABLED</code>
                    </td>
                    <td data-label="Défaut">
                      <code>true</code>
                    </td>
                    <td data-label="Description">
                      Écrit <code>databasus.log</code> à côté du reste des
                      données, avec rotation à 5 Mo et conservation de 3 anciens
                      fichiers. Mettez <code>false</code> si votre plateforme
                      collecte déjà stdout.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Valeurs pour les backends courants, chacune avec l&apos;en-tête
                qui l&apos;authentifie. Remplacez les hôtes, régions et clés par
                les vôtres :
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Backend</th>
                    <th>
                      <code>OPEN_TELEMETRY_URL</code>
                    </th>
                    <th>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>VictoriaLogs</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>
                        http://victoria-logs:9428/insert/opentelemetry/v1/logs
                      </code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Basic%20dXNlcjpwYXNzd29yZA==</code> :
                      les identifiants attendus par votre <code>vmauth</code> ou
                      votre reverse proxy, puisque VictoriaLogs lui-même
                      n&apos;a pas d&apos;authentification sur le chemin
                      d&apos;ingestion.
                    </td>
                  </tr>
                  <tr>
                    <td>OpenTelemetry Collector</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://otel-collector:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> :
                      correspond à une extension <code>bearertokenauth</code> ou{" "}
                      <code>basicauth</code> sur le récepteur. Un Collector
                      joignable uniquement dans votre réseau n&apos;en a en
                      général pas besoin.
                    </td>
                  </tr>
                  <tr>
                    <td>Graylog 6.2+</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://graylog:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> : le jeton
                      défini sur l&apos;input OpenTelemetry (gRPC). L&apos;input
                      accepte aussi le mTLS à la place.
                    </td>
                  </tr>
                  <tr>
                    <td>SigNoz Cloud</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpcs://ingest.eu.signoz.cloud:443</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>signoz-ingestion-key=your-ingestion-key</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Grafana Cloud</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>
                        https://otlp-gateway-prod-eu-west-0.grafana.net/otlp/v1/logs
                      </code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Basic%20&lt;base64&gt;</code> : base64
                      de <code>instance-id:api-token</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Honeycomb</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>https://api.honeycomb.io/v1/logs</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>x-honeycomb-team=your-api-key</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Datadog Agent</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://datadog-agent:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      Aucun : l&apos;Agent détient la clé API et transmet pour
                      vous.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Les valeurs d&apos;en-tête sont décodées en pourcent, donc
                l&apos;espace après <code>Basic</code> ou <code>Bearer</code>{" "}
                s&apos;écrit <code>%20</code> et une virgule dans une valeur{" "}
                <code>%2C</code>. L&apos;authentification Basic peut aussi aller
                directement dans l&apos;URL sous la forme{" "}
                <code>https://user:password@host/path</code> : Databasus la
                transforme en le même en-tête et la garde hors des journaux. Sur{" "}
                <code>http://</code> et <code>grpc://</code>, les clés et mots
                de passe circulent en clair, utilisez donc <code>https://</code>{" "}
                ou <code>grpcs://</code> en dehors d&apos;un réseau de
                confiance.
              </p>

              <h2 id="analytics-script">Script d&apos;analytique</h2>

              <p>
                Databasus peut injecter votre propre extrait d&apos;analytique
                ou de suivi (Google Analytics, Plausible, Umami et similaires)
                dans l&apos;application. Quand <code>ANALYTICS_SCRIPT</code> est
                défini, sa valeur est insérée dans le <code>&lt;head&gt;</code>{" "}
                de la page au démarrage.
              </p>

              <p>
                <strong>Avertissement de sécurité :</strong> la valeur est
                injectée telle quelle, comme HTML et JavaScript bruts, et
                s&apos;exécute avec un accès complet à l&apos;interface
                Databasus dans le navigateur de chaque visiteur. Ne la
                définissez que sur un extrait que vous contrôlez et auquel vous
                faites entièrement confiance.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Description</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>ANALYTICS_SCRIPT</code>
                    </td>
                    <td data-label="Description">
                      Balisage <code>&lt;script&gt;</code> personnalisé injecté
                      avant la balise fermante <code>&lt;/head&gt;</code>.
                      Laissez vide pour n&apos;ajouter aucune analytique.
                    </td>
                  </tr>
                </tbody>
              </table>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
