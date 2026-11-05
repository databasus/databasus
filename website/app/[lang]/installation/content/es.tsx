import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Instalación - Documentación de Databasus",
  description:
    "Aprenda a instalar Databasus con el script automatizado, Docker run, Docker Compose, Helm para Kubernetes o Caddy como reverse proxy. Instalación sencilla y sin configuración para su sistema self-hosted de copia de seguridad de PostgreSQL.",
  keywords: [
    "instalación de Databasus",
    "instalación con Docker",
    "configurar copia de seguridad de PostgreSQL",
    "respaldo self-hosted",
    "Docker Compose",
    "instalación de copias de seguridad de bases de datos",
    "configuración de pg_dump",
    "Kubernetes",
    "Helm chart",
    "despliegue en K8s",
    "Caddy reverse proxy",
    "configuración de HTTPS",
    "health check",
    "monitorización",
    "liveness probe",
  ],
  openGraph: {
    title: "Instalación - Documentación de Databasus",
    description:
      "Aprenda a instalar Databasus con el script automatizado, Docker run, Docker Compose, Helm para Kubernetes o Caddy como reverse proxy. Instalación sencilla y sin configuración para su sistema self-hosted de copia de seguridad de PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("es", "installation"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Instalación - Documentación de Databasus",
    description:
      "Aprenda a instalar Databasus con el script automatizado, Docker run, Docker Compose, Helm para Kubernetes o Caddy como reverse proxy. Instalación sencilla y sin configuración para su sistema self-hosted de copia de seguridad de PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "installation"),
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
            headline: "Instalación - Documentación de Databasus",
            description:
              "Aprenda a instalar Databasus con el script automatizado, Docker run, Docker Compose, Helm para Kubernetes o Caddy como reverse proxy. Instalación sencilla y sin configuración para su sistema self-hosted de copia de seguridad de PostgreSQL.",
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
            name: "Cómo instalar Databasus",
            description:
              "Guía paso a paso para instalar Databasus, la herramienta de copia de seguridad de PostgreSQL",
            step: [
              {
                "@type": "HowToStep",
                name: "Script de instalación automatizado",
                text: "Ejecute el script de instalación automatizado para instalar Docker y configurar Databasus con arranque automático.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Ejecute el comando curl para descargar y ejecutar el script de instalación",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Docker Run",
                text: "Use el comando docker run para arrancar rápidamente el contenedor de Databasus con persistencia de datos.",
              },
              {
                "@type": "HowToStep",
                name: "Docker Compose",
                text: "Cree un archivo docker-compose.yml y use Docker Compose para un despliegue gestionado.",
              },
              {
                "@type": "HowToStep",
                name: "Kubernetes con Helm",
                text: "Use el Helm chart oficial para desplegar Databasus en Kubernetes con StatefulSet, almacenamiento persistente e ingress opcional.",
              },
              {
                "@type": "HowToStep",
                name: "Ejecución con Caddy como reverse proxy",
                text: "Use Docker Compose con Caddy para despliegues en producción con certificados HTTPS automáticos.",
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent lang="es" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="es" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="installation">Instalación</h1>

              <p className="text-lg text-gray-400">
                Hay varias formas de instalar Databasus: script automatizado
                (recomendado), un simple docker run, Docker Compose, Helm para
                Kubernetes o Docker Compose con Caddy para despliegues en
                producción.
              </p>

              <h2 id="system-requirements">Requisitos del sistema</h2>

              <p>
                Databasus necesita los siguientes recursos mínimos para
                funcionar correctamente:
              </p>

              <ul>
                <li>
                  <strong>CPU</strong>: al menos 1 núcleo
                </li>
                <li>
                  <strong>RAM</strong>: mínimo 500 MB
                </li>
                <li>
                  <strong>Almacenamiento</strong>: 5 GB para la instalación más
                  el espacio que necesite para las copias de seguridad
                </li>
                <li>
                  <strong>Docker</strong>: Docker Engine 20.10+ y Docker Compose
                  v2.0+
                </li>
              </ul>

              <h2 id="option-1-automated-script">
                Opción 1: script de instalación (recomendado, solo Linux)
              </h2>

              <p>El script de instalación:</p>

              <ul>
                <li>
                  ✅ Instala Docker con Docker Compose (si aún no están
                  instalados)
                </li>
                <li>✅ Configura Databasus</li>
                <li>
                  ✅ Activa el arranque automático al reiniciar el sistema
                </li>
              </ul>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{installScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={installScript} />
                </div>
              </div>

              <p>
                En este caso, Databasus se instala en el directorio{" "}
                <code>/opt/databasus</code>.
              </p>

              <h2 id="option-2-docker-run">Opción 2: docker run simple</h2>

              <p>La forma más fácil de ejecutar Databasus:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRun}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={dockerRun} />
                </div>
              </div>

              <p>Este único comando:</p>

              <ul>
                <li>✅ Arranca Databasus</li>
                <li>
                  ✅ Guarda todos los datos en el directorio{" "}
                  <code>./databasus-data</code>
                </li>
                <li>
                  ✅ Reinicia el contenedor automáticamente al reiniciar el
                  sistema
                </li>
              </ul>

              <h2 id="option-3-docker-compose">
                Opción 3: configuración con Docker Compose
              </h2>

              <p>
                Cree un archivo <code>docker-compose.yml</code> con la siguiente
                configuración:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerCompose}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={dockerCompose} />
                </div>
              </div>

              <p>Después ejecute:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text="docker compose up -d" />
                </div>
              </div>

              <p>
                Tenga en cuenta que el arranque puede tardar hasta ~2 minutos.
              </p>

              <h2 id="option-4-helm">Opción 4: Kubernetes con Helm</h2>

              <p>
                Para despliegues en Kubernetes, instale directamente desde el
                registro OCI. Elija el método de acceso que mejor encaje con su
                entorno.
              </p>

              <h3 id="helm-clusterip">
                Con ClusterIP + port-forward (desarrollo)
              </h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallClusterIP}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={helmInstallClusterIP} />
                </div>
              </div>

              <p>Acceda mediante port-forward:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmPortForward}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={helmPortForward} />
                </div>
              </div>

              <h3 id="helm-loadbalancer">
                Con LoadBalancer (entornos en la nube)
              </h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallLoadBalancer}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={helmInstallLoadBalancer} />
                </div>
              </div>

              <p>Obtenga la IP externa y acceda a Databasus:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmGetSvc}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={helmGetSvc} />
                </div>
              </div>

              <h3 id="helm-ingress">Con Ingress (acceso por dominio)</h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallIngress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={helmInstallIngress} />
                </div>
              </div>

              <p>
                Para más opciones (NodePort, TLS, HTTPRoute para Gateway API),
                consulte la{" "}
                <a
                  href="https://github.com/databasus/databasus/tree/main/deploy/helm"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  documentación del Helm chart
                </a>
                .
              </p>

              <h2 id="caddy-reverse-proxy">
                Ejecución con Caddy como reverse proxy
              </h2>

              <p>
                Para despliegues en producción puede usar{" "}
                <a
                  href="https://caddyserver.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Caddy
                </a>{" "}
                como reverse proxy para obtener certificados HTTPS automáticos y
                acceso seguro a Databasus. A continuación tiene una
                configuración completa de Docker Compose con Caddy.
              </p>

              <h3 id="caddy-docker-compose">Docker Compose con Caddy</h3>

              <p>
                Cree un archivo <code>docker-compose.yml</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeCaddy}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={dockerComposeCaddy} />
                </div>
              </div>

              <p>
                Cree un <code>Caddyfile</code> en el mismo directorio:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{caddyfile}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={caddyfile} />
                </div>
              </div>

              <p>Después arranque los servicios:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text="docker compose up -d" />
                </div>
              </div>

              <p>Esta configuración proporciona:</p>

              <ul>
                <li>
                  ✅ HTTPS automático con certificados de Let&apos;s Encrypt
                </li>
                <li>✅ Redirección de HTTP a HTTPS</li>
                <li>✅ Reverse proxy hacia Databasus</li>
                <li>✅ Datos persistentes tanto de Caddy como de Databasus</li>
              </ul>

              <p>
                Sustituya <code>backup.example.com</code> por su dominio real.
                Asegúrese de que el DNS del dominio apunta a la dirección IP de
                su servidor antes de arrancar los servicios.
              </p>

              <h2 id="getting-started">Primeros pasos</h2>

              <p>Después de la instalación:</p>

              <ol>
                <li>
                  <strong>Arranque y acceda a Databasus</strong>: inicie
                  Databasus y abra <code>http://localhost:4005</code>
                </li>
                <li>
                  <strong>Cree su primera tarea de respaldo</strong>: haga clic
                  en &quot;New Backup&quot; y configure la conexión a su base de
                  datos PostgreSQL
                </li>
                <li>
                  <strong>Configure la programación</strong>: defina la
                  frecuencia de las copias (cada hora, día, semana, mes o cron)
                </li>
                <li>
                  <strong>Elija el destino de almacenamiento</strong>: indique
                  dónde guardar las copias (local, S3, Google Drive, etc.)
                </li>
                <li>
                  <strong>Configure las notificaciones</strong>: añada canales
                  (Slack, Telegram, Discord) para recibir alertas sobre el
                  estado de las copias
                </li>
                <li>
                  <strong>Empiece a respaldar</strong>: guarde la configuración
                  y observe cómo se ejecuta su primera copia de seguridad
                </li>
              </ol>

              <h2 id="health-checks">Health checks</h2>

              <h3 id="docker-health-check">Health check en Docker</h3>

              <p>
                Un health check integrado se activa automáticamente para{" "}
                <code>docker run</code> y Docker Compose. El contenedor pasa a{" "}
                <code>healthy</code> en cuanto Databasus empieza a atender
                peticiones (tras un breve periodo de gracia de arranque). Solo
                comprueba que la aplicación responde, así que el contenedor no
                se reinicia por condiciones no críticas como poco espacio en
                disco.
              </p>

              <h3 id="monitoring-endpoint">
                Endpoint de monitorización / estado
              </h3>

              <p>Para monitorización de disponibilidad y paneles de estado:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{healthEndpoint}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={healthEndpoint} />
                </div>
              </div>

              <p>
                Devuelve <code>200</code> cuando todo está bien, o{" "}
                <code>503</code> con el motivo cuando algo necesita atención:
                base de datos interna, uso de disco (por encima del 95%),
                herramientas cliente de bases de datos, tareas en segundo plano
                y actividad del agente de verificación. El endpoint no requiere
                autenticación y tiene CORS abierto para monitores basados en
                navegador.
              </p>

              <p>
                <strong>⚠️ Importante:</strong> solo para monitorización y
                alertas, no como liveness probe del contenedor o de Kubernetes:
                devuelve <code>503</code> en estados degradados pero operativos
                (por ejemplo, un disco casi lleno), lo que reiniciaría un
                contenedor que funciona.
              </p>

              <h3 id="kubernetes-health-check">Kubernetes</h3>

              <p>
                Use una liveness/readiness probe que ejecute{" "}
                <code>databasus healthcheck</code>; reserve el endpoint{" "}
                <a
                  href="#monitoring-endpoint"
                  className="text-blue-400 hover:text-blue-300"
                >
                  /api/v1/system/health
                </a>{" "}
                para la monitorización externa.
              </p>

              <h2 id="how-to-update">¿Cómo actualizar Databasus?</h2>

              <h3 id="update-docker">Actualizar la instalación con Docker</h3>

              <p>
                Para actualizar Databasus ejecutado con Docker, deténgalo,
                limpie la caché de Docker y reinicie el contenedor.
              </p>

              <ol>
                <li>
                  Vaya al directorio donde está instalado Databasus (normalmente{" "}
                  <code>/opt/databasus</code>)
                </li>
                <li>
                  Detenga el contenedor: <code>docker compose stop</code>
                </li>
                <li>
                  Limpie la caché de Docker: <code>docker system prune -a</code>
                </li>
                <li>
                  Reinicie el contenedor: <code>docker compose up -d</code>
                </li>
              </ol>

              <p>
                Se descargará la última versión de Databasus desde Docker Hub
                (si no ha fijado la versión en el archivo{" "}
                <code>docker-compose.yml</code>).
              </p>

              <h3 id="update-helm">Actualizar la instalación con Helm</h3>

              <p>
                Para actualizar Databasus ejecutado en Kubernetes con Helm, use
                el comando upgrade:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmUpgrade}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={helmUpgrade} />
                </div>
              </div>

              <p>
                Si tiene valores personalizados, añada{" "}
                <code>-f values.yaml</code> o use flags <code>--set</code> para
                conservar su configuración. Helm hará una actualización
                progresiva a la nueva versión.
              </p>

              <h2 id="postgresus-migration">Migración desde Postgresus</h2>

              <p>
                Databasus es el nuevo nombre de Postgresus. Si actualmente usa
                Postgresus, puede seguir usándolo o migrar a Databasus.
              </p>

              <p>
                <strong>Importante:</strong> renombrar la imagen de Docker no es
                suficiente, porque Postgresus y Databasus usan carpetas de datos
                y nombres de base de datos interna distintos.
              </p>

              <p>Para migrar:</p>

              <ol>
                <li>
                  Detenga el contenedor de Postgresus:{" "}
                  <code>docker compose stop</code>
                </li>
                <li>
                  Instale Databasus con cualquiera de los métodos anteriores
                  (use una ruta de volumen distinta,{" "}
                  <code>./databasus-data</code>)
                </li>
                <li>
                  Vuelva a crear manualmente sus bases de datos, almacenamientos
                  y notificadores en Databasus
                </li>
              </ol>

              <p>
                Durante la migración puede ejecutar Postgresus y Databasus en
                paralelo usando puertos y rutas de volumen distintos.
              </p>

              <h2 id="troubleshooting">Solución de problemas</h2>

              <h3 id="container-wont-start">El contenedor no arranca</h3>

              <p>Si el contenedor no arranca, revise los logs:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker logs databasus</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text="docker logs databasus" />
                </div>
              </div>

              <h3 id="port-already-in-use">Puerto ya en uso</h3>

              <p>
                Si el puerto 4005 ya está en uso, puede cambiarlo en su
                docker-compose.yml:
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

              <h3 id="permission-denied">Errores de permisos</h3>

              <p>Si tiene problemas de permisos con el directorio de datos:</p>

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
                    lang="es"
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
