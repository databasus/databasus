import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "Copia de seguridad de MongoDB",
  description:
    "Herramienta gratuita y de código abierto para copias de seguridad programadas de MongoDB. Automatice mongodump con interfaz web, guarde los archivos en S3, Azure Blob Storage o localmente. Reciba notificaciones por Slack, Discord, Telegram. Cifrado AES-256 para datos BSON.",
  keywords:
    "copia de seguridad de MongoDB, alternativa a mongodump, automatización de copias de seguridad de MongoDB, herramienta de copia de seguridad de MongoDB, copia de seguridad programada de MongoDB, copia de seguridad de MongoDB en la nube, copia de seguridad de MongoDB en S3, copia de seguridad de MongoDB en Docker, cifrado de copias de seguridad de MongoDB, copia de seguridad de MongoDB Atlas, copia de seguridad de replica set, copia de seguridad de base de datos documental, copia de seguridad BSON, copia de seguridad NoSQL",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("es", "mongodb-backup"),
    languages: getLanguageAlternates("mongodb-backup"),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("es", "mongodb-backup"),
    title: "Copia de seguridad de MongoDB",
    description:
      "Herramienta gratuita y de código abierto para copias de seguridad programadas de MongoDB. Automatice mongodump con interfaz web, almacenamiento en la nube, notificaciones y cifrado.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Interfaz del panel de Databasus para gestionar copias de seguridad de MongoDB",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary_large_image",
    title: "Copia de seguridad de MongoDB",
    description:
      "Herramienta gratuita y de código abierto para copias de seguridad programadas de MongoDB. Automatice mongodump con interfaz web, almacenamiento en la nube, notificaciones y cifrado.",
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
              "Herramienta gratuita y de código abierto para copias de seguridad programadas de MongoDB. Automatice mongodump con interfaz web, almacenamiento en la nube, notificaciones y cifrado.",
            url: "https://databasus.com/es/mongodb-backup/",
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
              "Copias de seguridad programadas de MongoDB mediante mongodump",
              "Múltiples destinos de almacenamiento (S3, Azure Blob Storage, FTP, SFTP, NAS, etc.)",
              "Notificaciones en tiempo real (Slack, Telegram, Discord, Webhook, correo electrónico, etc.)",
              "Monitorización del estado de la conexión a MongoDB",
              "Autoalojado mediante Docker",
              "De código abierto y gratuito",
              "Soporte para MongoDB 4, 5, 6, 7, 8 y 9",
              "Compresión de archivos BSON con gzip",
              "Cifrado AES-256-GCM de los archivos de respaldo",
              "Soporte para MongoDB Atlas y replica sets",
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
                name: "¿Qué es Databasus y cómo realiza copias de seguridad de bases de datos MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus es una herramienta de respaldo autoalojada con licencia Apache 2.0 que usa mongodump internamente para crear copias de seguridad consistentes de MongoDB. Envuelve mongodump con una interfaz web moderna, programación automatizada, integración con almacenamiento en la nube, notificaciones en tiempo real y cifrado AES-256-GCM, así que no necesita scripts de shell personalizados.",
                },
              },
              {
                "@type": "Question",
                name: "¿Databasus soporta replica sets de MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Sí, Databasus soporta completamente los replica sets de MongoDB. Puede conectarse a cualquier miembro de un replica set usando el formato estándar de URI de conexión de MongoDB. Databasus leerá desde el nodo especificado, lo que le permite respaldar desde nodos secundarios para reducir la carga del primario.",
                },
              },
              {
                "@type": "Question",
                name: "¿Puedo respaldar bases de datos de MongoDB Atlas con Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Sí, Databasus funciona sin problemas con MongoDB Atlas. Como Databasus usa respaldos lógicos mediante mongodump, solo requiere credenciales de conexión MongoDB estándar, sin permisos ni roles especiales de Atlas. Basta con proporcionar su cadena de conexión de Atlas y Databasus se encarga del resto.",
                },
              },
              {
                "@type": "Question",
                name: "¿Qué versiones de MongoDB soporta Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus soporta las versiones 4, 5, 6, 7, 8 y 9 de MongoDB. Todos los respaldos usan la herramienta nativa mongodump con las opciones --archive y --gzip para obtener archivos BSON comprimidos y eficientes que pueden restaurarse con mongorestore.",
                },
              },
              {
                "@type": "Question",
                name: "¿Cómo protege Databasus las credenciales y las copias de seguridad de MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus implementa seguridad en varias capas: (1) Todas las contraseñas y cadenas de conexión de MongoDB se cifran con AES-256-GCM antes de almacenarse; (2) Cada archivo de respaldo se cifra con una clave única derivada de la clave maestra, el ID del respaldo y una sal aleatoria; (3) Los URI de conexión se pasan de forma segura a mongodump, sin exponerse nunca en los registros ni en la salida de la línea de comandos.",
                },
              },
              {
                "@type": "Question",
                name: "¿Databasus soporta copias de seguridad incrementales de MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus se centra en respaldos lógicos completos con mongodump, no en respaldos incrementales. Para la mayoría de los casos de uso, los respaldos completos programados (por hora, día o semana) proporcionan puntos de recuperación suficientes. MongoDB Atlas ya ofrece recuperación nativa a un punto en el tiempo, y los respaldos incrementales externos no pueden restaurarse fácilmente en clústeres de Atlas.",
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        lang="es"
        sectionLinks={[
          { href: "#features", label: "Funciones" },
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
              Herramienta de copia de seguridad de MongoDB
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus es una herramienta gratuita, de código abierto y
              autoalojada para respaldar bases de datos documentales MongoDB.
              Automatice mongodump con programación y guarde los archivos BSON
              en S3, Azure Blob Storage o localmente. Reciba avisos por Slack, Discord
              o Telegram cuando los respaldos terminen
            </p>

            <div className="flex flex-col sm:flex-row sm:flex-wrap items-center justify-center gap-2 sm:gap-2 max-w-[400px] mx-auto pb-0 sm:pb-[50px] lg:pb-0">
              <a
                href="#installation"
                className="w-full sm:flex-1 inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-3"
              >
                Autoalojar con Docker
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
                alt="Interfaz del panel de Databasus para gestionar copias de seguridad de MongoDB"
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
              <span className="text-sm font-medium">Resumen</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Funciones para la copia de seguridad de MongoDB
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus envuelve mongodump con funciones de nivel empresarial:
              programación automatizada, integración con almacenamiento en la
              nube, notificaciones en tiempo real y cifrado AES-256-GCM. Ideal
              para desarrolladores y equipos de DevOps que gestionan bases de
              datos documentales y colecciones de MongoDB
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
                Volcados de MongoDB programados
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Configuración de copias de seguridad programadas de MongoDB"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Programe mongodump en horarios óptimos, cuando la carga de su
                aplicación es baja. Elija intervalos por hora, día, semana o
                mes, o use expresiones cron para un control preciso de los
                horarios
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Monitorización del estado de MongoDB
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Comprobaciones de estado de MongoDB"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Supervise la disponibilidad de la conexión a MongoDB con
                comprobaciones de estado configurables. Reciba avisos cuando su
                base de datos o su replica set quede inaccesible
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                Configure los intervalos de comprobación (cada minuto, cada 5
                minutos, etc.) y los umbrales de fallos antes de marcar la base
                de datos como no disponible
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Guarde los archivos BSON donde quiera
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Guarde los archivos de respaldo de MongoDB localmente, en
                almacenamiento compatible con S3, Azure Blob Storage, FTP, NAS u
                otros destinos. Sus datos documentales permanecen bajo su
                control.{" "}
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
                  alt="Destinos de almacenamiento de copias de seguridad de MongoDB"
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
                Notificaciones de respaldos
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Reciba alertas cuando los respaldos de MongoDB terminen o
                fallen. Envíe notificaciones al chat de su equipo de DevOps o a
                canales personales.{" "}
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
                  alt="Notificaciones de copias de seguridad de MongoDB"
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
                Autoalojado con Docker
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Ejecute Databasus en su propia infraestructura. Todas las
                cadenas de conexión de MongoDB y los datos de respaldo
                permanecen en servidores que usted controla. Despliegue en unos
                2 minutos mediante script, Docker o Kubernetes
              </p>

              <div className="flex">
                <img
                  src="/images/index/feature-deploy.svg"
                  alt="Despliegue con Docker"
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
                De código abierto y gratuito
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Databasus es totalmente de código abierto con licencia Apache
                2.0. Inspeccione cada línea de código y haga un fork. Gratuito
                para uso personal y empresarial
              </p>
              <div>
                <img
                  src="/images/index/feature-github.svg"
                  alt="Código abierto en GitHub"
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
                  Versiones de MongoDB soportadas
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Se soportan MongoDB 4, 5, 6, 7, 8 y 9. Databasus usa la
                  herramienta nativa mongodump para cada versión, lo que
                  garantiza compatibilidad total con su base de datos documental
                </p>

                <div>
                  <img
                    src="/images/index/database-mongodb.svg"
                    alt="Versiones de MongoDB"
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
                  Seguridad
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Las cadenas de conexión de MongoDB se cifran con AES-256-GCM
                  antes de almacenarse. Cada archivo BSON se cifra con una clave
                  única. Las credenciales se pasan de forma segura a mongodump,
                  sin exponerse nunca en los registros.{" "}
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
                    alt="Seguridad de las copias de seguridad de MongoDB"
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
                  alt="Gestión de accesos a copias de seguridad de MongoDB"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Controle quién puede ver o gestionar las bases de datos MongoDB.
                Cree espacios de trabajo para distintos proyectos. Asigne roles
                de lector, editor o administrador.{" "}
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
                  alt="Registros de auditoría de copias de seguridad de MongoDB"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Rastree todas las actividades: descargas de respaldos, cambios
                de programación, actualizaciones de configuración. Vea quién
                hizo qué y cuándo, para el cumplimiento normativo y la
                trazabilidad.{" "}
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
                Las cadenas de conexión de MongoDB se cifran con AES-256-GCM
                antes de almacenarse. Cada archivo BSON se cifra con una clave
                única. Las credenciales se pasan de forma segura a mongodump,
                sin exponerse nunca en los registros.{" "}
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
                  alt="Seguridad de las copias de seguridad de MongoDB"
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
                  Funciona con MongoDB Atlas y con instalaciones autoalojadas
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus se conecta a bases de datos MongoDB alojadas en la
                  nube, incluidas MongoDB Atlas, AWS DocumentDB y despliegues
                  autoalojados. Como usa respaldos lógicos mediante mongodump,
                  solo necesita credenciales de conexión estándar, sin permisos
                  de nube especiales ni acceso al sistema de archivos
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
                <span className="text-sm font-medium">
                  Construido sobre mongodump
                </span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Cómo funciona la copia de seguridad de MongoDB
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Databasus usa <strong>mongodump</strong> internamente, la
                  utilidad oficial de respaldo de MongoDB. Cuando inicia un
                  respaldo, Databasus ejecuta mongodump con parámetros
                  optimizados:
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --archive
                    </code>{" "}
                    para una salida BSON en un solo archivo en lugar de una
                    estructura de directorios
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --gzip
                    </code>{" "}
                    para archivos comprimidos que reducen el tamaño de
                    almacenamiento y transferencia
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --db
                    </code>{" "}
                    para respaldar bases de datos concretas de su instancia de
                    MongoDB
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --uri
                    </code>{" "}
                    para un manejo seguro de la cadena de conexión con
                    autenticación
                  </li>
                </ul>

                <p className="text-gray-400">
                  El flujo del respaldo se canaliza directamente al
                  almacenamiento configurado y puede cifrarse con AES-256-GCM
                  antes de escribirse. Este enfoque minimiza la E/S de disco y
                  funciona de forma eficiente con colecciones grandes.
                </p>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Versiones de MongoDB soportadas:
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
                  alt="Base de datos MongoDB"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Respaldo oficial de MongoDB con mongodump, con compresión
                  gzip, cifrado y almacenamiento en la nube
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
                <span className="text-sm font-medium">Primeros pasos</span>
              </div>

              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
                ¿Cómo instalarlo?
              </h2>

              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                Databasus admite varios métodos de instalación. Despliegue en su
                VPS, en su máquina local o en un clúster de Kubernetes en unos 2
                minutos. La misma instalación sirve para respaldos de MongoDB,
                PostgreSQL, MySQL y MariaDB
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
              Preguntas sobre la copia de seguridad de MongoDB
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Preguntas frecuentes sobre el respaldo de bases de datos
              documentales MongoDB con Databasus. Si tiene otras preguntas,
              únase a nuestra comunidad en Telegram
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="¿Qué es Databasus y cómo realiza copias de seguridad de bases de datos MongoDB?"
              answer="Databasus es una herramienta de respaldo autoalojada con licencia Apache 2.0 que usa mongodump internamente para crear copias de seguridad consistentes de MongoDB. Envuelve mongodump con una interfaz web moderna, programación automatizada, integración con almacenamiento en la nube (S3, Azure Blob Storage, FTP), notificaciones en tiempo real (Slack, Discord, Telegram) y cifrado AES-256-GCM, así que no necesita scripts de shell personalizados ni tareas cron."
            />
            <FaqItem
              number="2"
              question="¿Databasus soporta replica sets de MongoDB?"
              answer="Sí, Databasus soporta completamente los replica sets de MongoDB. Puede conectarse a cualquier miembro de un replica set usando el formato estándar de URI de conexión de MongoDB con las opciones de replica set. Databasus leerá desde el nodo especificado, lo que le permite respaldar desde nodos secundarios para reducir la carga del primario. Esto resulta especialmente útil en entornos de producción donde se quiere evitar afectar al rendimiento del nodo primario."
            />
            <FaqItem
              number="3"
              question="¿Puedo respaldar bases de datos de MongoDB Atlas con Databasus?"
              answer="Sí, Databasus funciona sin problemas con MongoDB Atlas. Como Databasus usa respaldos lógicos mediante mongodump, solo requiere credenciales de conexión MongoDB estándar, sin permisos especiales de Atlas ni roles administrativos; en la lista de IP permitidas solo hace falta la de su servidor de Databasus. Basta con proporcionar su cadena de conexión de Atlas (disponible en el panel de Atlas) y Databasus se encarga del resto."
            />
            <FaqItem
              number="4"
              question="¿Qué versiones de MongoDB soporta Databasus?"
              answer="Databasus soporta las versiones 4, 5, 6, 7, 8 y 9 de MongoDB. Todos los respaldos usan la herramienta nativa mongodump con las opciones --archive y --gzip para obtener archivos BSON comprimidos y eficientes. Los archivos pueden restaurarse con mongorestore en cualquier versión compatible de MongoDB, lo que simplifica las migraciones entre versiones."
            />
            <FaqItem
              number="5"
              question="¿Cómo maneja Databasus las colecciones grandes de MongoDB?"
              answer="Databasus transmite la salida de mongodump directamente al destino de almacenamiento, cifrando opcionalmente el flujo en tránsito. Este enfoque evita escribir archivos temporales en disco, por lo que resulta eficiente para bases de datos con colecciones grandes. La opción --archive crea un solo archivo comprimido en lugar de una estructura de directorios, lo que reduce la sobrecarga de E/S y simplifica la gestión del almacenamiento."
            />
            <FaqItem
              number="6"
              question="¿Puedo respaldar clústeres fragmentados (sharded) de MongoDB con Databasus?"
              answer={
                <>
                  Actualmente, Databasus se centra en respaldar bases de datos
                  MongoDB individuales, no en respaldos coordinados de clústeres
                  fragmentados.
                  <br />
                  <br />
                  Para clústeres fragmentados, puede:
                  <br />
                  <br />
                  • Respaldar cada shard por separado conectándose a los replica
                  sets de cada shard
                  <br />
                  • Respaldar a través de un router mongos (aunque esto puede
                  afectar al rendimiento)
                  <br />
                  <br />
                  Para clústeres fragmentados en producción, considere los
                  respaldos nativos de MongoDB Atlas o mongodump con --oplog
                  para obtener consistencia a un punto en el tiempo entre
                  shards.
                </>
              }
            />
            <FaqItem
              number="7"
              question="¿Cómo protege Databasus las credenciales y las copias de seguridad de MongoDB?"
              answer={
                <>
                  Databasus implementa seguridad en varias capas:
                  <br />
                  <br />
                  <strong>1. Cifrado de credenciales:</strong> todos los URI de
                  conexión, contraseñas y datos de autenticación de MongoDB se
                  cifran con AES-256-GCM antes de almacenarse.
                  <br />
                  <br />
                  <strong>2. Cifrado de respaldos:</strong> cada archivo BSON se
                  cifra con una clave única derivada de la clave maestra, el ID
                  del respaldo y una sal aleatoria.
                  <br />
                  <br />
                  <strong>3. Manejo seguro de credenciales:</strong> los URI de
                  conexión se pasan directamente a mongodump mediante parámetros
                  seguros, sin exponerse nunca en los registros ni en los
                  listados de procesos.
                </>
              }
            />
            <FaqItem
              number="8"
              question="¿Databasus soporta copias de seguridad incrementales de MongoDB o el seguimiento del oplog?"
              answer="Databasus se centra en respaldos lógicos completos con mongodump, no en respaldos incrementales ni en recuperación a un punto en el tiempo basada en el oplog. Para la mayoría de los casos de uso, los respaldos completos programados (por hora, día o semana) proporcionan puntos de recuperación suficientes sin la complejidad de gestionar el oplog. MongoDB Atlas ya ofrece respaldos continuos nativos con recuperación a un punto en el tiempo, y los respaldos incrementales externos no pueden restaurarse fácilmente en clústeres de Atlas."
            />
            <FaqItem
              number="9"
              question="¿Puedo restaurar copias de seguridad de MongoDB en otra versión u otro clúster?"
              answer="Sí, como Databasus crea archivos estándar de mongodump en formato BSON, puede restaurarlos en cualquier servidor MongoDB compatible: otra versión, otro proveedor de nube o una máquina de desarrollo local. Descargue el respaldo desde Databasus (se descifra automáticamente) y luego use mongorestore con las opciones --archive y --gzip. Databasus muestra el comando exacto de restauración para cada respaldo."
            />
            <FaqItem
              number="10"
              question="¿Cómo funciona la compresión de mongodump en Databasus?"
              answer="Databasus usa la opción --gzip integrada de mongodump, que comprime los datos BSON durante el proceso de volcado. Esto suele reducir el tamaño del archivo en un 60-80% en comparación con BSON sin comprimir. La compresión ocurre en el flujo de mongodump antes del cifrado opcional, de modo que tanto los archivos comprimidos como los cifrados siguen siendo eficientes. La descompresión es automática al usar mongorestore con la opción --gzip."
            />
            <FaqItem
              number="11"
              question="¿Puedo respaldar colecciones concretas de MongoDB en lugar de bases de datos completas?"
              answer="Actualmente, Databasus respalda bases de datos MongoDB completas, no colecciones individuales. Esto garantiza respaldos completos y consistentes que incluyen todas las colecciones, los índices y los metadatos. Si necesita respaldos por colección, puede crear bases de datos separadas para distintos dominios de datos, cada una con su propia programación de respaldo en Databasus."
            />
            <FaqItem
              number="12"
              question="¿Databasus funciona con MongoDB ejecutado en Docker o Kubernetes?"
              answer="Sí, Databasus se conecta a MongoDB a través de la red usando URI de conexión estándar, por lo que funciona con MongoDB sin importar dónde esté desplegado: contenedores Docker, pods de Kubernetes, máquinas virtuales o servidores físicos. Solo asegúrese de que haya conectividad de red entre Databasus y su instancia de MongoDB. En despliegues de Kubernetes puede usar nombres DNS de servicios internos o endpoints de balanceadores de carga externos."
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
