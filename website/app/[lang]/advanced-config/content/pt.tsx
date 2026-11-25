import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Configuração avançada - Documentação do Databasus",
  description:
    "Variáveis de ambiente opcionais para auto-hospedar o Databasus: login com Google e GitHub, email SMTP, captcha Cloudflare Turnstile, telemetria, exportação de logs por OpenTelemetry e script de análise próprio. Desnecessárias numa instalação padrão.",
  keywords: [
    "variáveis de ambiente do Databasus",
    "configuração avançada do Databasus",
    "configuração self-hosted",
    "GitHub OAuth",
    "Google OAuth",
    "configuração de email SMTP",
    "Cloudflare Turnstile",
    "variáveis de ambiente Docker",
    "logs OpenTelemetry",
  ],
  openGraph: {
    title: "Configuração avançada - Documentação do Databasus",
    description:
      "Variáveis de ambiente opcionais para auto-hospedar o Databasus: login com Google e GitHub, email SMTP, captcha Cloudflare Turnstile, telemetria, exportação de logs por OpenTelemetry e script de análise próprio. Desnecessárias numa instalação padrão.",
    type: "article",
    url: getLocalizedUrl("pt", "advanced-config"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Configuração avançada - Documentação do Databasus",
    description:
      "Variáveis de ambiente opcionais para auto-hospedar o Databasus: login com Google e GitHub, email SMTP, captcha Cloudflare Turnstile, telemetria, exportação de logs por OpenTelemetry e script de análise próprio. Desnecessárias numa instalação padrão.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "advanced-config"),
    languages: getLanguageAlternates("advanced-config"),
  },
  robots: "index, follow",
};

export default function AdvancedConfigPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Configuração avançada - Documentação do Databasus",
            description:
              "Variáveis de ambiente opcionais para auto-hospedar o Databasus: login com Google e GitHub, email SMTP, captcha Cloudflare Turnstile, telemetria, exportação de logs por OpenTelemetry e script de análise próprio. Desnecessárias numa instalação padrão.",
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
              <h1 id="advanced-config">Configuração avançada</h1>

              <p className="text-lg text-gray-400">
                O Databasus já vem com valores padrão sensatos: uma instalação
                típica de container único não precisa de nenhuma configuração.
                Cada variável desta página é <strong>opcional</strong> e
                desnecessária em 99% dos ambientes de produção.
              </p>

              <h2 id="oauth">OAuth</h2>

              <p>
                Por padrão o Databasus usa login com email e senha. Você também
                pode permitir o login com conta Google ou GitHub. O botão de um
                provedor aparece assim que o seu client ID é definido, mas o
                login só se completa quando <strong>tanto</strong> o client ID
                quanto o client secret estão presentes.
              </p>

              <p>
                Ao registrar a aplicação OAuth, defina a URL de redirecionamento
                (callback) como{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>. Por
                causa desse redirecionamento, o login OAuth exige que a sua
                instância seja servida por HTTPS num domínio público — veja a
                nota abaixo.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    <strong>HTTPS é obrigatório para login e email.</strong>{" "}
                    Tanto o login OAuth quanto o email precisam que a sua
                    instância esteja acessível por HTTPS num domínio público: os
                    provedores OAuth redirecionam o navegador de volta para{" "}
                    <code>https://&lt;your-domain&gt;/auth/callback</code>, e os
                    links dentro dos emails precisam abrir para quem os recebe.
                    Uma instância apenas em localhost ou em HTTP puro não pode
                    usar esses recursos. A forma mais simples de obter HTTPS é a
                    configuração do{" "}
                    <a
                      href="/pt/installation/#caddy-reverse-proxy"
                      className="text-blue-400 hover:text-blue-300"
                    >
                      reverse proxy Caddy
                    </a>
                    .
                  </p>
                </div>
              </div>

              <h3 id="oauth-google">Google</h3>

              <p>
                Crie um cliente OAuth no{" "}
                <a
                  href="https://console.cloud.google.com/apis/credentials"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Google Cloud Console
                </a>{" "}
                (APIs &amp; Services → Credentials → Create credentials → OAuth
                client ID, tipo de aplicação <em>Web application</em>) e
                adicione <code>https://&lt;your-domain&gt;/auth/callback</code>{" "}
                como URI de redirecionamento autorizada.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variável</th>
                    <th>Descrição</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_ID</code>
                    </td>
                    <td data-label="Descrição">
                      Client ID do seu cliente OAuth do Google. Defini-lo faz
                      aparecer o botão &quot;Sign in with Google&quot;.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Descrição">
                      Client secret do seu cliente OAuth do Google. Necessário
                      junto com o ID para o login funcionar.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h3 id="oauth-github">GitHub</h3>

              <p>
                Crie uma aplicação OAuth em{" "}
                <a
                  href="https://github.com/settings/developers"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  GitHub Developer settings
                </a>{" "}
                (Settings → Developer settings → OAuth Apps → New OAuth App) e
                defina a URL de callback de autorização como{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variável</th>
                    <th>Descrição</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_ID</code>
                    </td>
                    <td data-label="Descrição">
                      Client ID da sua aplicação OAuth do GitHub. Defini-lo faz
                      aparecer o botão &quot;Sign in with GitHub&quot;.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Descrição">
                      Client secret da sua aplicação OAuth do GitHub. Necessário
                      junto com o ID para o login funcionar.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="email-smtp">Email (SMTP)</h2>

              <p>
                Conecte um servidor de e-mail para que o Databasus possa enviar convites para espaços de trabalho, códigos de redefinição de senha e códigos de acesso. O servidor de e-mail é considerado configurado <strong>assim que <code>SMTP_HOST</code> é definido</strong>. Sem ele, a tela de configurações informa que o servidor de e-mail não está configurado, e a tela de login esconde o link de redefinição de senha.
              </p>

              <p>
                Este é o servidor de e-mail da instância. Os canais de notificação por e-mail, que avisam sobre os backups, têm configurações SMTP próprias no formulário de cada canal. Por isso, um canal de notificação que entrega e-mails não diz nada sobre este servidor. Para testar o servidor de e-mail da instância, abra <strong>Databasus settings → Mail server</strong> e clique em <strong>Send test email</strong>. A mensagem vai para o seu próprio endereço e, se a entrega falhar, você vê a resposta do servidor de e-mail.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variável</th>
                    <th>Descrição</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>SMTP_HOST</code>
                    </td>
                    <td data-label="Descrição">
                      Host do servidor de e-mail (por exemplo, <code>smtp.gmail.com</code>). Definir esta variável ativa o e-mail da instância.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PORT</code>
                    </td>
                    <td data-label="Descrição">
                      Porta do servidor de e-mail (por exemplo, <code>587</code>). Precisa ser um inteiro positivo quando <code>SMTP_HOST</code> está definido.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_SECURITY</code>
                    </td>
                    <td data-label="Descrição">
                      Segurança da conexão: <code>tls</code>, <code>starttls</code> ou <code>none</code>, descritos abaixo. O padrão é <code>tls</code> na porta 465 e <code>starttls</code> em qualquer outra porta. Com outro valor, a instância não inicia.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_USER</code>
                    </td>
                    <td data-label="Descrição">
                      Nome de usuário para a autenticação SMTP.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PASSWORD</code>
                    </td>
                    <td data-label="Descrição">
                      Senha para a autenticação SMTP. No Gmail, use uma App Password, não a senha da conta.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_FROM</code>
                    </td>
                    <td data-label="Descrição">
                      Endereço do remetente, só o endereço (<code>noreply@example.com</code>) ou com um nome (<code>Acme Backups &lt;noreply@example.com&gt;</code>). Sem nome, os destinatários veem <code>Databasus</code>. Quando não está definido, é usado <code>SMTP_USER</code> se ele for um endereço de e-mail; caso contrário, <code>noreply@</code> seguido de <code>SMTP_HOST</code>, então um usuário como <code>apikey</code> nunca vira o remetente. Se o valor não puder ser lido como endereço, a instância não inicia.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_HELO_NAME</code>
                    </td>
                    <td data-label="Descrição">
                      O nome com que a instância se apresenta ao servidor de e-mail. O padrão é o host de <code>DATABASUS_URL</code> e, na falta dele, o nome da máquina. Precisa ser um nome de host ou um literal de endereço como <code>[192.0.2.1]</code> ou <code>[IPv6:2001:db8::1]</code>; caso contrário, a instância não inicia.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_INSECURE_SKIP_VERIFY</code>
                    </td>
                    <td data-label="Descrição">
                      Defina como <code>true</code> para pular a verificação do certificado TLS nos modos <code>tls</code> e <code>starttls</code>. O padrão é <code>false</code>. Use apenas para servidores com certificado autoassinado numa rede confiável: desativa a proteção contra ataques man-in-the-middle.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>DATABASUS_URL</code>
                    </td>
                    <td data-label="Descrição">
                      URL base pública da sua instância (por exemplo, <code>https://backup.example.com</code>). Opcional: adiciona um link aos e-mails de convite e fornece o nome de saudação padrão. O e-mail funciona sem ela.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                <code>SMTP_SECURITY</code> escolhe um de três modos. Os canais de notificação por e-mail oferecem a mesma escolha em <strong>Advanced settings</strong>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Modo</th>
                    <th>Comportamento</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>tls</code>
                    </td>
                    <td data-label="Comportamento">
                      A conexão é criptografada desde o primeiro byte. Comum na porta 465.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>starttls</code>
                    </td>
                    <td data-label="Comportamento">
                      A conexão começa sem criptografia e passa a ser criptografada antes do envio da senha ou de qualquer mensagem. Se o servidor não oferecer esse recurso, a entrega falha em vez de continuar sem criptografia. Comum na porta 587.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>none</code>
                    </td>
                    <td data-label="Comportamento">
                      A conexão nunca é criptografada. Use para um relay que não oferece STARTTLS, como um Postfix local na porta 25.
                    </td>
                  </tr>
                </tbody>
              </table>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-red-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-red-500 mt-0.5 shrink-0"
                >
                  <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                  <path d="M12 9v4M12 17h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <code>none</code> envia a senha SMTP e todas as mensagens sem criptografia, e qualquer pessoa entre o Databasus e o relay consegue lê-las. Use apenas com um relay no mesmo host ou numa rede privada confiável.
                  </p>
                </div>
              </div>

              <h2 id="signup-captcha">
                Captcha no registro (Cloudflare Turnstile)
              </h2>

              <p>
                Se a sua instância está acessível pela internet pública, você
                pode colocar um desafio do{" "}
                <a
                  href="https://www.cloudflare.com/products/turnstile/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Cloudflare Turnstile
                </a>{" "}
                nos formulários de registro e de login para barrar bots. As duas
                chaves vêm do painel do Turnstile, e o desafio só é ativado
                quando ambas estão definidas.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    Para bloquear registros externos por completo, em vez de
                    apenas desafiá-los, você não precisa de captcha nenhum: abra{" "}
                    <strong>Databasus settings → Allow sign up</strong> na
                    interface e desligue a opção. Isso fecha completamente o
                    formulário de registro.
                  </p>
                </div>
              </div>

              <table>
                <thead>
                  <tr>
                    <th>Variável</th>
                    <th>Descrição</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SITE_KEY</code>
                    </td>
                    <td data-label="Descrição">
                      Chave pública do site no Turnstile, usada para renderizar
                      o widget no navegador.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SECRET_KEY</code>
                    </td>
                    <td data-label="Descrição">
                      Chave secreta do Turnstile, usada pelo backend para
                      validar as respostas ao desafio.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="docker-storage-permissions">Permissões do armazenamento Docker</h2>

              <p>
                O Databasus normalmente usa os IDs numéricos do usuário e do
                grupo do diretório pgdata existente, depois do mount de backups
                ou da raiz de dados e, por fim, usa <code>999</code>. Defina{" "}
                <code>PUID</code> ou <code>PGID</code> apenas quando essa escolha
                automática não servir para seu bind mount, compartilhamento CIFS
                ou exportação NFS. Os valores devem ser inteiros decimais de{" "}
                <code>1</code> a <code>4294967294</code>.
              </p>

              <table>
                <thead><tr><th>Variável</th><th>Valor automático</th><th>Conta</th></tr></thead>
                <tbody>
                  <tr><td><code>PUID</code></td><td>Proprietário do mount ou <code>999</code></td><td>Usuário <code>databasus</code></td></tr>
                  <tr><td><code>PGID</code></td><td>Grupo do mount ou <code>999</code></td><td>Grupo principal <code>databasus</code></td></tr>
                </tbody>
              </table>

              <p>
                Databasus e PostgreSQL usam a mesma conta do sistema
                operacional sem privilégios root chamada <code>databasus</code>.
              </p>

              <p>
                O entrypoint começa como root dentro do contêiner para escolher
                os IDs e tentar executar <code>chown</code> e <code>chmod</code>.
                Em seguida, ele verifica todo o ciclo de arquivos como{" "}
                <code>databasus</code>. Um mount pode funcionar mesmo que recuse
                mudanças de proprietário ou modo. Um valor arbitrário de Docker{" "}
                <code>user:</code> não é aceito.
              </p>

              <pre><code>{`ERROR: Databasus cannot write to local storage paths /databasus-data/temp and /databasus-data/backups as UID 999 and GID 999.
Required operation: save a file through local storage.
Set PUID and PGID or fix the mounted directory permissions: https://databasus.com/advanced-config/#docker-storage-permissions
Details: permission denied`}</code></pre>

              <p>
                As quatro variáveis anteriores específicas de cada serviço foram
                removidas de propósito. Remova-as da configuração ao atualizar.
                Se os IDs automáticos não conseguirem acessar um mount, a
                inicialização será interrompida e mostrará a operação que falhou
                e o link para esta documentação.
              </p>

              <h2 id="telemetry">Telemetria</h2>

              <p>
                O Databasus envia por padrão telemetria de uso anônima e não
                identificável. Ela não carrega dados pessoais e nos ajuda a
                entender como o projeto é usado. Você pode ler exatamente o que
                é coletado na{" "}
                <a
                  href="/privacy"
                  className="text-blue-400 hover:text-blue-300"
                >
                  política de privacidade
                </a>
                , e pode desativá-la por completo.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variável</th>
                    <th>Padrão</th>
                    <th>Descrição</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>IS_DISABLE_ANONYMOUS_TELEMETRY</code>
                    </td>
                    <td data-label="Padrão">
                      <code>false</code>
                    </td>
                    <td data-label="Descrição">
                      Defina como <code>true</code> para desativar a telemetria
                      de uso anônima.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="logging">Logs</h2>

              <p>
                O Databasus escreve os seus logs no stdout e os espelha como
                JSON em <code>databasus.log</code> no volume de dados. Defina{" "}
                <code>OPEN_TELEMETRY_URL</code> e ele também os exporta por
                OpenTelemetry para um backend como VictoriaLogs, Graylog,
                SigNoz, Grafana Loki, Datadog ou Honeycomb, ou para um
                OpenTelemetry Collector, que é ele próprio um receptor OTLP.
              </p>

              <ul>
                <li>
                  <strong>O transporte</strong> segue o esquema da URL.{" "}
                  <code>http://</code> e <code>https://</code> enviam OTLP/HTTP
                  e usam a URL tal como está, caminho incluído;{" "}
                  <code>grpc://</code> e <code>grpcs://</code> enviam OTLP/gRPC
                  e usam apenas o host e a porta.
                </li>
                <li>
                  <strong>A autenticação</strong> vai em{" "}
                  <code>OPEN_TELEMETRY_HEADERS</code> ou na própria URL como{" "}
                  <code>user:password@host</code>.
                </li>
                <li>
                  <strong>Segredos</strong> (senhas, tokens, credenciais) dentro
                  de URLs são censurados antes de uma entrada de log sair do
                  processo.
                </li>
                <li>
                  <strong>As entradas de auditoria</strong> seguem junto com os
                  logs da aplicação marcadas com <code>log_type=audit</code> e
                  ignoram <code>LOG_LEVEL</code>, então subir o nível nunca
                  descarta a trilha de auditoria.
                </li>
              </ul>

              <table>
                <thead>
                  <tr>
                    <th>Variável</th>
                    <th>Padrão</th>
                    <th>Descrição</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_URL</code>
                    </td>
                    <td data-label="Padrão">—</td>
                    <td data-label="Descrição">
                      URL completa do endpoint OTLP, incluindo o caminho. Deixe
                      sem definir para manter os logs no container. Uma query
                      string, um host ausente ou um esquema desconhecido impede
                      o container de iniciar, em vez de exportar para lugar
                      nenhum.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </td>
                    <td data-label="Padrão">—</td>
                    <td data-label="Descrição">
                      Pares <code>key=value</code> separados por vírgula,
                      enviados em cada exportação, normalmente uma chave de API.
                      Os valores são decodificados de percent-encoding, seguindo
                      o formato padrão <code>OTEL_EXPORTER_OTLP_HEADERS</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_LEVEL</code>
                    </td>
                    <td data-label="Padrão">
                      <code>info</code>
                    </td>
                    <td data-label="Descrição">
                      Um de <code>debug</code>, <code>info</code>,{" "}
                      <code>warn</code> ou <code>error</code>. Um valor não
                      reconhecido volta a <code>info</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_FILE_IS_ENABLED</code>
                    </td>
                    <td data-label="Padrão">
                      <code>true</code>
                    </td>
                    <td data-label="Descrição">
                      Escreve <code>databasus.log</code> junto com o resto dos
                      dados, com rotação a 5 MB e 3 versões antigas guardadas.
                      Defina como <code>false</code> se a sua plataforma já
                      coleta o stdout.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Valores para backends comuns, cada um com o cabeçalho que o
                autentica. Substitua hosts, regiões e chaves pelos seus:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Backend</th>
                    <th>
                      <code>OPEN_TELEMETRY_URL</code>
                    </th>
                    <th>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>VictoriaLogs</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>
                        http://victoria-logs:9428/insert/opentelemetry/v1/logs
                      </code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Basic%20dXNlcjpwYXNzd29yZA==</code> —
                      as credenciais que o seu <code>vmauth</code> ou reverse
                      proxy espera, já que o próprio VictoriaLogs não tem
                      autenticação no caminho de ingestão.
                    </td>
                  </tr>
                  <tr>
                    <td>OpenTelemetry Collector</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://otel-collector:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> —
                      corresponde a uma extensão <code>bearertokenauth</code> ou{" "}
                      <code>basicauth</code> no receptor. Um Collector acessível
                      apenas dentro da sua rede normalmente não precisa de
                      nenhum.
                    </td>
                  </tr>
                  <tr>
                    <td>Graylog 6.2+</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://graylog:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> — o token
                      definido no input OpenTelemetry (gRPC). O input também
                      aceita mTLS em alternativa.
                    </td>
                  </tr>
                  <tr>
                    <td>SigNoz Cloud</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpcs://ingest.eu.signoz.cloud:443</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>signoz-ingestion-key=your-ingestion-key</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Grafana Cloud</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>
                        https://otlp-gateway-prod-eu-west-0.grafana.net/otlp/v1/logs
                      </code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Basic%20&lt;base64&gt;</code> — base64
                      de <code>instance-id:api-token</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Honeycomb</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>https://api.honeycomb.io/v1/logs</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>x-honeycomb-team=your-api-key</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Datadog Agent</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://datadog-agent:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      Nenhum — o Agent guarda a chave de API e reenvia por conta
                      própria.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Os valores dos cabeçalhos são decodificados de percent-encoding,
                então o espaço depois de <code>Basic</code> ou{" "}
                <code>Bearer</code> se escreve como <code>%20</code> e uma
                vírgula dentro de um valor como <code>%2C</code>. A autenticação
                Basic também pode ir direto na URL como{" "}
                <code>https://user:password@host/path</code> — o Databasus a
                transforma no mesmo cabeçalho e a mantém fora dos logs. Por{" "}
                <code>http://</code> e <code>grpc://</code> as chaves e senhas
                viajam em claro, então use <code>https://</code> ou{" "}
                <code>grpcs://</code> fora de uma rede confiável.
              </p>

              <h2 id="analytics-script">Script de análise</h2>

              <p>
                O Databasus pode injetar na aplicação o seu próprio snippet de
                análise ou rastreamento: Google Analytics, Plausible, Umami e
                semelhantes. Quando <code>ANALYTICS_SCRIPT</code> está definido,
                o seu valor é inserido no <code>&lt;head&gt;</code> da página na
                inicialização.
              </p>

              <p>
                <strong>Aviso de segurança:</strong> o valor é injetado tal como
                está, como HTML e JavaScript puros, e executa com acesso total à
                interface do Databasus no navegador de cada visitante. Só o
                defina com um snippet que controla e no qual confia totalmente.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variável</th>
                    <th>Descrição</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>ANALYTICS_SCRIPT</code>
                    </td>
                    <td data-label="Descrição">
                      Marcação <code>&lt;script&gt;</code> própria injetada
                      antes da tag de fechamento <code>&lt;/head&gt;</code>.
                      Deixe sem definir para não adicionar nenhuma análise.
                    </td>
                  </tr>
                </tbody>
              </table>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
