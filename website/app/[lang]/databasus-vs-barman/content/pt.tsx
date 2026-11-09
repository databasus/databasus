import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs Barman: comparação de ferramentas de backup PostgreSQL",
  description:
    "Compare as ferramentas de backup PostgreSQL Databasus e Barman. Veja as diferenças na abordagem de backup, capacidades de PITR, facilidade de uso, recursos de equipe e quando escolher cada ferramenta.",
  keywords: [
    "Databasus vs Barman",
    "comparação de backup PostgreSQL",
    "alternativa ao Barman",
    "ferramentas de backup PostgreSQL",
    "comparação de backup de bases de dados",
    "pg_dump vs backup físico",
    "backup self-hosted",
    "PITR PostgreSQL",
    "arquivamento de WAL",
    "recuperação de desastres PostgreSQL",
  ],
  openGraph: {
    title:
      "Databasus vs Barman: comparação de ferramentas de backup PostgreSQL",
    description:
      "Compare as ferramentas de backup PostgreSQL Databasus e Barman. Veja as diferenças na abordagem de backup, capacidades de PITR, facilidade de uso, recursos de equipe e quando escolher cada ferramenta.",
    type: "article",
    url: getLocalizedUrl("pt", "databasus-vs-barman"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs Barman: comparação de ferramentas de backup PostgreSQL",
    description:
      "Compare as ferramentas de backup PostgreSQL Databasus e Barman. Veja as diferenças na abordagem de backup, capacidades de PITR, facilidade de uso, recursos de equipe e quando escolher cada ferramenta.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "databasus-vs-barman"),
    languages: getLanguageAlternates("databasus-vs-barman"),
  },
  robots: "index, follow",
};

export default function DatabasusVsBarmanPage() {
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
              "Databasus vs Barman: comparação de ferramentas de backup PostgreSQL",
            description:
              "Uma comparação completa das ferramentas de backup PostgreSQL Databasus e Barman, cobrindo abordagem de backup, capacidades de PITR, facilidade de uso, recursos de equipe e quando escolher cada ferramenta.",
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

      <DocsNavbarComponent lang="pt" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="pt" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="databasus-vs-barman">Databasus vs Barman</h1>

              <p className="text-lg text-gray-400">
                Databasus e Barman foram criados para recuperação de desastres
                com RTO e RPO mínimos, e ambos suportam backups físicos,
                arquivamento de WAL e recuperação a um ponto no tempo (PITR). O
                Databasus executa esses backups remotamente sobre a pilha nativa
                do PostgreSQL 17, reutilizando as ferramentas já comprovadas do
                próprio PostgreSQL em vez de reinventá-las, tudo por trás de uma
                interface web intuitiva com recursos de equipe e suporte a
                vários motores de base de dados. Funciona com bases de dados de
                qualquer tamanho e complexidade. Backups físicos exigem
                PostgreSQL 17 ou mais recente; em versões anteriores só estão
                disponíveis backups lógicos com <code>pg_dump</code>. O Barman
                (Backup and Recovery Manager) traz motor de backup próprio, por
                isso cobre backups físicos em versões bem mais antigas do
                PostgreSQL e acrescenta recursos avançados como backups
                incrementais baseados em rsync, integração com replicação por
                streaming e georredundância Barman a Barman.
              </p>

              <h2 id="quick-comparison">Comparação rápida</h2>

              <p>Veja as principais diferenças entre Databasus e Barman:</p>

              <table>
                <thead>
                  <tr>
                    <th>Recurso</th>
                    <th>Databasus</th>
                    <th>Barman</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Público-alvo</td>
                    <td data-label="Databasus">
                      Indivíduos, equipes, DBAs, empresas
                    </td>
                    <td data-label="Barman">DBAs, empresas</td>
                  </tr>
                  <tr>
                    <td>Suporte a outras bases de dados</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="Barman">❌ Apenas PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>Interface</td>
                    <td data-label="Databasus">Interface web</td>
                    <td data-label="Barman">Apenas linha de comando</td>
                  </tr>
                  <tr>
                    <td>Tipo de backup</td>
                    <td data-label="Databasus">Lógico + físico</td>
                    <td data-label="Barman">Físico (nível de arquivo)</td>
                  </tr>
                  <tr>
                    <td>Versão do PostgreSQL para backups físicos</td>
                    <td data-label="Databasus">17+ (nativo)</td>
                    <td data-label="Barman">9.x+ (motor próprio)</td>
                  </tr>
                  <tr>
                    <td>Opções de recuperação</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="Barman">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Backups incrementais</td>
                    <td data-label="Databasus">
                      ✅ Em nível de bloco (PG 17+)
                    </td>
                    <td data-label="Barman">Incremental baseado em rsync</td>
                  </tr>
                  <tr>
                    <td>Backups remotos</td>
                    <td data-label="Databasus">✅ Sim</td>
                    <td data-label="Barman">
                      ❌ Não (requer acesso ao sistema de arquivos)
                    </td>
                  </tr>
                  <tr>
                    <td>Gestão de vários servidores</td>
                    <td data-label="Databasus">
                      Agendamento por base de dados
                    </td>
                    <td data-label="Barman">Servidor de backup centralizado</td>
                  </tr>
                  <tr>
                    <td>Recursos de equipe</td>
                    <td data-label="Databasus">
                      ✅ Workspaces, RBAC, logs de auditoria
                    </td>
                    <td data-label="Barman">
                      ❌ Apenas permissões do sistema operacional
                    </td>
                  </tr>
                  <tr>
                    <td>Notificações</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, e-mail
                    </td>
                    <td data-label="Barman">❌ Requer scripts próprios</td>
                  </tr>
                  <tr>
                    <td>Curva de aprendizado</td>
                    <td data-label="Databasus">Mínima</td>
                    <td data-label="Barman">Exige experiência de DBA</td>
                  </tr>
                  <tr>
                    <td>Instalação</td>
                    <td data-label="Databasus">
                      Script de uma linha ou Docker
                    </td>
                    <td data-label="Barman">Requer configuração manual</td>
                  </tr>
                  <tr>
                    <td>Gestão de backups</td>
                    <td data-label="Databasus">✅ Sim</td>
                    <td data-label="Barman">❌ Não</td>
                  </tr>
                  <tr>
                    <td>Adequado para bases de dados self-hosted</td>
                    <td data-label="Databasus">✅ Sim</td>
                    <td data-label="Barman">✅ Sim</td>
                  </tr>
                  <tr>
                    <td>Adequado para bases de dados na nuvem</td>
                    <td data-label="Databasus">
                      ✅ Sim (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="Barman">
                      ❌ Não (requer acesso ao sistema de arquivos)
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="target-audience">Público-alvo</h2>

              <p>
                A diferença mais significativa entre as ferramentas é para quem
                cada uma foi projetada:
              </p>

              <h3 id="audience-databasus">Público do Databasus</h3>

              <p>
                O Databasus foi criado para um público amplo, de desenvolvedores
                individuais a grandes empresas:
              </p>

              <ul>
                <li>
                  <strong>Desenvolvedores individuais</strong>: instalação
                  simples e interface intuitiva facilitam proteger projetos
                  pessoais sem conhecimento profundo de PostgreSQL.
                </li>
                <li>
                  <strong>Equipes de desenvolvimento</strong>: workspaces,
                  controle de acesso baseado em papéis e logs de auditoria
                  permitem colaboração segura entre os membros da equipe.
                </li>
                <li>
                  <strong>Empresas</strong>: escala para atender necessidades
                  corporativas com segurança completa, vários destinos de
                  armazenamento e canais de notificação.
                </li>
                <li>
                  <strong>DBAs e recuperação de desastres</strong>: backups
                  físicos, arquivamento de WAL e PITR para sistemas críticos com
                  exigência de perda de dados quase nula.
                </li>
              </ul>

              <h3 id="audience-barman">Público do Barman</h3>

              <p>
                O Barman foi projetado especificamente para administradores de
                bases de dados (DBAs) que administram infraestrutura PostgreSQL
                corporativa:
              </p>

              <ul>
                <li>
                  <strong>DBAs corporativos</strong>: profissionais que precisam
                  de gestão centralizada de backups de vários servidores
                  PostgreSQL a partir de um servidor de backup dedicado.
                </li>
                <li>
                  <strong>Equipes que precisam de incremental via rsync</strong>
                  : a comparação em nível de arquivo reduz o tempo de backup e o
                  uso de rede em clusters grandes.
                </li>
                <li>
                  <strong>Exigências de georredundância</strong>: replicação
                  Barman a Barman para redundância geográfica entre data
                  centers.
                </li>
              </ul>

              <h2 id="backup-approach">Abordagem de backup</h2>

              <p>
                As ferramentas usam estratégias de backup fundamentalmente
                diferentes, cada uma com vantagens próprias:
              </p>

              <h3 id="backup-databasus">
                Databasus: backups lógicos + físicos
              </h3>

              <p>
                O Databasus suporta tanto a estratégia lógica quanto a física de
                cópia de segurança:
              </p>

              <ul>
                <li>
                  <strong>Backups físicos, incrementais e de WAL</strong>:
                  executados remotamente pelo protocolo de replicação do
                  PostgreSQL sobre a pilha nativa do PostgreSQL 17:{" "}
                  <code>pg_basebackup</code>,{" "}
                  <code>pg_basebackup --incremental</code> em nível de bloco
                  guiado pelos resumos de WAL do servidor,{" "}
                  <code>pg_receivewal</code> e <code>pg_combinebackup</code>. O
                  Databasus reutiliza as ferramentas já comprovadas do próprio
                  PostgreSQL em vez de reinventá-las. Requer PostgreSQL 17 ou
                  mais recente.
                </li>
                <li>
                  <strong>Backups lógicos</strong>: usa <code>pg_dump</code>{" "}
                  para backups portáteis que podem ser restaurados em versões
                  diferentes do PostgreSQL. É também o único tipo de backup em
                  PostgreSQL anterior à versão 17 e o caminho para MySQL,
                  MariaDB e MongoDB.
                </li>
                <li>
                  <strong>Nada instalado na base de dados</strong>: os backups
                  se conectam remotamente; redes fechadas são alcançadas por um
                  túnel SSH até um host interno ou um bastion, de modo que a
                  base de dados nunca precisa ficar exposta publicamente.
                </li>
                <li>
                  <strong>Compressão eficiente</strong>: usa compressão zstd
                  (nível 5) tanto para backups lógicos quanto físicos.
                </li>
                <li>
                  <strong>Acesso somente leitura</strong>: backups lógicos
                  exigem apenas permissões SELECT, minimizando riscos de
                  segurança.
                </li>
              </ul>

              <h3 id="backup-barman">Barman: backups físicos</h3>

              <p>
                O Barman realiza backups em nível de arquivo (físicos) do
                diretório de dados do PostgreSQL:
              </p>

              <ul>
                <li>
                  <strong>Backup completo do cluster</strong>: captura o cluster
                  inteiro da base de dados no nível do sistema de arquivos,
                  usando rsync ou pg_basebackup.
                </li>
                <li>
                  <strong>Arquivamento de WAL</strong>: arquiva continuamente os
                  Write-Ahead Logs para recuperação a um ponto no tempo.
                </li>
                <li>
                  <strong>Incremental com rsync</strong>: usa rsync para
                  transferir apenas os arquivos alterados, reduzindo o tempo de
                  backup e o uso de rede.
                </li>
                <li>
                  <strong>Integração com replicação por streaming</strong>: pode
                  receber arquivos de WAL pelo protocolo de replicação por
                  streaming, para arquivamento em tempo real.
                </li>
              </ul>

              <h2 id="recovery-options">Opções de recuperação</h2>

              <p>
                As duas ferramentas oferecem recuperação flexível, mas com
                granularidade diferente:
              </p>

              <h3 id="recovery-databasus">Recuperação com o Databasus</h3>

              <ul>
                <li>
                  <strong>Recuperação a um ponto no tempo</strong>: restaure
                  para qualquer segundo específico com a reaplicação do WAL.
                </li>
                <li>
                  <strong>Restauração completa do cluster</strong>: restaure o
                  cluster inteiro da base de dados para um ponto específico no
                  tempo a partir de backups físicos.
                </li>
                <li>
                  <strong>Restauração lógica</strong>: restaure a partir de
                  backups lógicos agendados para qualquer ponto de backup.
                </li>
                <li>
                  <strong>Restauração com um clique</strong>: baixe e restaure
                  backups lógicos diretamente pela interface web.
                </li>
                <li>
                  <strong>Compatibilidade entre versões</strong>: backups
                  lógicos podem ser restaurados em versões diferentes do
                  PostgreSQL.
                </li>
              </ul>

              <h3 id="recovery-barman">Recuperação com o Barman</h3>

              <ul>
                <li>
                  <strong>Recuperação a um ponto no tempo (PITR)</strong>:
                  restaure para qualquer segundo específico com a reaplicação do
                  WAL, minimizando a perda de dados.
                </li>
                <li>
                  <strong>Restauração completa do cluster</strong>: restaure o
                  cluster inteiro da base de dados para um ponto específico no
                  tempo.
                </li>
                <li>
                  <strong>Recuperação remota</strong>: recupere bases de dados
                  em servidores remotos via SSH.
                </li>
                <li>
                  <strong>Criação de standby</strong>: crie réplicas do
                  PostgreSQL a partir de backups para configurações de alta
                  disponibilidade.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Nota:</strong> as duas
                  ferramentas suportam PITR. O Barman oferece adicionalmente a
                  criação de standby a partir de backups e a recuperação remota
                  via SSH em outros servidores, o que pode ser útil em
                  configurações de alta disponibilidade.{" "}
                  <a
                    href="/pt/faq#pitr"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Saiba como o Databasus suporta PITR →
                  </a>
                </p>
              </div>

              <h2 id="ease-of-use">Facilidade de uso</h2>

              <p>
                As ferramentas diferem drasticamente na abordagem à experiência
                do usuário:
              </p>

              <h3 id="ease-databasus">Experiência de uso do Databasus</h3>

              <ul>
                <li>
                  <strong>Interface web</strong>: configuração de todos os
                  ajustes de backup por cliques. Nenhuma linha de comando
                  necessária.
                </li>
                <li>
                  <strong>Instalação em 2 minutos</strong>: um script cURL de
                  uma linha ou um comando Docker simples coloca tudo em
                  funcionamento imediatamente.
                </li>
                <li>
                  <strong>Monitoramento visual</strong>: o painel mostra o
                  status dos backups, verificações de saúde e histórico num
                  relance.
                </li>
                <li>
                  <strong>Notificações integradas</strong>: configure alertas
                  por Slack, Teams, Telegram, e-mail ou webhook diretamente na
                  interface.
                </li>
                <li>
                  <strong>Sem exigir conhecimento de PostgreSQL</strong>:
                  projetado para desenvolvedores que querem backups confiáveis
                  sem se tornarem especialistas em bases de dados.
                </li>
              </ul>

              <h3 id="ease-barman">Experiência de uso do Barman</h3>

              <ul>
                <li>
                  <strong>Interface de linha de comando</strong>: todas as
                  operações são feitas por comandos de terminal como{" "}
                  <code>barman backup</code>, <code>barman recover</code>.
                </li>
                <li>
                  <strong>Arquivos de configuração</strong>: requer edição
                  manual de arquivos de configuração no estilo INI para cada
                  servidor.
                </li>
                <li>
                  <strong>Configuração do arquivamento de WAL</strong>: é
                  preciso configurar o <code>archive_command</code> do
                  PostgreSQL ou os ajustes de replicação por streaming.
                </li>
                <li>
                  <strong>Gestão de chaves SSH</strong>: requer configurar
                  chaves SSH entre o servidor Barman e os servidores PostgreSQL.
                </li>
                <li>
                  <strong>Experiência de DBA esperada</strong>: a documentação
                  presume familiaridade com o funcionamento interno do
                  PostgreSQL e a mecânica do WAL.
                </li>
              </ul>

              <p>
                <a
                  href="/pt/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Veja o guia de instalação do Databasus →
                </a>
              </p>

              <h2 id="team-features">Recursos de equipe</h2>

              <p>
                Para organizações em que vários membros da equipe administram
                backups:
              </p>

              <h3 id="team-databasus">Recursos de equipe do Databasus</h3>

              <ul>
                <li>
                  <strong>Workspaces</strong>: organize bases de dados,
                  notificadores e armazenamentos por projeto ou equipe. Os
                  usuários só veem os workspaces para os quais foram convidados.
                </li>
                <li>
                  <strong>Controle de acesso baseado em papéis</strong>: atribua
                  permissões de visualizador, editor ou administrador para
                  controlar o que cada membro da equipe pode fazer.
                </li>
                <li>
                  <strong>Logs de auditoria</strong>: acompanhe todas as
                  atividades e mudanças do sistema. Essencial para conformidade
                  de segurança e responsabilização.
                </li>
                <li>
                  <strong>Notificações compartilhadas</strong>: os canais da
                  equipe recebem automaticamente as atualizações de status dos
                  backups.
                </li>
              </ul>

              <h3 id="team-barman">Recursos de equipe do Barman</h3>

              <p>
                O Barman é uma ferramenta de linha de comando sem recursos de
                equipe integrados:
              </p>

              <ul>
                <li>Sem gestão de usuários nem controle de acesso</li>
                <li>Sem log de auditoria das operações</li>
                <li>
                  A coordenação da equipe requer ferramentas e processos
                  externos
                </li>
                <li>
                  Acesso controlado por permissões do sistema operacional e
                  chaves SSH
                </li>
              </ul>

              <p>
                <a
                  href="/pt/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Saiba mais sobre a gestão de acessos do Databasus →
                </a>
              </p>

              <h2 id="security">Segurança</h2>

              <p>
                As duas ferramentas oferecem recursos de segurança, mas com
                abordagens diferentes:
              </p>

              <h3 id="security-databasus">Segurança do Databasus</h3>

              <ul>
                <li>
                  <strong>Criptografia AES-256-GCM</strong>: todas as senhas,
                  tokens e credenciais são criptografados. A chave de
                  criptografia é guardada separada da base de dados.
                </li>
                <li>
                  <strong>Criptografia única por backup</strong>: cada arquivo
                  de backup é criptografado com uma chave única derivada da
                  chave mestra, do ID do backup e de um salt aleatório.
                </li>
                <li>
                  <strong>Acesso somente leitura à base de dados</strong>: exige
                  apenas permissões SELECT, evitando corrupção de dados mesmo em
                  caso de comprometimento.
                </li>
              </ul>

              <h3 id="security-barman">Segurança do Barman</h3>

              <ul>
                <li>
                  <strong>Comunicação via SSH</strong>: usa SSH para a
                  comunicação segura entre o servidor Barman e os servidores
                  PostgreSQL.
                </li>
                <li>
                  <strong>Sem criptografia integrada</strong>: o Barman não
                  oferece criptografia de backup integrada. É preciso usar
                  ferramentas externas ou armazenamento criptografado.
                </li>
                <li>
                  <strong>Segurança no nível do sistema operacional</strong>:
                  depende de permissões do sistema de arquivos e da gestão de
                  chaves SSH para controlar o acesso.
                </li>
                <li>
                  <strong>Verificação por checksum</strong>: valida a
                  integridade dos backups com checksums.
                </li>
              </ul>

              <p>
                <a
                  href="/pt/security"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Saiba mais sobre a segurança do Databasus →
                </a>
              </p>

              <h2 id="storage-options">Opções de armazenamento</h2>

              <p>
                As ferramentas suportam destinos de armazenamento diferentes:
              </p>

              <h3 id="storage-databasus">Armazenamento no Databasus</h3>

              <p>Opções acessíveis para vários casos de uso:</p>

              <ul>
                <li>Armazenamento local</li>
                <li>Amazon S3 e serviços compatíveis com S3</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (armazenamento conectado à rede)</li>
                <li>FTP</li>
              </ul>

              <h3 id="storage-barman">Armazenamento no Barman</h3>

              <p>Opções de armazenamento com foco corporativo:</p>

              <ul>
                <li>Armazenamento local (sistemas de arquivos POSIX)</li>
                <li>Amazon S3 e armazenamento de objetos compatível com S3</li>
                <li>Redundância geográfica via replicação Barman a Barman</li>
              </ul>

              <p>
                <a
                  href="/pt/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Veja todas as opções de armazenamento do Databasus →
                </a>
              </p>

              <h2 id="notifications">Notificações</h2>

              <p>Para ficar a par do status dos backups:</p>

              <h3 id="notifications-databasus">Notificações do Databasus</h3>

              <p>Suporte integrado a vários canais de notificação:</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>E-mail</li>
                <li>Webhooks</li>
              </ul>

              <h3 id="notifications-barman">Notificações do Barman</h3>

              <p>
                O Barman não tem suporte integrado a notificações. Notificações
                exigem:
              </p>

              <ul>
                <li>Scripts próprios em torno dos comandos de backup</li>
                <li>Integração com ferramentas externas de monitoramento</li>
                <li>Análise manual de logs e configuração de alertas</li>
                <li>
                  Integração com ferramentas como Nagios, Zabbix ou soluções
                  próprias
                </li>
              </ul>

              <p>
                <a
                  href="/pt/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Veja todos os canais de notificação do Databasus →
                </a>
              </p>

              <h2 id="multi-server-management">Gestão de vários servidores</h2>

              <p>
                As duas ferramentas conseguem administrar backups de vários
                servidores PostgreSQL, mas com abordagens diferentes:
              </p>

              <h3 id="multi-databasus">Abordagem do Databasus</h3>

              <ul>
                <li>
                  <strong>Agendamento por base de dados</strong>: cada base de
                  dados pode ter agenda de backup e destino de armazenamento
                  próprios.
                </li>
                <li>
                  <strong>Organização por workspaces</strong>: agrupe bases de
                  dados relacionadas em workspaces para facilitar a gestão.
                </li>
                <li>
                  <strong>Painel unificado</strong>: veja todos os backups de
                  bases de dados e o status deles em uma única interface web.
                </li>
              </ul>

              <h3 id="multi-barman">Abordagem do Barman</h3>

              <ul>
                <li>
                  <strong>Servidor de backup centralizado</strong>: um servidor
                  Barman dedicado administra os backups de várias instâncias
                  PostgreSQL.
                </li>
                <li>
                  <strong>Configuração por servidor</strong>: cada servidor
                  PostgreSQL requer arquivo de configuração próprio no servidor
                  Barman.
                </li>
                <li>
                  <strong>Georredundância</strong>: servidores Barman podem
                  replicar para outros servidores Barman para redundância
                  geográfica.
                </li>
              </ul>

              <h2 id="conclusion">Conclusão</h2>

              <p>
                Databasus e Barman atendem necessidades diferentes no
                ecossistema de backup PostgreSQL. A escolha certa depende dos
                seus requisitos de recuperação, da estrutura da equipe e do
                conhecimento técnico disponível.
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Escolha o Databasus se:
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Você é um desenvolvedor individual, uma equipe ou uma
                    empresa em busca de uma solução de backup intuitiva
                  </li>
                  <li>
                    Você prefere uma interface web a ferramentas de linha de
                    comando
                  </li>
                  <li>
                    Você precisa de recursos de colaboração em equipe
                    (workspaces, RBAC, logs de auditoria)
                  </li>
                  <li>
                    Você quer notificações integradas para Slack, Teams,
                    Telegram etc.
                  </li>
                  <li>
                    Você quer administrar backups de várias bases de dados a
                    partir de um único painel, com agendamento, notificações e
                    recursos de equipe
                  </li>
                  <li>
                    Você quer uma configuração rápida com o mínimo de
                    conhecimento de PostgreSQL
                  </li>
                  <li>
                    A criptografia integrada de backups é importante para você
                  </li>
                  <li>
                    Você usa bases de dados gerenciadas na nuvem (AWS RDS,
                    Google Cloud SQL, Azure) ou PostgreSQL self-hosted
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Escolha o Barman se:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Você precisa de backups físicos ou incrementais em
                    PostgreSQL anterior à versão 17 (o Barman traz motor de
                    backup próprio)
                  </li>
                  <li>
                    Você precisa de backups incrementais baseados em rsync
                    (comparação em nível de arquivo) para reduzir o tempo de
                    transferência
                  </li>
                  <li>
                    Você precisa de integração com replicação por streaming para
                    arquivamento de WAL em tempo real
                  </li>
                  <li>
                    Você precisa de redundância geográfica Barman a Barman
                  </li>
                  <li>
                    Você precisa criar standby a partir de backups para
                    configurações de alta disponibilidade
                  </li>
                  <li>
                    Você está à vontade com ferramentas de linha de comando e o
                    funcionamento interno do PostgreSQL
                  </li>
                  <li>Sua organização conta com experiência de DBA dedicada</li>
                </ul>
              </div>

              <p>
                As duas ferramentas suportam backups físicos, arquivamento de
                WAL e PITR, e as duas foram criadas para recuperação de
                desastres com RTO e RPO mínimos. O Databasus funciona com bases
                de dados de qualquer tamanho e complexidade e oferece interface
                web, recursos de equipe e backups lógicos e físicos tanto em
                bases self-hosted quanto gerenciadas na nuvem. O Barman é a
                melhor opção quando você precisa de backups incrementais
                baseados em rsync, integração com replicação por streaming,
                georredundância Barman a Barman ou criação de standby a partir
                de backups.
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
