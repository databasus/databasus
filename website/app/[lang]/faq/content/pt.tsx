import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

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
  const startupRecoveryCommand = `docker run -d \\
  --name databasus \\
  -p 4005:4005 \\
  -v ./databasus-data:/databasus-data \\
  -e RECOVERY_DUMP_PATH=/databasus-data/recovery/backup.dbbackup \\
  -e RECOVERY_MASTER_KEY_PATH=/databasus-data/recovery/master.key \\
  --restart unless-stopped \\
  databasus/databasus:<version>`;

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
                  text: "O Databasus executa backups físicos remotamente a partir do seu próprio host, conectando-se ao seu PostgreSQL pelo protocolo padrão de replicação, então nada precisa ser instalado no servidor da base de dados. Bases em redes fechadas podem ser alcançadas por um túnel SSH. Os backups físicos usam a pilha nativa do PostgreSQL 17: backups completos via pg_basebackup, incrementais em nível de bloco via pg_basebackup --incremental com base nos resumos de WAL do servidor (summarize_wal = on) e streaming contínuo de WAL via pg_receivewal. Backups físicos exigem PostgreSQL 17 ou mais recente; em versões anteriores usam-se backups lógicos com pg_dump. Para restaurar para um ponto no tempo, o pg_combinebackup reconstrói um diretório de dados utilizável a partir do backup completo e da sua cadeia de incrementais, e o PostgreSQL então reproduz o WAL até o momento alvo escolhido, recuperando para qualquer segundo entre backups. A interface do Databasus dá instruções passo a passo para restaurar num host ou numa base em Docker, seja por um script pronto que torna a restauração um único comando, seja baixando os backups e reconstruindo a cadeia de partes completas, incrementais e WAL por conta própria. Incrementais e WAL são opcionais: você pode fazer apenas um backup completo, e o WAL não é obrigatório. Usamos os backups nativos do PostgreSQL 17 porque reutilizam o mecanismo de backup já comprovado do próprio PostgreSQL em vez de reinventá-lo e dão perda de dados quase nula.",
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
                  text: "O Databasus foi feito para continuar recuperável mesmo que o próprio servidor seja perdido ou corrompido. Ele pode fazer backup da própria base de dados interna: contas, configurações, as credenciais criptografadas das bases de dados, dos armazenamentos e dos canais de notificação, os agendamentos e o catálogo de backups. Ative isso em Databasus settings → Databasus backups escolhendo um ou mais armazenamentos existentes; por padrão, o backup do Databasus roda a cada hora e é mantido por 30 dias. O arquivo é criptografado com a chave mestra e não contém nem a chave nem os arquivos de backup das suas bases de dados, então guarde a chave mestra à parte: o Databasus pede que você a baixe, e ela também fica em . /databasus-data/secret. key no host. Para recuperar, você precisa do arquivo de backup, da chave mestra dele e exatamente da mesma versão do Databasus, incluindo o patch. Você pode colocar os dois arquivos no diretório de dados montado e iniciar o Databasus com RECOVERY_DUMP_PATH e RECOVERY_MASTER_KEY_PATH em um servidor sem contas e sem configurações alteradas, ou enviar o arquivo e colar a chave em Databasus settings → Databasus backups, o que substitui a instalação atual. Pare a instalação antiga antes de recuperar. A instalação recuperada inicia na hora os backups, as notificações e as gravações nos armazenamentos, então duas instalações rodando ao mesmo tempo gravariam nos mesmos armazenamentos e bancos de dados.",
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
              {
                "@type": "Question",
                name: "Como a IA é usada no desenvolvimento do Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "A IA nos ajuda a desenvolver o Databasus, do planejamento e da implementação aos testes e à documentação. As pessoas definem o rumo e tomam as decisões finais. Uma pessoa lê cada linha de código antes de integrar uma mudança. Usamos desenvolvimento baseado em especificações (SDD) com OpenSpec para definir o comportamento antes da implementação. Revisão independente em cada etapa Os hooks automáticos iniciam sessões de revisão separadas com acesso somente para leitura ao repositório. Cada agente inspeciona o código pertinente e apresenta seus próprios apontamentos. O agente que implementa não pode aprovar o próprio trabalho. A revisão começa antes da implementação. Agentes de planejamento separados procuram soluções mais simples, verificam os nomes propostos e avaliam o projeto técnico. Também conferem se os requisitos e as tarefas de verificação estão de acordo. Revisor de código: Procura bugs, casos extremos esquecidos, falhas no tratamento de erros, condições de corrida, vazamentos de recursos e problemas de segurança. Revisor de requisitos: Localiza a implementação e os testes de cada requisito. Verifica se os testes demonstram o comportamento prometido. Revisor de arquitetura: Verifica os limites dos módulos, a direção das dependências, a responsabilidade pelos dados, as migrações e as regras de API. Revisor da estrutura do código: Verifica nomes, responsabilidades de arquivos e funções, comentários e legibilidade. Revisor de testes: Avalia se os testes detectariam um comportamento ausente ou uma regressão. Também verifica esperas por condições, limpeza e evidências de verificação. Revisor de documentação e tarefas: Confere se a documentação corresponde ao código, se as tarefas estão completas e se as verificações declaradas têm evidências. As revisões completas de implementação rodam em paralelo. Cada especificação alterada tem seu próprio revisor de requisitos. Arquitetura, testes e documentação têm revisores separados. Uma mudança em duas especificações recebe, portanto, cinco agentes de revisão completa. Apontamentos bloqueantes impedem a aceitação do trabalho. O agente que implementa corrige os problemas ou explica por que um apontamento está incorreto. Um revisor posterior confere essa resolução e as alterações seguintes. Uma revisão que não pôde rodar não é considerada aprovada. Verificações automáticas e restores reais A revisão independente é complementada por verificações automáticas rigorosas. golangci-lint e nosso analisador de arquitetura verificam as dependências do backend, os limites dos módulos e as regras de migração. As verificações de API validam os contratos OpenAPI e o código gerado. O frontend verifica TypeScript, React, importações entre camadas, estilos e localização. Ruff e ty verificam formatação, regras de código e tipos do Python. ShellCheck verifica scripts de shell, e actionlint verifica os fluxos do GitHub Actions. Os hooks de pré-commit verificam os arquivos afetados do backend, frontend e Python sem reescrevê-los. A análise estática também roda no CI. Os testes verificam comportamentos reais. O backend é testado pela API da aplicação, e os mecanismos de banco de dados são verificados com bancos reais em contêineres. Os cenários de ponta a ponta criam backups, fazem o restore e verificam os dados restaurados. Outros testes cobrem atualizações, telas do frontend e fluxos no navegador com comparações visuais. As verificações são escolhidas conforme a mudança. As suítes obrigatórias de publicação rodam em amd64 e arm64. O agente que implementa executa os analisadores e os testes pertinentes antes da revisão. Os revisores inspecionam as evidências sem executar esses comandos novamente. Antes da publicação, CodeQL e Semgrep analisam o código, Gitleaks procura segredos e as dependências são verificadas para detectar vulnerabilidades. Trivy analisa as imagens de contêiner e os Dockerfiles. As verificações obrigatórias de CI precisam passar antes da publicação.",
                },
              },
              {
                "@type": "Question",
                name: "O Databasus tem uma comunidade?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Sim. A comunidade do Databasus conversa nos chats do Telegram (https://t.me/databasus_community) e Discord (https://databasus.com/discord). Lá você pode tirar dúvidas, contar como usa o Databasus, discutir funcionalidades e ficar sabendo das novas versões antes de todo mundo. Escolha o aplicativo que você já usa.",
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

              <h2 id="ai-usage">Como a IA é usada no desenvolvimento do Databasus?</h2>

              <p>
                A IA nos ajuda a desenvolver o Databasus, do planejamento e da implementação aos testes e à
                documentação. As pessoas definem o rumo e tomam as decisões finais.{" "}
                <strong>Uma pessoa lê cada linha de código antes de integrar uma mudança.</strong>
              </p>

              <p>
                Usamos desenvolvimento baseado em especificações (SDD) com{" "}
                <a href="https://github.com/databasus/databasus/tree/main/openspec">OpenSpec</a> para definir o
                comportamento antes da implementação.
              </p>

              <h3 id="ai-review-hooks">Revisão independente em cada etapa</h3>

              <p>
                Os hooks automáticos iniciam sessões de revisão separadas com acesso somente para leitura ao
                repositório. Cada agente inspeciona o código pertinente e apresenta seus próprios apontamentos.
                O agente que implementa não pode aprovar o próprio trabalho.
              </p>

              <p>
                A revisão começa antes da implementação. Agentes de planejamento separados procuram soluções
                mais simples, verificam os nomes propostos e avaliam o projeto técnico. Também conferem se os
                requisitos e as tarefas de verificação estão de acordo.
              </p>

              <ul>
                <li>
                  <strong>Revisor de código:</strong> Procura bugs, casos extremos esquecidos, falhas no
                  tratamento de erros, condições de corrida, vazamentos de recursos e problemas de segurança.
                </li>
                <li>
                  <strong>Revisor de requisitos:</strong> Localiza a implementação e os testes de cada
                  requisito. Verifica se os testes demonstram o comportamento prometido.
                </li>
                <li>
                  <strong>Revisor de arquitetura:</strong> Verifica os limites dos módulos, a direção das
                  dependências, a responsabilidade pelos dados, as migrações e as regras de API.
                </li>
                <li>
                  <strong>Revisor da estrutura do código:</strong> Verifica nomes, responsabilidades de arquivos
                  e funções, comentários e legibilidade.
                </li>
                <li>
                  <strong>Revisor de testes:</strong> Avalia se os testes detectariam um comportamento ausente
                  ou uma regressão. Também verifica esperas por condições, limpeza e evidências de verificação.
                </li>
                <li>
                  <strong>Revisor de documentação e tarefas:</strong> Confere se a documentação corresponde ao
                  código, se as tarefas estão completas e se as verificações declaradas têm evidências.
                </li>
              </ul>

              <p>
                As revisões completas de implementação rodam em paralelo. Cada especificação alterada tem seu
                próprio revisor de requisitos. Arquitetura, testes e documentação têm revisores separados. Uma
                mudança em duas especificações recebe, portanto, cinco agentes de revisão completa.
              </p>

              <p>
                <strong>Apontamentos bloqueantes impedem a aceitação do trabalho.</strong> O agente que
                implementa corrige os problemas ou explica por que um apontamento está incorreto. Um revisor
                posterior confere essa resolução e as alterações seguintes. Uma revisão que não pôde rodar não é
                considerada aprovada.
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/.agent-hooks/review_gate.py">
                  Implementação dos hooks de revisão
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/tree/main/.agent-hooks">
                  Instruções dos revisores
                </a>
              </p>

              <h3 id="ai-linting">Verificações automáticas e restores reais</h3>

              <p>
                A revisão independente é complementada por verificações automáticas rigorosas. golangci-lint e
                nosso analisador de arquitetura verificam as dependências do backend, os limites dos módulos e
                as regras de migração. As verificações de API validam os contratos OpenAPI e o código gerado. O
                frontend verifica TypeScript, React, importações entre camadas, estilos e localização.
              </p>

              <p>
                Ruff e ty verificam formatação, regras de código e tipos do Python. ShellCheck verifica scripts
                de shell, e actionlint verifica os fluxos do GitHub Actions. Os hooks de pré-commit verificam os
                arquivos afetados do backend, frontend e Python sem reescrevê-los. A análise estática também
                roda no CI.
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/mise-tasks/backend/lint">
                  Análise do backend
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/blob/main/frontend/lint/check.sh">
                  Análise do frontend
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/blob/main/scripts/python_quality.py">
                  Verificações de Python
                </a>
              </p>

              <p>
                Os testes verificam comportamentos reais. O backend é testado pela API da aplicação, e os
                mecanismos de banco de dados são verificados com bancos reais em contêineres. Os cenários de
                ponta a ponta criam backups, fazem o restore e verificam os dados restaurados. Outros testes
                cobrem atualizações, telas do frontend e fluxos no navegador com comparações visuais.
              </p>

              <p>
                As verificações são escolhidas conforme a mudança. As suítes obrigatórias de publicação rodam em
                amd64 e arm64. O agente que implementa executa os analisadores e os testes pertinentes antes da
                revisão. Os revisores inspecionam as evidências sem executar esses comandos novamente.
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/docs/testing.md">
                  Estratégia de testes e comandos
                </a>
              </p>

              <p>
                Antes da publicação, CodeQL e Semgrep analisam o código, Gitleaks procura segredos e as
                dependências são verificadas para detectar vulnerabilidades. Trivy analisa as imagens de
                contêiner e os Dockerfiles.
              </p>

              <p>
                <strong>As verificações obrigatórias de CI precisam passar antes da publicação.</strong>{" "}
                <a href={`${getLocalizedHref("pt", "security")}#security-and-reliability-engineering`}>
                  Segurança e confiabilidade
                </a>
                .
              </p>

              <h2 id="backup-databasus">
                Como fazer backup do próprio Databasus?
              </h2>

              <p>
                O Databasus foi feito para continuar recuperável mesmo que o próprio servidor seja perdido ou corrompido.
                Ele pode fazer backup da própria base de dados interna:
                contas, configurações, as credenciais criptografadas das suas
                bases de dados, dos armazenamentos e dos canais de notificação,
                os agendamentos e o catálogo de backups. Ative isso em{" "}
                <strong>Databasus settings → Databasus backups</strong>{" "}
                escolhendo um ou mais armazenamentos existentes. Por padrão, o
                backup do Databasus roda a cada hora e é mantido por 30 dias.
                Cada arquivo é criptografado com a chave mestra.
              </p>

              <p>
                O backup do Databasus não contém a chave mestra nem os arquivos
                de backup das suas bases de dados, que ficam nos próprios
                armazenamentos. <strong>Guarde a chave mestra à
                parte.</strong>{" "}
                O Databasus pede que você a baixe no primeiro acesso, e a chave
                também é o arquivo <code>./databasus-data/secret.key</code> no
                host.
              </p>

              <ul>
                <li>
                  <strong>Escolha um armazenamento remoto.</strong> Um
                  armazenamento local no mesmo servidor se perde junto com esse
                  servidor.
                </li>
                <li>
                  <strong>Alertas.</strong> O Databasus mostra um aviso quando
                  um backup do Databasus falha, é salvo só em parte dos
                  armazenamentos, para ou está atrasado, ou seja, nenhuma cópia
                  deu certo 15 minutos depois de uma execução agendada. Os
                  mesmos alertas são enviados por e-mail pelo{" "}
                  <a href={`${getLocalizedHref("pt", "advanced-config")}#email-smtp`}>
                  servidor de e-mail da instância
                  </a>{" "}
                  quando ele está configurado.
                </li>
              </ul>

              <h3 id="recover-databasus">
                Como recuperar o Databasus a partir de um backup do Databasus?
              </h3>

              <p>
                Você precisa de duas coisas: um arquivo de backup do Databasus
                (<code>.dbbackup</code>), baixado do histórico ou pego direto do
                armazenamento, e a chave mestra da instalação que o criou.{" "}
                <strong>
                A recuperação só funciona exatamente na mesma versão do
                Databasus, incluindo o patch.
                </strong>{" "}
                Em qualquer outra versão, o Databasus recusa e informa qual
                versão iniciar: recupere nessa versão e depois atualize
                normalmente.
              </p>

              <p>
                <strong>Pare a instalação antiga antes de recuperar.</strong> A
                instalação recuperada inicia na hora os backups, as notificações
                e as gravações nos armazenamentos, então duas instalações
                rodando ao mesmo tempo gravariam nos mesmos armazenamentos e
                bancos de dados.
              </p>

              <p>
                <strong>Opção 1: arquivos na inicialização.</strong> Use em um
                servidor novo. Coloque os dois arquivos no diretório de dados
                montado, por exemplo <code>./databasus-data/recovery/</code>, e
                inicie a imagem da versão que criou o backup com duas
                configurações a mais. Troque <code>&lt;version&gt;</code> por
                essa versão:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startupRecoveryCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={startupRecoveryCommand} />
                </div>
              </div>

              <ul>
                <li>
                  Se você já iniciou o Databasus normalmente neste servidor, mas
                  não criou nenhuma conta nem mudou nenhuma configuração, não
                  precisa apagar os dados dele. Pare-o, adicione os dois
                  arquivos e as duas configurações e inicie-o de novo.
                </li>
                <li>
                  A recuperação na inicialização nunca substitui uma instalação
                  que já tem pelo menos uma conta, nem uma que tem outros dados
                  de usuários ou configurações alteradas. Nesse caso, o
                  Databasus não inicia. O log diz que a recuperação não rodou,
                  por quê, que a instalação atual ficou como estava e o que
                  fazer em seguida. Para iniciar a instalação atual, remova as
                  duas configurações ou use o envio pela interface.
                </li>
                <li>
                  Depois de uma recuperação bem-sucedida, os arquivos e as
                  configurações podem ficar onde estão. O Databasus guarda o
                  SHA-256 do arquivo aplicado, então reinicializações e
                  atualizações futuras o ignoram e avisam isso no log, mesmo que
                  o arquivo tenha sido renomeado. Um arquivo diferente no mesmo
                  caminho não é ignorado.
                </li>
              </ul>

              <p>
                <strong>Opção 2: envio pela interface.</strong> Use em uma
                instalação que já está rodando. Em uma instalação nova, primeiro
                faça o cadastro. Abra{" "}
                <strong>Databasus settings → Databasus backups</strong>, escolha
                o arquivo de backup e cole ou digite a chave mestra da
                instalação que o criou.
                Tudo na instalação atual é substituído: contas, configurações
                e o catálogo de backups. Depois, entre com uma conta que está
                no backup.
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

              <h2 id="community">O Databasus tem uma comunidade?</h2>

              <p>
                Sim. A comunidade do Databasus conversa nos chats do{" "}
                <a
                  href="https://t.me/databasus_community"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Telegram
                </a>{" "}
                e{" "}
                <a
                  href="/discord"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  Discord
                </a>
                . Lá você pode tirar dúvidas, contar como usa o Databasus, discutir funcionalidades e ficar sabendo das novas versões antes de todo mundo.
              </p>

              <p>Escolha o aplicativo que você já usa.</p>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
