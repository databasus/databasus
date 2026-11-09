import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Databasus vs pgBackRest: comparação de ferramentas de backup PostgreSQL",
  description:
    "Compare as ferramentas de backup PostgreSQL Databasus e pgBackRest. Veja as diferenças na abordagem de backup, público-alvo, facilidade de uso, opções de recuperação e quando escolher cada ferramenta.",
  keywords: [
    "Databasus vs pgBackRest",
    "comparação de backup PostgreSQL",
    "alternativa ao pgBackRest",
    "ferramentas de backup PostgreSQL",
    "comparação de backup de bases de dados",
    "pg_dump vs backup físico",
    "backup self-hosted",
    "PITR PostgreSQL",
    "backup de bases de dados grandes",
    "ferramentas de backup para DBAs",
  ],
  openGraph: {
    title:
      "Databasus vs pgBackRest: comparação de ferramentas de backup PostgreSQL",
    description:
      "Compare as ferramentas de backup PostgreSQL Databasus e pgBackRest. Veja as diferenças na abordagem de backup, público-alvo, facilidade de uso, opções de recuperação e quando escolher cada ferramenta.",
    type: "article",
    url: getLocalizedUrl("pt", "databasus-vs-pgbackrest"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs pgBackRest: comparação de ferramentas de backup PostgreSQL",
    description:
      "Compare as ferramentas de backup PostgreSQL Databasus e pgBackRest. Veja as diferenças na abordagem de backup, público-alvo, facilidade de uso, opções de recuperação e quando escolher cada ferramenta.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "databasus-vs-pgbackrest"),
    languages: getLanguageAlternates("databasus-vs-pgbackrest"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackRestPage() {
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
              "Databasus vs pgBackRest: comparação de ferramentas de backup PostgreSQL",
            description:
              "Uma comparação completa das ferramentas de backup PostgreSQL Databasus e pgBackRest, cobrindo abordagem de backup, público-alvo, facilidade de uso, opções de recuperação e quando escolher cada ferramenta.",
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
              <h1 id="databasus-vs-pgbackrest">Databasus vs pgBackRest</h1>

              <p className="text-lg text-gray-400">
                Databasus e pgBackRest foram criados para recuperação de
                desastres com RTO e RPO mínimos, e ambos suportam backups
                físicos, arquivamento de WAL e recuperação a um ponto no tempo
                (PITR). O Databasus executa esses backups remotamente sobre a
                pilha nativa do PostgreSQL 17, reutilizando as ferramentas já
                comprovadas do próprio PostgreSQL em vez de reinventá-las, tudo
                por trás de uma interface web intuitiva. Funciona com bases de
                dados de qualquer tamanho e complexidade. Backups físicos exigem
                PostgreSQL 17 ou mais recente; em versões anteriores só estão
                disponíveis backups lógicos com <code>pg_dump</code>. O
                pgBackRest traz motor de backup próprio, por isso cobre também
                backups físicos e incrementais em versões bem mais antigas do
                PostgreSQL e acrescenta recursos avançados como backups
                diferenciais e restauração delta.
              </p>

              <h2 id="quick-comparison">Comparação rápida</h2>

              <p>Veja as principais diferenças entre Databasus e pgBackRest:</p>

              <table>
                <thead>
                  <tr>
                    <th>Recurso</th>
                    <th>Databasus</th>
                    <th>pgBackRest</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Público-alvo</td>
                    <td data-label="Databasus">
                      Indivíduos, equipes, DBAs, empresas
                    </td>
                    <td data-label="pgBackRest">
                      DBAs, fluxos de linha de comando e IaC
                    </td>
                  </tr>
                  <tr>
                    <td>Gestão de backups</td>
                    <td data-label="Databasus">✅ Várias bases de dados</td>
                    <td data-label="pgBackRest">❌ Apenas uma base de dados</td>
                  </tr>
                  <tr>
                    <td>Suporte a outras bases de dados</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="pgBackRest">❌ Apenas PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>Interface</td>
                    <td data-label="Databasus">Interface web</td>
                    <td data-label="pgBackRest">
                      Linha de comando, arquivos de configuração
                    </td>
                  </tr>
                  <tr>
                    <td>Tipo de backup</td>
                    <td data-label="Databasus">Lógico + físico</td>
                    <td data-label="pgBackRest">Físico (nível de arquivo)</td>
                  </tr>
                  <tr>
                    <td>Versão do PostgreSQL para backups físicos</td>
                    <td data-label="Databasus">17+ (nativo)</td>
                    <td data-label="pgBackRest">9.4+ (motor próprio)</td>
                  </tr>
                  <tr>
                    <td>Opções de recuperação</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="pgBackRest">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Operações em paralelo</td>
                    <td data-label="Databasus">✅ Sim</td>
                    <td data-label="pgBackRest">✅ Sim</td>
                  </tr>
                  <tr>
                    <td>Backups incrementais</td>
                    <td data-label="Databasus">
                      ✅ Em nível de bloco (PG 17+)
                    </td>
                    <td data-label="pgBackRest">
                      Incremental em nível de bloco
                    </td>
                  </tr>
                  <tr>
                    <td>Backups diferenciais</td>
                    <td data-label="Databasus">❌ Não</td>
                    <td data-label="pgBackRest">✅ Sim</td>
                  </tr>
                  <tr>
                    <td>Restauração delta</td>
                    <td data-label="Databasus">❌ Não</td>
                    <td data-label="pgBackRest">
                      ✅ Sim (apenas arquivos alterados)
                    </td>
                  </tr>
                  <tr>
                    <td>Backups remotos</td>
                    <td data-label="Databasus">✅ Sim</td>
                    <td data-label="pgBackRest">
                      ❌ Não (requer acesso ao sistema de arquivos)
                    </td>
                  </tr>
                  <tr>
                    <td>Recursos de equipe</td>
                    <td data-label="Databasus">
                      ✅ Workspaces, RBAC, logs de auditoria
                    </td>
                    <td data-label="pgBackRest">❌ Usuário único</td>
                  </tr>
                  <tr>
                    <td>Curva de aprendizado</td>
                    <td data-label="Databasus">Mínima</td>
                    <td data-label="pgBackRest">Exige experiência de DBA</td>
                  </tr>
                  <tr>
                    <td>Instalação</td>
                    <td data-label="Databasus">
                      Script de uma linha ou Docker
                    </td>
                    <td data-label="pgBackRest">Requer configuração manual</td>
                  </tr>
                  <tr>
                    <td>Adequado para bases de dados self-hosted</td>
                    <td data-label="Databasus">✅ Sim</td>
                    <td data-label="pgBackRest">✅ Sim</td>
                  </tr>
                  <tr>
                    <td>Adequado para bases de dados na nuvem</td>
                    <td data-label="Databasus">
                      ✅ Sim (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="pgBackRest">
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

              <h3 id="audience-pgbackrest">Público do pgBackRest</h3>

              <p>
                O pgBackRest é uma ferramenta de linha de comando voltada a
                equipes que querem operar o próprio motor de backup:
              </p>

              <ul>
                <li>
                  <strong>Fluxos de linha de comando e IaC</strong>: equipes que
                  preferem configurar backups por arquivos e scripts em vez de
                  uma interface web.
                </li>
                <li>
                  <strong>Versões antigas do PostgreSQL</strong>: o pgBackRest
                  traz motor próprio, por isso executa backups físicos e
                  incrementais em versões do PostgreSQL anteriores à 17.
                </li>
                <li>
                  <strong>Recursos avançados em grande escala</strong>: quando
                  backups diferenciais, restauração delta e criação de standby
                  compensam a configuração adicional.
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
                  diferentes do PostgreSQL, com restauração seletiva de tabelas
                  ou esquemas específicos. É também o único tipo de backup em
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

              <h3 id="backup-pgbackrest">pgBackRest: backups físicos</h3>

              <p>
                O pgBackRest realiza backups em nível de arquivo (físicos) do
                diretório de dados do PostgreSQL:
              </p>

              <ul>
                <li>
                  <strong>Incremental em nível de bloco</strong>: apenas os
                  blocos alterados são copiados, reduzindo o tempo de backup e o
                  armazenamento em bases de dados muito grandes.
                </li>
                <li>
                  <strong>Arquivamento de WAL</strong>: o arquivamento contínuo
                  dos Write-Ahead Logs permite recuperação precisa a um ponto no
                  tempo.
                </li>
                <li>
                  <strong>Completo, diferencial, incremental</strong>: várias
                  estratégias de backup para diferentes cenários de recuperação.
                </li>
                <li>
                  <strong>Otimizado para grande escala</strong>: projetado para
                  bases de dados em que backups lógicos demorariam demais.
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

              <h3 id="recovery-pgbackrest">Recuperação com o pgBackRest</h3>

              <ul>
                <li>
                  <strong>Recuperação a um ponto no tempo (PITR)</strong>:
                  restaure para qualquer segundo específico com a reaplicação do
                  WAL.
                </li>
                <li>
                  <strong>Restauração completa do cluster</strong>: restaure o
                  cluster inteiro da base de dados para um ponto específico no
                  tempo a partir de backups físicos.
                </li>
                <li>
                  <strong>Restauração em paralelo</strong>: restauração
                  multithread para recuperação mais rápida de bases de dados
                  grandes.
                </li>
                <li>
                  <strong>Restauração delta</strong>: restaure apenas os
                  arquivos alterados, reduzindo o tempo de recuperação.
                </li>
                <li>
                  <strong>Criação de standby</strong>: crie réplicas do
                  PostgreSQL a partir de backups.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Nota:</strong> as duas
                  ferramentas suportam PITR. O pgBackRest oferece adicionalmente
                  a restauração delta (buscando apenas os arquivos alterados),
                  backups diferenciais e criação de standby a partir de backups.{" "}
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

              <h3 id="ease-pgbackrest">Experiência de uso do pgBackRest</h3>

              <ul>
                <li>
                  <strong>Interface de linha de comando</strong>: todas as
                  operações são feitas por comandos de terminal.
                </li>
                <li>
                  <strong>Arquivos de configuração</strong>: requer edição
                  manual de arquivos de configuração no estilo INI.
                </li>
                <li>
                  <strong>Configuração do arquivamento de WAL</strong>: é
                  preciso configurar o <code>archive_command</code> do
                  PostgreSQL e os ajustes relacionados.
                </li>
                <li>
                  <strong>Curva de aprendizado íngreme</strong>: exige
                  compreensão do funcionamento interno do PostgreSQL, da
                  mecânica do WAL e das estratégias de backup.
                </li>
                <li>
                  <strong>Experiência de DBA esperada</strong>: a documentação
                  presume familiaridade com conceitos de administração de bases
                  de dados.
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

              <h3 id="team-pgbackrest">Recursos de equipe do pgBackRest</h3>

              <p>
                O pgBackRest é uma ferramenta de linha de comando sem recursos
                de equipe integrados:
              </p>

              <ul>
                <li>Sem gestão de usuários nem controle de acesso</li>
                <li>Sem log de auditoria das operações</li>
                <li>
                  A coordenação da equipe requer ferramentas e processos
                  externos
                </li>
                <li>
                  Acesso controlado por permissões do sistema operacional nos
                  arquivos de configuração
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

              <p>As duas ferramentas oferecem recursos sólidos de segurança:</p>

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

              <h3 id="security-pgbackrest">Segurança do pgBackRest</h3>

              <ul>
                <li>
                  <strong>Criptografia do repositório</strong>: os repositórios
                  de backup podem ser criptografados com AES-256.
                </li>
                <li>
                  <strong>Transporte TLS/SSH</strong>: comunicação segura para
                  operações remotas.
                </li>
                <li>
                  <strong>Verificação por checksum</strong>: valida a
                  integridade dos backups durante a criação e a restauração.
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
                As duas ferramentas suportam vários destinos de armazenamento; o
                Databasus tem as opções mais acessíveis:
              </p>

              <h3 id="storage-databasus">Armazenamento no Databasus</h3>

              <ul>
                <li>Armazenamento local</li>
                <li>Amazon S3 e serviços compatíveis com S3</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (armazenamento conectado à rede)</li>
                <li>FTP</li>
              </ul>

              <h3 id="storage-pgbackrest">Armazenamento no pgBackRest</h3>

              <ul>
                <li>Armazenamento local (POSIX, CIFS)</li>
                <li>Amazon S3 e serviços compatíveis com S3</li>
                <li>Cloudflare R2 (compatível com S3)</li>
                <li>Azure Blob Storage</li>
                <li>NAS (armazenamento conectado à rede)</li>
                <li>Google Cloud Storage</li>
                <li>SFTP</li>
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

              <h3 id="notifications-pgbackrest">Notificações do pgBackRest</h3>

              <p>
                O pgBackRest não tem suporte integrado a notificações.
                Notificações exigem:
              </p>

              <ul>
                <li>Scripts próprios em torno dos comandos de backup</li>
                <li>Integração com ferramentas externas de monitoramento</li>
                <li>Análise manual de logs e configuração de alertas</li>
              </ul>

              <p>
                <a
                  href="/pt/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Veja todos os canais de notificação do Databasus →
                </a>
              </p>

              <h2 id="conclusion">Conclusão</h2>

              <p>
                Databasus e pgBackRest atendem necessidades diferentes no
                ecossistema de backup PostgreSQL. A escolha certa depende do
                tamanho das suas bases de dados, da estrutura da equipe e dos
                requisitos técnicos.
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
                    Você usa bases de dados gerenciadas na nuvem (AWS RDS,
                    Google Cloud SQL, Azure) ou PostgreSQL self-hosted
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Escolha o pgBackRest se:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Você precisa de backups físicos ou incrementais em
                    PostgreSQL anterior à versão 17 (o pgBackRest traz motor de
                    backup próprio)
                  </li>
                  <li>
                    Você precisa de backups diferenciais ou restauração delta
                    (apenas arquivos alterados)
                  </li>
                  <li>
                    Você precisa criar standby a partir de backups para alta
                    disponibilidade
                  </li>
                  <li>
                    Você prefere ferramentas de linha de comando e fluxos de
                    infraestrutura como código
                  </li>
                  <li>
                    Sua equipe tem o conhecimento de PostgreSQL para operar e
                    ajustar a ferramenta
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
                <br />O pgBackRest é a melhor opção quando você quer operar o
                próprio motor de backup, precisa de backups físicos em
                PostgreSQL anterior à versão 17 ou depende dos backups
                diferenciais e da restauração delta dele.
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
