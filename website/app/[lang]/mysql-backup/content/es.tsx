import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "Copia de seguridad de MySQL con soporte para MariaDB",
  description:
    "Herramienta gratuita y de código abierto para copias de seguridad programadas de MySQL y MariaDB. Alternativa a mysqldump con interfaz web, almacenamiento en la nube (S3, Azure Blob Storage), notificaciones (Slack, Discord, Telegram) y cifrado AES-256.",
  keywords:
    "copia de seguridad de MySQL, copia de seguridad de MariaDB, alternativa a mysqldump, mysqldump GUI, automatización de copias de seguridad de MySQL, herramienta de copia de seguridad de MySQL, herramienta de copia de seguridad de MariaDB, copia de seguridad programada de MySQL, copia de seguridad de MySQL en la nube, copia de seguridad de MySQL en S3, copia de seguridad de MySQL en Docker, cifrado de copias de seguridad de MySQL",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("es", "mysql-backup"),
    languages: getLanguageAlternates("mysql-backup"),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("es", "mysql-backup"),
    title: "Copia de seguridad de MySQL con soporte para MariaDB",
    description:
      "Herramienta gratuita y de código abierto para copias de seguridad programadas de MySQL y MariaDB. Alternativa a mysqldump con interfaz web, almacenamiento en la nube, notificaciones y cifrado.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Interfaz del panel de Databasus para gestionar copias de seguridad de MySQL",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary_large_image",
    title: "Copia de seguridad de MySQL con soporte para MariaDB",
    description:
      "Herramienta gratuita y de código abierto para copias de seguridad programadas de MySQL y MariaDB. Alternativa a mysqldump con interfaz web, almacenamiento en la nube, notificaciones y cifrado.",
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

export default function MysqlBackupPage() {
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
              "Herramienta gratuita y de código abierto para copias de seguridad programadas de MySQL y MariaDB. Alternativa a mysqldump con interfaz web, almacenamiento en la nube, notificaciones y cifrado.",
            url: "https://databasus.com/es/mysql-backup/",
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
              "Copias de seguridad programadas de MySQL y MariaDB",
              "Múltiples destinos de almacenamiento (S3, Azure Blob Storage, FTP, SFTP, NAS, etc.)",
              "Notificaciones en tiempo real (Slack, Telegram, Discord, Webhook, correo electrónico, etc.)",
              "Monitorización del estado de la base de datos MySQL",
              "Autoalojado mediante Docker",
              "De código abierto y gratuito",
              "Soporte para MySQL 5.7, 8.0, 8.4, 9, 26",
              "Soporte para MariaDB 5.5, 10, 11, 12, 13",
              "Compresión de copias de seguridad y cifrado AES-256-GCM",
              "Alternativa a mysqldump con interfaz web",
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
                name: "¿Qué es Databasus y cómo realiza copias de seguridad de bases de datos MySQL?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus es una herramienta de respaldo autoalojada con licencia Apache 2.0 que usa mysqldump internamente para crear copias de seguridad consistentes de MySQL. Envuelve mysqldump con una interfaz web moderna, programación automatizada, integración con almacenamiento en la nube, notificaciones en tiempo real y cifrado AES-256-GCM, así que no necesita scripts de shell personalizados.",
                },
              },
              {
                "@type": "Question",
                name: "¿Es Databasus una alternativa o un reemplazo de mysqldump?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus extiende mysqldump en lugar de reemplazarlo. Internamente, ejecuta mysqldump con parámetros optimizados (--single-transaction, --routines, --triggers, --events) y luego se encarga de la compresión, el cifrado y la subida al almacenamiento configurado. Obtiene toda la fiabilidad de mysqldump más programación, almacenamiento en la nube, notificaciones y funciones para equipos.",
                },
              },
              {
                "@type": "Question",
                name: "¿Qué versiones de MySQL soporta Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus soporta MySQL 5.7, 8.0, 8.4, 9 y 26. Para MySQL 8.0 y versiones posteriores, Databasus usa compresión de red zstd para transferencias más rápidas. Para MySQL 5.7 se usa automáticamente la compresión heredada. MySQL 5.7 solo se soporta en x86-64.",
                },
              },
              {
                "@type": "Question",
                name: "¿Qué versiones de MariaDB soporta Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus soporta MariaDB 5.5, 10, 11, 12 y 13. Para los respaldos de MariaDB, Databasus usa la herramienta nativa mariadb-dump en lugar de mysqldump, lo que garantiza compatibilidad total con las funciones y optimizaciones específicas de MariaDB.",
                },
              },
              {
                "@type": "Question",
                name: "¿Puedo hacer copias de seguridad de bases de datos MySQL alojadas en AWS RDS, Google Cloud SQL o Azure?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Sí, Databasus funciona sin problemas con bases de datos MySQL alojadas en la nube, incluidas AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL y PlanetScale. Como Databasus usa respaldos lógicos mediante mysqldump, solo requiere credenciales de conexión MySQL estándar, sin permisos de nube especiales.",
                },
              },
              {
                "@type": "Question",
                name: "¿Cómo protege Databasus las credenciales y las copias de seguridad de MySQL?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus implementa seguridad en varias capas: (1) Todas las contraseñas y credenciales de MySQL se cifran con AES-256-GCM antes de almacenarse; (2) Cada archivo de respaldo se cifra con una clave única derivada de la clave maestra, el ID del respaldo y una sal aleatoria; (3) Las credenciales se pasan mediante archivos .my.cnf temporales con permisos 0600, sin exponerse nunca en la línea de comandos ni en los registros.",
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
          { href: "#mysql-backup", label: "MySQL" },
          { href: "#mariadb-backup", label: "MariaDB" },
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
              Herramienta de copia de seguridad de MySQL con soporte para
              MariaDB
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus es una herramienta gratuita, de código abierto y
              autoalojada para respaldar bases de datos MySQL y MariaDB.
              Programe volcados automáticos y guárdelos en S3, Azure Blob Storage o
              localmente. Reciba notificaciones por Slack, Discord o Telegram
              cuando los respaldos terminen
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
                alt="Interfaz del panel de Databasus para gestionar copias de seguridad de MySQL"
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
              Funciones para la copia de seguridad de MySQL
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus envuelve mysqldump con funciones de nivel empresarial:
              programación automatizada, integración con almacenamiento en la
              nube, notificaciones en tiempo real y cifrado AES-256-GCM. Ideal
              para desarrolladores, equipos de DevOps y organizaciones que
              gestionan bases de datos MySQL
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
                Volcados de MySQL programados
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Configuración de copias de seguridad programadas de MySQL"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Programe mysqldump en horarios óptimos, cuando la carga de la
                base de datos es baja. Elija intervalos por hora, día, semana o
                mes, o use expresiones cron para un control preciso
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Monitorización del estado de MySQL
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Comprobaciones de estado de MySQL"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Supervise la disponibilidad de la conexión a MySQL con
                comprobaciones de estado configurables. Reciba avisos cuando su
                base de datos quede inaccesible o se recupere
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                Configure los intervalos de comprobación (cada minuto, cada 5
                minutos, etc.) y los umbrales de fallos antes de marcar la base
                de datos como caída
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Guarde los volcados de MySQL donde quiera
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Guarde los archivos de respaldo de MySQL localmente, en
                almacenamiento compatible con S3, Azure Blob Storage, FTP, NAS u
                otros destinos. Sus datos permanecen bajo su control.{" "}
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
                  alt="Destinos de almacenamiento de copias de seguridad de MySQL"
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
                Reciba alertas cuando los respaldos de MySQL terminen o fallen.
                Envíe notificaciones al chat de su equipo de DevOps o a canales
                personales.{" "}
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
                  alt="Notificaciones de copias de seguridad de MySQL"
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
                credenciales de MySQL y los datos de respaldo permanecen en
                servidores que usted controla. Despliegue en unos 2 minutos
                mediante script, Docker o Kubernetes
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

            {/* Card 7: Many MySQL versions - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Many MySQL versions */}
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Versiones de MySQL y MariaDB
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Se soportan MySQL 5.7, 8.0, 8.4, 9 y 26. MariaDB 5.5, 10, 11, 12 y 13
                  se soportan con la herramienta nativa mariadb-dump para una
                  compatibilidad total
                </p>

                <div>
                  <img
                    src="/images/index/database-mysql.svg"
                    alt="Versiones de MySQL"
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
                  Las credenciales de MySQL se cifran con AES-256-GCM antes de
                  almacenarse. Cada archivo de respaldo se cifra con una clave
                  única. Las contraseñas se pasan mediante archivos de
                  configuración temporales, sin exponerse nunca en la línea de
                  comandos.{" "}
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
                    alt="Seguridad de las copias de seguridad de MySQL"
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
                  alt="Gestión de accesos a copias de seguridad de MySQL"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Controle quién puede ver o gestionar las bases de datos MySQL.
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
                  alt="Registros de auditoría de copias de seguridad de MySQL"
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
                Las credenciales de MySQL se cifran con AES-256-GCM antes de
                almacenarse. Cada archivo de respaldo se cifra con una clave
                única. Las contraseñas se pasan mediante archivos de
                configuración temporales, sin exponerse nunca en la línea de
                comandos.{" "}
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
                  alt="Seguridad de las copias de seguridad de MySQL"
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
                  Funciona con bases de datos MySQL en la nube
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus se conecta a bases de datos MySQL alojadas en la
                  nube, incluidas AWS RDS for MySQL, Google Cloud SQL, Azure
                  Database for MySQL y PlanetScale. Como usa respaldos lógicos
                  mediante mysqldump, solo necesita credenciales de conexión
                  estándar, sin permisos de nube especiales ni acceso al sistema
                  de archivos
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* MYSQL BACKUP SECTION */}
      <section
        id="mysql-backup"
        className="py-12 md:py-20 px-4 md:px-6 lg:px-0"
      >
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            {/* Left side: Info */}
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">
                  Construido sobre mysqldump
                </span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Copia de seguridad de MySQL
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Databasus usa <strong>mysqldump</strong> internamente, la
                  utilidad oficial de respaldo de MySQL. Cuando inicia un
                  respaldo, Databasus ejecuta mysqldump con parámetros
                  optimizados:
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --single-transaction
                    </code>{" "}
                    para instantáneas consistentes sin bloquear tablas
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --routines
                    </code>{" "}
                    para incluir procedimientos almacenados y funciones
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --triggers
                    </code>{" "}
                    para incluir los triggers de la base de datos
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --events
                    </code>{" "}
                    para incluir los eventos programados
                  </li>
                </ul>

                <p className="text-gray-400">
                  Para MySQL 8.0 y versiones posteriores, Databasus activa la{" "}
                  <strong>compresión de red zstd</strong> para transferencias
                  más rápidas en conexiones lentas. MySQL 5.7 usa la compresión
                  heredada automáticamente.
                </p>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Versiones de MySQL soportadas:
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 5.7
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 8.0
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 8.4
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 9
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 26
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Right side: Image */}
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mysql.svg"
                  alt="Base de datos MySQL"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Respaldo oficial de MySQL con mysqldump, con compresión,
                  cifrado y almacenamiento en la nube
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <div className="border-b border-[#ffffff20] max-w-[calc(100%-2rem)] md:max-w-[calc(100%-3rem)] lg:max-w-[1000px] 2xl:max-w-[1200px] mx-auto" />

      {/* MARIADB BACKUP SECTION */}
      <section
        id="mariadb-backup"
        className="py-12 md:py-20 px-4 md:px-6 lg:px-0"
      >
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row-reverse gap-8 lg:gap-16">
            {/* Right side: Info */}
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">
                  Herramienta nativa mariadb-dump
                </span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Copia de seguridad de MariaDB
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Para bases de datos MariaDB, Databasus usa la herramienta
                  nativa <strong>mariadb-dump</strong> en lugar de mysqldump.
                  Esto garantiza compatibilidad total con las funciones y
                  optimizaciones específicas de MariaDB.
                </p>

                <p className="text-gray-400">
                  mariadb-dump se ejecuta con los mismos parámetros optimizados:
                  modo de transacción única para respaldos consistentes, con
                  procedimientos, triggers y eventos incluidos por defecto.
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>Compatibilidad total con las funciones de MariaDB</li>
                  <li>Soporte de conexiones SSL/TLS</li>
                  <li>Compresión de red para transferencias más rápidas</li>
                  <li>
                    Las mismas opciones de cifrado y almacenamiento que MySQL
                  </li>
                </ul>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Versiones de MariaDB soportadas:
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 5.5
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 10
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 11
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 12
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 13
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Left side: Image */}
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mariadb.svg"
                  alt="Base de datos MariaDB"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Respaldo nativo de MariaDB con mariadb-dump y compatibilidad
                  total de funciones
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
                minutos. La misma instalación sirve para respaldos de MySQL,
                MariaDB, PostgreSQL y MongoDB
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
              Preguntas sobre la copia de seguridad de MySQL
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Preguntas frecuentes sobre el respaldo de bases de datos MySQL y
              MariaDB con Databasus. Si tiene otras preguntas, únase a nuestra
              comunidad en Telegram
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="¿Qué es Databasus y cómo realiza copias de seguridad de bases de datos MySQL?"
              answer="Databasus es una herramienta de respaldo autoalojada con licencia Apache 2.0 que usa mysqldump internamente para crear copias de seguridad consistentes de MySQL. Envuelve mysqldump con una interfaz web moderna, programación automatizada, integración con almacenamiento en la nube (S3, Azure Blob Storage, FTP), notificaciones en tiempo real (Slack, Discord, Telegram) y cifrado AES-256-GCM, así que no necesita scripts de shell personalizados ni tareas cron."
            />
            <FaqItem
              number="2"
              question="¿Es Databasus una alternativa o un reemplazo de mysqldump?"
              answer="Databasus extiende mysqldump en lugar de reemplazarlo. Internamente, ejecuta mysqldump con parámetros optimizados (--single-transaction, --routines, --triggers, --events) y luego se encarga de la compresión, el cifrado y la subida al almacenamiento configurado. Obtiene toda la fiabilidad de mysqldump más programación, almacenamiento en la nube, notificaciones y funciones de colaboración en equipo mediante una interfaz web."
            />
            <FaqItem
              number="3"
              question="¿Qué versiones de MySQL soporta Databasus?"
              answer="Databasus soporta MySQL 5.7, 8.0, 8.4, 9 y 26. Para MySQL 8.0 y versiones posteriores, Databasus usa compresión de red zstd para transferencias más rápidas en conexiones lentas. Para MySQL 5.7 se usa automáticamente la compresión heredada. MySQL 5.7 solo se soporta en x86-64. Todas las versiones cuentan con las mismas funciones: programación, cifrado, almacenamiento en la nube y notificaciones."
            />
            <FaqItem
              number="4"
              question="¿Qué versiones de MariaDB soporta Databasus?"
              answer="Databasus soporta MariaDB 5.5, 10, 11, 12 y 13. Para los respaldos de MariaDB, Databasus usa la herramienta nativa mariadb-dump en lugar de mysqldump, lo que garantiza compatibilidad total con las funciones y optimizaciones específicas de MariaDB. La misma interfaz web y las mismas funciones de programación, almacenamiento y notificaciones sirven tanto para MySQL como para MariaDB."
            />
            <FaqItem
              number="5"
              question="¿Puedo hacer copias de seguridad de bases de datos MySQL alojadas en AWS RDS, Google Cloud SQL o Azure?"
              answer="Sí, Databasus funciona sin problemas con bases de datos MySQL alojadas en la nube, incluidas AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL y PlanetScale. Como Databasus usa respaldos lógicos mediante mysqldump, solo requiere credenciales de conexión MySQL estándar (host, puerto, usuario, contraseña), sin permisos de nube especiales, roles IAM ni acceso al sistema de archivos."
            />
            <FaqItem
              number="6"
              question="¿Cómo funciona la programación de respaldos de MySQL en Databasus?"
              answer="Puede programar respaldos de MySQL con intervalos por hora, día, semana o mes, o usar expresiones cron para un control preciso. Elija horas concretas (como las 3 de la madrugada, cuando el tráfico es bajo) y días. Databasus se encarga del resto: conectarse a MySQL, ejecutar mysqldump, comprimir la salida, cifrarla y subirla al destino de almacenamiento."
            />
            <FaqItem
              number="7"
              question="¿Dónde se guardan mis copias de seguridad de MySQL?"
              answer="Usted elige dónde guardar los archivos de respaldo de MySQL: localmente en su servidor, en almacenamiento compatible con S3 (AWS S3, MinIO, Backblaze B2), Azure Blob Storage, Cloudflare R2, FTP, NAS por SFTP. Cada base de datos puede tener su propio destino de almacenamiento. Sus archivos de respaldo permanecen bajo su control."
            />
            <FaqItem
              number="8"
              question="¿Cómo protege Databasus las credenciales y las copias de seguridad de MySQL?"
              answer={
                <>
                  Databasus implementa seguridad en varias capas:
                  <br />
                  <br />
                  <strong>1. Cifrado de credenciales:</strong> todas las
                  contraseñas, tokens y cadenas de conexión de MySQL se cifran
                  con AES-256-GCM antes de almacenarse. La clave de cifrado se
                  guarda separada de la base de datos.
                  <br />
                  <br />
                  <strong>2. Cifrado de respaldos:</strong> cada archivo de
                  respaldo se cifra con una clave única derivada de la clave
                  maestra, el ID del respaldo y una sal aleatoria.
                  <br />
                  <br />
                  <strong>3. Manejo seguro de contraseñas:</strong> las
                  contraseñas de MySQL se pasan mediante archivos .my.cnf
                  temporales con permisos 0600, sin exponerse nunca en los
                  argumentos de la línea de comandos ni en los registros.
                </>
              }
            />
            <FaqItem
              number="9"
              question="¿Puedo restaurar copias de seguridad de MySQL en otro servidor u otro proveedor de nube?"
              answer="Sí, como Databasus crea respaldos lógicos estándar de mysqldump, puede restaurarlos en cualquier servidor MySQL: otra versión, otro proveedor de nube o una máquina de desarrollo local. Descargue el respaldo desde Databasus (se descifra automáticamente) y luego use el comando mysql para restaurarlo. Databasus muestra el comando exacto de restauración para cada respaldo."
            />
            <FaqItem
              number="10"
              question="¿Databasus permite respaldar réplicas (esclavos) de replicación de MySQL?"
              answer="Sí, puede respaldar servidores réplica (esclavos) de MySQL. Databasus usa --single-transaction, que crea una instantánea consistente sin detener la replicación. Esto resulta útil para descargar el trabajo de respaldo de su servidor primario. Basta con apuntar Databasus a los datos de conexión de su réplica."
            />
            <FaqItem
              number="11"
              question="¿En qué se diferencia Databasus de MySQL Enterprise Backup o Percona XtraBackup?"
              answer={
                <>
                  MySQL Enterprise Backup y Percona XtraBackup crean respaldos
                  físicos (binarios) que requieren acceso directo al sistema de
                  archivos del directorio de datos de MySQL. Son más rápidos
                  para bases de datos muy grandes, pero:
                  <br />
                  <br />
                  • No pueden respaldar MySQL alojado en la nube (RDS, Cloud
                  SQL, Azure)
                  <br />
                  • Requieren instalarse en el propio servidor de base de datos
                  <br />
                  • Tienen un proceso de configuración y restauración más
                  complejo
                  <br />
                  <br />
                  Databasus usa respaldos lógicos (mysqldump), que funcionan con
                  cualquier servidor MySQL al que pueda conectarse, incluidos
                  todos los proveedores de nube. Para la mayoría de las bases de
                  datos de menos de 100 GB, los respaldos lógicos son prácticos
                  y mucho más sencillos de gestionar.
                </>
              }
            />
            <FaqItem
              number="12"
              question="¿Puedo respaldar bases de datos MySQL y PostgreSQL con la misma instancia de Databasus?"
              answer="Sí, Databasus soporta MySQL, MariaDB, PostgreSQL y MongoDB en una sola instalación. Puede gestionar los respaldos de todas sus bases de datos desde una única interfaz web, con distintas programaciones, destinos de almacenamiento y canales de notificación para cada una. Esto resulta útil para equipos que gestionan infraestructuras de bases de datos diversas."
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
