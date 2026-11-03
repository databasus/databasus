import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "Backup MySQL com suporte a MariaDB",
  description:
    "Ferramenta gratuita e open source para backups agendados de MySQL e MariaDB. Alternativa ao mysqldump com interface web, armazenamento em nuvem (S3, Google Drive), notificações (Slack, Discord, Telegram) e criptografia AES-256.",
  keywords:
    "backup MySQL, backup MariaDB, alternativa ao mysqldump, mysqldump GUI, automação de backup MySQL, ferramenta de backup MySQL, ferramenta de backup MariaDB, backup MySQL agendado, backup MySQL na nuvem, backup MySQL S3, backup MySQL Docker, criptografia de backup MySQL",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("pt", "mysql-backup"),
    languages: getLanguageAlternates("mysql-backup"),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("pt", "mysql-backup"),
    title: "Backup MySQL com suporte a MariaDB",
    description:
      "Ferramenta gratuita e open source para backups agendados de MySQL e MariaDB. Alternativa ao mysqldump com interface web, armazenamento em nuvem, notificações e criptografia.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Interface do Databasus para gestão de backups MySQL",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary_large_image",
    title: "Backup MySQL com suporte a MariaDB",
    description:
      "Ferramenta gratuita e open source para backups agendados de MySQL e MariaDB. Alternativa ao mysqldump com interface web, armazenamento em nuvem, notificações e criptografia.",
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
              "Ferramenta gratuita e open source para backups agendados de MySQL e MariaDB. Alternativa ao mysqldump com interface web, armazenamento em nuvem, notificações e criptografia.",
            url: "https://databasus.com/pt/mysql-backup/",
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
              "Backups agendados de MySQL e MariaDB",
              "Vários destinos de armazenamento (S3, Google Drive, Dropbox, SFTP, rclone, etc.)",
              "Notificações em tempo real (Slack, Telegram, Discord, Webhook, email, etc.)",
              "Monitoramento de saúde da base de dados MySQL",
              "Auto-hospedado via Docker",
              "Open source e gratuito",
              "Suporte a MySQL 5.7, 8.0, 8.4, 9, 26",
              "Suporte a MariaDB 5.5, 10, 11, 12, 13",
              "Compressão de backups e criptografia AES-256-GCM",
              "Alternativa ao mysqldump com interface web",
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
                name: "O que é o Databasus e como ele faz backup de bases de dados MySQL?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus é uma ferramenta de backup auto-hospedada, com licença Apache 2.0, que usa o mysqldump internamente para criar backups consistentes de MySQL. Por cima do mysqldump, ele adiciona uma interface web moderna, agendamento automático, integração com armazenamento em nuvem, notificações em tempo real e criptografia AES-256-GCM, dispensando scripts shell personalizados.",
                },
              },
              {
                "@type": "Question",
                name: "O Databasus é uma alternativa ou um substituto do mysqldump?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus estende o mysqldump em vez de substituí-lo. Internamente, ele executa o mysqldump com parâmetros otimizados (--single-transaction, --routines, --triggers, --events) e depois trata da compressão, criptografia e envio para o armazenamento configurado. Você fica com toda a confiabilidade do mysqldump mais agendamento, armazenamento em nuvem, notificações e recursos para equipes.",
                },
              },
              {
                "@type": "Question",
                name: "Quais versões de MySQL o Databasus suporta?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus suporta MySQL 5.7, 8.0, 8.4, 9 e 26. Para MySQL 8.0 e mais recentes, o Databasus usa compressão de rede zstd para transferências mais rápidas. Para MySQL 5.7, a compressão legada é usada automaticamente. MySQL 5.7 é suportado apenas em x86-64.",
                },
              },
              {
                "@type": "Question",
                name: "Quais versões de MariaDB o Databasus suporta?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus suporta MariaDB 5.5, 10, 11, 12 e 13. Para backups de MariaDB, o Databasus usa a ferramenta nativa mariadb-dump em vez do mysqldump, garantindo compatibilidade total com recursos e otimizações específicos do MariaDB.",
                },
              },
              {
                "@type": "Question",
                name: "Posso fazer backup de bases de dados MySQL hospedadas em AWS RDS, Google Cloud SQL ou Azure?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Sim, o Databasus funciona sem problemas com bases de dados MySQL na nuvem, incluindo AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL e PlanetScale. Como o Databasus usa backups lógicos via mysqldump, ele só precisa das credenciais padrão de conexão MySQL, sem permissões especiais de nuvem.",
                },
              },
              {
                "@type": "Question",
                name: "Como o Databasus protege as credenciais e os backups da base de dados MySQL?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus implementa segurança em várias camadas: (1) Todas as senhas e credenciais MySQL são criptografadas com AES-256-GCM antes do armazenamento; (2) Cada arquivo de backup é criptografado com uma chave única derivada da chave mestra, do ID do backup e de um salt aleatório; (3) As credenciais são passadas por arquivos .my.cnf temporários com permissões 0600, nunca expostas na linha de comando ou nos logs.",
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        lang="pt"
        sectionLinks={[
          { href: "#features", label: "Funcionalidades" },
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
              Ferramenta de backup MySQL com suporte a MariaDB
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              O Databasus é uma ferramenta gratuita, open source e
              auto-hospedada para fazer cópia de segurança de bases de dados
              MySQL e MariaDB. Agende dumps automáticos e guarde-os no S3, no
              Google Drive ou localmente. Receba notificações via Slack, Discord
              ou Telegram quando os backups terminarem
            </p>

            <div className="flex flex-col sm:flex-row sm:flex-wrap items-center justify-center gap-2 sm:gap-2 max-w-[400px] mx-auto pb-0 sm:pb-[50px] lg:pb-0">
              <a
                href="#installation"
                className="w-full sm:flex-1 inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-3"
              >
                Auto-hospedagem via Docker
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
                alt="Interface do Databasus para gestão de backups MySQL"
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
              <span className="text-sm font-medium">Visão geral</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Funcionalidades para backup MySQL
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              O Databasus adiciona ao mysqldump recursos de nível empresarial:
              agendamento automático, integração com armazenamento em nuvem,
              notificações em tempo real e criptografia AES-256-GCM. Ideal para
              desenvolvedores, equipes DevOps e organizações que administram
              bases de dados MySQL
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
                Dumps MySQL agendados
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Configuração de backups MySQL agendados"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Agende o mysqldump para os horários em que a carga da base de
                dados é baixa. Escolha intervalos de hora em hora, diários,
                semanais, mensais ou use expressões cron para controle preciso
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Monitoramento de saúde do MySQL
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Verificações de saúde do MySQL"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Monitore a disponibilidade da conexão MySQL com verificações de
                saúde configuráveis. Seja notificado quando a base de dados
                ficar inacessível ou se recuperar
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                Defina intervalos de verificação (a cada minuto, 5 minutos,
                etc.) e limites de falhas antes de marcar a base de dados como
                indisponível
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Guarde dumps MySQL em qualquer lugar
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Mantenha os arquivos de backup MySQL localmente, em
                armazenamento compatível com S3, Google Drive, Dropbox, NAS ou
                outros destinos. Os seus dados ficam sob o seu controle.{" "}
                <a
                  href="/pt/storages"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Ver todos →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-destinations.svg"
                  alt="Destinos de armazenamento de backup MySQL"
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
                Notificações de backup
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Receba alertas quando os backups MySQL terminarem ou falharem.
                Envie notificações para o chat da sua equipe DevOps ou canais
                pessoais.{" "}
                <a
                  href="/pt/notifiers"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Ver todos →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-notifications.svg"
                  alt="Notificações de backup MySQL"
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
                Auto-hospedado via Docker
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Execute o Databasus na sua própria infraestrutura. Todas as
                credenciais MySQL e os dados de backup ficam em servidores que
                você controla. Instale em cerca de 2 minutos via script, Docker
                ou Kubernetes
              </p>

              <div className="flex">
                <img
                  src="/images/index/feature-deploy.svg"
                  alt="Instalação via Docker"
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
                Open source e gratuito
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                O Databasus é totalmente open source com licença Apache 2.0.
                Inspecione cada linha de código e faça fork. Gratuito
                para uso pessoal e empresarial
              </p>
              <div>
                <img
                  src="/images/index/feature-github.svg"
                  alt="Open source no GitHub"
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
                  Versões de MySQL e MariaDB
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  MySQL 5.7, 8.0, 8.4, 9 e 26 são suportados. MariaDB 5.5, 10, 11, 12 e
                  13 são suportados com a ferramenta nativa mariadb-dump para
                  compatibilidade total
                </p>

                <div>
                  <img
                    src="/images/index/database-mysql.svg"
                    alt="Versões de MySQL"
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
                  Segurança
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  As credenciais MySQL são criptografadas com AES-256-GCM antes
                  do armazenamento. Cada arquivo de backup é criptografado com
                  uma chave única. As senhas são passadas por arquivos de
                  configuração temporários, nunca expostas na linha de comando.{" "}
                  <a
                    href="/pt/security"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Saiba mais →
                  </a>
                </p>

                <div>
                  <img
                    src="/images/index/feature-encryption.svg"
                    alt="Segurança de backup MySQL"
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
                  Gestão de acesso
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  para equipes
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-access-management.svg"
                  alt="Gestão de acesso a backups MySQL"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Controle quem pode ver ou administrar as bases de dados MySQL.
                Crie espaços de trabalho para projetos diferentes. Atribua
                papéis de visualizador, editor ou administrador.{" "}
                <a
                  href="/pt/access-management#settings"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Saiba mais →
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
                  Logs de auditoria
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  para equipes
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-audit-logs.svg"
                  alt="Logs de auditoria de backup MySQL"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Acompanhe todas as atividades: downloads de backups, mudanças de
                agendamento, atualizações de configuração. Veja quem fez o quê e
                quando, para conformidade e responsabilização.{" "}
                <a
                  href="/pt/access-management#audit-logs"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Saiba mais →
                </a>
              </p>
            </div>

            {/* Card 10: Security - Mobile/Tablet only */}
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1 lg:hidden">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                10
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Segurança
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                As credenciais MySQL são criptografadas com AES-256-GCM antes do
                armazenamento. Cada arquivo de backup é criptografado com uma
                chave única. As senhas são passadas por arquivos de configuração
                temporários, nunca expostas na linha de comando.{" "}
                <a
                  href="/pt/security"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Saiba mais →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-encryption.svg"
                  alt="Segurança de backup MySQL"
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
                  Funciona com bases de dados MySQL na nuvem
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  O Databasus se conecta a bases de dados MySQL hospedadas na
                  nuvem, incluindo AWS RDS for MySQL, Google Cloud SQL, Azure
                  Database for MySQL e PlanetScale. Como usa backups lógicos via
                  mysqldump, você só precisa das credenciais padrão de conexão,
                  sem permissões especiais de nuvem nem acesso ao sistema de
                  arquivos
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
                  Baseado no mysqldump
                </span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Backup MySQL
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  O Databasus usa o <strong>mysqldump</strong> internamente, o
                  utilitário oficial de backup do MySQL. Quando você inicia um
                  backup, o Databasus executa o mysqldump com parâmetros
                  otimizados:
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --single-transaction
                    </code>{" "}
                    para snapshots consistentes sem bloquear tabelas
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --routines
                    </code>{" "}
                    para incluir procedimentos armazenados e funções
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --triggers
                    </code>{" "}
                    para incluir os triggers da base de dados
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --events
                    </code>{" "}
                    para incluir eventos agendados
                  </li>
                </ul>

                <p className="text-gray-400">
                  Para MySQL 8.0 e mais recentes, o Databasus ativa a{" "}
                  <strong>compressão de rede zstd</strong> para transferências
                  mais rápidas em conexões lentas. O MySQL 5.7 usa a compressão
                  legada automaticamente.
                </p>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Versões de MySQL suportadas:
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
                  alt="Base de dados MySQL"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Backup MySQL oficial via mysqldump com compressão,
                  criptografia e armazenamento em nuvem
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
                  Ferramenta nativa mariadb-dump
                </span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Backup MariaDB
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Para bases de dados MariaDB, o Databasus usa a ferramenta
                  nativa <strong>mariadb-dump</strong> em vez do mysqldump. Isso
                  garante compatibilidade total com recursos e otimizações
                  específicos do MariaDB.
                </p>

                <p className="text-gray-400">
                  O mariadb-dump é executado com os mesmos parâmetros
                  otimizados: modo single-transaction para backups consistentes,
                  com procedimentos, triggers e eventos incluídos por padrão.
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>Compatibilidade total com os recursos do MariaDB</li>
                  <li>Suporte a conexões SSL/TLS</li>
                  <li>Compressão de rede para transferências mais rápidas</li>
                  <li>
                    As mesmas opções de criptografia e armazenamento do MySQL
                  </li>
                </ul>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Versões de MariaDB suportadas:
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
                  alt="Base de dados MariaDB"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Backup MariaDB nativo via mariadb-dump com compatibilidade
                  total de recursos
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
                <span className="text-sm font-medium">Comece agora</span>
              </div>

              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
                Como instalar?
              </h2>

              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                O Databasus suporta vários métodos de instalação. Instale no seu
                VPS, na sua máquina local ou em um cluster Kubernetes em cerca
                de 2 minutos. A mesma instalação funciona para backups de MySQL,
                MariaDB, PostgreSQL e MongoDB
              </p>
            </div>

            <InstallationComponent lang="pt" />
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
              Perguntas sobre backup MySQL
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Perguntas comuns sobre cópia de segurança de bases de dados MySQL
              e MariaDB com o Databasus. Se tiver outras dúvidas, junte-se à
              nossa comunidade no Telegram
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="O que é o Databasus e como ele faz backup de bases de dados MySQL?"
              answer="O Databasus é uma ferramenta de backup auto-hospedada, com licença Apache 2.0, que usa o mysqldump internamente para criar backups consistentes de MySQL. Por cima do mysqldump, ele adiciona uma interface web moderna, agendamento automático, integração com armazenamento em nuvem (S3, Google Drive, Dropbox), notificações em tempo real (Slack, Discord, Telegram) e criptografia AES-256-GCM, dispensando scripts shell personalizados e cron jobs."
            />
            <FaqItem
              number="2"
              question="O Databasus é uma alternativa ou um substituto do mysqldump?"
              answer="O Databasus estende o mysqldump em vez de substituí-lo. Internamente, ele executa o mysqldump com parâmetros otimizados (--single-transaction, --routines, --triggers, --events) e depois trata da compressão, criptografia e envio para o armazenamento configurado. Você fica com toda a confiabilidade do mysqldump mais agendamento, armazenamento em nuvem, notificações e recursos de colaboração em equipe através de uma interface web."
            />
            <FaqItem
              number="3"
              question="Quais versões de MySQL o Databasus suporta?"
              answer="O Databasus suporta MySQL 5.7, 8.0, 8.4, 9 e 26. Para MySQL 8.0 e mais recentes, o Databasus usa compressão de rede zstd para transferências mais rápidas em conexões lentas. Para MySQL 5.7, a compressão legada é usada automaticamente. MySQL 5.7 é suportado apenas em x86-64. Todas as versões têm os mesmos recursos: agendamento, criptografia, armazenamento em nuvem e notificações."
            />
            <FaqItem
              number="4"
              question="Quais versões de MariaDB o Databasus suporta?"
              answer="O Databasus suporta MariaDB 5.5, 10, 11, 12 e 13. Para backups de MariaDB, o Databasus usa a ferramenta nativa mariadb-dump em vez do mysqldump, garantindo compatibilidade total com recursos e otimizações específicos do MariaDB. A mesma interface web e os mesmos recursos de agendamento, armazenamento e notificações funcionam tanto para MySQL como para MariaDB."
            />
            <FaqItem
              number="5"
              question="Posso fazer backup de bases de dados MySQL hospedadas em AWS RDS, Google Cloud SQL ou Azure?"
              answer="Sim, o Databasus funciona sem problemas com bases de dados MySQL na nuvem, incluindo AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL e PlanetScale. Como o Databasus usa backups lógicos via mysqldump, ele só precisa das credenciais padrão de conexão MySQL (host, porta, nome de usuário, senha), sem permissões especiais de nuvem, papéis IAM ou acesso ao sistema de arquivos."
            />
            <FaqItem
              number="6"
              question="Como funciona o agendamento de backup MySQL no Databasus?"
              answer="Você pode agendar backups MySQL em intervalos de hora em hora, diários, semanais ou mensais, ou usar expressões cron para controle preciso. Escolha horários específicos (como 3 da manhã, quando o tráfego é baixo) e dias. O Databasus cuida do resto: conectar ao MySQL, executar o mysqldump, comprimir o resultado, criptografá-lo e enviá-lo para o destino de armazenamento."
            />
            <FaqItem
              number="7"
              question="Onde ficam guardados os meus backups MySQL?"
              answer="Você escolhe onde guardar os arquivos de backup MySQL: localmente no seu servidor, em armazenamento compatível com S3 (AWS S3, MinIO, Backblaze B2), Google Drive, Cloudflare R2, Dropbox, NAS via SFTP, ou qualquer destino suportado pelo rclone. Cada base de dados pode ter o seu próprio destino de armazenamento. Os arquivos de backup ficam sob o seu controle."
            />
            <FaqItem
              number="8"
              question="Como o Databasus protege as credenciais e os backups da base de dados MySQL?"
              answer={
                <>
                  O Databasus implementa segurança em várias camadas:
                  <br />
                  <br />
                  <strong>1. Criptografia de credenciais:</strong> todas as
                  senhas, tokens e strings de conexão MySQL são criptografados
                  com AES-256-GCM antes do armazenamento. A chave de
                  criptografia é guardada separada da base de dados.
                  <br />
                  <br />
                  <strong>2. Criptografia de backups:</strong> cada arquivo de
                  backup é criptografado com uma chave única derivada da chave
                  mestra, do ID do backup e de um salt aleatório.
                  <br />
                  <br />
                  <strong>3. Tratamento seguro de senhas:</strong> as senhas
                  MySQL são passadas por arquivos .my.cnf temporários com
                  permissões 0600, nunca expostas em argumentos de linha de
                  comando ou logs.
                </>
              }
            />
            <FaqItem
              number="9"
              question="Posso restaurar backups MySQL em um servidor ou provedor de nuvem diferente?"
              answer="Sim, como o Databasus cria backups lógicos padrão do mysqldump, você pode restaurá-los em qualquer servidor MySQL: outra versão, outro provedor de nuvem ou uma máquina de desenvolvimento local. Baixe o backup do Databasus (descriptografado automaticamente) e use o comando mysql para restaurar. O Databasus mostra o comando exato de restauração para cada backup."
            />
            <FaqItem
              number="10"
              question="O Databasus suporta backup de réplicas MySQL?"
              answer="Sim, você pode fazer backup de servidores réplica (slave) MySQL. O Databasus usa --single-transaction, que cria um snapshot consistente sem parar a replicação. Isso é útil para tirar a carga de backup do servidor primário. Basta apontar o Databasus para os dados de conexão da sua réplica."
            />
            <FaqItem
              number="11"
              question="Qual é a diferença entre o Databasus e o MySQL Enterprise Backup ou o Percona XtraBackup?"
              answer={
                <>
                  O MySQL Enterprise Backup e o Percona XtraBackup criam backups
                  físicos (binários) que exigem acesso direto ao sistema de
                  arquivos do diretório de dados do MySQL. Eles são mais rápidos
                  para bases de dados muito grandes, mas:
                  <br />
                  <br />
                  • Não conseguem fazer backup de MySQL na nuvem (RDS, Cloud
                  SQL, Azure)
                  <br />
                  • Exigem instalação no próprio servidor da base de dados
                  <br />
                  • Têm configuração e processo de restauração mais complexos
                  <br />
                  <br />O Databasus usa backups lógicos (mysqldump), que
                  funcionam com qualquer servidor MySQL ao qual você consiga se
                  conectar, incluindo todos os provedores de nuvem. Para a
                  maioria das bases de dados com menos de 100 GB, os backups
                  lógicos são práticos e muito mais simples de administrar.
                </>
              }
            />
            <FaqItem
              number="12"
              question="Posso fazer backup de bases de dados MySQL e PostgreSQL com a mesma instância do Databasus?"
              answer="Sim, o Databasus suporta MySQL, MariaDB, PostgreSQL e MongoDB em uma única instalação. Você pode administrar os backups de todas as suas bases de dados a partir de uma única interface web, com agendamentos, destinos de armazenamento e canais de notificação diferentes para cada uma. Isso é útil para equipes que administram infraestruturas de bases de dados variadas."
            />
          </div>
        </div>
      </section>

      <LandingFooterComponent lang="pt" />
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
