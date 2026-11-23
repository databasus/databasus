import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

const PAGE_URL = getLocalizedUrl("pt", "storages/neon-object-storage");

export const metadata: Metadata = {
  title: "Backup PostgreSQL no Neon Object Storage: guia passo a passo | Databasus",
  description:
    "Guia passo a passo com capturas de tela: crie um bucket e uma credencial no Neon Object Storage, adicione-o ao Databasus como armazenamento S3 e mantenha os backups dos seus bancos de dados no Neon com agendamento.",
  keywords: [
    "Databasus",
    "backup no Neon Object Storage",
    "Neon Object Storage",
    "Neon S3",
    "bucket do Neon",
    "backup PostgreSQL no S3",
    "armazenamento compatível com S3",
    "backup de banco de dados",
  ],
  openGraph: {
    title: "Backup PostgreSQL no Neon Object Storage: guia passo a passo | Databasus",
    description:
      "Guia passo a passo com capturas de tela: crie um bucket e uma credencial no Neon Object Storage, adicione-o ao Databasus como armazenamento S3 e mantenha os backups dos seus bancos de dados no Neon com agendamento.",
    type: "article",
    url: PAGE_URL,
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Backup PostgreSQL no Neon Object Storage: guia passo a passo | Databasus",
    description:
      "Guia passo a passo com capturas de tela: crie um bucket e uma credencial no Neon Object Storage, adicione-o ao Databasus como armazenamento S3 e mantenha os backups dos seus bancos de dados no Neon com agendamento.",
  },
  alternates: {
    canonical: PAGE_URL,
    languages: getLanguageAlternates("storages/neon-object-storage"),
  },
  robots: "index, follow",
};

export default function NeonObjectStoragePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Como fazer backup no Neon Object Storage com o Databasus",
            description:
              "Guia passo a passo para criar um bucket e uma credencial no Neon Object Storage e adicioná-lo ao Databasus como armazenamento de backups",
            step: [
              {
                "@type": "HowToStep",
                name: "Crie um bucket no Neon",
                text: "No Neon Console, abra o projeto dos backups, clique em Object storage na barra lateral e depois em Create bucket.",
                url: `${PAGE_URL}#create-bucket`,
              },
              {
                "@type": "HowToStep",
                name: "Copie as credenciais do armazenamento",
                text: "Clique em Connect, abra a aba Storage, escolha Parameters only e clique em Copy credentials.",
                url: `${PAGE_URL}#copy-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "Escolha Neon Object Storage no Databasus",
                text: "No Databasus, abra Storages, clique em Add storage e escolha Neon Object Storage na lista de tipos.",
                url: `${PAGE_URL}#choose-neon-object-storage`,
              },
              {
                "@type": "HowToStep",
                name: "Cole as credenciais e digite o nome do bucket",
                text: "Clique em Paste from clipboard para preencher o endpoint, a região, a access key e a secret key, e depois digite o nome do bucket.",
                url: `${PAGE_URL}#paste-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "Teste a conexão e salve",
                text: "Clique em Test connection e, quando o teste passar, em Save.",
                url: `${PAGE_URL}#test-connection`,
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="neon-object-storage-backup">Como fazer backup no Neon Object Storage</h1>

              <p className="text-lg text-gray-400">
                <a href="https://neon.com/docs/storage/overview">Neon Object Storage</a> é um
                armazenamento de objetos compatível com S3 que já vem em todo projeto do Neon.
                Para guardar seus backups lá, crie um bucket e uma credencial no Neon Console e
                depois adicione o bucket ao Databasus como armazenamento. O Databasus vai enviar
                para lá todo backup dos bancos de dados que usam esse armazenamento, estejam
                eles no Neon ou em qualquer outro lugar.
              </p>

              <p>
                Você só precisa de um Databasus rodando. Se ainda não tem, siga
                o <a href={getLocalizedHref("pt", "installation")}>guia de instalação</a>.
                A <a href="https://neon.com/docs/storage/s3-compatibility">página de
                compatibilidade com S3</a> da documentação do Neon lista o que ele suporta.
              </p>

              <h2 id="create-bucket">Passo 1. Crie um bucket no Neon</h2>

              <p>
                Abra no Neon Console o projeto que vai guardar os backups. Prefira um projeto
                diferente daquele dos bancos de dados que você protege: excluir um projeto
                também apaga os buckets dele. Clique em <strong>Object storage</strong> na barra
                lateral, depois em <strong>Create bucket</strong>, e digite um nome para o
                bucket. Um bucket novo é privado e pertence à branch atual do projeto.
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-object-storage-buckets.png"
                alt="Página Object storage de um projeto do Neon com um bucket e o botão Create bucket"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
              />

              <h2 id="copy-credentials">Passo 2. Copie as credenciais do armazenamento</h2>

              <p>
                Clique em <strong>Connect</strong> e abra a aba <strong>Storage</strong>.
                Com <strong>Parameters only</strong> selecionado, o Neon mostra o endpoint, a
                access key, a secret key e a região. Se o bloco mostrar marcadores no lugar das
                chaves, gere uma credencial primeiro, como pede a primeira linha. Depois clique
                em <strong>Copy credentials</strong>.
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-connect-storage-credentials.png"
                alt="Aba Storage da janela Connect do Neon com o endpoint, as chaves, a região e o botão Copy credentials"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                O bloco fica assim. Ele não traz o nome do bucket, então guarde o nome do passo
                1.
              </p>

              <pre>
                <code>{`AWS_ENDPOINT_URL_S3="https://br-<branch>.storage.c-9.us-east-2.aws.neon.tech"
AWS_ACCESS_KEY_ID="nak_live_..."
AWS_SECRET_ACCESS_KEY="..."
AWS_REGION="us-east-2"`}</code>
              </pre>

              <h2 id="add-storage">Passo 3. Adicione o armazenamento ao Databasus</h2>

              <h3 id="choose-neon-object-storage">Escolha Neon Object Storage no Databasus</h3>

              <p>
                No Databasus, abra <strong>Storages</strong> e clique em <strong>Add
                storage</strong>. Dê um nome ao armazenamento e escolha <strong>Neon Object
                Storage</strong> na lista de tipos. O Databasus o salva como um armazenamento S3
                comum, por isso a lista de armazenamentos mostra o tipo S3.
              </p>

              <h3 id="paste-credentials">Cole as credenciais e digite o nome do bucket</h3>

              <p>
                Clique em <strong>Paste from clipboard</strong>. O Databasus lê o bloco que você
                copiou no Neon e preenche o endpoint, a região, a access key e a secret key. O
                bloco fica no seu navegador: as chaves só saem dele quando você testa ou salva o
                armazenamento. Digite você mesmo o nome do bucket em <strong>S3 Bucket</strong>.
                Também dá para preencher todos os campos à mão abaixo do divisor <strong>or
                enter the details manually</strong>.
              </p>

              <img
                src="/images/storages/neon-object-storage/databasus-neon-object-storage-form.png"
                alt="Formulário do Neon Object Storage no Databasus com endpoint, região e chaves colados e o nome do bucket digitado"
                width={1435}
                height={897}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="test-connection">Teste a conexão e salve</h3>

              <p>
                Clique em <strong>Test connection</strong>. Quando o teste passar, clique
                em <strong>Save</strong>. Agora escolha esse armazenamento nas configurações de
                backup de qualquer banco de dados. Se o próprio banco está rodando no Neon,
                o <a href={getLocalizedHref("pt", "faq/neon")}>guia de backup do Neon</a> mostra
                o restante da configuração.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("pt", "storages")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Voltar aos armazenamentos
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
