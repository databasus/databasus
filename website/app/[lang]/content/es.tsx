import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl, getLocalizedHref } from "@/app/i18n";
import HeroStatsComponent from "@/app/components/HeroStatsComponent";
import InstallationComponent from "@/app/components/InstallationComponent";
import LiteYouTubeEmbed from "@/app/components/LiteYouTubeEmbed";

export const metadata: Metadata = {
  title: "Copia de seguridad de PostgreSQL | Databasus",
  description:
    "Herramienta gratuita y de código abierto para copias de seguridad programadas de PostgreSQL (con soporte para MySQL y MongoDB). Guárdelas en local y en la nube. Notificaciones a Slack, Discord, Telegram, correo, webhook y más.",
  keywords:
    "PostgreSQL, copia de seguridad, respaldo, monitoreo, base de datos, copias de seguridad programadas, Docker, self-hosted, código abierto, S3, Azure Blob Storage, notificaciones en Slack, Discord, DevOps, monitoreo de bases de datos, pg_dump, restauración de base de datos, cifrado, AES-256, cifrado de copias de seguridad",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("es", ""),
    languages: getLanguageAlternates(""),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("es", ""),
    title: "Copia de seguridad de PostgreSQL | Databasus",
    description:
      "Herramienta gratuita y de código abierto para copias de seguridad programadas de PostgreSQL (con soporte para MySQL y MongoDB). Guárdelas en local y en la nube. Notificaciones a Slack, Discord, Telegram, correo, webhook y más.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Interfaz del panel de Databasus para gestionar copias de seguridad",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary_large_image",
    title: "Copia de seguridad de PostgreSQL | Databasus",
    description:
      "Herramienta gratuita y de código abierto para copias de seguridad programadas de PostgreSQL (con soporte para MySQL y MongoDB). Guárdelas en local y en la nube. Notificaciones a Slack, Discord, Telegram, correo, webhook y más.",
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
              "Herramienta gratuita y de código abierto para copias de seguridad programadas de PostgreSQL (con soporte para MySQL y MongoDB). Guárdelas en local y en la nube. Notificaciones a Slack, Discord, Telegram, correo, webhook y más.",
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
              "Copias de seguridad programadas de PostgreSQL",
              "Múltiples destinos de almacenamiento (S3, Azure Blob Storage, FTP, SFTP, NAS, etc.)",
              "Notificaciones en tiempo real (Slack, Telegram, Discord, webhook, correo, etc.)",
              "Monitoreo del estado de las bases de datos",
              "Autoalojado mediante Docker",
              "Código abierto y gratuito",
              "Soporte para PostgreSQL 12-18",
              "Compresión de copias de seguridad y cifrado AES-256-GCM",
              "Soporte para PostgreSQL, MySQL, MariaDB y MongoDB",
              "Políticas de retención: por periodo de tiempo, cantidad, GFS y límites de tamaño",
              "Point-in-Time Recovery (PITR) con archivado de WAL",
              "Verificación de restauración: pruebas de restauración automatizadas en contenedores Docker con bases de datos reales",
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
            description: "Herramienta de copia de seguridad de PostgreSQL",
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
                name: "¿Qué es Databasus y por qué usarlo en lugar de scripts escritos a mano?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus es un servicio autoalojado con licencia Apache 2.0 que respalda PostgreSQL, de la v13 a la v18. Se diferencia de los scripts de shell en que ofrece una interfaz para programar tareas, comprimir y guardar los archivos en varios destinos (disco local, S3, Azure Blob Storage, FTP, SFTP, NAS, etc.), configurar políticas de retención que eliminan automáticamente las copias antiguas y avisar a su equipo cuando las tareas terminan o fallan, todo sin escribir código a mano",
                },
              },
              {
                "@type": "Question",
                name: "¿Cuál es la forma más rápida de instalar Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "La vía más directa es ejecutar el instalador de una línea con cURL. Descarga la imagen Docker actual y levanta un único contenedor de PostgreSQL. Después crea un docker-compose.yml y arranca el servicio, que volverá a levantarse automáticamente tras cada reinicio. El tiempo total suele ser inferior a dos minutos en un VPS típico.",
                },
              },
              {
                "@type": "Question",
                name: "¿Cómo funciona la verificación de restauración?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus ejecuta un pequeño agente de verificación en un host que usted controla. En cada ejecución programada, el agente descarga la copia de seguridad más reciente. La restaura en un contenedor de base de datos desechable. Luego compara la base de datos restaurada con la de origen. El resultado se informa de vuelta, incluyendo el código de salida de la restauración y el recuento de filas por tabla. Las programaciones admiten después de cada copia, cada hora, diaria, semanal, mensual o una expresión cron en UTC. Los fallos pueden enviarse por cualquier notificador conectado a la base de datos: Slack, Teams, Discord, correo y otros.",
                },
              },
              {
                "@type": "Question",
                name: "¿Dónde se guardan mis copias de seguridad y cuánto espacio ocuparán?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Los archivos pueden guardarse en volúmenes locales, buckets compatibles con S3, Azure Blob Storage, FTP y otros destinos en la nube. Databasus aplica una compresión equilibrada que normalmente reduce el tamaño del volcado entre 4 y 8 veces con solo un 20% de sobrecarga adicional en el tiempo de ejecución, de modo que ahorra almacenamiento y ancho de banda.",
                },
              },
              {
                "@type": "Question",
                name: "¿Cómo sabré si una copia de seguridad tuvo éxito o, peor aún, falló?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus puede notificar en tiempo real por correo, Slack, Telegram, webhooks, Mattermost, Discord y más. Usted elige qué canales avisar para que su equipo DevOps se entere de los éxitos y los fallos en tiempo real, lo que facilita las rutinas de recuperación y las auditorías de cumplimiento.",
                },
              },
              {
                "@type": "Question",
                name: "¿Cómo garantiza Databasus la seguridad?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus aplica seguridad en tres niveles: (1) Cifrado de datos sensibles: todas las contraseñas, tokens y credenciales se cifran con AES-256-GCM y se guardan separados de la base de datos; (2) Cifrado de las copias de seguridad: cada archivo de respaldo se cifra con una clave única derivada de una clave maestra, el ID de la copia y una sal aleatoria, de modo que las copias resultan inútiles sin su clave de cifrado aunque alguien acceda al almacenamiento; (3) Acceso de solo lectura a la base de datos: Databasus solo requiere permisos SELECT y realiza comprobaciones exhaustivas para asegurar que no existen privilegios de escritura, lo que evita la corrupción de datos incluso si la herramienta se ve comprometida. El propio inicio de sesión puede exigir un segundo factor: con él activado, tras la contraseña correcta llega un código de seis dígitos al correo de la cuenta, mientras que el inicio de sesión con Google o GitHub sigue apoyándose en las comprobaciones de esos proveedores. Antes de cada versión, las comprobaciones automáticas analizan código, dependencias, secretos e imágenes de contenedores. Las pruebas verifican ciclos completos de copia de seguridad y restauración en contenedores reales de PostgreSQL, MySQL, MariaDB y MongoDB. Todas las comprobaciones obligatorias deben pasar antes de publicar. Las auditorías periódicas de Codex Security de OpenAI las complementan y se realizan por separado del proceso de publicación. Todas las operaciones se ejecutan en contenedores que usted controla, en servidores que le pertenecen, y al ser código abierto, su equipo de seguridad puede auditar cada línea de código antes del despliegue.",
                },
              },
              {
                "@type": "Question",
                name: "¿Databasus cuenta con el apoyo de los programas OSS de Anthropic y OpenAI?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Sí, en marzo de 2026 Databasus fue aceptado tanto en Claude for Open Source de Anthropic como en Codex for Open Source de OpenAI. El proyecto fue evaluado de forma independiente y reconocido por líderes de la industria como infraestructura de código abierto crítica que merece apoyo.",
                },
              },
              {
                "@type": "Question",
                name: "¿En qué se diferencia Databasus de PgBackRest, Barman o pg_dump?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus apuesta por la simplicidad: ofrece una interfaz web moderna para gestionar las copias de seguridad de muchas bases de datos a la vez, con programación integrada, compresión, múltiples destinos de almacenamiento, monitoreo del estado y notificaciones en tiempo real. A la vez, a diferencia de pgBackRest y WAL-G, Databasus realiza copias físicas, incrementales y de WAL sobre el mecanismo nativo de PostgreSQL 17, así que no reinventa su propio motor de respaldo. Se conecta a sus bases de datos de forma remota y alcanza redes cerradas mediante un túnel SSH al servidor o a un bastión, de modo que las bases de datos sin exposición pública también pueden respaldarse y gestionarse desde un único panel.",
                },
              },
              {
                "@type": "Question",
                name: "¿Qué bases de datos admite Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus admite PostgreSQL, MySQL, MariaDB y MongoDB. Sin embargo, Databasus fue creado originalmente para PostgreSQL y mantiene su enfoque principal en él: soporte excelente al 100% y la máxima eficiencia para las copias de seguridad de PostgreSQL. Aunque MySQL, MariaDB y MongoDB están soportados, PostgreSQL sigue siendo la prioridad central, con las funciones más optimizadas y desarrollo continuo. Por ejemplo, Databasus ofrece soporte nativo de copias físicas y de WAL para la recuperación ante desastres de PostgreSQL. En realidad, Databasus es una herramienta de copia de seguridad de PostgreSQL y las demás bases de datos son solo extensiones.",
                },
              },
              {
                "@type": "Question",
                name: "¿Cuál es el nivel de adopción de Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus es hoy la herramienta de código abierto para copias de seguridad de PostgreSQL con mayor adopción. A 17 de junio de 2026, DBA, ingenieros DevOps, desarrolladores y equipos de todo el mundo la han descargado más de 1.800.000 veces en Docker. Con más de 8.500 estrellas en GitHub, supera a pgBackRest (~4.200 estrellas, disponible desde 2014) y a WAL-G (~4.100 estrellas, disponible desde 2017). Databasus se lanzó en 2025 y superó a ambos en su primer año.",
                },
              },
              {
                "@type": "Question",
                name: "¿Qué tipos de copia de seguridad admite Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus admite copias físicas, completas, incrementales, de WAL y lógicas. Las copias físicas son una copia a nivel de archivos de todo el clúster de la base de datos: más rápidas de crear y restaurar en conjuntos de datos grandes que los volcados lógicos, y construidas sobre el mecanismo nativo de respaldo de PostgreSQL 17, así que nos apoyamos en las herramientas probadas de PostgreSQL en lugar de reinventarlas. Las copias completas son una copia íntegra y autocontenida del clúster, la base de la que parte cada cadena de respaldos. Las copias incrementales guardan solo lo que cambió desde la copia anterior, así que los respaldos se mantienen pequeños y rápidos. El streaming de WAL captura de forma continua el flujo de escritura de la base de datos, lo que habilita la recuperación a un punto en el tiempo (PITR) para recuperación ante desastres y una pérdida de datos casi nula. Las copias lógicas son un volcado nativo de la base de datos en su formato binario específico del motor, comprimido y transmitido directamente al almacenamiento sin archivos intermedios. Todas estas copias pueden ejecutarse por un túnel SSH si necesita conexiones no públicas, de modo que la base de datos nunca tiene que exponerse públicamente. El túnel SSH viene integrado.",
                },
              },
              {
                "@type": "Question",
                name: "Olvidé el correo o la contraseña del administrador",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: 'La primera cuenta creada en la instancia es la que la administra. Si no recuerda cuál es esa dirección, ejecute docker exec -it databasus ./main --list-admins en el servidor donde se ejecuta Databasus: nombra cada cuenta de administrador con su correo electrónico, nombre visible, fecha de creación y estado de actividad, y no imprime ninguna contraseña. Para establecer una nueva contraseña para esa cuenta, ejecute docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com". Una instancia creada antes de que la primera cuenta la administrara todavía lleva la dirección provisional admin hasta que su propietario la reemplace. Si además la instancia exige un código al iniciar sesión y el servidor de correo ha dejado de entregarlo, ejecute docker exec -it databasus ./main --disable-2fa para que deje de pedirlo. Los tres comandos están documentados en la página de contraseña.',
                },
              },
              {
                "@type": "Question",
                name: "¿Cómo se utiliza la IA en el desarrollo de Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "La IA nos ayuda a desarrollar Databasus siguiendo un proceso de ingeniería definido. Las personas definen el rumbo y toman las decisiones finales. Una persona lee cada línea de código antes de integrar un cambio. Agentes independientes revisan la corrección, la arquitectura, las pruebas y la documentación en sesiones separadas de solo lectura. El análisis estático estricto y las pruebas con bases de datos reales complementan sus revisiones. Las comprobaciones obligatorias de CI deben superarse antes de publicar.",
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        lang="es"
        sectionLinks={[
          { href: "#how-to-use", label: "Cómo usarlo" },
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
              Copia de seguridad de PostgreSQL
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus es una herramienta gratuita, de código abierto y
              autoalojada para respaldar PostgreSQL. Cree copias de seguridad en
              distintos almacenamientos (S3, Azure Blob Storage, FTP, etc.) con
              notificaciones sobre el progreso (Slack, Discord, Telegram, etc.).
              Con un enfoque en Point-in-Time Recovery{" "}
              <span className="underline decoration-2 underline-offset-2 decoration-blue-600">
                con RPO/RTO bajos
              </span>
            </p>

            <div>
              <div className="flex flex-col items-center justify-center gap-2 max-w-[370px] sm:max-w-[340px] mx-auto">
                <a
                  href="#installation"
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-1"
                >
                  Autoalójelo con Docker
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
                alt="Interfaz del panel de Databasus"
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
                alt="Con el apoyo de los programas OSS de Anthropic y OpenAI"
              />

              <div className="flex justify-center text-base md:text-xl mt-4 md:mt-0 md:ml-10">
                <div className="max-w-[370px] text-gray-400 text-center md:text-left">
                  Con el apoyo de los programas de código abierto de Anthropic y
                  OpenAI.{" "}
                  <a
                    href="/es/faq#oss-programs"
                    target="_blank"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Más&nbsp;información&nbsp;→
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
              <span className="text-sm font-medium">Resumen</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Funcionalidades
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus ofrece todo lo necesario para gestionar respaldos de
              producción de forma fiable. Desde la programación automática hasta
              el cifrado de las copias de seguridad. Es adecuado tanto para
              desarrolladores individuales con proyectos personales como para
              equipos DevOps y empresas
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
                Copias de seguridad programadas
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Copias de seguridad programadas"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Un respaldo debe ejecutarse a la hora indicada y de forma
                regular. Por eso ofrecemos muchas opciones: cada hora, diaria,
                semanal, mensual, cron, etc.
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Comprobaciones de estado configurables
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Comprobaciones de estado"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Cada minuto (o el intervalo que prefiera) el sistema hará ping a
                su base de datos y le mostrará el historial de intentos
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                La base de datos puede considerarse caída tras 3 intentos
                fallidos, por ejemplo. Cuando vuelve a estar operativa, usted
                también recibe una notificación
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Muchos destinos de almacenamiento
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Los archivos se guardan en su VPS, en almacenamientos en la nube
                y en otros lugares. Puede elegir el almacenamiento que quiera.
                Los archivos siempre le pertenecen a usted.{" "}
                <a
                  href="/es/storages"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Ver todos →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-destinations.svg"
                  alt="Almacenamiento"
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
                Notificaciones
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Puede recibir notificaciones sobre el éxito o el fallo del
                proceso. Resulta útil para desarrolladores y equipos DevOps.{" "}
                <a
                  href="/es/notifiers"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Ver todos →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-notifications.svg"
                  alt="Notificaciones"
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
                Autoalojado mediante Docker
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Databasus se ejecuta en su PC o VPS. Por lo tanto, todos sus
                datos le pertenecen y están protegidos. El despliegue lleva unos
                2 minutos mediante script, Docker o k8s
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
                Código abierto y gratuito
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                El proyecto es totalmente de código abierto, gratuito y con
                licencia Apache 2.0. Puede copiar y bifurcar el código por su
                cuenta. También está abierto al uso empresarial
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
                  Verificación de restauración
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Una copia de seguridad que termina sin errores no es lo mismo
                  que una copia que se puede restaurar. Databasus descarga
                  periódicamente el último respaldo, lo restaura en un
                  contenedor de base de datos desechable e informa del
                  resultado.{" "}
                  <a
                    href="/es/restore-verification"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Leer más →
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
                  Seguridad
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  El cifrado de nivel empresarial protege los datos sensibles y
                  las copias de seguridad. El acceso de solo lectura a la base
                  de datos evita la corrupción de datos. Todo esto no requiere
                  ningún conocimiento y funciona de forma automática desde el
                  primer momento.{" "}
                  <a
                    href="/es/security"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Leer más →
                  </a>
                </p>

                <div>
                  <img
                    src="/images/index/feature-encryption.svg"
                    alt="Seguridad"
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
                  Gestión de accesos
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  para equipos
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-access-management.svg"
                  alt="Gestión de accesos"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Otorgue a los usuarios acceso para ver o gestionar las bases de
                datos. Separe equipos y proyectos. Adecuado para equipos DevOps
                y desarrolladores.{" "}
                <a
                  href="/es/access-management#settings"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Leer más →
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
                  Registros de auditoría
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  para equipos
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-audit-logs.svg"
                  alt="Registros de auditoría"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Siga toda la actividad del sistema con registros de auditoría
                completos. Puede ver el historial de accesos y cambios de cada
                usuario (descargas de copias, cambios de programación,
                actualizaciones de configuración, etc.).{" "}
                <a
                  href="/es/access-management#audit-logs"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Leer más →
                </a>
              </p>
            </div>

            {/* Card 10: Security - Mobile/Tablet only */}
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1 lg:hidden">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                10
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Seguridad
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                El cifrado de nivel empresarial protege los datos sensibles y
                las copias de seguridad. El acceso de solo lectura a la base de
                datos evita la corrupción de datos. Todo esto no requiere ningún
                conocimiento y funciona de forma automática desde el primer
                momento.{" "}
                <a
                  href="/es/security"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Leer más →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-encryption.svg"
                  alt="Seguridad"
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
                  Copias lógicas, físicas, incrementales y de WAL
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus admite copias lógicas y físicas (completas e
                  incrementales) con streaming de WAL para Point-in-Time
                  Recovery. Esto hace que Databasus sirva para la recuperación
                  ante desastres y funcione igual de bien con bases de datos
                  autoalojadas y en la nube: use el modo remoto para bases de
                  datos gestionadas en la nube o accesibles públicamente. Las
                  copias físicas se realizan sobre el mecanismo nativo de
                  respaldo de PG 17.{" "}
                  <a
                    href="/es/faq/#pitr"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Leer más →
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
                <span className="text-sm font-medium">
                  Resumen de 4 minutos
                </span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                ¿Cómo usar Databasus?
              </h2>

              <p className="text-gray-200 max-w-[450px] leading-relaxed mb-6 md:mb-8 text-sm sm:text-base">
                Vea en este video cómo conectar su base de datos, cómo
                configurar la programación de las copias de seguridad, cómo
                descargarlas y restaurarlas, cómo añadir miembros al equipo y
                qué son los registros de auditoría de usuarios
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
                    Desarrollador de Databasus
                  </p>
                </div>
              </a>
            </div>

            {/* Right side: Video */}
            <div className="flex-1 relative">
              <div className="rounded-lg overflow-hidden shadow-lg border border-[#ffffff20]">
                <LiteYouTubeEmbed
                  videoId="1qsAnijJfJE"
                  title="¿Cómo usar Databasus? (resumen)"
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
              <span className="text-sm font-medium">Bases de datos</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Bases de datos compatibles
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              Databasus admite PostgreSQL, MySQL, MariaDB y MongoDB. Puede
              respaldarlas y restaurarlas todas con la misma herramienta. El
              enfoque principal está en PostgreSQL, pero MySQL, MariaDB y
              MongoDB también están soportadas
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
                PostgreSQL es la base de datos principal soportada por
                Databasus. Se admiten todas las versiones de la 12 a la 18
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
                MySQL es la segunda base de datos más popular del mundo. Puede
                respaldar y restaurar sus bases de datos MySQL con la misma
                simplicidad
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/es/mysql-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Leer más →
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
                MariaDB está soportada con las mismas funciones que MySQL. Puede
                respaldar y restaurar sus bases de datos MariaDB sin
                complicaciones
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/es/mysql-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Leer más →
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
                MongoDB es la base de datos NoSQL más popular. Puede respaldar y
                restaurar sus bases de datos MongoDB con la misma interfaz fácil
                de usar
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/es/mongodb-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Leer más →
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
              <span className="text-sm font-medium">Proceso</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              ¿Cómo crear una copia de seguridad de PostgreSQL?
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              La prioridad principal de Databasus es la simplicidad; ahora mismo
              es la herramienta más fácil del mundo para respaldar PostgreSQL.
              Para crear copias de seguridad solo hay que seguir 4 pasos.
              Después podrá restaurar con un clic
            </p>
          </div>

          {/* Steps */}
          <div className="space-y-6 md:space-y-10 max-w-[1000px] mx-auto">
            {/* Step 1 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Paso 1
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Seleccione la programación necesaria
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Puede elegir el momento que necesite: diario, semanal,
                    mensual, una hora concreta (como las 4 AM) y ciclos cron
                  </p>
                  <p>
                    Para el intervalo semanal debe indicar un día concreto; para
                    el mensual, también un día concreto
                  </p>
                  <p>
                    Si su base de datos es grande, le recomendamos elegir la
                    hora en la que baja el tráfico
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Paso 1"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 2 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Paso 2
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Introduzca los datos de su base de datos
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Introduzca las credenciales de su base de datos PostgreSQL,
                    seleccione la versión y la base de datos de destino. Elija
                    también si se requiere SSL
                  </p>
                  <p>
                    De forma predeterminada, Databasus comprime las copias en un
                    nivel equilibrado para no ralentizar el proceso (~20% más
                    lento) y ahorrar de 4 a 8 veces el espacio (lo que reduce el
                    tráfico de red)
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-2.svg"
                  alt="Paso 2"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 3 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Paso 3
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Elija el almacenamiento para las copias
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Puede guardar los archivos con las copias de seguridad en
                    local, en S3, en Azure Blob Storage, NAS, FTP y otros
                    servicios
                  </p>
                  <p>
                    Tenga en cuenta que necesita espacio suficiente en el
                    almacenamiento
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-3.svg"
                  alt="Paso 3"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 4 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Paso 4
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Elija dónde recibir las notificaciones (opcional)
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Cuando una copia de seguridad termina con éxito o falla,
                    Databasus puede enviarle una notificación. Puede ser el chat
                    con DevOps, su correo o incluso un webhook de su equipo
                  </p>
                  <p>
                    Vamos a dar soporte a la mayoría de las plataformas y
                    mensajerías populares
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-4.svg"
                  alt="Paso 4"
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
              Comenzar
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
                <span className="text-sm font-medium">Primeros pasos</span>
              </div>

              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
                ¿Cómo instalar?
              </h2>

              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                Databasus admite muchas formas de instalación. Se soporta tanto
                en local como en la nube. Ambas vías son extremadamente
                sencillas y fáciles de usar, incluso para quien no tiene
                experiencia en administración o DevOps
              </p>
            </div>

            <InstallationComponent lang="es" />
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
              Preguntas frecuentes
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              El objetivo de Databasus es hacer que respaldar sea lo más simple
              posible tanto para desarrolladores individuales (y DevOps) como
              para equipos. La interfaz facilita crear copias de seguridad,
              visualiza el progreso y restaura cualquier cosa en un par de clics
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="¿Qué es Databasus y por qué usarlo en lugar de scripts escritos a mano?"
              answer="Databasus es un servicio autoalojado con licencia Apache 2.0 que respalda bases de datos. Se diferencia de los scripts de shell en que ofrece una interfaz para programar tareas, comprimir y guardar los archivos en varios destinos (disco local, S3, Azure Blob Storage, NAS, FTP, SFTP, etc.), configurar políticas de retención que eliminan automáticamente las copias antiguas y avisar a su equipo cuando las tareas terminan o fallan, todo sin escribir código a mano"
            />
            <FaqItem
              number="2"
              question="¿Cuál es la forma más rápida de instalar Databasus?"
              answer="Databasus admite varios métodos de instalación: script automatizado, Docker, Docker Compose y Kubernetes con Helm. La vía más rápida es ejecutar el instalador de una línea con cURL, que descarga la imagen Docker actual, crea un docker-compose.yml y arranca el servicio, que volverá a levantarse automáticamente tras cada reinicio del servidor. Para entornos Kubernetes, use el chart oficial de Helm para despliegues listos para producción. El tiempo total suele ser inferior a dos minutos en un VPS típico."
            />
            <FaqItem
              number="3"
              question="¿Cómo funciona la verificación de restauración?"
              answer="Databasus ejecuta un pequeño agente de verificación en un host que usted controla. En cada ejecución programada, el agente descarga la copia de seguridad más reciente. La restaura en un contenedor de base de datos desechable. Luego compara la base de datos restaurada con la de origen. El resultado se informa de vuelta, incluyendo el código de salida de la restauración y el recuento de filas por tabla. Las programaciones admiten después de cada copia, cada hora, diaria, semanal, mensual o una expresión cron en UTC. Los fallos pueden enviarse por cualquier notificador conectado a la base de datos: Slack, Teams, Discord, correo y otros."
            />
            <FaqItem
              number="4"
              question="¿Cómo garantiza Databasus la seguridad?"
              answer={
                <>
                  Databasus aplica seguridad en tres niveles: (1) Cifrado de
                  datos sensibles: todas las contraseñas, tokens y credenciales
                  se cifran con AES-256-GCM y se guardan separados de la base de
                  datos; (2) Cifrado de las copias de seguridad: cada archivo de
                  respaldo se cifra con una clave única derivada de una clave
                  maestra, el ID de la copia y una sal aleatoria, de modo que
                  las copias resultan inútiles sin su clave de cifrado aunque
                  alguien acceda al almacenamiento; (3) Acceso de solo lectura a
                  la base de datos: Databasus solo requiere permisos SELECT y
                  realiza comprobaciones exhaustivas para asegurar que no
                  existen privilegios de escritura, lo que evita la corrupción
                  de datos incluso si la herramienta se ve comprometida.
                  <br />
                  <br />
                  El propio inicio de sesión puede exigir un segundo factor: con
                  él activado, tras la contraseña correcta llega un código de
                  seis dígitos al correo de la cuenta, mientras que el inicio de
                  sesión con Google o GitHub sigue apoyándose en las
                  comprobaciones de esos proveedores.
                  <br />
                  <br />
                  Antes de cada versión, las comprobaciones automáticas analizan código,
                  dependencias, secretos e imágenes de contenedores. Las pruebas verifican
                  ciclos completos de copia de seguridad y restauración en contenedores reales
                  de PostgreSQL, MySQL, MariaDB y MongoDB. Todas las comprobaciones
                  obligatorias deben pasar antes de publicar. Las auditorías periódicas de
                  Codex Security de OpenAI las complementan y se realizan por separado del
                  proceso de publicación.
                  <br />
                  <br />
                  Consulte{" "}
                  <a
                    href={`${getLocalizedHref("es", "security")}#security-and-reliability-engineering`}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Ingeniería de seguridad y fiabilidad
                  </a>{" "}
                  para ver el flujo completo.
                  <br />
                  <br />
                  Además, todos los registros pueden exportarse a cualquier
                  sistema externo (VictoriaLogs, SigNoz, Graylog, etc.) mediante
                  el estándar OpenTelemetry. De forma predeterminada, los
                  registros (incluidos los de auditoría) también se escriben en
                  archivos locales, así que los registros de auditoría no pueden
                  perderse. Consulte la{" "}
                  <a
                    href="/es/advanced-config#logging"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    configuración avanzada
                  </a>{" "}
                  aquí.
                </>
              }
            />
            <FaqItem
              number="5"
              question="¿Cómo configuro y ejecuto mi primera tarea de respaldo en Databasus?"
              answer={
                <>
                  Para iniciar su primera copia de seguridad con Databasus,
                  inicie sesión en el panel, haga clic en New Backup y
                  seleccione un intervalo: cada hora, diario, semanal, mensual o
                  cron. Después indique la hora exacta de ejecución (por
                  ejemplo, 02:30 para horas de poca actividad).
                  <br />
                  <br />
                  A continuación introduzca el host de su PostgreSQL, el número
                  de puerto, el nombre de la base de datos, las credenciales y
                  la preferencia de SSL. Elija adónde debe enviarse el archivo
                  (ruta local, bucket de S3, Azure Blob Storage, FTP,
                  etc.). <br />
                  <br />
                  Si lo necesita, añada canales de notificación como correo,
                  Slack, Telegram o un webhook y haga clic en Save. Databasus
                  valida los datos al instante, activa la programación, ejecuta
                  la primera tarea y envía el estado en vivo. Así podrá
                  restaurar con un toque cuando la copia esté completa.
                </>
              }
            />
            <FaqItem
              number="6"
              question="¿Cuál es el nivel de adopción de Databasus?"
              answer="Databasus es hoy la herramienta de código abierto para copias de seguridad de PostgreSQL con mayor adopción. A 17 de junio de 2026, DBA, ingenieros DevOps, desarrolladores y equipos de todo el mundo la han descargado más de 1.800.000 veces en Docker. Con más de 8.500 estrellas en GitHub, supera a pgBackRest (~4.200 estrellas, disponible desde 2014) y a WAL-G (~4.100 estrellas, disponible desde 2017). Databasus se lanzó en 2025 y superó a ambos en su primer año."
            />
            <FaqItem
              number="7"
              question="¿En qué se diferencia Databasus de PgBackRest, Barman o pg_dump? ¿Dónde puedo leer comparativas?"
              answer={
                <>
                  Databasus apuesta por la simplicidad: ofrece una interfaz web
                  moderna para gestionar las copias de seguridad de muchas bases
                  de datos a la vez, en lugar de archivos de configuración
                  complejos y herramientas de línea de comandos. A diferencia de
                  los scripts de pg_dump puros, incluye programación integrada,
                  compresión, múltiples destinos de almacenamiento, monitoreo
                  del estado y notificaciones en tiempo real, todo gestionado
                  desde una interfaz web sencilla.
                  <br />
                  <br />A la vez, a diferencia de pgBackRest y WAL-G, Databasus
                  realiza copias físicas, incrementales y de WAL sobre el
                  mecanismo nativo de PostgreSQL 17, así que no reinventa su
                  propio motor de respaldo. Se conecta a sus bases de datos de
                  forma remota y alcanza redes cerradas mediante un túnel SSH al
                  servidor o a un bastión, de modo que las bases de datos sin
                  exposición pública también pueden respaldarse y gestionarse
                  desde un único panel.{" "}
                  <a
                    href="/es/faq/#pitr"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Lea cómo están implementadas las copias físicas y PITR
                  </a>
                  .
                  <br />
                  <br />
                  Tenemos páginas de comparación detalladas para las
                  herramientas de respaldo más populares:{" "}
                  <a
                    href="/es/pgdump-alternative"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pg_dump
                  </a>
                  ,{" "}
                  <a
                    href="/es/databasus-vs-pgbackrest"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pgBackRest
                  </a>
                  ,{" "}
                  <a
                    href="/es/databasus-vs-barman"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs Barman
                  </a>
                  ,{" "}
                  <a
                    href="/es/databasus-vs-wal-g"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs WAL-G
                  </a>{" "}
                  y{" "}
                  <a
                    href="/es/databasus-vs-pgbackweb"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pgBackWeb
                  </a>
                  . Cada comparación explica las diferencias clave, los pros y
                  los contras, y le ayuda a elegir la herramienta adecuada para
                  sus necesidades.
                </>
              }
            />
            <FaqItem
              number="8"
              question="¿Databasus cuenta con el apoyo de los programas OSS de Anthropic y OpenAI?"
              answer={
                <>
                  Sí, estamos orgullosos de que Databasus haya sido reconocido
                  como un proyecto de código abierto valioso por dos de las
                  empresas de IA líderes del mundo. En marzo de 2026, Databasus
                  fue aceptado tanto en{" "}
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Claude for Open Source
                  </a>{" "}
                  de Anthropic como en{" "}
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Codex for Open Source
                  </a>{" "}
                  de OpenAI. Es una confirmación independiente de fiabilidad
                  para nosotros: el proyecto fue evaluado y reconocido como
                  infraestructura crítica que merece apoyo.{" "}
                  <a
                    href="/es/faq#oss-programs"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Leer más →
                  </a>
                </>
              }
            />
            <FaqItem
              number="9"
              question="¿Es Databasus una alternativa a pg_dump?"
              answer="No exactamente. Databasus se centra en la recuperación ante desastres con RTO y RPO bajos, así que está más cerca de ser una alternativa a pgBackRest o WAL-G: intenta hacer la recuperación ante desastres tan simple como pg_dump. Dicho esto, para copias lógicas sí sirve como alternativa a pg_dump y usa pg_dump por debajo, añadiendo una interfaz web fácil de usar, programación automática, múltiples destinos de almacenamiento, notificaciones en tiempo real, monitoreo del estado y cifrado de las copias. Las copias lógicas también están disponibles para MySQL, MariaDB y MongoDB."
            />
            <FaqItem
              number="10"
              question="¿Qué bases de datos admite Databasus?"
              answer={
                <>
                  Databasus admite PostgreSQL, MySQL, MariaDB y MongoDB. Sin
                  embargo, Databasus fue creado originalmente para PostgreSQL y
                  mantiene su enfoque principal en él: soporte excelente al 100%
                  y la máxima eficiencia para las copias de seguridad de
                  PostgreSQL.
                  <br />
                  <br />
                  Aunque MySQL, MariaDB y MongoDB están soportados, PostgreSQL
                  sigue siendo la prioridad central, con las funciones más
                  optimizadas y desarrollo continuo.
                  <br />
                  <br />
                  Por ejemplo, Databasus ofrece soporte nativo de copias físicas
                  y de WAL para la recuperación ante desastres de PostgreSQL. En
                  realidad, Databasus es una herramienta de copia de seguridad
                  de PostgreSQL y las demás bases de datos son solo extensiones.
                </>
              }
            />
            <FaqItem
              number="11"
              question="¿Qué tipos de copia de seguridad admite Databasus?"
              answer={
                <>
                  Databasus admite copias físicas, completas, incrementales, de
                  WAL y lógicas, así que sirve tanto para quien quiere volcados
                  lógicos simples como para quien necesita una herramienta
                  sólida de recuperación ante desastres.
                  <ul className="list-disc list-inside mt-3 space-y-2">
                    <li>
                      <strong>Física</strong>: copia a nivel de archivos de todo
                      el clúster de la base de datos. Copia y restauración más
                      rápidas para conjuntos de datos grandes que los volcados
                      lógicos. Construida sobre el mecanismo nativo de respaldo
                      de PostgreSQL 17, así que nos apoyamos en las herramientas
                      probadas de PostgreSQL en lugar de reinventarlas
                    </li>
                    <li>
                      <strong>Completa</strong>: una copia íntegra y
                      autocontenida del clúster, la base de la que parte cada
                      cadena de respaldos
                    </li>
                    <li>
                      <strong>Incremental</strong>: guarda solo lo que cambió
                      desde la copia anterior, así que los respaldos se
                      mantienen pequeños y rápidos
                    </li>
                    <li>
                      <strong>Streaming de WAL</strong>: captura de forma
                      continua el flujo de escritura de la base de datos, lo que
                      habilita la recuperación a un punto en el tiempo (PITR).
                      Pensado para recuperación ante desastres y una pérdida de
                      datos casi nula
                    </li>
                    <li>
                      <strong>Lógica</strong>: volcado nativo de la base de
                      datos en su formato binario específico del motor.
                      Comprimido y transmitido directamente al almacenamiento
                      sin archivos intermedios
                    </li>
                  </ul>
                  <br />
                  Las copias físicas, incrementales y de WAL se apoyan en el
                  mecanismo nativo de PostgreSQL 17, así que requieren
                  PostgreSQL 17 o superior; en versiones anteriores solo están
                  disponibles las copias lógicas. Es una decisión deliberada: la
                  mayoría de las bases de datos en producción ya funcionan con
                  PostgreSQL 17 o superior, y en unos dos años las versiones más
                  antiguas llegan al final de su vida útil. Databasus aspira a
                  convertirse en la herramienta de respaldo estándar para bases
                  de datos desde PostgreSQL 17 en adelante.
                  <br />
                  <br />
                  Todas estas copias pueden ejecutarse por un túnel SSH si
                  necesita conexiones no públicas, de modo que la base de datos
                  nunca tiene que exponerse públicamente. El túnel SSH viene
                  integrado.
                </>
              }
            />
            <FaqItem
              number="12"
              question="¿Cómo se utiliza la IA en el desarrollo de Databasus?"
              answer={
                <>
                  <p>
                    La IA nos ayuda a desarrollar Databasus siguiendo un proceso de ingeniería definido. Las
                    personas definen el rumbo y toman las decisiones finales.{" "}
                    <strong>Una persona lee cada línea de código antes de integrar un cambio.</strong>
                  </p>
                  <p className="mt-3">
                    Agentes independientes revisan la corrección, la arquitectura, las pruebas y la documentación en
                    sesiones separadas de solo lectura. El análisis estático estricto y las pruebas con bases de
                    datos reales complementan sus revisiones. Las comprobaciones obligatorias de CI deben superarse
                    antes de publicar.{" "}
                    <a
                      href={`${getLocalizedHref("es", "faq")}#ai-usage`}
                      className="text-blue-400 hover:text-blue-600"
                    >
                      Conozca el proceso de desarrollo
                    </a>
                    .
                  </p>
                </>
              }
            />
            <FaqItem
              number="13"
              question="¿Cómo puedo unirme a la comunidad de Databasus?"
              answer={
                <>
                  Puede unirse a nuestra gran comunidad de desarrolladores, DBA
                  e ingenieros DevOps en{" "}
                  <a
                    href="https://t.me/databasus_community"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    t.me/databasus_community
                  </a>
                  . La comunidad es un gran lugar para hacer preguntas,
                  compartir experiencias, obtener ayuda con la configuración y
                  estar al día de las últimas funciones y versiones.
                </>
              }
            />
            <FaqItem
              number="14"
              question="¿Cuál es el nivel de adopción de Databasus?"
              answer={
                <>
                  Databasus supera los 1,8 millones de descargas en Docker y las
                  8,5k estrellas en GitHub. Como comparación, pgBackRest y WAL-G
                  rondan las 4,2k estrellas cada uno y Barman está en torno a
                  las 3,1k, lo que convierte a Databasus en la herramienta de
                  respaldo de bases de datos más popular de GitHub.
                  <br />
                  <br />
                  Ha sido aceptado en los programas de código abierto de
                  Anthropic y OpenAI como proyecto importante y crítico. Hoy
                  Databasus lo usan empresas, equipos e ingenieros DevOps,
                  respaldado por una comunidad grande y activa.
                  <br />
                  <br />
                  Databasus se desarrolla y se usa desde 2023, y es de código
                  abierto con uso extendido desde principios de 2025. Lleva
                  tiempo en producción real, así que está probado en muchos
                  casos límite. Un punto clave: Databasus no inventa formas
                  propias de respaldar sus datos; se apoya en la implementación
                  nativa y probada de PostgreSQL en lugar de construir sus
                  propias soluciones improvisadas para los casos límite.
                  <br />
                  <br />
                  Nuestro objetivo es convertirnos en la herramienta de respaldo
                  estándar para PostgreSQL desde la versión 17 en adelante.
                  Databasus es la primera herramienta de respaldo construida
                  sobre el protocolo de respaldo nativo, eficiente y ya estándar
                  de PostgreSQL, en lugar de escribir sus propias
                  implementaciones.
                </>
              }
            />
            <FaqItem
              number="15"
              question="Olvidé el correo o la contraseña del administrador"
              answer={
                <>
                  La primera cuenta creada en la instancia es la que la
                  administra. Si no recuerda cuál es esa dirección, ejecute
                  docker exec -it databasus ./main --list-admins en el servidor
                  donde se ejecuta Databasus: nombra cada cuenta de
                  administrador con su correo electrónico, nombre visible, fecha
                  de creación y estado de actividad, y no imprime ninguna
                  contraseña. Para establecer una nueva contraseña para esa
                  cuenta, ejecute docker exec -it databasus ./main
                  --new-password=&quot;YourNewSecurePassword123&quot;
                  --email=&quot;owner@example.com&quot;. Una instancia creada
                  antes de que la primera cuenta la administrara todavía lleva
                  la dirección provisional admin hasta que su propietario la
                  reemplace. Si además la instancia exige un código al iniciar
                  sesión y el servidor de correo ha dejado de entregarlo,
                  ejecute docker exec -it databasus ./main --disable-2fa para
                  que deje de pedirlo. Los tres comandos están documentados en
                  <a
                    href="/es/password"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    la página de contraseña
                  </a>
                  .
                </>
              }
            />
          </div>
        </div>
      </section>

      <LandingFooterComponent lang="es" />
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
