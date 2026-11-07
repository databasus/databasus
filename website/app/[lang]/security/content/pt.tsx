import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "Segurança - Como o Databasus protege os seus dados | Databasus",
  description:
    "Saiba como o Databasus garante segurança de nível empresarial com criptografia AES-256-GCM de dados sensíveis e backups, acesso só de leitura à base de dados e registro de auditoria completo.",
  keywords: [
    "segurança do Databasus",
    "segurança de backup PostgreSQL",
    "criptografia AES-256-GCM",
    "criptografia de base de dados",
    "criptografia de backups",
    "acesso só de leitura à base de dados",
    "segurança empresarial",
    "proteção de dados",
    "backups seguros",
  ],
  openGraph: {
    title: "Segurança - Como o Databasus protege os seus dados | Databasus",
    description:
      "Saiba como o Databasus garante segurança de nível empresarial com criptografia AES-256-GCM de dados sensíveis e backups, acesso só de leitura à base de dados e registro de auditoria completo.",
    type: "article",
    url: getLocalizedUrl("pt", "security"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Segurança - Como o Databasus protege os seus dados | Databasus",
    description:
      "Saiba como o Databasus garante segurança de nível empresarial com criptografia AES-256-GCM de dados sensíveis e backups, acesso só de leitura à base de dados e registro de auditoria completo.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "security"),
    languages: getLanguageAlternates("security"),
  },
  robots: "index, follow",
};

export default function SecurityPage() {
  const encryptionPipeline = `PostgreSQL pg_dump → Compression → Encryption → Cloud Storage`;
  const disableTwoFactorCommand = `docker exec -it databasus ./main --disable-2fa`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Segurança - Como o Databasus protege os seus dados",
            description:
              "Saiba como o Databasus garante segurança de nível empresarial com criptografia AES-256-GCM de dados sensíveis e backups, acesso só de leitura à base de dados e registro de auditoria completo.",
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
              <h1 id="security">Como o Databasus garante a segurança?</h1>

              <p className="text-lg text-gray-400">
                O Databasus é responsável por dados sensíveis:
              </p>

              <ul>
                <li>acessa a sua base de dados;</li>
                <li>faz backup dela (ou seja, cria uma cópia dos dados);</li>
                <li>
                  guarda credenciais para poder acessar a sua base de dados de
                  forma regular;
                </li>
                <li>
                  salva backups no seu S3 ou em outros armazenamentos na nuvem
                  (se você ativar);
                </li>
              </ul>

              <p>
                Por isso,{" "}
                <strong>
                  a principal prioridade do Databasus é ser seguro e confiável
                  em nível empresarial
                </strong>
                .
              </p>

              <p>Para garantir que:</p>

              <ul>
                <li>
                  os dados sensíveis nunca ficam expostos e estão sempre
                  criptografados;
                </li>
                <li>
                  os backups são criptografados e inúteis mesmo que alguém os
                  veja no armazenamento na nuvem;
                </li>
                <li>
                  o Databasus nem sequer recebe acesso de escrita ou alteração à
                  base de dados;
                </li>
                <li>todas as ações são registradas e podem ser auditadas;</li>
              </ul>

              <p>
                Todos esses passos protegem os seus dados. Não existe sistema
                100% seguro, mas fazemos o possível para chegar perto disso.
                Mesmo em caso de invasão, ninguém conseguirá corromper os seus
                dados.
              </p>

              <p>O Databasus aplica segurança em três níveis:</p>

              <ol>
                <li>Criptografia de dados sensíveis;</li>
                <li>Criptografia de backups;</li>
                <li>Acesso só de leitura à base de dados.</li>
              </ol>

              <h2 id="level-1-sensitive-data-encryption">
                Nível 1: criptografia de dados sensíveis
              </h2>

              <p>
                Internamente, o Databasus usa uma base de dados PostgreSQL para
                guardar detalhes de conexão e as configurações de notificadores
                e armazenamentos (S3, Google Drive, Dropbox, etc.).
              </p>

              <p>Todos os dados sensíveis são criptografados. Por exemplo:</p>

              <ul>
                <li>senhas</li>
                <li>tokens</li>
                <li>webhooks com segredos</li>
              </ul>

              <p>
                Assim, na base de dados o Databasus guarda apenas hashes ou
                valores codificados. Para a criptografia é usado o algoritmo{" "}
                <strong>AES-256-GCM</strong>. Além disso, apesar da
                criptografia, esses valores nunca são expostos via API ou
                interface.
              </p>

              <p>
                A chave secreta usada na criptografia fica no armazenamento
                local (<code>./databasus-data/secret.key</code> por padrão) e
                não está presente na própria base de dados. Assim, comprometer a
                base de dados não dá acesso aos dados sensíveis.
              </p>

              <h2 id="level-2-backups-encryption">
                Nível 2: criptografia de backups
              </h2>

              <p>
                Cada arquivo de backup é criptografado em tempo real durante a
                criação do backup. O Databasus usa o algoritmo{" "}
                <strong>AES-256-GCM</strong>, que garante que os dados do backup
                não podem ser lidos sem a chave de criptografia e que qualquer
                adulteração é detectada durante a descriptografia.
              </p>

              <p>Os backups passam por este pipeline:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{encryptionPipeline}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={encryptionPipeline} />
                </div>
              </div>

              <p>
                Cada backup recebe uma chave de criptografia única derivada de:
              </p>

              <ul>
                <li>
                  Chave mestra (guardada em{" "}
                  <code>./databasus-data/secret.key</code>)
                </li>
                <li>ID do backup</li>
                <li>Salt aleatório (único por backup)</li>
              </ul>

              <p>
                <strong>Resultado</strong>: mesmo que alguém consiga acesso ao
                seu armazenamento na nuvem (S3, Google Drive, etc.), não
                consegue ler os backups sem a sua chave mestra.
              </p>

              <h2 id="level-3-read-only-access">
                Nível 3: acesso só de leitura à base de dados
              </h2>

              <p>
                O Databasus aplica o princípio do menor privilégio: só precisa
                de acesso de leitura para criar backups, nunca de acesso de
                escrita. Isso protege a sua base de dados contra corrupção de
                dados acidental ou maliciosa através da ferramenta de backup.
              </p>

              <p>
                Antes de aceitar credenciais de base de dados, o Databasus faz
                verificações em três níveis:
              </p>

              <ol>
                <li>
                  <strong>Nível de role</strong>: confirma que o usuário NÃO é
                  superuser e não pode criar roles nem bases de dados
                </li>
                <li>
                  <strong>Nível de base de dados</strong>: garante que não há
                  privilégios CREATE nem TEMP
                </li>
                <li>
                  <strong>Nível de tabela</strong>: confirma zero permissões de
                  escrita (INSERT, UPDATE, DELETE, TRUNCATE, etc.)
                </li>
              </ol>

              <p>
                O usuário da base de dados precisa passar nas três verificações
                para ser considerado só de leitura. Se for detectado qualquer
                privilégio de escrita, o Databasus avisa.
              </p>

              <p>
                O Databasus sugere a criação de usuários só de leitura com as
                permissões adequadas:
              </p>

              <ul>
                <li>Concede SELECT em todas as tabelas atuais e futuras</li>
                <li>Concede USAGE nos esquemas (mas não CREATE)</li>
                <li>Revoga explicitamente todos os privilégios de escrita</li>
              </ul>

              <p>
                <strong>Resultado</strong>: mesmo que o Databasus seja
                comprometido, o servidor invadido, a chave secreta roubada e as
                credenciais descriptografadas, os atacantes não conseguem
                corromper a sua base de dados.
              </p>

              <h2 id="two-factor-authentication">
                Autenticação de dois fatores no login
              </h2>

              <p>
                Só a senha já abre todas as credenciais de bases de dados, as
                chaves de armazenamento e os backups que a instância guarda, por
                isso um administrador pode exigir um segundo fator no login por
                senha. Com a opção ligada, a senha correta envia um código de
                seis dígitos para o e-mail da conta e o login termina apenas
                quando esse código é informado. O código para de funcionar dez
                minutos depois do envio ou após cinco tentativas erradas, e
                pedir outro cancela o anterior. Ligar a opção protege o próximo
                login e mantém funcionando as sessões já abertas.
              </p>

              <p>
                A configuração vale para o login por senha. Entrar pelo Google
                ou pelo GitHub continua entregando um token direto, porque esses
                provedores fazem as próprias verificações de vários fatores: uma
                instância que queira o segundo fator para todo mundo
                simplesmente não os configura.
              </p>

              <p>
                Ela só pode ser ligada quando a instância tem servidor de e-mail
                e todo administrador ativo tem um endereço de verdade, para que
                ninguém ligue a opção e tranque a instância. Se o e-mail parar
                de ser entregue depois, ninguém entra com senha, e isso é
                proposital: um servidor de e-mail quebrado não pode rebaixar o
                login para um fator em silêncio. Um operador com acesso ao
                servidor desliga a opção de novo:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{disableTwoFactorCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={disableTwoFactorCommand} />
                </div>
              </div>

              <p>
                O comando informa o que mudou e registra isso no log de
                auditoria, então uma instância que parou de pedir códigos
                explica o motivo. Ele está documentado ao lado da recuperação de
                senha na{" "}
                <a
                  href="/pt/password#disable-two-factor"
                  className="text-blue-400 hover:text-blue-600"
                >
                  página de senha
                </a>
                .
              </p>

              <h2 id="security-and-reliability-engineering">
                🛡️ Engenharia de segurança e confiabilidade
              </h2>

              <p>
                O Databasus trabalha com dados sensíveis, por isso prevenir
                vulnerabilidades, acessos não autorizados e vazamentos de
                dados é uma preocupação central. Investimos nisso dos dois
                lados do sistema: no próprio código (verificações de
                permissões, criptografia, tratamento cuidadoso de segredos) e
                na infraestrutura ao redor dele (análise de dependências,
                resposta a CVEs, práticas de DevSecOps). O pipeline abaixo
                roda automaticamente antes de cada release. Nenhuma camada
                chega sozinha, mas juntas reduzem a probabilidade de código
                vulnerável, dependências inseguras, imagens quebradas ou
                backups não restauráveis chegarem a uma release.
              </p>

              <h3 id="static-analysis">Análise estática</h3>

              <p>
                A análise estática roda em várias passagens independentes. O
                CodeQL analisa todo o código em busca de problemas de segurança.
                Antes de cada release, o <strong>gitleaks</strong> procura segredos vazados e o{" "}
                <strong>semgrep</strong> verifica o código com regras de segurança.
                Os
                Dockerfiles e os workflows de CI têm regras adicionais próprias
                (referências de actions fixadas, permissões de menor privilégio,
                imagens base suspeitas), por isso padrões inseguros são
                sinalizados antes da release.
              </p>

              <p>
                Além dessas verificações antes da release, o{" "}
                <strong>Codex Security</strong> da OpenAI executa auditorias
                regulares e mais profundas de todo o código. É um programa
                separado que encontra problemas arquiteturais e transversais que
                as análises automáticas mais restritas não veem.
              </p>

              <h3 id="dependency-management">Gestão de dependências</h3>

              <p>
                O Dependabot vigia todas as nossas dependências contra a GitHub
                Advisory Database e sinaliza CVEs minutos depois da publicação.
                As atualizações passam por um período de espera para que as
                versões recém-publicadas amadureçam antes de as adotarmos, uma
                defesa deliberada contra incidentes de pacotes comprometidos,
                como ataques à cadeia de suprimentos.
              </p>

              <p>
                A <strong>Dependency Review Action</strong> bloqueia uma release que introduza um novo CVE <strong>HIGH</strong> ou{" "}
                <strong>CRITICAL</strong>.
              </p>

              <h3 id="container-and-ci-hardening">
                Endurecimento de containers e CI
              </h3>

              <ul>
                <li>
                  A imagem da aplicação é analisada com <strong>Trivy</strong> nas
                  duas arquiteturas antes de cada release.
                </li>
                <li>
                  Uma passagem separada do Trivy sobre o Dockerfile detecta
                  configurações erradas antes de entrarem numa imagem.
                </li>
                <li>
                  Todas as GitHub Actions estão fixadas a SHAs de commit
                  completos, em vez de tags flutuantes como <code>@v4</code> ou{" "}
                  <code>@main</code>, que foram um vetor de ataque ativo em
                  2025.
                </li>
                <li>
                  Os workflows usam permissões de menor privilégio por padrão e
                  só as elevam por job quando é realmente necessário.
                </li>
              </ul>

              <h3 id="testing-and-verification">Testes e verificação</h3>

              <p>
                Os caminhos críticos estão cobertos por testes unitários e de
                integração, executados contra containers de bases de dados reais
                para cada motor suportado e cada versão principal.
              </p>

              <p>
                A restauração é o caminho que mais importa numa ferramenta de
                backup, por isso a testamos explicitamente: antes de cada
                release, executamos ciclos completos de backup e restauração
                contra esses mesmos containers reais, confirmando que os
                backups podem realmente ser restaurados de ponta a ponta, e
                não apenas escritos com sucesso.
              </p>

              <p>
                O resto do pipeline de CI/CD executa lint, verificação de
                tipos, a suíte completa de testes, smoke tests das imagens e
                builds multiarquitetura antes de cada release. Uma release só
                sai depois que tudo isso e as varreduras de segurança passam,
                sem nenhum alerta crítico do CodeQL em aberto.
              </p>

              <h3 id="reporting-a-vulnerability">
                Reportar uma vulnerabilidade
              </h3>

              <p>
                Encontrou uma vulnerabilidade? Reporte-a pela aba Security do
                GitHub. Veja o{" "}
                <a
                  href="https://github.com/databasus/databasus?tab=security-ov-file#readme"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  SECURITY.md
                </a>
                . Os relatórios de segurança são a fila de trabalho com maior
                prioridade.
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
