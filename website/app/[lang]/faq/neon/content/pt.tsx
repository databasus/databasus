import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

export const metadata: Metadata = {
  title: "Backup PostgreSQL no Neon: guia passo a passo | Databasus",
  description:
    "Guia passo a passo com capturas de tela: configure backups agendados de um banco PostgreSQL do Neon no Databasus, guarde os backups no seu próprio armazenamento e restaure no Neon quando precisar.",
  keywords: [
    "Databasus",
    "backup do Neon",
    "backup PostgreSQL no Neon",
    "backup de banco de dados Neon",
    "restauração no Neon",
    "usuário somente leitura Neon",
    "backup PostgreSQL",
    "backup de banco de dados",
  ],
  openGraph: {
    title: "Backup PostgreSQL no Neon: guia passo a passo | Databasus",
    description:
      "Guia passo a passo com capturas de tela: configure backups agendados de um banco PostgreSQL do Neon no Databasus, guarde os backups no seu próprio armazenamento e restaure no Neon quando precisar.",
    type: "article",
    url: getLocalizedUrl("pt", "faq/neon"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Backup PostgreSQL no Neon: guia passo a passo | Databasus",
    description:
      "Guia passo a passo com capturas de tela: configure backups agendados de um banco PostgreSQL do Neon no Databasus, guarde os backups no seu próprio armazenamento e restaure no Neon quando precisar.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "faq/neon"),
    languages: getLanguageAlternates("faq/neon"),
  },
  robots: "index, follow",
};

export default function NeonPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Como fazer backup PostgreSQL no Neon com o Databasus",
            description:
              "Guia passo a passo para fazer backup de um banco PostgreSQL do Neon com o Databasus e restaurá-lo no Neon",
            step: [
              {
                "@type": "HowToStep",
                name: "Copie a string de conexão no Neon",
                text: "No console do Neon, abra o projeto, clique em Connect e depois em Copy snippet.",
                url: `${getLocalizedUrl("pt", "faq/neon")}#copy-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "Adicione o banco do Neon ao Databasus",
                text: "Clique em Add database, escolha Neon, cole a string de conexão com Paste from clipboard e clique em Test connection.",
                url: `${getLocalizedUrl("pt", "faq/neon")}#paste-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "Crie um usuário somente leitura",
                text: "Clique em Yes, create read-only user. Assim o Databasus guarda credenciais que não conseguem alterar seus dados.",
                url: `${getLocalizedUrl("pt", "faq/neon")}#read-only-user`,
              },
              {
                "@type": "HowToStep",
                name: "Defina o agendamento, o armazenamento e a retenção",
                text: "Escolha com que frequência fazer o backup, onde guardar os arquivos e por quanto tempo mantê-los.",
                url: `${getLocalizedUrl("pt", "faq/neon")}#schedule-storage-retention`,
              },
              {
                "@type": "HowToStep",
                name: "Escolha os canais de notificação",
                text: "Marque os canais que vão receber as notificações de backup e clique em Complete. O Databasus inicia o primeiro backup.",
                url: `${getLocalizedUrl("pt", "faq/neon")}#notifiers`,
              },
              {
                "@type": "HowToStep",
                name: "Restaure um backup no Neon",
                text: "Clique no ícone de restauração ao lado do backup, cole a string de conexão do banco de destino no Neon e clique em Restore to this DB.",
                url: `${getLocalizedUrl("pt", "faq/neon")}#restore`,
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
              <h1 id="neon-backup">Como fazer backup PostgreSQL no Neon</h1>

              <p className="text-lg text-gray-400">
                O Neon é um PostgreSQL gerenciado e serverless. Para fazer
                backup dele com o Databasus, copie a string de conexão no
                console do Neon, adicione o banco no Databasus com o tipo Neon
                e escolha onde guardar os backups. A partir daí, o Databasus
                faz backups no horário que você definir e
                consegue restaurá-los em qualquer banco do Neon.
              </p>

              <p>
                Você só precisa do Databasus rodando. Se ainda não tiver, siga o{" "}
                <a href={getLocalizedHref("pt", "installation")}>guia de instalação</a>
                . O armazenamento e os canais de notificação podem ser criados durante a configuração.
              </p>

              <h2 id="copy-connection-string">
                Passo 1. Copie a string de conexão no Neon
              </h2>

              <p>
                Abra o projeto no console do Neon e clique em{" "}
                <strong>Connect</strong>.
              </p>

              <img
                src="/images/faq/neon/neon-connect-button.png"
                alt="Botão Connect na página Computes de um projeto do Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
              />

              <p>
                Na janela <strong>Connection details</strong>, escolha o
                branch, o banco e o role do backup e clique em{" "}
                <strong>Copy snippet</strong>.
              </p>

              <img
                src="/images/faq/neon/neon-copy-connection-string.png"
                alt="Janela Connection details do Neon com o botão Copy snippet"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="add-database">Passo 2. Adicione o banco do Neon ao Databasus</h2>

              <h3 id="paste-connection-string">
                Escolha Neon e cole a string de conexão
              </h3>

              <p>
                No Databasus, clique em <strong>Add database</strong>, dê um
                nome ao banco, escolha <strong>Neon</strong> e clique em{" "}
                <strong>Continue</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-choose-neon.png"
                alt="Janela Add database do Databasus com o tipo Neon selecionado"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Clique em <strong>Paste from clipboard</strong>. O Databasus lê
                a string de conexão que você copiou no Neon e preenche host,
                porta, usuário, senha, nome do banco e modo SSL.
              </p>

              <img
                src="/images/faq/neon/databasus-paste-connection-string.png"
                alt="Formulário de conexão do Neon no Databasus com o botão Paste from clipboard"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Clique em <strong>Test connection</strong>. Quando o teste
                passar, clique em <strong>Continue</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-test-connection.png"
                alt="Formulário de conexão do Neon preenchido no Databasus depois de um teste de conexão bem-sucedido"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="read-only-user">Crie um usuário somente leitura</h3>

              <p>
                O Databasus sugere criar um{" "}
                <a
                  href={`${getLocalizedHref("pt", "security")}#level-3-read-only-access`}
                >
                  usuário somente leitura
                </a>{" "}
                para os backups. Clique em{" "}
                <strong>Yes, create read-only user</strong>. Assim o Databasus
                guarda apenas credenciais que leem seus dados, mas não
                conseguem alterá-los.
              </p>

              <img
                src="/images/faq/neon/databasus-create-read-only-user.png"
                alt="O Databasus sugere criar um usuário somente leitura para os backups do Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="schedule-storage-retention">
                Defina o agendamento, o armazenamento e a retenção
              </h3>

              <p>Escolha:</p>

              <ul>
                <li>
                  <strong>Backup interval</strong> e horário: com que
                  frequência o Databasus faz o backup, por exemplo todo dia às
                  03:00.
                </li>
                <li>
                  <strong>Storage</strong>: para onde vão os backups. Dá para{" "}
                  <a href={getLocalizedHref("pt", "storages")}>
                    adicionar um armazenamento
                  </a>{" "}
                  direto nessa lista. Para guardar os backups também no Neon, veja{" "}
                  <a href={getLocalizedHref("pt", "storages/neon-object-storage")}>
                    como fazer backup no Neon Object Storage
                  </a>
                  .
                </li>
                <li>
                  <strong>Encryption</strong>: se o Databasus deve{" "}
                  <a
                    href={`${getLocalizedHref("pt", "security")}#level-2-backups-encryption`}
                  >
                    criptografar os arquivos de backup
                  </a>{" "}
                  antes de enviá-los.
                </li>
                <li>
                  <strong>Retention policy</strong>: por quanto tempo manter os
                  backups antigos, por exemplo 3 meses.
                </li>
                <li>
                  <strong>Notifications</strong>: sobre quais eventos avisar,
                  só sobre backups com falha ou também sobre os bem-sucedidos.
                </li>
              </ul>

              <img
                src="/images/faq/neon/databasus-schedule-storage-retention.png"
                alt="Configurações de agendamento, armazenamento, criptografia e retenção de backups no Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="notifiers">Escolha os canais de notificação</h3>

              <p>
                Marque os{" "}
                <a href={getLocalizedHref("pt", "notifiers")}>
                  canais de notificação
                </a>{" "}
                que vão receber esses avisos, por exemplo{" "}
                <a href={getLocalizedHref("pt", "notifiers/slack")}>Slack</a>{" "}
                ou{" "}
                <a href={getLocalizedHref("pt", "notifiers/teams")}>
                  Microsoft Teams
                </a>
                . Você pode marcar vários, ou nenhum e adicionar depois. Clique
                em <strong>Complete</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-choose-notifiers.png"
                alt="Escolha dos canais de notificação no Databasus antes de concluir a configuração do banco"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="first-backup">Passo 3. Confira o primeiro backup</h2>

              <p>
                O Databasus inicia o primeiro backup assim que você clica em{" "}
                <strong>Complete</strong>. Ele aparece na aba{" "}
                <strong>Backups</strong> com o tamanho e a duração. Os próximos
                seguem o agendamento, e o botão{" "}
                <strong>Make backup right now</strong> faz um backup na hora,
                sempre que você quiser.
              </p>

              <img
                src="/images/faq/neon/databasus-first-neon-backup.png"
                alt="Primeiro backup do Neon concluído na aba Backups do Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Um backup bem-sucedido ainda não garante que a restauração vai
                funcionar. Para conferir se cada backup pode mesmo ser
                restaurado, configure a{" "}
                <a href={getLocalizedHref("pt", "restore-verification")}>
                  verificação de restauração
                </a>
                .
              </p>

              <h2 id="restore">Como restaurar um backup do Neon</h2>

              <p>
                Clique no ícone de restauração (uma nuvem com uma seta) ao lado
                do backup. No console do Neon, copie a string de conexão do
                banco de destino do mesmo jeito que no{" "}
                <a href="#copy-connection-string">passo 1</a>, clique em{" "}
                <strong>Paste from clipboard</strong> e depois em{" "}
                <strong>Restore to this DB</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-restore-neon-backup.png"
                alt="Janela de restauração do backup no Databasus com o banco de destino no Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Antes da restauração, o Databasus limpa o banco de destino. O
                destino pode ser:
              </p>

              <ul>
                <li>um banco novo no mesmo projeto do Neon</li>
                <li>um banco em um branch do Neon</li>
                <li>um banco existente, inclusive o próprio banco de origem</li>
              </ul>

              <p>
                O caminho mais seguro é um branch novo do Neon. Lá você confere
                os dados restaurados, e o banco de produção fica intacto até
                você decidir fazer a troca.
              </p>

              <p>Quando a restauração termina, o status muda para Successful.</p>

              <img
                src="/images/faq/neon/databasus-restore-done.png"
                alt="Restauração de um backup do Neon concluída no Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="health-check">
                A verificação de disponibilidade mantém o compute do Neon
                acordado?
              </h2>

              <p>
                Por padrão, não. O Neon coloca o compute para dormir quando
                ninguém está conectado e não cobra por ele enquanto está
                dormindo. A verificação de disponibilidade se conecta a cada
                minuto, o que manteria o compute acordado e sendo cobrado 24
                horas por dia. Por isso o Databasus desativa essa verificação
                nos bancos do Neon. Se você quiser ativá-la, o Databasus pede
                uma confirmação antes. Um backup agendado acorda o compute só
                enquanto está rodando, e depois o Neon o coloca para dormir de
                novo.
              </p>

              <p>
                Também faz backup do Supabase? Veja{" "}
                <a href={getLocalizedHref("pt", "faq/supabase")}>
                  como fazer backup do Supabase
                </a>
                .
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("pt", "faq")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Voltar às perguntas frequentes
                </a>
              </div>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
