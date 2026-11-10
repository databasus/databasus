import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl, getLocalizedHref } from "@/app/i18n";
import HeroStatsComponent from "@/app/components/HeroStatsComponent";
import InstallationComponent from "@/app/components/InstallationComponent";
import LiteYouTubeEmbed from "@/app/components/LiteYouTubeEmbed";

export const metadata: Metadata = {
  title: "Backup PostgreSQL | Databasus",
  description:
    "Ferramenta gratuita e de código aberto para backups agendados de PostgreSQL (com suporte a MySQL e MongoDB). Guarde as cópias localmente e na nuvem. Notificações no Slack, Discord, Telegram, e-mail, webhook e mais.",
  keywords:
    "PostgreSQL, backup, monitoramento, base de dados, backups agendados, Docker, self-hosted, código aberto, S3, Azure Blob Storage, notificações no Slack, Discord, DevOps, monitoramento de bases de dados, pg_dump, restauração de base de dados, criptografia, AES-256, criptografia de backups",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("pt", ""),
    languages: getLanguageAlternates(""),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("pt", ""),
    title: "Backup PostgreSQL | Databasus",
    description:
      "Ferramenta gratuita e de código aberto para backups agendados de PostgreSQL (com suporte a MySQL e MongoDB). Guarde as cópias localmente e na nuvem. Notificações no Slack, Discord, Telegram, e-mail, webhook e mais.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Interface do painel do Databasus mostrando a gestão de backups",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary_large_image",
    title: "Backup PostgreSQL | Databasus",
    description:
      "Ferramenta gratuita e de código aberto para backups agendados de PostgreSQL (com suporte a MySQL e MongoDB). Guarde as cópias localmente e na nuvem. Notificações no Slack, Discord, Telegram, e-mail, webhook e mais.",
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
              "Ferramenta gratuita e de código aberto para backups agendados de PostgreSQL (com suporte a MySQL e MongoDB). Guarde as cópias localmente e na nuvem. Notificações no Slack, Discord, Telegram, e-mail, webhook e mais.",
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
              "Backups agendados de PostgreSQL",
              "Vários destinos de armazenamento (S3, Azure Blob Storage, FTP, SFTP, NAS, etc.)",
              "Notificações em tempo real (Slack, Telegram, Discord, webhook, e-mail, etc.)",
              "Monitoramento da saúde das bases de dados",
              "Self-hosted via Docker",
              "Código aberto e gratuito",
              "Suporte a PostgreSQL 12-18",
              "Compressão de backups e criptografia AES-256-GCM",
              "Suporte a PostgreSQL, MySQL, MariaDB e MongoDB",
              "Políticas de retenção: período de tempo, quantidade, GFS e limites de tamanho",
              "Point-in-Time Recovery (PITR) com arquivamento de WAL",
              "Verificação de restauração: testes automáticos de restauração em contêineres Docker com bases de dados reais",
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
            description: "Ferramenta de backup PostgreSQL",
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
                name: "O que é o Databasus e por que usá-lo em vez de scripts escritos à mão?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus é um serviço self-hosted com licença Apache 2.0 que faz backup de PostgreSQL, da v13 à v18. A diferença em relação a scripts shell é que ele tem uma interface para agendar tarefas, comprimir e guardar os arquivos em vários destinos (disco local, S3, Azure Blob Storage, FTP, SFTP, NAS, etc.), configurar políticas de retenção para apagar automaticamente backups antigos e notificar a sua equipe quando as tarefas terminam ou falham — tudo sem código escrito à mão",
                },
              },
              {
                "@type": "Question",
                name: "Qual é a forma mais rápida de instalar o Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O caminho mais direto é executar o instalador de uma linha via cURL. Ele baixa a imagem Docker atual e sobe um único contêiner PostgreSQL. Depois cria um docker-compose.yml e inicia o serviço, que passa a iniciar automaticamente após reinicializações. O tempo total costuma ser inferior a dois minutos em um VPS típico.",
                },
              },
              {
                "@type": "Question",
                name: "Como funciona a verificação de restauração?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus executa um pequeno agente de verificação em um host sob o seu controle. A cada execução agendada, o agente baixa o backup mais recente. Ele restaura o backup em um contêiner de base de dados descartável. Em seguida, compara a base restaurada com a original. O resultado é reportado de volta, incluindo o código de saída da restauração e a contagem de linhas por tabela. Os agendamentos aceitam: após o backup, de hora em hora, diário, semanal, mensal ou uma expressão cron em UTC. As falhas podem ser enviadas por qualquer notificador ligado à base de dados: Slack, Teams, Discord, e-mail e outros.",
                },
              },
              {
                "@type": "Question",
                name: "Onde ficam os meus backups e quanto espaço vão ocupar?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Os arquivos podem ser guardados em volumes locais, buckets compatíveis com S3, Azure Blob Storage, FTP e outros destinos na nuvem. O Databasus aplica compressão balanceada, que normalmente reduz o tamanho do dump em 4-8x com apenas cerca de 20% de tempo de execução adicional, gerando economia de armazenamento e de tráfego de rede.",
                },
              },
              {
                "@type": "Question",
                name: "Como vou saber que um backup foi concluído — ou pior, que falhou?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus notifica em tempo real por e-mail, Slack, Telegram, webhooks, Mattermost, Discord e mais. Você escolhe quais canais avisar, para que a sua equipe de DevOps saiba dos sucessos e das falhas em tempo real, o que facilita as rotinas de recuperação e as auditorias de conformidade.",
                },
              },
              {
                "@type": "Question",
                name: "Como o Databasus garante a segurança?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus aplica segurança em três níveis: (1) Criptografia de dados sensíveis — todas as senhas, tokens e credenciais são criptografados com AES-256-GCM e guardados separadamente da base de dados; (2) Criptografia de backups — cada arquivo de backup é criptografado com uma chave única derivada de uma chave mestra, do ID do backup e de um salt aleatório, tornando os backups inúteis sem a sua chave de criptografia mesmo que alguém obtenha acesso ao armazenamento; (3) Acesso somente leitura à base de dados — o Databasus exige apenas permissões SELECT e faz verificações completas para garantir que não existem privilégios de escrita, evitando corrupção de dados mesmo que a ferramenta seja comprometida. O próprio login pode exigir um segundo fator: com ele ativado, depois da senha correta chega um código de seis dígitos ao e-mail da conta, enquanto o login pelo Google ou pelo GitHub continua contando com as verificações desses provedores. Antes de cada versão, verificações automáticas analisam código, dependências, segredos e imagens de contêineres. Os testes verificam ciclos completos de backup e restauração em contêineres reais de PostgreSQL, MySQL, MariaDB e MongoDB. As verificações obrigatórias precisam passar antes da publicação. Auditorias periódicas do Codex Security da OpenAI complementam essas verificações e são independentes do processo de publicação. Todas as operações rodam em contêineres sob o seu controle, em servidores seus, e como o código é aberto, a sua equipe de segurança pode auditar cada linha antes da implantação.",
                },
              },
              {
                "@type": "Question",
                name: "O Databasus tem o apoio dos programas OSS da Anthropic e da OpenAI?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Sim, em março de 2026 o Databasus foi aceito tanto no Claude for Open Source da Anthropic quanto no Codex for Open Source da OpenAI. O projeto foi avaliado de forma independente e reconhecido por líderes do setor como infraestrutura de código aberto crítica que merece apoio.",
                },
              },
              {
                "@type": "Question",
                name: "Qual é a diferença entre o Databasus e o PgBackRest, o Barman ou o pg_dump?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus prioriza a simplicidade: oferece uma interface web moderna para gerenciar backups de muitas bases de dados ao mesmo tempo, com agendamento integrado, compressão, vários destinos de armazenamento, monitoramento de saúde e notificações em tempo real. Ao mesmo tempo, diferente do pgBackRest e do WAL-G, o Databasus faz backups físicos, incrementais e de WAL sobre o mecanismo nativo do PostgreSQL 17, ou seja, não reinventa o próprio motor de backup. Ele se conecta às suas bases de dados remotamente, alcançando redes fechadas por meio de um túnel SSH até o servidor ou um bastion, para que bases que não estão expostas publicamente também possam ser copiadas e gerenciadas a partir de um único painel.",
                },
              },
              {
                "@type": "Question",
                name: "Quais bases de dados o Databasus suporta?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus suporta PostgreSQL, MySQL, MariaDB e MongoDB. No entanto, o Databasus foi criado originalmente para o PostgreSQL e mantém o foco principal nele, com suporte completo e máxima eficiência para backups de PostgreSQL. MySQL, MariaDB e MongoDB são suportados, mas o PostgreSQL continua sendo a prioridade central, com os recursos mais otimizados e desenvolvimento contínuo. Por exemplo, o Databasus oferece suporte nativo a backups físicos e de WAL para recuperação de desastres do PostgreSQL. Ou seja, o Databasus é de fato uma ferramenta de backup PostgreSQL; as demais bases vêm como um bônus.",
                },
              },
              {
                "@type": "Question",
                name: "Qual é o nível de adoção do Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus é hoje a ferramenta de backup PostgreSQL de código aberto mais adotada. Em 17 de junho de 2026, já havia sido baixado mais de 1.800.000 vezes no Docker por DBAs, engenheiros de DevOps, desenvolvedores e equipes do mundo todo. Com mais de 8.500 estrelas no GitHub, supera o pgBackRest (~4.200 estrelas, disponível desde 2014) e o WAL-G (~4.100 estrelas, disponível desde 2017). O Databasus foi lançado em 2025 e ultrapassou ambos no primeiro ano.",
                },
              },
              {
                "@type": "Question",
                name: "Quais tipos de backup o Databasus suporta?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus suporta backups físicos, completos, incrementais, de WAL e lógicos. Os backups físicos são uma cópia no nível de arquivos de todo o cluster da base de dados — mais rápidos de fazer e de restaurar para grandes volumes de dados do que dumps lógicos, e construídos sobre o mecanismo nativo de backup do PostgreSQL 17, então usamos as ferramentas testadas do próprio PostgreSQL em vez de reinventá-las. Os backups completos são uma cópia integral e autossuficiente do cluster, a base de onde parte toda cadeia de backups. Os backups incrementais guardam apenas o que mudou desde o backup anterior, mantendo os backups pequenos e rápidos. O streaming de WAL captura continuamente o fluxo de escrita da base, permitindo Point-in-Time Recovery (PITR) para recuperação de desastres e perda de dados próxima de zero. Os backups lógicos são um dump nativo da base no formato binário do próprio motor, comprimido e enviado diretamente ao armazenamento, sem arquivos intermediários. Todos esses backups podem passar por um túnel SSH se você precisar de conexões não públicas, então a base de dados nunca precisa ficar exposta publicamente. O túnel SSH já vem integrado.",
                },
              },
              {
                "@type": "Question",
                name: "Esqueci o email ou a senha do administrador",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: 'A primeira conta criada na instância é quem a administra. Se você não lembra qual é esse endereço, execute docker exec -it databasus ./main --list-admins no servidor onde o Databasus está em execução: ele nomeia cada conta de administrador com seu endereço de email, nome de exibição, data de criação e estado de atividade, e não imprime nenhuma senha. Para definir uma nova senha para essa conta, execute docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com". Uma instância criada antes de a primeira conta administrá-la ainda carrega o endereço provisório admin até que seu dono o substitua. Se a instância também exigir um código ao entrar e o servidor de e-mail tiver parado de entregá-lo, execute docker exec -it databasus ./main --disable-2fa para que ela pare de pedir o código. Os três comandos estão documentados na página de senha.',
                },
              },
              {
                "@type": "Question",
                name: "Como a IA é usada no desenvolvimento do Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "A IA nos ajuda a desenvolver o Databasus seguindo um processo de engenharia definido. As pessoas definem o rumo e tomam as decisões finais. Uma pessoa lê cada linha de código antes de integrar uma mudança. Agentes independentes revisam a correção, a arquitetura, os testes e a documentação em sessões separadas, somente para leitura. A análise estática rigorosa e os testes com bancos reais complementam suas revisões. As verificações obrigatórias de CI precisam passar antes da publicação.",
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        lang="pt"
        sectionLinks={[
          { href: "#how-to-use", label: "Como usar" },
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
              Backup PostgreSQL
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              O Databasus é uma ferramenta gratuita, de código aberto e
              self-hosted para fazer backup de PostgreSQL. Crie cópias de
              segurança em diferentes armazenamentos (S3, Azure Blob Storage, FTP,
              etc.) com notificações sobre o progresso (Slack, Discord,
              Telegram, etc.). Com foco em Point-in-Time Recovery{" "}
              <span className="underline decoration-2 underline-offset-2 decoration-blue-600">
                com RPO/RTO baixos
              </span>
            </p>

            <div>
              <div className="flex flex-col items-center justify-center gap-2 max-w-[370px] sm:max-w-[340px] mx-auto">
                <a
                  href="#installation"
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-1"
                >
                  Hospede você mesmo via Docker
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
                alt="Interface do painel do Databasus"
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
                alt="Apoio da Anthropic e da OpenAI OSS"
              />

              <div className="flex justify-center text-base md:text-xl mt-4 md:mt-0 md:ml-10">
                <div className="max-w-[370px] text-gray-400 text-center md:text-left">
                  Apoiado pelos programas de código aberto da Anthropic e da
                  OpenAI.{" "}
                  <a
                    href="/pt/faq#oss-programs"
                    target="_blank"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Saiba&nbsp;mais&nbsp;→
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
              <span className="text-sm font-medium">Visão geral</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Recursos
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              O Databasus oferece tudo o que você precisa para gerenciar backups
              de produção com confiança. Do agendamento automático à
              criptografia de backups. Funciona bem tanto para quem desenvolve
              projetos pessoais quanto para equipes de DevOps e empresas
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
                Backups agendados
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Backups agendados"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Backup é algo que precisa acontecer em horários definidos, de
                forma regular. Por isso oferecemos várias opções: de hora em
                hora, diário, semanal, mensal, cron, etc.
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Verificações de saúde configuráveis
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Verificações de saúde"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                A cada minuto (ou qualquer outro intervalo), o sistema faz ping
                na sua base de dados e mostra o histórico das tentativas
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                A base pode ser considerada fora do ar após 3 tentativas
                falhadas, por exemplo. Quando ela volta a ficar saudável, você
                também recebe uma notificação
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Vários destinos de armazenamento
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Os arquivos ficam em VPS, armazenamentos na nuvem e outros
                lugares. Você escolhe o armazenamento que quiser. Os arquivos
                são sempre seus.{" "}
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
                  alt="Armazenamento"
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
                Notificações
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Você pode receber notificações sobre o sucesso ou a falha do
                processo. Útil para desenvolvedores e equipes de DevOps.{" "}
                <a
                  href="/pt/notifiers"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Ver todas →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-notifications.svg"
                  alt="Notificações"
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
                Self-hosted via Docker
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                O Databasus roda no seu PC ou VPS. Assim, todos os seus dados
                ficam com você, em segurança. A implantação leva cerca de 2
                minutos via script, Docker ou k8s
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
                Código aberto e gratuito
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                O projeto é totalmente de código aberto, gratuito e com licença
                Apache 2.0. Você pode copiar e fazer fork do código por conta
                própria. Aberto também para uso corporativo
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
                  Verificação de restauração
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Um backup que termina sem erro não é o mesmo que um backup que
                  você consegue restaurar. O Databasus baixa periodicamente o
                  backup mais recente, restaura-o em um contêiner de base de
                  dados descartável e reporta o resultado.{" "}
                  <a
                    href="/pt/restore-verification"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Leia mais →
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
                  Segurança
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Criptografia de nível corporativo protege dados sensíveis e
                  backups. O acesso somente leitura à base de dados evita
                  corrupção de dados. Tudo isso não exige conhecimento nenhum e
                  já vem pronto e ativado automaticamente desde o início.{" "}
                  <a
                    href="/pt/security"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Leia mais →
                  </a>
                </p>

                <div>
                  <img
                    src="/images/index/feature-encryption.svg"
                    alt="Segurança"
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
                  Gestão de acessos
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  para equipes
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-access-management.svg"
                  alt="Gestão de acessos"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Conceda acesso para ver ou gerenciar bases de dados. Separe
                equipes e projetos. Ideal para equipes de DevOps e
                desenvolvedores.{" "}
                <a
                  href="/pt/access-management#settings"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Leia mais →
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
                  alt="Logs de auditoria"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Acompanhe todas as atividades do sistema com logs de auditoria
                completos. Você pode ver o histórico de acessos e alterações de
                cada pessoa (downloads de backups, mudanças de agendamento,
                atualizações de configuração, etc.).{" "}
                <a
                  href="/pt/access-management#audit-logs"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Leia mais →
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
                Criptografia de nível corporativo protege dados sensíveis e
                backups. O acesso somente leitura à base de dados evita
                corrupção de dados. Tudo isso não exige conhecimento nenhum e já
                vem pronto e ativado automaticamente desde o início.{" "}
                <a
                  href="/pt/security"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Leia mais →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-encryption.svg"
                  alt="Segurança"
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
                  Backups lógicos, físicos, incrementais e de WAL
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  O Databasus suporta backups lógicos e físicos (completos e
                  incrementais) com streaming de WAL para Point-in-Time
                  Recovery. Isso torna o Databasus adequado para recuperação de
                  desastres e funciona igualmente bem com bases self-hosted e na
                  nuvem: use o modo remoto para bases gerenciadas na nuvem ou
                  acessíveis publicamente. Os backups físicos são feitos sobre
                  os backups nativos do PG 17.{" "}
                  <a
                    href="/pt/faq/#pitr"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Leia mais →
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
                  Visão geral em 4 minutos
                </span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Como usar o Databasus?
              </h2>

              <p className="text-gray-200 max-w-[450px] leading-relaxed mb-6 md:mb-8 text-sm sm:text-base">
                Veja neste vídeo como conectar a sua base de dados, configurar o
                agendamento dos backups, baixar e restaurar backups, adicionar
                membros à equipe e o que são os logs de auditoria
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
                    Desenvolvedor do Databasus
                  </p>
                </div>
              </a>
            </div>

            {/* Right side: Video */}
            <div className="flex-1 relative">
              <div className="rounded-lg overflow-hidden shadow-lg border border-[#ffffff20]">
                <LiteYouTubeEmbed
                  videoId="1qsAnijJfJE"
                  title="Como usar o Databasus (visão geral)?"
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
              <span className="text-sm font-medium">Bases de dados</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Bases de dados suportadas
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              O Databasus suporta PostgreSQL, MySQL, MariaDB e MongoDB. Você
              pode fazer backup e restaurar todas com a mesma ferramenta. O foco
              principal é o PostgreSQL, mas MySQL, MariaDB e MongoDB também são
              suportados
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
                O PostgreSQL é a principal base de dados suportada pelo
                Databasus. Todas as versões de 12 a 18 são suportadas
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
                O MySQL é a segunda base de dados mais popular do mundo. Você
                pode fazer backup e restaurar as suas bases MySQL com a mesma
                simplicidade
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/pt/mysql-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Leia mais →
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
                O MariaDB é suportado com os mesmos recursos do MySQL. Você pode
                fazer backup e restaurar as suas bases MariaDB sem complicações
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/pt/mysql-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Leia mais →
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
                O MongoDB é a base de dados NoSQL mais popular. Você pode fazer
                backup e restaurar as suas bases MongoDB com a mesma interface
                fácil de usar
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/pt/mongodb-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Leia mais →
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
              <span className="text-sm font-medium">Processo</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Como fazer backups?
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              A principal prioridade do Databasus é a simplicidade; hoje é a
              ferramenta mais fácil do mundo para fazer backup de PostgreSQL.
              Para criar backups, você segue 4 passos. Depois disso, poderá
              restaurar com um clique
            </p>
          </div>

          {/* Steps */}
          <div className="space-y-6 md:space-y-10 max-w-[1000px] mx-auto">
            {/* Step 1 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Passo 1
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Escolha o agendamento necessário
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Você pode escolher o horário que quiser: diário, semanal,
                    mensal, um horário específico (como 4h da manhã) e ciclos
                    cron
                  </p>
                  <p>
                    Para o intervalo semanal é preciso indicar o dia da semana;
                    para o mensal, o dia do mês
                  </p>
                  <p>
                    Se a sua base de dados for grande, recomendamos escolher um
                    horário de menor tráfego
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Passo 1"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 2 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Passo 2
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Informe os dados da sua base
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Insira as credenciais da sua base PostgreSQL, escolha a
                    versão e a base de destino. Indique também se SSL é
                    necessário
                  </p>
                  <p>
                    Por padrão, o Databasus comprime os backups em nível
                    balanceado para não deixar o processo lento (~20% mais
                    demorado) e economizar de 4x a 8x de espaço (o que reduz o
                    tráfego de rede)
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-2.svg"
                  alt="Passo 2"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 3 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Passo 3
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Escolha o armazenamento dos backups
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Você pode guardar os arquivos de backup localmente, no S3,
                    no Azure Blob Storage, em NAS, no FTP e em outros serviços
                  </p>
                  <p>
                    Lembre-se de que é preciso ter espaço suficiente no
                    armazenamento
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-3.svg"
                  alt="Passo 3"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 4 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Passo 4
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Escolha onde quer receber notificações (opcional)
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Quando um backup termina com sucesso ou falha, o Databasus
                    pode enviar uma notificação. Pode ser o chat do DevOps, o
                    seu e-mail ou até um webhook da sua equipe
                  </p>
                  <p>
                    Pretendemos suportar a maioria dos mensageiros e plataformas
                    populares
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-4.svg"
                  alt="Passo 4"
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
              Começar agora
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
                <span className="text-sm font-medium">Comece agora</span>
              </div>

              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
                Como instalar?
              </h2>

              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                O Databasus suporta várias formas de instalação, tanto local
                quanto na nuvem. Todas são extremamente simples e fáceis de
                usar, mesmo para quem não tem experiência em administração de
                sistemas ou DevOps
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
              Perguntas frequentes
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              O objetivo do Databasus é tornar a cópia de segurança o mais
              simples possível para quem desenvolve sozinho (e também para
              DevOps) e para equipes. A interface facilita criar backups,
              visualiza o progresso e restaura qualquer coisa em poucos cliques
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="O que é o Databasus e por que usá-lo em vez de scripts escritos à mão?"
              answer="O Databasus é um serviço self-hosted com licença Apache 2.0 que faz backup de bases de dados. A diferença em relação a scripts shell é que ele tem uma interface para agendar tarefas, comprimir e guardar os arquivos em vários destinos (disco local, S3, Azure Blob Storage, NAS, FTP, SFTP, etc.), configurar políticas de retenção para apagar automaticamente backups antigos e notificar a sua equipe quando as tarefas terminam ou falham — tudo sem código escrito à mão"
            />
            <FaqItem
              number="2"
              question="Qual é a forma mais rápida de instalar o Databasus?"
              answer="O Databasus suporta vários métodos de instalação: script automatizado, Docker, Docker Compose e Kubernetes com Helm. O caminho mais rápido é executar o instalador de uma linha via cURL, que baixa a imagem Docker atual, cria um docker-compose.yml e inicia o serviço, que passa a reiniciar automaticamente após reinicializações. Para ambientes Kubernetes, use o Helm chart oficial para implantações prontas para produção. O tempo total costuma ser inferior a dois minutos em um VPS típico."
            />
            <FaqItem
              number="3"
              question="Como funciona a verificação de restauração?"
              answer="O Databasus executa um pequeno agente de verificação em um host sob o seu controle. A cada execução agendada, o agente baixa o backup mais recente. Ele restaura o backup em um contêiner de base de dados descartável. Em seguida, compara a base restaurada com a original. O resultado é reportado de volta, incluindo o código de saída da restauração e a contagem de linhas por tabela. Os agendamentos aceitam: após o backup, de hora em hora, diário, semanal, mensal ou uma expressão cron em UTC. As falhas podem ser enviadas por qualquer notificador ligado à base de dados: Slack, Teams, Discord, e-mail e outros."
            />
            <FaqItem
              number="4"
              question="Como o Databasus garante a segurança?"
              answer={
                <>
                  O Databasus aplica segurança em três níveis: (1) Criptografia
                  de dados sensíveis: todas as senhas, tokens e credenciais são
                  criptografados com AES-256-GCM e guardados separadamente da
                  base de dados; (2) Criptografia de backups: cada arquivo de
                  backup é criptografado com uma chave única derivada de uma
                  chave mestra, do ID do backup e de um salt aleatório, tornando
                  os backups inúteis sem a sua chave de criptografia mesmo que
                  alguém obtenha acesso ao armazenamento; (3) Acesso somente
                  leitura à base de dados: o Databasus exige apenas permissões
                  SELECT e faz verificações completas para garantir que não
                  existem privilégios de escrita, evitando corrupção de dados
                  mesmo que a ferramenta seja comprometida.
                  <br />
                  <br />
                  O próprio login pode exigir um segundo fator: com ele ativado,
                  depois da senha correta chega um código de seis dígitos ao
                  e-mail da conta, enquanto o login pelo Google ou pelo GitHub
                  continua contando com as verificações desses provedores.
                  <br />
                  <br />
                  Antes de cada versão, verificações automáticas analisam código,
                  dependências, segredos e imagens de contêineres. Os testes verificam ciclos
                  completos de backup e restauração em contêineres reais de PostgreSQL, MySQL,
                  MariaDB e MongoDB. As verificações obrigatórias precisam passar antes da
                  publicação. Auditorias periódicas do Codex Security da OpenAI complementam
                  essas verificações e são independentes do processo de publicação.
                  <br />
                  <br />
                  Veja{" "}
                  <a
                    href={`${getLocalizedHref("pt", "security")}#security-and-reliability-engineering`}
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Engenharia de segurança e confiabilidade
                  </a>{" "}
                  para conhecer o pipeline completo.
                  <br />
                  <br />
                  Além disso, todos os logs podem ser exportados para qualquer
                  sistema externo (VictoriaLogs, SigNoz, Graylog, etc.) via o
                  padrão OpenTelemetry. Por padrão, os logs (incluindo os de
                  auditoria) também são gravados em arquivos locais, então os
                  logs de auditoria não se perdem. Veja a{" "}
                  <a
                    href="/pt/advanced-config#logging"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    configuração avançada
                  </a>{" "}
                  aqui.
                </>
              }
            />
            <FaqItem
              number="5"
              question="Como configuro e executo o meu primeiro backup no Databasus?"
              answer={
                <>
                  Para o seu primeiro backup no Databasus, basta entrar no
                  painel, clicar em New Backup e escolher um intervalo: de hora
                  em hora, diário, semanal, mensal ou cron. Depois defina o
                  horário exato da execução (por exemplo, 02:30, fora do horário
                  de pico).
                  <br />
                  <br />
                  Em seguida, informe o host do PostgreSQL, a porta, o nome da
                  base, as credenciais e a preferência de SSL. Escolha para onde
                  o arquivo deve ser enviado (caminho local, bucket S3, pasta do
                  Azure Blob Storage, FTP, etc.). <br />
                  <br />
                  Se quiser, adicione canais de notificação como e-mail, Slack,
                  Telegram ou um webhook e clique em Save. O Databasus valida os
                  dados na hora, ativa o agendamento, executa a primeira tarefa
                  e envia o status em tempo real. Assim, você pode restaurar com
                  um toque quando o backup estiver concluído.
                </>
              }
            />
            <FaqItem
              number="6"
              question="Qual é o nível de adoção do Databasus?"
              answer="O Databasus é hoje a ferramenta de backup PostgreSQL de código aberto mais adotada. Em 17 de junho de 2026, já havia sido baixado mais de 1.800.000 vezes no Docker por DBAs, engenheiros de DevOps, desenvolvedores e equipes do mundo todo. Com mais de 8.500 estrelas no GitHub, supera o pgBackRest (~4.200 estrelas, disponível desde 2014) e o WAL-G (~4.100 estrelas, disponível desde 2017). O Databasus foi lançado em 2025 e ultrapassou ambos no primeiro ano."
            />
            <FaqItem
              number="7"
              question="Qual é a diferença entre o Databasus e o PgBackRest, o Barman ou o pg_dump? Onde posso ler comparações?"
              answer={
                <>
                  O Databasus prioriza a simplicidade: oferece uma interface web
                  moderna para gerenciar backups de muitas bases de dados ao
                  mesmo tempo, em vez de arquivos de configuração complexos e
                  ferramentas de linha de comando. Diferente de scripts com
                  pg_dump puro, inclui agendamento integrado, compressão, vários
                  destinos de armazenamento, monitoramento de saúde e
                  notificações em tempo real, tudo gerenciado por uma interface
                  web simples.
                  <br />
                  <br />
                  Ao mesmo tempo, diferente do pgBackRest e do WAL-G, o
                  Databasus faz backups físicos, incrementais e de WAL sobre o
                  mecanismo nativo do PostgreSQL 17, ou seja, não reinventa o
                  próprio motor de backup. Ele se conecta às suas bases de dados
                  remotamente, alcançando redes fechadas por meio de um túnel
                  SSH até o servidor ou um bastion, para que bases que não estão
                  expostas publicamente também possam ser copiadas e gerenciadas
                  a partir de um único painel.{" "}
                  <a
                    href="/pt/faq/#pitr"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Leia como os backups físicos e o PITR são implementados
                  </a>
                  .
                  <br />
                  <br />
                  Temos páginas de comparação detalhadas para as ferramentas de
                  backup mais conhecidas:{" "}
                  <a
                    href="/pt/pgdump-alternative"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pg_dump
                  </a>
                  ,{" "}
                  <a
                    href="/pt/databasus-vs-pgbackrest"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pgBackRest
                  </a>
                  ,{" "}
                  <a
                    href="/pt/databasus-vs-barman"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs Barman
                  </a>
                  ,{" "}
                  <a
                    href="/pt/databasus-vs-wal-g"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs WAL-G
                  </a>{" "}
                  e{" "}
                  <a
                    href="/pt/databasus-vs-pgbackweb"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus vs pgBackWeb
                  </a>
                  . Cada comparação explica as principais diferenças, prós e
                  contras, e ajuda a escolher a ferramenta certa para o seu
                  caso.
                </>
              }
            />
            <FaqItem
              number="8"
              question="O Databasus tem o apoio dos programas OSS da Anthropic e da OpenAI?"
              answer={
                <>
                  Sim, temos orgulho de que o Databasus tenha sido reconhecido
                  como um projeto de código aberto valioso por duas das
                  principais empresas de IA do mundo. Em março de 2026, o
                  Databasus foi aceito tanto no{" "}
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Claude for Open Source
                  </a>{" "}
                  da Anthropic quanto no{" "}
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Codex for Open Source
                  </a>{" "}
                  da OpenAI. Para nós, é uma confirmação independente de
                  confiabilidade: o projeto foi avaliado e reconhecido como
                  infraestrutura crítica que merece apoio.{" "}
                  <a
                    href="/pt/faq#oss-programs"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Leia mais →
                  </a>
                </>
              }
            />
            <FaqItem
              number="9"
              question="O Databasus é uma alternativa ao pg_dump?"
              answer="Não exatamente. O Databasus foca em recuperação de desastres com RTO e RPO baixos, então está mais próximo de uma alternativa ao pgBackRest ou ao WAL-G — ele tenta tornar a recuperação de desastres tão simples quanto o pg_dump. Dito isso, para backups lógicos ele funciona sim como alternativa ao pg_dump e usa o pg_dump por baixo, acrescentando uma interface web amigável, agendamento automático, vários destinos de armazenamento, notificações em tempo real, monitoramento de saúde e criptografia de backups. Os backups lógicos também estão disponíveis para MySQL, MariaDB e MongoDB."
            />
            <FaqItem
              number="10"
              question="Quais bases de dados o Databasus suporta?"
              answer={
                <>
                  O Databasus suporta PostgreSQL, MySQL, MariaDB e MongoDB. No
                  entanto, o Databasus foi criado originalmente para o
                  PostgreSQL e mantém o foco principal nele, com suporte
                  completo e máxima eficiência para backups de PostgreSQL.
                  <br />
                  <br />
                  MySQL, MariaDB e MongoDB são suportados, mas o PostgreSQL
                  continua sendo a prioridade central, com os recursos mais
                  otimizados e desenvolvimento contínuo.
                  <br />
                  <br />
                  Por exemplo, o Databasus oferece suporte nativo a backups
                  físicos e de WAL para recuperação de desastres do PostgreSQL.
                  Ou seja, o Databasus é de fato uma ferramenta de backup
                  PostgreSQL; as demais bases vêm como um bônus.
                </>
              }
            />
            <FaqItem
              number="11"
              question="Quais tipos de backup o Databasus suporta?"
              answer={
                <>
                  O Databasus suporta backups físicos, completos, incrementais,
                  de WAL e lógicos — então serve tanto para quem quer dumps
                  lógicos simples quanto para quem precisa de uma ferramenta
                  sólida de recuperação de desastres.
                  <ul className="list-disc list-inside mt-3 space-y-2">
                    <li>
                      <strong>Físico</strong> — cópia no nível de arquivos de
                      todo o cluster da base de dados. Backup e restauração mais
                      rápidos para grandes volumes de dados do que dumps
                      lógicos. Construído sobre o mecanismo nativo de backup do
                      PostgreSQL 17, então usamos as ferramentas testadas do
                      próprio PostgreSQL em vez de reinventá-las
                    </li>
                    <li>
                      <strong>Completo</strong> — uma cópia integral e
                      autossuficiente do cluster, a base de onde parte toda
                      cadeia de backups
                    </li>
                    <li>
                      <strong>Incremental</strong> — guarda apenas o que mudou
                      desde o backup anterior, mantendo os backups pequenos e
                      rápidos
                    </li>
                    <li>
                      <strong>Streaming de WAL</strong> — captura continuamente
                      o fluxo de escrita da base, permitindo Point-in-Time
                      Recovery (PITR). Projetado para recuperação de desastres e
                      perda de dados próxima de zero
                    </li>
                    <li>
                      <strong>Lógico</strong> — dump nativo da base no formato
                      binário do próprio motor. Comprimido e enviado diretamente
                      ao armazenamento, sem arquivos intermediários
                    </li>
                  </ul>
                  <br />
                  Os backups físicos, incrementais e de WAL usam o mecanismo
                  nativo do PostgreSQL 17, então exigem PostgreSQL 17 ou mais
                  recente; em versões anteriores, apenas backups lógicos estão
                  disponíveis. Isso é intencional: a maioria das bases de
                  produção já roda PostgreSQL 17 ou superior, e em cerca de dois
                  anos as versões mais antigas chegam ao fim da vida útil. O
                  Databasus quer se tornar a ferramenta de backup padrão para
                  bases de dados a partir do PostgreSQL 17.
                  <br />
                  <br />
                  Todos esses backups podem passar por um túnel SSH se você
                  precisar de conexões não públicas, então a base de dados nunca
                  precisa ficar exposta publicamente. O túnel SSH já vem
                  integrado.
                </>
              }
            />
            <FaqItem
              number="12"
              question="Como a IA é usada no desenvolvimento do Databasus?"
              answer={
                <>
                  <p>
                    A IA nos ajuda a desenvolver o Databasus seguindo um processo de engenharia definido. As pessoas
                    definem o rumo e tomam as decisões finais.{" "}
                    <strong>Uma pessoa lê cada linha de código antes de integrar uma mudança.</strong>
                  </p>
                  <p className="mt-3">
                    Agentes independentes revisam a correção, a arquitetura, os testes e a documentação em sessões
                    separadas, somente para leitura. A análise estática rigorosa e os testes com bancos reais
                    complementam suas revisões. As verificações obrigatórias de CI precisam passar antes da
                    publicação.{" "}
                    <a
                      href={`${getLocalizedHref("pt", "faq")}#ai-usage`}
                      className="text-blue-400 hover:text-blue-600"
                    >
                      Conheça o processo de desenvolvimento
                    </a>
                    .
                  </p>
                </>
              }
            />
            <FaqItem
              number="13"
              question="Como posso entrar para a comunidade do Databasus?"
              answer={
                <>
                  Você pode entrar para a nossa grande comunidade de
                  desenvolvedores, DBAs e engenheiros de DevOps em{" "}
                  <a
                    href="https://t.me/databasus_community"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    t.me/databasus_community
                  </a>
                  . A comunidade é um ótimo lugar para tirar dúvidas, trocar
                  experiências, receber ajuda com configuração e acompanhar os
                  recursos e lançamentos mais recentes.
                </>
              }
            />
            <FaqItem
              number="14"
              question="Qual é o nível de adoção do Databasus?"
              answer={
                <>
                  O Databasus tem mais de 1,8 milhão de downloads no Docker e
                  8,5 mil estrelas no GitHub. Para comparação, o pgBackRest e o
                  WAL-G ficam em torno de 4,2 mil estrelas cada e o Barman em
                  cerca de 3,1 mil, o que faz do Databasus a ferramenta de
                  backup de bases de dados mais popular do GitHub.
                  <br />
                  <br />
                  Ele foi aceito nos programas de código aberto da Anthropic e
                  da OpenAI como um projeto importante e crítico. Hoje o
                  Databasus é usado por empresas, equipes e engenheiros de
                  DevOps, com o apoio de uma comunidade grande e ativa.
                  <br />
                  <br />
                  O Databasus é desenvolvido e usado desde 2023, e em código
                  aberto com uso amplo desde o início de 2025. Está em uso real
                  de produção há bastante tempo, então já foi testado em muitos
                  casos extremos. O ponto essencial: o Databasus não inventa
                  formas próprias de copiar os seus dados — ele usa a
                  implementação nativa e testada do PostgreSQL em vez de criar
                  as próprias soluções improvisadas para casos extremos.
                  <br />
                  <br />
                  Nosso objetivo é nos tornarmos a ferramenta de backup padrão
                  para PostgreSQL a partir da versão 17. O Databasus é a
                  primeira ferramenta de backup construída sobre o protocolo de
                  backup nativo, eficiente e agora padrão do PostgreSQL, em vez
                  de escrever implementações próprias.
                </>
              }
            />
            <FaqItem
              number="15"
              question="Esqueci o email ou a senha do administrador"
              answer={
                <>
                  A primeira conta criada na instância é quem a administra. Se
                  você não lembra qual é esse endereço, execute docker exec -it
                  databasus ./main --list-admins no servidor onde o Databasus
                  está em execução: ele nomeia cada conta de administrador com
                  seu endereço de email, nome de exibição, data de criação e
                  estado de atividade, e não imprime nenhuma senha. Para definir
                  uma nova senha para essa conta, execute docker exec -it
                  databasus ./main
                  --new-password=&quot;YourNewSecurePassword123&quot;
                  --email=&quot;owner@example.com&quot;. Uma instância criada
                  antes de a primeira conta administrá-la ainda carrega o
                  endereço provisório admin até que seu dono o substitua. Se a
                  instância também exigir um código ao entrar e o servidor de
                  e-mail tiver parado de entregá-lo, execute docker exec -it
                  databasus ./main --disable-2fa para que ela pare de pedir o
                  código. Os três comandos estão documentados na
                  <a
                    href="/pt/password"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    página de senha
                  </a>
                  .
                </>
              }
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
