import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Databasus vs PgBackWeb: comparação de ferramentas de backup PostgreSQL",
  description:
    "Compare as ferramentas de backup PostgreSQL Databasus e PgBackWeb. Veja as diferenças em recursos, segurança, suporte a equipes, opções de armazenamento, notificações e facilidade de uso.",
  keywords: [
    "Databasus vs PgBackWeb",
    "comparação de backup PostgreSQL",
    "alternativa ao PgBackWeb",
    "ferramentas de backup PostgreSQL",
    "comparação de backup de bases de dados",
    "interface gráfica para pg_dump",
    "backup self-hosted",
    "segurança de backup PostgreSQL",
  ],
  openGraph: {
    title:
      "Databasus vs PgBackWeb: comparação de ferramentas de backup PostgreSQL",
    description:
      "Compare as ferramentas de backup PostgreSQL Databasus e PgBackWeb. Veja as diferenças em recursos, segurança, suporte a equipes, opções de armazenamento, notificações e facilidade de uso.",
    type: "article",
    url: getLocalizedUrl("pt", "databasus-vs-pgbackweb"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs PgBackWeb: comparação de ferramentas de backup PostgreSQL",
    description:
      "Compare as ferramentas de backup PostgreSQL Databasus e PgBackWeb. Veja as diferenças em recursos, segurança, suporte a equipes, opções de armazenamento, notificações e facilidade de uso.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "databasus-vs-pgbackweb"),
    languages: getLanguageAlternates("databasus-vs-pgbackweb"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackWebPage() {
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
              "Databasus vs PgBackWeb: comparação de ferramentas de backup PostgreSQL",
            description:
              "Uma comparação completa das ferramentas de backup PostgreSQL Databasus e PgBackWeb, cobrindo recursos, segurança, suporte a equipes, opções de armazenamento e facilidade de uso.",
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
              <h1 id="databasus-vs-pgbackweb">Databasus vs PgBackWeb</h1>

              <p className="text-lg text-gray-400">
                Databasus e PgBackWeb são ferramentas de código aberto criadas
                para simplificar a gestão de backups do PostgreSQL por meio de
                interfaces web. Embora compartilhem o objetivo de tornar as
                cópias de segurança mais acessíveis, diferem bastante em
                recursos, segurança, suporte a equipes e facilidade de uso.
              </p>

              <h2 id="quick-comparison">Comparação rápida</h2>

              <p>Veja as principais diferenças entre Databasus e PgBackWeb:</p>

              <table>
                <thead>
                  <tr>
                    <th>Recurso</th>
                    <th>Databasus</th>
                    <th>PgBackWeb</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Licença</td>
                    <td data-label="Databasus">Apache 2.0</td>
                    <td data-label="PgBackWeb">AGPL-3.0</td>
                  </tr>
                  <tr>
                    <td>Gestão de backups</td>
                    <td data-label="Databasus">✅ Várias bases de dados</td>
                    <td data-label="PgBackWeb">✅ Várias bases de dados</td>
                  </tr>
                  <tr>
                    <td>Suporte a outras bases de dados</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="PgBackWeb">❌ Apenas PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>Opções de armazenamento</td>
                    <td data-label="Databasus">
                      Local, S3, Cloudflare R2, Azure, NAS,
                      FTP
                    </td>
                    <td data-label="PgBackWeb">
                      Apenas local e compatível com S3
                    </td>
                  </tr>
                  <tr>
                    <td>Notificações</td>
                    <td data-label="Databasus">
                      Slack, Discord, Telegram, Teams, e-mail, webhooks
                    </td>
                    <td data-label="PgBackWeb">Apenas webhooks</td>
                  </tr>
                  <tr>
                    <td>Segurança</td>
                    <td data-label="Databasus">
                      ✅ AES-256-GCM, chaves únicas por backup, acesso somente
                      leitura obrigatório
                    </td>
                    <td data-label="PgBackWeb">✅ Criptografia PGP</td>
                  </tr>
                  <tr>
                    <td>Recursos de equipe</td>
                    <td data-label="Databasus">
                      ✅ Workspaces, controle de acesso por papéis, logs de
                      auditoria
                    </td>
                    <td data-label="PgBackWeb">❌ Não disponível</td>
                  </tr>
                  <tr>
                    <td>Monitoramento de saúde</td>
                    <td data-label="Databasus">✅ Integrado</td>
                    <td data-label="PgBackWeb">❌ Não disponível</td>
                  </tr>
                  <tr>
                    <td>Instalação</td>
                    <td data-label="Databasus">
                      Script de uma linha, Docker ou Helm
                    </td>
                    <td data-label="PgBackWeb">
                      Configuração manual do Docker
                    </td>
                  </tr>
                  <tr>
                    <td>Backups físicos</td>
                    <td data-label="Databasus">✅ Sim</td>
                    <td data-label="PgBackWeb">❌ Não disponível</td>
                  </tr>
                  <tr>
                    <td>Backups incrementais</td>
                    <td data-label="Databasus">
                      ✅ Em nível de bloco (PG 17+)
                    </td>
                    <td data-label="PgBackWeb">❌ Não disponível</td>
                  </tr>
                  <tr>
                    <td>Arquivamento de WAL</td>
                    <td data-label="Databasus">✅ Streaming contínuo</td>
                    <td data-label="PgBackWeb">❌ Não disponível</td>
                  </tr>
                  <tr>
                    <td>Recuperação a um ponto no tempo</td>
                    <td data-label="Databasus">✅ Sim</td>
                    <td data-label="PgBackWeb">❌ Não disponível</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="backup-features">Recursos de backup</h2>

              <p>
                As duas ferramentas suportam backups agendados com horários
                flexíveis:
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong>: suporta agendas horárias, diárias,
                  semanais, mensais ou por cron, com horário preciso (por
                  exemplo, 4 da manhã). Implementa{" "}
                  <strong>compressão equilibrada com zstd (nível 5)</strong>,
                  reduzindo o tamanho dos backups em 4 a 8 vezes com apenas ~20%
                  de tempo adicional de execução. Bem mais eficiente que o gzip.
                </li>
                <li>
                  <strong>PgBackWeb</strong>: suporta agendamento por cron para
                  a execução dos backups. Usa compressão gzip, mais lenta e
                  menos eficiente que o zstd.
                </li>
              </ul>

              <p>
                Além dos backups lógicos, o Databasus também suporta backups
                físicos, incrementais e de WAL. Eles são construídos sobre a
                pilha nativa de backup do PostgreSQL 17 e executam remotamente,
                de modo que nada é instalado no servidor da base de dados e
                redes fechadas podem ser alcançadas por um túnel SSH. Isso lhe
                dá backups incrementais em nível de bloco, streaming contínuo de
                WAL e recuperação a um ponto no tempo para recuperação de
                desastres com perda de dados quase nula, restaurando para
                qualquer segundo entre backups. O PgBackWeb não oferece nada
                disso.
              </p>

              <h2 id="storage-options">Opções de armazenamento</h2>

              <p>
                A flexibilidade de armazenamento é crucial para as estratégias
                de cópia de segurança. Veja como as duas ferramentas se
                comparam:
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong>: suporta uma ampla gama de destinos
                  de armazenamento:
                  <ul>
                    <li>Armazenamento local</li>
                    <li>Amazon S3 e serviços compatíveis com S3</li>
                    <li>Cloudflare R2</li>
                    <li>Azure Blob Storage</li>
                    <li>NAS (armazenamento conectado à rede)</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>: limitado a armazenamento local e
                  compatível com S3.
                </li>
              </ul>

              <p>
                <a
                  href="/pt/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Veja todas as opções de armazenamento do Databasus →
                </a>
              </p>

              <h2 id="security">Segurança</h2>

              <p>
                A segurança é um aspecto crítico da gestão de backups. O
                Databasus implementa segurança de nível corporativo em três
                camadas:
              </p>

              <h3 id="security-databasus">Modelo de segurança do Databasus</h3>

              <ol>
                <li>
                  <strong>Criptografia de dados sensíveis</strong>: todas as
                  senhas, tokens e credenciais são criptografados com
                  AES-256-GCM. A chave de criptografia é guardada separada da
                  base de dados, de modo que, mesmo se a base de dados for
                  comprometida, os dados sensíveis continuam protegidos.
                </li>
                <li>
                  <strong>Criptografia dos backups</strong>: cada arquivo de
                  backup é criptografado com uma chave única derivada da chave
                  mestra, do ID do backup e de um salt aleatório. Mesmo que
                  alguém ganhe acesso ao seu armazenamento em nuvem, não
                  consegue ler os backups sem a sua chave de criptografia.
                </li>
                <li>
                  <strong>Acesso somente leitura à base de dados</strong>: o
                  Databasus impõe acesso somente leitura verificando permissões
                  no nível do papel, da base de dados e da tabela. Exige apenas
                  permissões SELECT e avisa se detectar privilégios de escrita.
                  Isso evita corrupção de dados mesmo que o Databasus seja
                  comprometido.
                </li>
              </ol>

              <h3 id="security-pgbackweb">Modelo de segurança do PgBackWeb</h3>

              <ul>
                <li>
                  <strong>Criptografia PGP</strong>: o PgBackWeb oferece
                  criptografia PGP para os arquivos de backup.
                </li>
                <li>
                  <strong>Sem imposição de somente leitura</strong>: o PgBackWeb
                  não impõe nem verifica o acesso somente leitura à base de
                  dados, ou seja, os backups podem ser criados com usuários que
                  têm permissões de escrita.
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

              <h2 id="notifications">Notificações</h2>

              <p>
                Ficar a par do status dos backups é essencial para uma operação
                confiável:
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong>: fornece notificações em tempo real
                  por vários canais:
                  <ul>
                    <li>Slack</li>
                    <li>Discord</li>
                    <li>Telegram</li>
                    <li>Microsoft Teams</li>
                    <li>E-mail</li>
                    <li>Webhooks</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>: suporta apenas webhooks para
                  notificações. Para receber alertas por Slack, Telegram ou
                  outras plataformas, é preciso montar serviços ou
                  intermediários adicionais.
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

              <h2 id="team-features">Recursos de equipe</h2>

              <p>
                Para organizações e equipes de DevOps, os recursos de
                colaboração são essenciais. É aqui que o Databasus supera com
                folga o PgBackWeb:
              </p>

              <h3 id="team-databasus">Recursos de equipe do Databasus</h3>

              <ul>
                <li>
                  <strong>Workspaces</strong>: agrupe bases de dados,
                  notificadores e armazenamentos por projeto ou equipe. Os
                  usuários só veem os workspaces para os quais foram convidados.
                </li>
                <li>
                  <strong>Controle de acesso baseado em papéis</strong>: níveis
                  de permissão para controlar o que cada membro da equipe pode
                  fazer dentro dos workspaces.
                </li>
                <li>
                  <strong>Logs de auditoria</strong>: acompanhe todas as
                  atividades do sistema e as mudanças feitas pelos usuários.
                  Essencial para conformidade de segurança e responsabilização
                  da equipe.
                </li>
              </ul>

              <h3 id="team-pgbackweb">Recursos de equipe do PgBackWeb</h3>

              <p>
                O PgBackWeb não tem gestão de usuários, workspaces nem logs de
                auditoria integrados. Foi projetado principalmente para cenários
                de usuário único.
              </p>

              <p>
                <a
                  href="/pt/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Saiba mais sobre a gestão de acessos do Databasus →
                </a>
              </p>

              <h2 id="ease-of-use">Facilidade de uso</h2>

              <p>
                <strong>
                  O Databasus foi projetado para ser bem mais fácil de usar
                </strong>{" "}
                que o PgBackWeb, com foco em uma experiência intuitiva e tempo
                mínimo de configuração:
              </p>

              <h3 id="ease-databasus">Experiência de uso do Databasus</h3>

              <ul>
                <li>
                  <strong>Instalação fácil</strong>: use o Docker diretamente ou
                  execute um script de uma linha que instala o Docker (se
                  necessário), configura o Databasus e ativa a inicialização
                  automática. Tempo total: ~2 minutos.
                </li>
                <li>
                  <strong>Interface web intuitiva</strong>: interface refinada
                  que guia a configuração dos backups passo a passo. Nenhum
                  conhecimento de PostgreSQL necessário.
                </li>
                <li>
                  <strong>Temas claro e escuro</strong>: escolha o visual que
                  combina com o seu fluxo de trabalho.
                </li>
                <li>
                  <strong>Adaptado a dispositivos móveis</strong>: verifique
                  seus backups de qualquer lugar, em qualquer dispositivo.
                </li>
                <li>
                  <strong>Monitoramento de saúde integrado</strong>:
                  verificações de saúde configuráveis com gráficos visuais de
                  disponibilidade.
                </li>
                <li>
                  <strong>Restauração com um clique</strong>: baixe e restaure
                  qualquer backup com um único clique.
                </li>
              </ul>

              <h3 id="ease-pgbackweb">Experiência de uso do PgBackWeb</h3>

              <ul>
                <li>
                  <strong>Configuração manual do Docker</strong>: requer
                  configurar variáveis de ambiente e montar uma base de dados
                  PostgreSQL externa para guardar a configuração.
                </li>
                <li>
                  <strong>Interface web básica</strong>: funcional, mas menos
                  refinada que a do Databasus. Tema escuro disponível.
                </li>
                <li>
                  <strong>Sem monitoramento de saúde</strong>: o monitoramento
                  da disponibilidade da base de dados precisa ser montado à
                  parte.
                </li>
              </ul>

              <h2 id="installation">Instalação e implantação</h2>

              <h3 id="install-databasus">Instalando o Databasus</h3>

              <p>
                O Databasus oferece três métodos de instalação, sendo o script
                automatizado o mais rápido:
              </p>

              <ul>
                <li>
                  <strong>Script automatizado (recomendado)</strong>: um comando
                  cURL de uma linha que instala o Docker, configura o Databasus
                  e ativa a inicialização automática.
                </li>
                <li>
                  <strong>Docker run</strong>: um único comando para iniciar o
                  Databasus com PostgreSQL embutido.
                </li>
                <li>
                  <strong>Docker Compose</strong>: para mais controle sobre a
                  implantação.
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

              <h3 id="install-pgbackweb">Instalando o PgBackWeb</h3>

              <p>
                O PgBackWeb requer Docker e configuração manual de variáveis de
                ambiente. Também é preciso montar uma base de dados PostgreSQL
                externa para guardar a configuração do PgBackWeb.
              </p>

              <h2 id="licensing">Licenciamento</h2>

              <p>
                A licença define como você pode usar e modificar o software:
              </p>

              <ul>
                <li>
                  <strong>Databasus (Apache 2.0)</strong>: licença permissiva
                  que autoriza uso comercial, modificação e distribuição sem
                  restrições. Você pode usar o Databasus em projetos
                  proprietários sem qualquer preocupação de licenciamento.
                </li>
                <li>
                  <strong>PgBackWeb (AGPL-3.0)</strong>: licença copyleft que
                  exige que trabalhos derivados ou modificações também sejam de
                  código aberto sob AGPL-3.0. Se você modificar o PgBackWeb e
                  oferecê-lo como serviço, precisa publicar suas modificações.
                </li>
              </ul>

              <h2 id="conclusion">Conclusão</h2>

              <p>
                Databasus e PgBackWeb dão conta do backup de PostgreSQL, mas
                atendem necessidades diferentes:
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Escolha o Databasus se você precisa de:
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Segurança de nível corporativo com proteção em 3 camadas
                  </li>
                  <li>
                    Colaboração em equipe com workspaces e logs de auditoria
                  </li>
                  <li>
                    Vários destinos de armazenamento (FTP, SFTP, Azure etc.)
                  </li>
                  <li>
                    Notificações integradas para Slack, Teams, Telegram etc.
                  </li>
                  <li>Instalação rápida com script de uma linha ou Docker</li>
                  <li>
                    Interface moderna e intuitiva com curva de aprendizado
                    mínima
                  </li>
                  <li>Licença permissiva Apache 2.0 para uso comercial</li>
                  <li>
                    Backups físicos, backups incrementais, arquivamento de WAL e
                    PITR para recuperação de desastres
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>Escolha o PgBackWeb se:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Você precisa de uma solução de backup simples para um único
                    usuário
                  </li>
                  <li>Armazenamento local ou S3 é suficiente</li>
                  <li>Notificações apenas por webhook são suficientes</li>
                  <li>A licença AGPL-3.0 é aceitável para o seu caso de uso</li>
                </ul>
              </div>

              <p>
                Para a maioria dos usuários, especialmente equipes e
                organizações que exigem segurança sólida, várias opções de
                armazenamento e canais completos de notificação,{" "}
                <strong>o Databasus é a escolha recomendada</strong>.
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
