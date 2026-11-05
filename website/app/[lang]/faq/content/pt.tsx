import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "FAQ - Perguntas frequentes | Databasus",
  description:
    "Perguntas frequentes sobre o Databasus, ferramenta de backup PostgreSQL com suporte a MySQL, MariaDB e MongoDB. Saiba como fazer backup de bases de dados em localhost, entenda formatos de backup, métodos de compressão e mais.",
  keywords: [
    "FAQ do Databasus",
    "perguntas sobre backup PostgreSQL",
    "backup de base de dados localhost",
    "formatos de backup",
    "compressão do pg_dump",
    "compressão zstd",
    "ajuda com backup PostgreSQL",
    "guia de backup de bases de dados",
  ],
  openGraph: {
    title: "FAQ - Perguntas frequentes | Databasus",
    description:
      "Perguntas frequentes sobre o Databasus, ferramenta de backup PostgreSQL com suporte a MySQL, MariaDB e MongoDB. Saiba como fazer backup de bases de dados em localhost, entenda formatos de backup, métodos de compressão e mais.",
    type: "article",
    url: getLocalizedUrl("pt", "faq"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "FAQ - Perguntas frequentes | Databasus",
    description:
      "Perguntas frequentes sobre o Databasus, ferramenta de backup PostgreSQL com suporte a MySQL, MariaDB e MongoDB. Saiba como fazer backup de bases de dados em localhost, entenda formatos de backup, métodos de compressão e mais.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "faq"),
    languages: getLanguageAlternates("faq"),
  },
  robots: "index, follow",
};

export default function FAQPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "Por que o Databasus não usa o formato de dump SQL puro para backups lógicos de PostgreSQL?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Para backups lógicos, o Databasus usa o formato custom do pg_dump com compressão zstd porque, depois de testes extensivos, é o que oferece a melhor velocidade de backup e restauração. O formato custom com compressão zstd de nível 5 dá o equilíbrio ideal entre velocidade de criação do backup, velocidade de restauração e tamanho do dump.",
                },
              },
              {
                "@type": "Question",
                name: "Onde o Databasus é instalado?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus é instalado em /opt/databasus/",
                },
              },
              {
                "@type": "Question",
                name: "Como funcionam os backups físicos e PITR (Point-in-Time Recovery)?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "O Databasus executa backups físicos remotamente a partir do seu próprio host, conectando-se ao seu PostgreSQL pelo protocolo padrão de replicação, então nada precisa ser instalado no servidor da base de dados. Bases em redes fechadas podem ser alcançadas por um túnel SSH. Os backups físicos usam a pilha nativa do PostgreSQL 17: backups completos via pg_basebackup, incrementais em nível de bloco via pg_basebackup --incremental com base nos resumos de WAL do servidor (summarize_wal = on) e streaming contínuo de WAL via pg_receivewal. Backups físicos exigem PostgreSQL 17 ou mais recente; em versões anteriores usam-se backups lógicos com pg_dump. Para restaurar para um ponto no tempo, o pg_combinebackup reconstrói um diretório de dados utilizável a partir do backup completo e da sua cadeia de incrementais, e o PostgreSQL então reproduz o WAL até o momento alvo escolhido, recuperando para qualquer segundo entre backups. A interface do Databasus dá instruções passo a passo para restaurar num host ou numa base em Docker, seja por um script pronto que torna a restauração um único comando, seja baixando os backups e reconstruindo a cadeia de partes completas, incrementais e WAL por conta própria. Incrementais e WAL são opcionais: você pode fazer apenas um backup completo, e o WAL não é obrigatório. Usamos os backups nativos do PostgreSQL 17 porque reutilizam o mecanismo de backup já comprovado do próprio PostgreSQL em vez de reinventá-lo, funcionam com bases remotas incluindo serviços gerenciados como RDS e Cloud SQL e dão perda de dados quase nula.",
                },
              },
              {
                "@type": "Question",
                name: "Por que o Databasus abandonou os backups baseados em agente?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Uma versão anterior do Databasus trazia um agente de backup: um binário que rodava no host da base de dados para transmitir WAL e criar backups físicos localmente. Essa primeira implementação se revelou um erro e foi removida. Era uma implementação ingênua que só copiava WAL por cima de backups completos, o que levava a um RTO longo. Era preciso configurar tanto o Databasus quanto um agente separado, quando fazer tudo remotamente de um só lugar é bem mais simples. Como o agente vivia fora do sistema principal, era difícil cobrir todos os casos de teste. Na verdade, só existe um problema que um agente resolve: alcançar uma base de dados inacessível de fora, e para 99% das pessoas isso já é resolvido executando o Databasus dentro da rede privada ou conectando por SSH, então o agente reinventava a roda e tornava um problema simples muito mais complicado do que o necessário. Ele também não podia rodar em bases gerenciadas como RDS e Cloud SQL, que proíbem instalações no host mas já expõem o protocolo de replicação, então um caminho remoto era necessário de qualquer forma. Além disso, trazia muitos casos-limite em torno de conexões quebradas, gestão de atualizações do agente e coleta de logs de um processo separado, e quanto menos partes móveis um sistema tem, mais confiável ele é no dia a dia. Os backups físicos agora rodam remotamente a partir do host do Databasus. Os backups existentes ficam seguros: ao atualizar de uma versão que ainda tem backups via agente, o Databasus não o faz em silêncio, mas avisa sobre a mudança e deixa você escolher entre ficar na versão suportada 3.42.0 ou remover os backups antigos do agente antes de atualizar. A implementação baseada em agente permanece disponível até a versão 3.42.0 e continuará funcionando por muito tempo.",
                },
              },
              {
                "@type": "Question",
                name: "Como fazer backup do próprio Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Para fazer backup do Databasus, vá a /opt/databasus (ou à pasta onde o instalou) e depois ao diretório databasus-data. É preciso guardar o secret.key (chave de criptografia das credenciais) e a pasta /pgdata (base de dados interna com as configurações e os metadados dos backups). Há dois cenários de recuperação: 1) Você pode recuperar os backups das bases usando apenas o secret.key, sem a interface do Databasus (veja o guia de recuperação manual); 2) Para restaurar a interface do Databasus com todas as configurações e o histórico, precisa tanto do secret.key quanto da pasta /pgdata. Para restaurar, recrie essa estrutura de pastas em outro servidor.",
                },
              },
              {
                "@type": "Question",
                name: "Como o Databasus é apoiado pelos programas open source da Anthropic e da OpenAI?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Em março de 2026, o Databasus foi aceito tanto no Claude for Open Source da Anthropic quanto no Codex for Open Source da OpenAI. Ter o apoio desses programas é um sinal de confiabilidade — o projeto foi avaliado de forma independente e reconhecido por líderes do setor como infraestrutura open source crítica que vale a pena apoiar.",
                },
              },
            ],
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
              <h1 id="faq">Perguntas frequentes</h1>

              <p className="text-lg text-gray-400">
                Encontre respostas às perguntas mais comuns sobre o Databasus,
                incluindo instalação, configuração e estratégias de backup.
              </p>

              <h2 id="why-no-raw-sql-dump">
                Por que o Databasus não usa o formato de dump SQL puro para
                backups lógicos de PostgreSQL?
              </h2>

              <p>
                Para backups lógicos, o Databasus usa o{" "}
                <strong>formato custom</strong> do <code>pg_dump</code> com{" "}
                <strong>compressão zstd de nível 5</strong> em vez do formato
                SQL simples, porque é o que dá o equilíbrio mais eficiente
                entre:
              </p>

              <ul>
                <li>Velocidade de criação do backup</li>
                <li>Velocidade de restauração</li>
                <li>
                  Compressão do tamanho (até 20x menor do que o formato SQL
                  simples)
                </li>
              </ul>

              <p>
                Essa decisão foi tomada depois de testes e benchmarks extensivos
                de diferentes formatos de backup e métodos de compressão do
                PostgreSQL. Você pode ler mais sobre os testes aqui:{" "}
                <a
                  href="https://dev.to/rostislav_dugin/postgresql-backups-comparing-pgdump-speed-in-different-formats-and-with-different-compression-4pmd"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  PostgreSQL backups: comparing pg_dump speed in different
                  formats and with different compression
                </a>
                .
              </p>

              <p>
                O Databasus não incluirá o formato de dump SQL puro, porque:
              </p>

              <ul>
                <li>variedade extra prejudica a experiência de uso;</li>
                <li>torna o código mais difícil de manter;</li>
                <li>o formato de dump atual serve para 99% dos casos</li>
              </ul>

              <h2 id="installation-directory">
                Onde o Databasus é instalado quando instalado via script .sh?
              </h2>

              <p>
                O Databasus é instalado no diretório{" "}
                <code>/opt/databasus/</code>.
              </p>

              <h2 id="pitr">
                Como funcionam os backups físicos e PITR (Point-in-Time
                Recovery)?
              </h2>

              <p>
                O Databasus executa backups físicos{" "}
                <strong>remotamente a partir do seu próprio host</strong>,
                conectando-se ao seu PostgreSQL pelo{" "}
                <strong>protocolo padrão de replicação</strong>, então nada
                precisa ser instalado no servidor da base de dados. Se a base
                vive numa rede fechada, o Databasus pode alcançá-la por um túnel
                SSH até um host interno ou um bastion, então a base de dados
                nunca precisa ficar exposta publicamente.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>Por que isso é possível agora:</strong> durante
                    anos, ferramentas como pgBackRest e WAL-G tiveram de
                    construir os seus próprios motores de backup incremental em
                    nível de bloco, porque o PostgreSQL não tinha um nativo.
                    Isso mudou com o PostgreSQL 17, onde a funcionalidade foi
                    desenvolvida por <strong>Robert Haas</strong> com a ajuda de{" "}
                    <strong>David Steele</strong>, o autor do pgBackRest. O
                    PostgreSQL agora traz backups incrementais nativos em nível
                    de bloco do lado do servidor (
                    <code>pg_basebackup --incremental</code> e{" "}
                    <code>summarize_wal</code>), então o Databasus se apoia
                    nisso em vez de reinventar um motor próprio.
                  </p>
                </div>
              </div>

              <p>
                <strong>Como funcionam os backups:</strong>
              </p>

              <ul>
                <li>
                  Os backups completos são criados com{" "}
                  <code>pg_basebackup</code>, transmitidos diretamente para o
                  Databasus
                </li>
                <li>
                  Os incrementais em nível de bloco usam{" "}
                  <code>pg_basebackup --incremental</code>, em que os resumos de
                  WAL do lado do servidor do PostgreSQL 17 (
                  <code>summarize_wal = on</code>) acompanham as mudanças para
                  transferir apenas os blocos alterados
                </li>
                <li>
                  O WAL é transmitido continuamente via{" "}
                  <code>pg_receivewal</code> para manter a cadeia de recuperação
                  completa entre backups
                </li>
                <li>
                  Backups físicos exigem{" "}
                  <strong>PostgreSQL 17 ou mais recente</strong>; em versões
                  anteriores usam-se backups lógicos com <code>pg_dump</code>
                </li>
              </ul>

              <p>
                <strong>Como funciona a restauração:</strong>
              </p>

              <ul>
                <li>
                  O <code>pg_combinebackup</code> reconstrói um diretório de
                  dados utilizável a partir do backup completo e da sua cadeia
                  de incrementais
                </li>
                <li>
                  O PostgreSQL então reproduz o WAL até o momento alvo que você
                  escolher, recuperando para qualquer segundo entre backups
                </li>
                <li>
                  Ao iniciar o PostgreSQL, ele conclui a recuperação, se promove
                  a primário e retoma a operação normal
                </li>
              </ul>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>Você não precisa fazer isso à mão.</strong> A
                    interface do Databasus dá instruções passo a passo para
                    restaurar num host ou numa base em Docker, seja por um
                    script pronto, seja baixando os backups manualmente.
                    Preparamos o script para que uma restauração seja um único
                    comando, mas você também pode reconstruir a cadeia de partes
                    completas, incrementais e WAL por conta própria, se
                    preferir. Incrementais e WAL também são opcionais: você pode
                    fazer apenas um backup completo, sem incrementais, e o WAL
                    não é obrigatório.
                  </p>
                </div>
              </div>

              <p>
                <strong>Por que usamos os backups nativos do PG 17:</strong>
              </p>

              <ul>
                <li>
                  Reutilizam o mecanismo de backup do próprio PostgreSQL em vez
                  de reinventá-lo, então você aproveita um mecanismo comprovado,
                  com milhares de testes e casos-limite por trás
                </li>
                <li>
                  Funcionam com bases de dados remotas, incluindo serviços
                  gerenciados como Amazon RDS e Google Cloud SQL, que expõem o
                  protocolo de replicação mas proíbem instalar software no host
                </li>
                <li>
                  Dão perda de dados quase nula, permitindo restaurar para
                  qualquer segundo entre backups
                </li>
              </ul>

              <h2 id="why-no-agent">
                Por que o Databasus abandonou os backups baseados em agente?
              </h2>

              <p>
                Uma versão anterior do Databasus trazia um{" "}
                <strong>agente</strong> de backup: um binário que rodava no host
                da base de dados para transmitir WAL e criar backups físicos
                localmente. Essa primeira implementação se revelou um erro, e
                nós a removemos. Os backups físicos agora rodam remotamente a
                partir do host do Databasus, como descrito acima.
              </p>

              <p>
                <strong>Por que o agente era a abordagem errada:</strong>
              </p>

              <ul>
                <li>
                  Era uma implementação ingênua que só copiava WAL por cima de
                  backups completos, o que levava a um RTO longo
                </li>
                <li>
                  Era preciso configurar tanto o Databasus quanto um agente
                  separado, quando fazer tudo remotamente de um só lugar é bem
                  mais simples
                </li>
                <li>
                  Como o agente vivia fora do sistema principal, era difícil
                  cobrir todos os casos de teste
                </li>
                <li>
                  Na verdade, só existe um problema que um agente resolve:
                  alcançar uma base de dados inacessível de fora. Para 99% das
                  pessoas isso já é resolvido executando o Databasus dentro da
                  rede privada ou conectando por SSH, então o agente reinventava
                  a roda e tornava um problema simples muito mais complicado do
                  que precisava ser
                </li>
                <li>
                  Ele não podia rodar em bases gerenciadas como RDS e Cloud SQL,
                  que proíbem instalações no host mas já expõem o protocolo de
                  replicação, então um caminho remoto era necessário de qualquer
                  forma
                </li>
                <li>
                  Também trazia muitos casos-limite. Conexões quebradas, gestão
                  das atualizações do agente e coleta de logs de um processo
                  separado eram todos dolorosos, e quanto menos partes móveis um
                  sistema tem, mais confiável ele é no dia a dia
                </li>
              </ul>

              <p>
                <strong>
                  Garantimos que os backups existentes ficam seguros.
                </strong>{" "}
                Se você atualizar de uma versão que ainda tem backups via
                agente, o Databasus não o fará em silêncio: ele avisa sobre a
                mudança e deixa você escolher entre ficar na{" "}
                <strong>versão suportada 3.42.0</strong> ou remover os backups
                antigos do agente antes de atualizar. A implementação baseada em
                agente permanece disponível até a versão 3.42.0 e continuará
                funcionando por muito tempo, então nada quebra.
              </p>

              <h2 id="backup-databasus">
                Como fazer backup do próprio Databasus?
              </h2>

              <p>
                Se você quiser fazer uma cópia de segurança da sua instância do
                Databasus (incluindo todas as configurações, bases de dados e
                credenciais), siga estes passos:
              </p>

              <ol>
                <li>
                  Vá a <code>/opt/databasus</code> (ou à pasta onde instalou o
                  Databasus)
                </li>
                <li>
                  Entre no diretório <code>databasus-data</code>
                </li>
              </ol>

              <p>
                <strong>Você precisa guardar:</strong>
              </p>

              <ul>
                <li>
                  <code>secret.key</code> — chave de criptografia das suas
                  credenciais
                </li>
                <li>
                  <code>/pgdata</code> — base de dados PostgreSQL interna do
                  Databasus, que contém todas as suas configurações e os
                  metadados dos backups
                </li>
              </ul>

              <p>
                Se você usa armazenamento local para os backups, também pode
                guardar a pasta <code>backups</code>.
              </p>

              <p>
                <strong>Importante:</strong> há dois cenários diferentes de
                recuperação:
              </p>

              <ul>
                <li>
                  <strong>
                    Recuperar backups sem a interface do Databasus:
                  </strong>{" "}
                  você pode recuperar os backups das suas bases usando apenas o{" "}
                  <code>secret.key</code>, sem precisar do Databasus nem dos
                  seus dados internos. Veja o{" "}
                  <a href="/pt/how-to-recover-without-databasus">
                    guia de recuperação manual
                  </a>{" "}
                  para instruções detalhadas.
                </li>
                <li>
                  <strong>
                    Restaurar a interface do Databasus e todas as configurações:
                  </strong>{" "}
                  se você quiser restaurar a interface do Databasus com todas as
                  suas configurações, backups agendados e histórico de backups,
                  precisa guardar tanto o <code>secret.key</code> quanto a pasta{" "}
                  <code>/pgdata</code> (que contém os metadados de criptografia
                  e todas as configurações do Databasus).
                </li>
              </ul>

              <p>
                <strong>Para restaurar o Databasus em outro servidor:</strong>{" "}
                basta recriar a estrutura da pasta <code>databasus-data</code>{" "}
                com os dados guardados e iniciar o Databasus.
              </p>

              <h2 id="oss-programs">
                Como o Databasus é apoiado pelos programas open source da
                Anthropic e da OpenAI?
              </h2>

              <p>
                Em março de 2026, o Databasus foi aceito tanto no{" "}
                <strong>
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Claude for Open Source
                  </a>
                </strong>{" "}
                da Anthropic quanto no{" "}
                <strong>
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Codex for Open Source
                  </a>
                </strong>{" "}
                da OpenAI. É muito valioso para nós que o projeto tenha sido
                reconhecido como software open source importante para o setor
                por duas das principais empresas de IA do mundo — especialmente
                considerando os requisitos de elegibilidade exigentes dos dois
                programas.
              </p>

              <p>
                O que isso significa para quem usa o Databasus? É mais uma
                confirmação de confiabilidade: o projeto foi avaliado de forma
                independente e reconhecido por líderes do setor como
                infraestrutura crítica que vale a pena apoiar. Então temos
                qualidade de código ainda mais alta, revisões de segurança mais
                rápidas e desenvolvimento ativo contínuo graças ao acesso às IAs
                mais recentes sem limites.
              </p>

              <img
                src="/images/faq/anthropic-email.png"
                alt="Databasus aceito no programa Claude for Open Source da Anthropic"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <img
                src="/images/faq/openai-email.png"
                alt="Databasus aceito no programa Codex for Open Source da OpenAI"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
