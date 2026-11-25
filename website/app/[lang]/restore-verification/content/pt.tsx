import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Verificação de restauração de backup - Documentação do Databasus",
  description:
    "Prove que os backups da sua base de dados são realmente restauráveis. O Databasus obtém o backup mais recente, restaura-o num container de base de dados descartável, compara a base restaurada com a origem e reporta a contagem de linhas por tabela em cada execução.",
  keywords: [
    "verificação de restauração",
    "restauração de base de dados",
    "verificação de backup",
    "recuperação de desastres",
    "teste de backup de base de dados",
    "agente de verificação do Databasus",
    "integridade de backup",
    "teste de restauração automatizado",
  ],
  openGraph: {
    title: "Verificação de restauração de backup - Documentação do Databasus",
    description:
      "Prove que os backups da sua base de dados são realmente restauráveis. O Databasus obtém o backup mais recente, restaura-o num container de base de dados descartável, compara a base restaurada com a origem e reporta a contagem de linhas por tabela em cada execução.",
    type: "article",
    url: getLocalizedUrl("pt", "restore-verification"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Verificação de restauração de backup - Documentação do Databasus",
    description:
      "Prove que os backups da sua base de dados são realmente restauráveis. O Databasus obtém o backup mais recente, restaura-o num container de base de dados descartável, compara a base restaurada com a origem e reporta a contagem de linhas por tabela em cada execução.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "restore-verification"),
    languages: getLanguageAlternates("restore-verification"),
  },
  robots: "index, follow",
};

export default function RestoreVerificationPage() {
  const downloadAgent = `curl -L -o verification-agent "https://your-databasus-host/api/v1/system/verification-agent?arch=amd64" \\
  && chmod +x verification-agent`;

  const startAgent = `./verification-agent start \\
  --databasus-host=https://your-databasus-host \\
  --agent-id=<AGENT_ID> \\
  --token=<TOKEN> \\
  --max-cpu=2 \\
  --max-ram-mb=2048 \\
  --max-disk-gb=20 \\
  --max-concurrent-jobs=1`;

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
              "Verificação de restauração de backup - Documentação do Databasus",
            description:
              "Prove que os backups da sua base de dados são realmente restauráveis. O Databasus obtém o backup mais recente, restaura-o num container de base de dados descartável, compara a base restaurada com a origem e reporta a contagem de linhas por tabela em cada execução.",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Como configurar a verificação de restauração no Databasus",
            description:
              "Guia passo a passo para registrar um agente de verificação, iniciá-lo no seu servidor e configurar a verificação de restauração agendada.",
            step: [
              {
                "@type": "HowToStep",
                name: "Criar um agente de verificação na interface",
                text: "Vá a Settings → Verification agents, clique em Create verification agent, dê um nome e copie o token e o ID do agente na janela.",
              },
              {
                "@type": "HowToStep",
                name: "Baixar o binário do agente",
                text: "Execute o comando curl no servidor onde a verificação deve rodar, escolhendo amd64 ou arm64 conforme a arquitetura.",
              },
              {
                "@type": "HowToStep",
                name: "Iniciar o agente",
                text: "Inicie o agente com --agent-id, --token e os limites de recursos (--max-cpu, --max-ram-mb, --max-disk-gb, --max-concurrent-jobs).",
              },
              {
                "@type": "HowToStep",
                name: "Agendar as verificações",
                text: "Abra as configurações de verificação da base de dados, ative Scheduled verification e escolha um intervalo (After backup, Hourly, Daily, Weekly, Monthly ou Cron).",
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
              <h1 id="restore-verification">
                Verificação de restauração de backup
              </h1>

              <p className="text-lg text-gray-400">
                Um backup que termina sem erros não é o mesmo que um backup que
                consegue realmente restaurar. A única prova real é restaurá-lo.
                O Databasus faz isso por você, de forma agendada:
              </p>

              <ul>
                <li>pega o backup mais recente</li>
                <li>
                  executa a restauração num container de base de dados
                  descartável
                </li>
                <li>compara a base de dados restaurada com a origem</li>
                <li>destrói o container</li>
                <li>reporta o resultado</li>
              </ul>

              <img
                src="/images/restore-verification/verified-backups.png"
                alt="Verified backups tab"
                loading="lazy"
              />

              <img
                className="mt-5"
                src="/images/restore-verification/verifications.png"
                alt="Verifications tab"
                loading="lazy"
              />

              <h2 id="what-is-verification-agent">
                O que é um agente de verificação?
              </h2>

              <p>
                O agente de verificação é um pequeno binário em Go que roda numa
                máquina sob o seu controle. Serve qualquer máquina com CPU, RAM
                e disco livres. O agente se conecta ao Databasus, coleta
                trabalhos de verificação de uma fila, executa-os localmente e
                retorna os resultados.
              </p>

              <h3 id="what-you-need">O que você precisa</h3>

              <ul>
                <li>
                  Um servidor com acesso HTTPS de saída para a URL do seu
                  Databasus.
                </li>
                <li>
                  Docker disponível nesse servidor. Para cada trabalho, o agente
                  cria containers de base de dados efêmeros da mesma versão
                  principal.
                </li>
                <li>
                  Capacidade de disco para cada trabalho de verificação que
                  cubra o <strong>tamanho do arquivo de backup</strong>, o{" "}
                  <strong>tamanho bruto da base de dados</strong>, uma reserva e os{" "}
                  <strong>maiores índices</strong> construídos ao mesmo tempo.
                </li>
                <li>
                  Pelo menos 1 núcleo de CPU e 512 MB de RAM disponíveis por
                  trabalho simultâneo.
                </li>
              </ul>

              <h3 id="why-not-just-checksums">Por que checksums não bastam?</h3>

              <p>
                Checksums e códigos de saída pegam alguns modos de falha, mas
                deixam escapar outros por completo:
              </p>

              <ul>
                <li>
                  <strong>Checksums</strong> detectam corrupção de bits no
                  arquivo, mas não dizem nada sobre se o dump em si está
                  completo ou é semanticamente válido.
                </li>
                <li>
                  <strong>O código de saída do dump</strong> diz que o comando
                  de dump rodou. Não detecta uma role sem permissões de leitura
                  em certos objetos, uma extensão ausente na origem ou uma
                  incompatibilidade de tablespaces. Qualquer um desses casos
                  pode fazer com que objetos sejam silenciosamente ignorados ou
                  removidos.
                </li>
                <li>
                  <strong>A verificação de restauração</strong> passa realmente
                  o arquivo pela ferramenta de restauração nativa da base de
                  dados e conta as linhas por tabela. É a única verificação que
                  pega tudo o que foi descrito acima. Se um backup não
                  restaurar, você descobre antes de precisar dele, e não durante
                  um desastre.
                </li>
              </ul>

              <h2 id="configuration">Configuração</h2>

              <h3 id="create-on-ui">Criar um agente na interface</h3>

              <p>
                Abra <strong>Settings → Verification agents</strong> e clique em{" "}
                <strong>Create verification agent</strong>. Escolha um nome
                descritivo como <code>staging-verifier</code> ou{" "}
                <code>eu-west-host-1</code>. A janela seguinte mostra o{" "}
                <strong>token</strong> e o <strong>ID</strong> do agente.
              </p>

              <p>
                O token é mostrado <strong>exatamente uma vez</strong>: copie-o
                antes de fechar a janela. Se você o perder mais tarde, use a
                ação <strong>Rotate token</strong> na linha do agente para
                emitir um novo; o token antigo deixa de funcionar no heartbeat
                seguinte do agente. A janela que se segue mostra os comandos de
                instalação para a arquitetura do seu servidor, os mesmos
                comandos descritos abaixo.
              </p>

              <h3 id="launch">Iniciar o agente no seu servidor</h3>

              <p>
                Acesse por SSH a máquina que vai executar as verificações.
                Primeiro, baixe o binário do agente. Substitua{" "}
                <code>https://your-databasus-host</code> pela URL do seu
                Databasus e troque <code>amd64</code> por <code>arm64</code> se
                o servidor for ARM:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={downloadAgent} />
                </div>
              </div>

              <p>
                Depois inicie o agente. O ID do agente e o token vêm da janela
                do passo anterior:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={startAgent} />
                </div>
              </div>

              <p>
                O <code>start</code> transforma o agente em daemon e grava as
                flags em <code>databasus-verification.json</code> no diretório
                de trabalho, por isso reinícios posteriores podem usar{" "}
                <code>./verification-agent start</code> sem flag nenhuma. Os
                logs ficam em <code>databasus-verification.log</code>, ao lado
                do binário.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] mb-3 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    O host do Databasus precisa ser <code>https://</code>. HTTP
                    simples só é permitido se você adicionar{" "}
                    <code>--allow-insecure-http</code>, e se destina a testes
                    locais. Nunca exponha um agente de produção por HTTP sem
                    criptografia.
                  </p>
                </div>
              </div>

              <p>
                As quatro flags <code>--max-*</code> são{" "}
                <strong>orçamentos</strong>, não alocações por trabalho. O
                agente as reporta ao Databasus em cada heartbeat, e o Databasus
                as divide pelos trabalhos simultâneos que permitir. Com{" "}
                <code>
                  --max-cpu=2 --max-ram-mb=2048 --max-concurrent-jobs=1
                </code>{" "}
                o único trabalho recebe os 2 CPUs e os 2 GB de RAM. Com{" "}
                <code>--max-concurrent-jobs=2</code>, cada trabalho recebe 1 CPU
                e 1 GB. O mínimo é 1 CPU e 512 MB por trabalho: se o orçamento
                não for suficiente para esse mínimo, o agente anuncia menos
                concorrência. O orçamento de disco é o mais fácil de errar: cada
                trabalho precisa de espaço para o{" "}
                <strong>tamanho do arquivo de backup</strong>, o{" "}
                <strong>tamanho bruto da base de dados</strong>, uma{" "}
                <strong>reserva de cerca de 1,5 GB</strong> e os seus{" "}
                <strong>maiores índices</strong>, um por trabalho de restauração em
                paralelo, porque cada índice em construção ocupa no disco uma
                ordenação de tamanho parecido com o dele. O Databasus reduz o
                número de trabalhos de restauração em paralelo até tudo isso caber
                no disco livre do agente. Por isso, um <code>--max-cpu</code> maior
                precisa de mais disco, e um <code>--max-disk-gb</code> maior
                permite restaurar uma base grande mais rápido.
              </p>

              <h3 id="manage">Gerenciar o agente</h3>

              <p>O mesmo binário oferece quatro subcomandos:</p>

              <ul>
                <li>
                  <code>./verification-agent status</code> — mostra se o daemon
                  está rodando e em que trabalhos está trabalhando.
                </li>
                <li>
                  <code>./verification-agent stop</code> — para o daemon. As
                  verificações em curso são reportadas ao Databasus como
                  falhadas e voltam para a fila.
                </li>
                <li>
                  <code>./verification-agent start</code> — reinicia o daemon.
                  As flags são lembradas da primeira inicialização; passe{" "}
                  <code>--token=&lt;NEW&gt;</code> depois de uma rotação para
                  atualizar o token salvo.
                </li>
                <li>
                  <code>./verification-agent run</code> — executa em primeiro
                  plano em vez de como daemon. Use este modo ao envolver o
                  agente numa unidade systemd ou num container Docker: esses
                  supervisores esperam que o processo não faça fork.
                </li>
              </ul>

              <p>
                A página Settings mostra três ações em cada linha de agente: ver
                novamente os comandos de instalação (sem revelar o token),
                rotacionar o token e apagar o agente. Apagar é seguro: as
                verificações atribuídas a esse agente voltam para a fila e são
                assumidas por outro agente, se houver algum disponível.
              </p>

              <h2 id="schedules-and-notifications">
                Agendamentos e notificações
              </h2>

              <p>
                A verificação de restauração é configurada por base de dados.
                Abra as configurações de verificação da base de dados, ative{" "}
                <strong>Scheduled verification</strong> e escolha um intervalo.
              </p>

              <h3 id="interval-options">Opções de intervalo</h3>

              <ul>
                <li>
                  <strong>After backup</strong> — a garantia mais forte: cada
                  backup bem-sucedido é verificado assim que termina.
                </li>
                <li>
                  <strong>Hourly, daily, weekly, monthly</strong> — escolha uma
                  frequência e uma hora do dia.
                </li>
                <li>
                  <strong>Cron</strong> — uma expressão cron em UTC para tudo o
                  que as opções predefinidas não cobrem. Exemplos:{" "}
                  <code>0 4 * * 0</code> (todos os domingos às 4:00 UTC) e{" "}
                  <code>0 */6 * * *</code> (a cada seis horas).
                </li>
              </ul>

              <h3 id="how-the-queue-works">
                Como a fila trata o &quot;After backup&quot;
              </h3>

              <p>
                Uma verificação costuma ser mais lenta do que o backup que a
                originou, por isso, se os backups chegarem mais rápido do que as
                verificações terminam, a fila cresceria sem parar. O Databasus
                evita isso{" "}
                <strong>
                  cancelando qualquer verificação pendente da mesma base de
                  dados sempre que chega um backup novo
                </strong>
                : só o backup mais recente espera na fila. O compromisso é
                intencional: é melhor pular a verificação de um backup
                desatualizado do que passar horas verificando algo a partir do
                qual você nunca restauraria.
              </p>

              <h3 id="manual-runs">Execuções manuais</h3>

              <p>
                Você também pode disparar uma verificação pontual a partir da
                aba <strong>Restore verifications</strong> da base de dados, sem
                mudar o agendamento. É útil para conferir um backup específico
                ou testar um agente novo de ponta a ponta antes de confiar a ele
                a carga agendada.
              </p>

              <h3 id="notifications">Notificações</h3>

              <p>
                Sucesso e falha podem ser enviados por qualquer notificador já
                conectado à base de dados. As duas caixas de seleção,{" "}
                <strong>Verification success</strong> e{" "}
                <strong>Verification failed</strong>, são independentes. A
                maioria das equipes ativa só a de falha, para evitar fadiga de
                notificações. Consulte a{" "}
                <a
                  href="/pt/notifiers"
                  className="text-blue-400 hover:text-blue-300"
                >
                  documentação de notificadores
                </a>{" "}
                para conectar Slack, Microsoft Teams, Discord, email e outros.
              </p>

              <h3 id="results">Ler os resultados</h3>

              <p>
                Cada tentativa de verificação aparece como uma linha na aba{" "}
                <strong>Restore verifications</strong> da base de dados. O
                status é um de <strong>Pending</strong>,{" "}
                <strong>Running</strong>, <strong>Successful</strong>,{" "}
                <strong>Failed</strong> ou <strong>Canceled</strong>. Ao clicar
                numa linha abre-se um painel com a cronologia completa, o código
                de saída da restauração, o tamanho da base de dados restaurada,
                as contagens de esquemas e tabelas e a contagem de linhas por
                tabela. As execuções falhadas mostram a mensagem de erro no topo
                do painel.
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
