import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Databasus vs WAL-G: comparação de ferramentas de backup PostgreSQL",
  description:
    "Compare as ferramentas de backup PostgreSQL Databasus e WAL-G. Veja as diferenças na abordagem de backup, suporte a várias bases de dados, facilidade de uso, recursos de equipe e quando escolher cada ferramenta.",
  keywords: [
    "Databasus vs WAL-G",
    "comparação de backup PostgreSQL",
    "alternativa ao WAL-G",
    "ferramentas de backup PostgreSQL",
    "comparação de backup de bases de dados",
    "pg_dump vs arquivamento de WAL",
    "backup self-hosted",
    "PITR PostgreSQL",
    "arquivamento de WAL",
    "backup de várias bases de dados",
  ],
  openGraph: {
    title: "Databasus vs WAL-G: comparação de ferramentas de backup PostgreSQL",
    description:
      "Compare as ferramentas de backup PostgreSQL Databasus e WAL-G. Veja as diferenças na abordagem de backup, suporte a várias bases de dados, facilidade de uso, recursos de equipe e quando escolher cada ferramenta.",
    type: "article",
    url: getLocalizedUrl("pt", "databasus-vs-wal-g"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Databasus vs WAL-G: comparação de ferramentas de backup PostgreSQL",
    description:
      "Compare as ferramentas de backup PostgreSQL Databasus e WAL-G. Veja as diferenças na abordagem de backup, suporte a várias bases de dados, facilidade de uso, recursos de equipe e quando escolher cada ferramenta.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "databasus-vs-wal-g"),
    languages: getLanguageAlternates("databasus-vs-wal-g"),
  },
  robots: "index, follow",
};

export default function DatabasusVsWalGPage() {
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
              "Databasus vs WAL-G: comparação de ferramentas de backup PostgreSQL",
            description:
              "Uma comparação completa das ferramentas de backup PostgreSQL Databasus e WAL-G, cobrindo abordagem de backup, suporte a várias bases de dados, facilidade de uso, recursos de equipe e quando escolher cada ferramenta.",
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
              <h1 id="databasus-vs-wal-g">Databasus vs WAL-G</h1>

              <p className="text-lg text-gray-400">
                Databasus e WAL-G foram criados para recuperação de desastres
                com RTO e RPO mínimos, e ambos suportam backups físicos do
                PostgreSQL, arquivamento de WAL e recuperação a um ponto no
                tempo (PITR). O Databasus executa esses backups remotamente
                sobre a pilha nativa do PostgreSQL 17, reutilizando as
                ferramentas já comprovadas do próprio PostgreSQL em vez de
                reinventá-las, tudo por trás de uma interface web intuitiva.
                Funciona com bases de dados de qualquer tamanho e complexidade.
                Backups físicos exigem PostgreSQL 17 ou mais recente; em versões
                anteriores só estão disponíveis backups lógicos com{" "}
                <code>pg_dump</code>. O WAL-G é uma ferramenta de linha de
                comando com motor próprio, por isso cobre backups físicos em
                versões bem mais antigas do PostgreSQL, usa um protocolo de
                streaming próprio com desempenho um pouco melhor, suporta
                backups delta (apenas páginas alteradas) e cobre mais motores de
                base de dados, incluindo MS SQL, FoundationDB e Greenplum.
              </p>

              <h2 id="quick-comparison">Comparação rápida</h2>

              <p>Veja as principais diferenças entre Databasus e WAL-G:</p>

              <table>
                <thead>
                  <tr>
                    <th>Recurso</th>
                    <th>Databasus</th>
                    <th>WAL-G</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Gestão de backups</td>
                    <td data-label="Databasus">
                      ✅ Sim (várias bases de dados)
                    </td>
                    <td data-label="WAL-G">
                      ❌ Não (apenas uma base de dados)
                    </td>
                  </tr>
                  <tr>
                    <td>Suporte a outras bases de dados</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="WAL-G">✅ PostgreSQL, MySQL, MS SQL</td>
                  </tr>
                  <tr>
                    <td>Interface</td>
                    <td data-label="Databasus">Interface web</td>
                    <td data-label="WAL-G">Apenas linha de comando</td>
                  </tr>
                  <tr>
                    <td>Tipo de backup</td>
                    <td data-label="Databasus">Lógico + físico</td>
                    <td data-label="WAL-G">Físico (arquivamento de WAL)</td>
                  </tr>
                  <tr>
                    <td>Versão do PostgreSQL para backups físicos</td>
                    <td data-label="Databasus">17+ (nativo)</td>
                    <td data-label="WAL-G">9.x+ (motor próprio)</td>
                  </tr>
                  <tr>
                    <td>Agendamento de backups</td>
                    <td data-label="Databasus">✅ Agendador integrado</td>
                    <td data-label="WAL-G">Requer ferramenta externa (cron)</td>
                  </tr>
                  <tr>
                    <td>Opções de recuperação</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="WAL-G">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Backups incrementais</td>
                    <td data-label="Databasus">
                      ✅ Em nível de bloco (PG 17+)
                    </td>
                    <td data-label="WAL-G">
                      Backups delta (apenas páginas alteradas)
                    </td>
                  </tr>
                  <tr>
                    <td>Backups remotos</td>
                    <td data-label="Databasus">✅ Sim</td>
                    <td data-label="WAL-G">❌ Não (executa localmente)</td>
                  </tr>
                  <tr>
                    <td>Recursos de equipe</td>
                    <td data-label="Databasus">
                      ✅ Workspaces, RBAC, logs de auditoria
                    </td>
                    <td data-label="WAL-G">
                      ❌ Apenas permissões do sistema operacional
                    </td>
                  </tr>
                  <tr>
                    <td>Notificações</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, e-mail
                    </td>
                    <td data-label="WAL-G">❌ Requer scripts próprios</td>
                  </tr>
                  <tr>
                    <td>Criptografia</td>
                    <td data-label="Databasus">AES-256-GCM integrada</td>
                    <td data-label="WAL-G">GPG ou libsodium</td>
                  </tr>
                  <tr>
                    <td>Curva de aprendizado</td>
                    <td data-label="Databasus">Mínima</td>
                    <td data-label="WAL-G">Exige domínio de CLI</td>
                  </tr>
                  <tr>
                    <td>Instalação</td>
                    <td data-label="Databasus">
                      Script de uma linha ou Docker
                    </td>
                    <td data-label="WAL-G">
                      Download do binário + configuração
                    </td>
                  </tr>
                  <tr>
                    <td>Adequado para bases de dados self-hosted</td>
                    <td data-label="Databasus">✅ Sim</td>
                    <td data-label="WAL-G">✅ Sim</td>
                  </tr>
                  <tr>
                    <td>Adequado para bases de dados na nuvem</td>
                    <td data-label="Databasus">
                      ✅ Sim (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="WAL-G">
                      ❌ Apenas backup (sem restauração para a nuvem)
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="database-focus">Foco em bases de dados</h2>

              <p>
                Uma das diferenças mais significativas entre as ferramentas é o
                escopo de bases de dados que cobrem:
              </p>

              <h3 id="focus-databasus">
                Databasus: gestão completa de backups
              </h3>

              <p>
                O Databasus foi criado para a gestão completa de backups de
                vários sistemas de bases de dados, com foco na facilidade de
                uso:
              </p>

              <ul>
                <li>
                  <strong>Suporte a várias bases de dados</strong>: administre
                  backups de PostgreSQL, MySQL, MariaDB e MongoDB a partir de
                  uma única interface.
                </li>
                <li>
                  <strong>Experiência unificada</strong>: a interface, os fluxos
                  de trabalho e os recursos funcionam de forma consistente em
                  todas as bases de dados suportadas.
                </li>
                <li>
                  <strong>Suporte a versões</strong>: suporta as versões 12 a 18
                  do PostgreSQL, com otimizações específicas por versão.
                </li>
                <li>
                  <strong>Gestão simplificada</strong>: todo o esforço de
                  desenvolvimento vai para melhorar a experiência de gestão de
                  backups.
                </li>
              </ul>

              <h3 id="focus-wal-g">WAL-G: suporte a várias bases de dados</h3>

              <p>
                O WAL-G começou como uma ferramenta de backup para PostgreSQL,
                mas se expandiu para suportar vários sistemas de bases de dados:
              </p>

              <ul>
                <li>
                  <strong>PostgreSQL</strong>: a implementação original e mais
                  madura.
                </li>
                <li>
                  <strong>MySQL/MariaDB</strong>: suporta backups baseados em
                  binlog.
                </li>
                <li>
                  <strong>MS SQL Server</strong>: backups de SQL Server em
                  Windows.
                </li>
                <li>
                  <strong>MongoDB</strong>: suporte a backup de base de dados de
                  documentos.
                </li>
                <li>
                  <strong>FoundationDB</strong>: suporte a base de dados
                  distribuída.
                </li>
                <li>
                  <strong>Greenplum</strong>: suporte a backup de data
                  warehouse.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">
                    Quando a gestão completa importa:
                  </strong>{" "}
                  se você precisa administrar backups de várias bases de dados
                  com uma interface unificada, o Databasus oferece uma
                  experiência simplificada. Você tem gestão centralizada de
                  backups, com recursos de equipe, sem a complexidade de lidar
                  com ferramentas diferentes para cada base de dados.
                </p>
              </div>

              <h2 id="target-audience">Público-alvo</h2>

              <p>
                As ferramentas atendem perfis de usuário diferentes, de acordo
                com a filosofia de design de cada uma:
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
                  <strong>Ambientes com várias bases de dados</strong>:
                  organizações que executam PostgreSQL, MySQL, MariaDB ou
                  MongoDB se beneficiam da gestão centralizada de backups.
                </li>
                <li>
                  <strong>DBAs e recuperação de desastres</strong>: backups
                  físicos, arquivamento de WAL e PITR para sistemas críticos com
                  exigência de perda de dados quase nula.
                </li>
                <li>
                  <strong>Engenheiros de DevOps</strong>: o modo agente se
                  integra à infraestrutura existente, enquanto a interface web e
                  a API dão visibilidade e controle sem scripts próprios.
                </li>
              </ul>

              <h3 id="audience-wal-g">Público do WAL-G</h3>

              <p>
                O WAL-G foi projetado para usuários à vontade com ferramentas de
                linha de comando:
              </p>

              <ul>
                <li>
                  <strong>Engenheiros de DevOps</strong>: quem prefere
                  infraestrutura como código e fluxos de trabalho baseados em
                  CLI.
                </li>
                <li>
                  <strong>Ambientes com várias bases de dados</strong>:
                  organizações que executam PostgreSQL junto com MySQL, MongoDB
                  ou outras bases de dados suportadas.
                </li>
                <li>
                  <strong>Implantações cloud-native</strong>: equipes que usam
                  Kubernetes ou ambientes em contêineres, onde ferramentas CLI
                  se integram bem.
                </li>
                <li>
                  <strong>Suporte estendido a bases de dados</strong>: equipes
                  que precisam de backup de MS SQL, FoundationDB ou Greenplum
                  além do PostgreSQL.
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
                  <strong>Compressão eficiente</strong>: usa zstd (nível 5) para
                  os dois tipos de backup, reduzindo o tamanho em 4 a 8 vezes.
                </li>
                <li>
                  <strong>Acesso somente leitura</strong>: backups lógicos
                  exigem apenas permissões SELECT, minimizando riscos de
                  segurança.
                </li>
              </ul>

              <h3 id="backup-wal-g">
                WAL-G: backups físicos com arquivamento de WAL
              </h3>

              <p>
                O WAL-G realiza backups em nível de arquivo (físicos) com
                arquivamento contínuo de WAL:
              </p>

              <ul>
                <li>
                  <strong>Backups base</strong>: cópias completas, em nível de
                  arquivo, do diretório de dados do PostgreSQL.
                </li>
                <li>
                  <strong>Backups delta</strong>: apenas as páginas alteradas
                  são copiadas, reduzindo armazenamento e tempo de
                  transferência.
                </li>
                <li>
                  <strong>Arquivamento de WAL</strong>: o arquivamento contínuo
                  dos Write-Ahead Logs permite a recuperação a um ponto no
                  tempo.
                </li>
                <li>
                  <strong>Otimização copy-on-write</strong>: tratamento
                  eficiente de blocos de dados não alterados.
                </li>
              </ul>

              <h2 id="recovery-options">Opções de recuperação</h2>

              <p>
                As duas ferramentas oferecem recuperação, mas com granularidade
                diferente:
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

              <h3 id="recovery-wal-g">Recuperação com o WAL-G</h3>

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
                  <strong>Restauração delta</strong>: recuperação mais rápida ao
                  buscar apenas as páginas alteradas.
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
                  ferramentas suportam PITR. O WAL-G oferece adicionalmente a
                  restauração delta (buscando apenas as páginas alteradas) e usa
                  um protocolo de streaming próprio com desempenho um pouco
                  melhor em grande escala.{" "}
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
                As ferramentas diferem bastante na abordagem à experiência do
                usuário:
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

              <h3 id="ease-wal-g">Experiência de uso do WAL-G</h3>

              <ul>
                <li>
                  <strong>Interface de linha de comando</strong>: todas as
                  operações são feitas por comandos de terminal como{" "}
                  <code>wal-g backup-push</code>,{" "}
                  <code>wal-g backup-fetch</code>.
                </li>
                <li>
                  <strong>Variáveis de ambiente</strong>: a configuração é feita
                  principalmente por variáveis de ambiente, não por arquivos de
                  configuração.
                </li>
                <li>
                  <strong>Agendamento externo</strong>: requer tarefas cron ou
                  orquestração externa para backups automáticos.
                </li>
                <li>
                  <strong>Configuração do arquivamento de WAL</strong>: é
                  preciso configurar o <code>archive_command</code> do
                  PostgreSQL para integrá-lo ao WAL-G.
                </li>
                <li>
                  <strong>Domínio de CLI esperado</strong>: a documentação
                  presume familiaridade com ferramentas de linha de comando e
                  scripts de shell.
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

              <h3 id="team-wal-g">Recursos de equipe do WAL-G</h3>

              <p>
                O WAL-G é uma ferramenta de linha de comando sem recursos de
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
                  políticas IAM da nuvem
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

              <h3 id="security-wal-g">Segurança do WAL-G</h3>

              <ul>
                <li>
                  <strong>Criptografia GPG</strong>: suporta criptografia
                  baseada em GPG para os arquivos de backup.
                </li>
                <li>
                  <strong>Criptografia libsodium</strong>: criptografia
                  alternativa com a biblioteca libsodium.
                </li>
                <li>
                  <strong>Integração com IAM da nuvem</strong>: usa o IAM do
                  provedor de nuvem para controlar o acesso ao armazenamento.
                </li>
                <li>
                  <strong>Sem gestão integrada de credenciais</strong>: depende
                  de variáveis de ambiente ou de gestão externa de segredos.
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
                As duas ferramentas suportam armazenamento em nuvem, com focos
                diferentes:
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

              <h3 id="storage-wal-g">Armazenamento no WAL-G</h3>

              <p>Opções de armazenamento cloud-native:</p>

              <ul>
                <li>Amazon S3</li>
                <li>Google Cloud Storage (GCS)</li>
                <li>Azure Blob Storage</li>
                <li>Swift (OpenStack)</li>
                <li>Sistema de arquivos local</li>
                <li>SSH/SFTP</li>
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

              <h3 id="notifications-wal-g">Notificações do WAL-G</h3>

              <p>
                O WAL-G não tem suporte integrado a notificações. Notificações
                exigem:
              </p>

              <ul>
                <li>Scripts próprios em torno dos comandos de backup</li>
                <li>Integração com ferramentas externas de monitoramento</li>
                <li>Análise manual de logs e configuração de alertas</li>
                <li>
                  Integração com ferramentas como Prometheus, Grafana ou
                  soluções próprias
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

              <h2 id="compression">Compressão</h2>

              <p>
                As duas ferramentas oferecem compressão para reduzir o tamanho
                dos backups:
              </p>

              <h3 id="compression-databasus">Compressão no Databasus</h3>

              <ul>
                <li>
                  <strong>Compressão zstd</strong>: usa zstd no nível 5,
                  equilibrando velocidade e taxa de compressão.
                </li>
                <li>
                  <strong>Redução de 4 a 8 vezes</strong>: taxas de compressão
                  típicas com apenas ~20% de tempo adicional de execução.
                </li>
                <li>
                  <strong>Automática</strong>: a compressão vem ativada por
                  padrão, sem necessidade de configuração.
                </li>
              </ul>

              <h3 id="compression-wal-g">Compressão no WAL-G</h3>

              <ul>
                <li>
                  <strong>Vários algoritmos</strong>: suporta LZ4, LZMA, Brotli
                  e zstd.
                </li>
                <li>
                  <strong>Níveis configuráveis</strong>: ajuste fino do
                  equilíbrio entre taxa de compressão e velocidade.
                </li>
                <li>
                  <strong>Compressão por arquivo</strong>: arquivos de WAL e
                  backups base podem usar configurações diferentes.
                </li>
              </ul>

              <h2 id="conclusion">Conclusão</h2>

              <p>
                Databasus e WAL-G atendem necessidades diferentes no ecossistema
                de backup PostgreSQL. A escolha certa depende do seu ambiente de
                bases de dados, da estrutura da equipe e das preferências
                operacionais.
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Escolha o Databasus se:
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Você precisa de gestão completa de backups do PostgreSQL a
                    partir de uma única interface
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
                    Você quer agendamento integrado, sem configurar cron externo
                  </li>
                  <li>
                    Você quer administrar backups de várias bases de dados a
                    partir de um único painel, com agendamento, notificações e
                    recursos de equipe
                  </li>
                  <li>
                    Você quer uma configuração rápida com o mínimo de
                    conhecimento de bases de dados
                  </li>
                  <li>
                    A criptografia integrada de backups é importante para você
                  </li>
                  <li>
                    Você usa bases de dados gerenciadas na nuvem (AWS RDS,
                    Google Cloud SQL, Azure) ou self-hosted
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Escolha o WAL-G se:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Você precisa de backups físicos ou incrementais em
                    PostgreSQL anterior à versão 17 (o WAL-G traz motor de
                    backup próprio)
                  </li>
                  <li>
                    Você precisa de backups delta (apenas páginas alteradas)
                    para reduzir armazenamento e tempo de transferência
                  </li>
                  <li>
                    Você precisa de suporte a MS SQL, FoundationDB ou Greenplum
                  </li>
                  <li>
                    Você prefere ferramentas de linha de comando e fluxos de
                    infraestrutura como código
                  </li>
                  <li>
                    Você quer vários algoritmos de compressão (LZ4, LZMA,
                    Brotli, zstd) com controle fino
                  </li>
                  <li>
                    Sua equipe tem experiência de DevOps para administrar
                    ferramentas por CLI
                  </li>
                </ul>
              </div>

              <p>
                As duas ferramentas suportam backups físicos, arquivamento de
                WAL e PITR, e as duas foram criadas para recuperação de
                desastres com RTO e RPO mínimos. O Databasus funciona com bases
                de dados de qualquer tamanho e complexidade e oferece interface
                web, recursos de equipe e backups lógicos e físicos tanto em
                bases self-hosted quanto gerenciadas na nuvem.
                <br />
                <br />O WAL-G continua sendo uma excelente escolha para equipes
                que preferem fluxos por CLI e precisam das vantagens exclusivas
                dele: backups delta (apenas páginas alteradas), um protocolo de
                streaming próprio com desempenho um pouco melhor e suporte a
                motores de base de dados além do PostgreSQL.
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
