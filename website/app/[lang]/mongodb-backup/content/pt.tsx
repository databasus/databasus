import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "Backup MongoDB",
  description:
    "Ferramenta gratuita e open source para backups agendados de MongoDB. Automatize o mongodump com interface web, guarde os arquivos no S3, no Azure Blob Storage ou localmente. Receba notificações via Slack, Discord, Telegram. Criptografia AES-256 para dados BSON.",
  keywords:
    "backup MongoDB, alternativa ao mongodump, automação de backup MongoDB, ferramenta de backup MongoDB, backup MongoDB agendado, backup MongoDB na nuvem, backup MongoDB S3, backup MongoDB Docker, criptografia de backup MongoDB, backup MongoDB Atlas, backup de replica set, backup de base de dados de documentos, backup BSON, backup NoSQL",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("pt", "mongodb-backup"),
    languages: getLanguageAlternates("mongodb-backup"),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("pt", "mongodb-backup"),
    title: "Backup MongoDB",
    description:
      "Ferramenta gratuita e open source para backups agendados de MongoDB. Automatize o mongodump com interface web, armazenamento em nuvem, notificações e criptografia.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Interface do Databasus para gestão de backups MongoDB",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary_large_image",
    title: "Backup MongoDB",
    description:
      "Ferramenta gratuita e open source para backups agendados de MongoDB. Automatize o mongodump com interface web, armazenamento em nuvem, notificações e criptografia.",
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
              "Ferramenta gratuita e open source para backups agendados de MongoDB. Automatize o mongodump com interface web, armazenamento em nuvem, notificações e criptografia.",
            url: "https://databasus.com/pt/mongodb-backup/",
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
              "Backups agendados de MongoDB via mongodump",
              "Vários destinos de armazenamento (S3, Azure Blob Storage, FTP, SFTP, NAS, etc.)",
              "Notificações em tempo real (Slack, Telegram, Discord, Webhook, email, etc.)",
              "Monitoramento de saúde da conexão MongoDB",
              "Auto-hospedado via Docker",
              "Open source e gratuito",
              "Suporte a MongoDB 4, 5, 6, 7, 8 e 9",
              "Compressão de arquivos BSON com gzip",
              "Criptografia AES-256-GCM para arquivos de backup",
              "Suporte a MongoDB Atlas e replica sets",
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
                name: "O que é o Databasus e como ele faz backup de bases de dados MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus é uma ferramenta de backup auto-hospedada, com licença Apache 2.0, que usa o mongodump internamente para criar backups consistentes de MongoDB. Por cima do mongodump, ele adiciona uma interface web moderna, agendamento automático, integração com armazenamento em nuvem, notificações em tempo real e criptografia AES-256-GCM, dispensando scripts shell personalizados.",
                },
              },
              {
                "@type": "Question",
                name: "O Databasus suporta replica sets de MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Sim, o Databasus suporta totalmente replica sets de MongoDB. Você pode se conectar a qualquer membro de um replica set usando o formato padrão de URI de conexão do MongoDB. O Databasus lê do nó especificado, permitindo fazer backup a partir de nós secundários para reduzir a carga no primário.",
                },
              },
              {
                "@type": "Question",
                name: "Posso fazer backup de bases de dados MongoDB Atlas com o Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Sim, o Databasus funciona sem problemas com o MongoDB Atlas. Como o Databasus usa backups lógicos via mongodump, ele só precisa das credenciais padrão de conexão MongoDB, sem permissões ou papéis especiais do Atlas. Basta fornecer a sua string de conexão do Atlas e o Databasus cuida do resto.",
                },
              },
              {
                "@type": "Question",
                name: "Quais versões de MongoDB o Databasus suporta?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus suporta as versões 4, 5, 6, 7, 8 e 9 do MongoDB. Todos os backups usam a ferramenta nativa mongodump com as flags --archive e --gzip, gerando arquivos BSON compactos e eficientes que podem ser restaurados com o mongorestore.",
                },
              },
              {
                "@type": "Question",
                name: "Como o Databasus protege as credenciais e os backups do MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus implementa segurança em várias camadas: (1) Todas as senhas e strings de conexão MongoDB são criptografadas com AES-256-GCM antes do armazenamento; (2) Cada arquivo de backup é criptografado com uma chave única derivada da chave mestra, do ID do backup e de um salt aleatório; (3) As URIs de conexão são passadas de forma segura ao mongodump, nunca expostas nos logs ou na saída da linha de comando.",
                },
              },
              {
                "@type": "Question",
                name: "O Databasus suporta backups incrementais de MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus foca em backups lógicos completos usando o mongodump, em vez de backups incrementais. Para a maioria dos casos, backups completos agendados (de hora em hora, diários, semanais) oferecem pontos de recuperação suficientes. O MongoDB Atlas já oferece recuperação nativa point-in-time, e backups incrementais externos não podem ser restaurados facilmente em clusters do Atlas.",
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
              Ferramenta de backup MongoDB
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              O Databasus é uma ferramenta gratuita, open source e
              auto-hospedada para fazer cópia de segurança de bases de dados de
              documentos MongoDB. Automatize o mongodump com agendamento, guarde
              os arquivos BSON no S3, no Azure Blob Storage ou localmente. Seja
              notificado via Slack, Discord ou Telegram quando os backups
              terminarem
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
                alt="Interface do Databasus para gestão de backups MongoDB"
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
              Funcionalidades para backup MongoDB
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              O Databasus adiciona ao mongodump recursos de nível empresarial:
              agendamento automático, integração com armazenamento em nuvem,
              notificações em tempo real e criptografia AES-256-GCM. Ideal para
              desenvolvedores e equipes DevOps que administram bases de dados de
              documentos e coleções MongoDB
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
                Dumps MongoDB agendados
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Configuração de backups MongoDB agendados"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Agende o mongodump para os horários em que a carga da sua
                aplicação é baixa. Escolha intervalos de hora em hora, diários,
                semanais, mensais ou use expressões cron para controle preciso
                dos horários
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Monitoramento de saúde do MongoDB
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Verificações de saúde do MongoDB"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Monitore a disponibilidade da conexão MongoDB com verificações
                de saúde configuráveis. Seja notificado quando a base de dados
                ou o replica set ficarem inacessíveis
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
                Guarde arquivos BSON em qualquer lugar
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Mantenha os arquivos de backup MongoDB localmente, em
                armazenamento compatível com S3, Azure Blob Storage, FTP, NAS ou
                outros destinos. Os seus dados de documentos ficam sob o seu
                controle.{" "}
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
                  alt="Destinos de armazenamento de backup MongoDB"
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
                Receba alertas quando os backups MongoDB terminarem ou falharem.
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
                  alt="Notificações de backup MongoDB"
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
                strings de conexão MongoDB e os dados de backup ficam em
                servidores que você controla. Instale em cerca de 2 minutos via
                script, Docker ou Kubernetes
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

            {/* Card 7: Many MongoDB versions - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Many MongoDB versions */}
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Versões de MongoDB suportadas
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  MongoDB 4, 5, 6, 7, 8 e 9 são suportados. O Databasus usa a
                  ferramenta nativa mongodump para cada versão, garantindo
                  compatibilidade total com a sua base de dados de documentos
                </p>

                <div>
                  <img
                    src="/images/index/database-mongodb.svg"
                    alt="Versões de MongoDB"
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
                  As strings de conexão MongoDB são criptografadas com
                  AES-256-GCM antes do armazenamento. Cada arquivo BSON é
                  criptografado com uma chave única. As credenciais são passadas
                  de forma segura ao mongodump, nunca expostas nos logs.{" "}
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
                    alt="Segurança de backup MongoDB"
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
                  alt="Gestão de acesso a backups MongoDB"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Controle quem pode ver ou administrar as bases de dados MongoDB.
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
                  alt="Logs de auditoria de backup MongoDB"
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
                As strings de conexão MongoDB são criptografadas com AES-256-GCM
                antes do armazenamento. Cada arquivo BSON é criptografado com
                uma chave única. As credenciais são passadas de forma segura ao
                mongodump, nunca expostas nos logs.{" "}
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
                  alt="Segurança de backup MongoDB"
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
                  Funciona com MongoDB Atlas e instalações próprias
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  O Databasus se conecta a bases de dados MongoDB hospedadas na
                  nuvem, incluindo MongoDB Atlas, AWS DocumentDB e instalações
                  auto-hospedadas. Como usa backups lógicos via mongodump, você
                  só precisa das credenciais padrão de conexão, sem permissões
                  especiais de nuvem nem acesso ao sistema de arquivos
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
                  Baseado no mongodump
                </span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Como funciona o backup MongoDB
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  O Databasus usa o <strong>mongodump</strong> internamente, o
                  utilitário oficial de backup do MongoDB. Quando você inicia um
                  backup, o Databasus executa o mongodump com parâmetros
                  otimizados:
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --archive
                    </code>{" "}
                    para saída BSON em um único arquivo em vez de uma estrutura
                    de diretórios
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --gzip
                    </code>{" "}
                    para arquivos comprimidos, reduzindo o tamanho de
                    armazenamento e de transferência
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --db
                    </code>{" "}
                    para fazer backup de bases de dados específicas da sua
                    instância MongoDB
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --uri
                    </code>{" "}
                    para tratamento seguro da string de conexão com autenticação
                  </li>
                </ul>

                <p className="text-gray-400">
                  O fluxo de backup é enviado diretamente para o armazenamento
                  configurado, com criptografia AES-256-GCM opcional antes da
                  escrita. Essa abordagem minimiza o I/O em disco e funciona de
                  forma eficiente com coleções grandes.
                </p>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Versões de MongoDB suportadas:
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
                  alt="Base de dados MongoDB"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Backup MongoDB oficial via mongodump com compressão gzip,
                  criptografia e armazenamento em nuvem
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
                de 2 minutos. A mesma instalação funciona para backups de
                MongoDB, PostgreSQL, MySQL e MariaDB
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
              Perguntas sobre backup MongoDB
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Perguntas comuns sobre cópia de segurança de bases de dados de
              documentos MongoDB com o Databasus. Se tiver outras dúvidas,
              junte-se à nossa comunidade no Telegram
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="O que é o Databasus e como ele faz backup de bases de dados MongoDB?"
              answer="O Databasus é uma ferramenta de backup auto-hospedada, com licença Apache 2.0, que usa o mongodump internamente para criar backups consistentes de MongoDB. Por cima do mongodump, ele adiciona uma interface web moderna, agendamento automático, integração com armazenamento em nuvem (S3, Azure Blob Storage, FTP), notificações em tempo real (Slack, Discord, Telegram) e criptografia AES-256-GCM, dispensando scripts shell personalizados e cron jobs."
            />
            <FaqItem
              number="2"
              question="O Databasus suporta replica sets de MongoDB?"
              answer="Sim, o Databasus suporta totalmente replica sets de MongoDB. Você pode se conectar a qualquer membro de um replica set usando o formato padrão de URI de conexão do MongoDB com as opções de replica set. O Databasus lê do nó especificado, permitindo fazer backup a partir de nós secundários para reduzir a carga no primário. Isso é especialmente útil em ambientes de produção onde você quer evitar impacto no desempenho do nó primário."
            />
            <FaqItem
              number="3"
              question="Posso fazer backup de bases de dados MongoDB Atlas com o Databasus?"
              answer="Sim, o Databasus funciona sem problemas com o MongoDB Atlas. Como o Databasus usa backups lógicos via mongodump, ele só precisa das credenciais padrão de conexão MongoDB, sem permissões especiais do Atlas, papéis administrativos ou liberação de IPs além do seu servidor Databasus. Basta fornecer a sua string de conexão do Atlas (disponível no painel do Atlas) e o Databasus cuida do resto."
            />
            <FaqItem
              number="4"
              question="Quais versões de MongoDB o Databasus suporta?"
              answer="O Databasus suporta as versões 4, 5, 6, 7, 8 e 9 do MongoDB. Todos os backups usam a ferramenta nativa mongodump com as flags --archive e --gzip, gerando arquivos BSON compactos e eficientes. Os arquivos podem ser restaurados com o mongorestore em qualquer versão compatível do MongoDB, o que simplifica migrações entre versões."
            />
            <FaqItem
              number="5"
              question="Como o Databasus lida com coleções MongoDB grandes?"
              answer="O Databasus envia a saída do mongodump diretamente para o destino de armazenamento, com criptografia opcional do fluxo em trânsito. Essa abordagem evita gravar arquivos temporários em disco, o que a torna eficiente para bases de dados com coleções grandes. A flag --archive cria um único arquivo comprimido em vez de uma estrutura de diretórios, reduzindo a sobrecarga de I/O e simplificando a gestão do armazenamento."
            />
            <FaqItem
              number="6"
              question="Posso fazer backup de clusters MongoDB com sharding usando o Databasus?"
              answer={
                <>
                  Atualmente o Databasus foca em fazer backup de bases de dados
                  MongoDB individuais, e não em backups coordenados de clusters
                  com sharding.
                  <br />
                  <br />
                  Para clusters com sharding, você pode:
                  <br />
                  <br />
                  • Fazer backup de cada shard individualmente, conectando-se
                  aos replica sets dos shards
                  <br />
                  • Fazer backup através de um roteador mongos (embora isso
                  possa afetar o desempenho)
                  <br />
                  <br />
                  Para clusters com sharding em produção, considere os backups
                  nativos do MongoDB Atlas ou o mongodump com --oplog para
                  consistência point-in-time entre os shards.
                </>
              }
            />
            <FaqItem
              number="7"
              question="Como o Databasus protege as credenciais e os backups do MongoDB?"
              answer={
                <>
                  O Databasus implementa segurança em várias camadas:
                  <br />
                  <br />
                  <strong>1. Criptografia de credenciais:</strong> todas as URIs
                  de conexão, senhas e dados de autenticação MongoDB são
                  criptografados com AES-256-GCM antes do armazenamento.
                  <br />
                  <br />
                  <strong>2. Criptografia de backups:</strong> cada arquivo BSON
                  é criptografado com uma chave única derivada da chave mestra,
                  do ID do backup e de um salt aleatório.
                  <br />
                  <br />
                  <strong>3. Tratamento seguro de credenciais:</strong> as URIs
                  de conexão são passadas diretamente ao mongodump por
                  parâmetros seguros, nunca expostas nos logs ou na listagem de
                  processos.
                </>
              }
            />
            <FaqItem
              number="8"
              question="O Databasus suporta backups incrementais de MongoDB ou leitura contínua do oplog?"
              answer="O Databasus foca em backups lógicos completos usando o mongodump, em vez de backups incrementais ou recuperação point-in-time baseada no oplog. Para a maioria dos casos, backups completos agendados (de hora em hora, diários, semanais) oferecem pontos de recuperação suficientes sem a complexidade de administrar o oplog. O MongoDB Atlas já oferece backups contínuos nativos com recuperação point-in-time, e backups incrementais externos não podem ser restaurados facilmente em clusters do Atlas."
            />
            <FaqItem
              number="9"
              question="Posso restaurar backups MongoDB em uma versão ou em um cluster diferente?"
              answer="Sim, como o Databasus cria arquivos padrão do mongodump em formato BSON, você pode restaurá-los em qualquer servidor MongoDB compatível: outra versão, outro provedor de nuvem ou uma máquina de desenvolvimento local. Baixe o backup do Databasus (descriptografado automaticamente) e use o mongorestore com as flags --archive e --gzip. O Databasus mostra o comando exato de restauração para cada backup."
            />
            <FaqItem
              number="10"
              question="Como funciona a compressão do mongodump no Databasus?"
              answer="O Databasus usa a flag nativa --gzip do mongodump, que comprime os dados BSON durante o processo de dump. Isso normalmente reduz o tamanho do arquivo em 60-80% em comparação com BSON sem compressão. A compressão acontece no fluxo do mongodump antes da criptografia opcional, então os arquivos comprimidos e criptografados continuam eficientes. A descompressão é automática ao usar o mongorestore com a flag --gzip."
            />
            <FaqItem
              number="11"
              question="Posso fazer backup de coleções MongoDB específicas em vez de bases de dados inteiras?"
              answer="Atualmente, o Databasus faz backup de bases de dados MongoDB inteiras, e não de coleções individuais. Isso garante backups completos e consistentes, incluindo todas as coleções, índices e metadados. Se você precisar de backups por coleção, pode criar bases de dados separadas para domínios de dados diferentes, cada uma com o seu próprio agendamento de backup no Databasus."
            />
            <FaqItem
              number="12"
              question="O Databasus funciona com MongoDB rodando em Docker ou Kubernetes?"
              answer="Sim, o Databasus se conecta ao MongoDB pela rede usando URIs de conexão padrão, então funciona com MongoDB em qualquer ambiente: containers Docker, pods Kubernetes, VMs ou servidores físicos. Basta garantir a conectividade de rede entre o Databasus e a sua instância MongoDB. Em instalações Kubernetes, você pode usar nomes DNS de serviços internos ou endpoints de balanceadores de carga externos."
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
