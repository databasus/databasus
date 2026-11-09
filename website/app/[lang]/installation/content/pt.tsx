import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Instalação - Documentação do Databasus",
  description:
    "Aprenda a instalar o Databasus com script automatizado, Docker run, Docker Compose, Helm para Kubernetes ou reverse proxy Caddy. Instalação simples, sem configuração, do seu sistema self-hosted de backup PostgreSQL.",
  keywords: [
    "instalação do Databasus",
    "instalação com Docker",
    "configuração de backup PostgreSQL",
    "backup self-hosted",
    "Docker Compose",
    "instalação de backup de base de dados",
    "configuração do pg_dump",
    "Kubernetes",
    "Helm chart",
    "deploy em K8s",
    "reverse proxy Caddy",
    "configuração de HTTPS",
    "health check",
    "monitoramento",
    "liveness probe",
  ],
  openGraph: {
    title: "Instalação - Documentação do Databasus",
    description:
      "Aprenda a instalar o Databasus com script automatizado, Docker run, Docker Compose, Helm para Kubernetes ou reverse proxy Caddy. Instalação simples, sem configuração, do seu sistema self-hosted de backup PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("pt", "installation"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title: "Instalação - Documentação do Databasus",
    description:
      "Aprenda a instalar o Databasus com script automatizado, Docker run, Docker Compose, Helm para Kubernetes ou reverse proxy Caddy. Instalação simples, sem configuração, do seu sistema self-hosted de backup PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "installation"),
    languages: getLanguageAlternates("installation"),
  },
  robots: "index, follow",
};

export default function InstallationPage() {
  const installScript = `sudo apt-get install -y curl && \\
sudo curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh | sudo bash`;

  const dockerRun = `docker run -d \\
  --name databasus \\
  -p 4005:4005 \\
  -v ./databasus-data:/databasus-data \\
  --restart unless-stopped \\
  databasus/databasus:latest`;

  const dockerCompose = `services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped`;

  const helmInstallClusterIP = `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace`;

  const helmPortForward = `kubectl port-forward svc/databasus-service 4005:4005 -n databasus
# Access at http://localhost:4005`;

  const helmInstallLoadBalancer = `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace \\
  --set service.type=LoadBalancer`;

  const helmGetSvc = `kubectl get svc databasus-service -n databasus
# Access at http://<EXTERNAL-IP>:4005`;

  const helmInstallIngress = `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace \\
  --set ingress.enabled=true \\
  --set ingress.hosts[0].host=backup.example.com`;

  const helmUpgrade = `helm upgrade databasus oci://ghcr.io/databasus/charts/databasus -n databasus`;

  const dockerComposeCaddy = `services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped
    # No port exposed - Caddy handles external access

  caddy:
    container_name: caddy
    image: caddy:latest
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile
      - ./caddy-data:/data
      - ./caddy-config:/config
    restart: unless-stopped
    depends_on:
      - databasus`;

  const caddyfile = `backup.example.com {
    reverse_proxy databasus:4005
}`;

  const healthEndpoint = `GET http://<host>:4005/api/v1/system/health`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Instalação - Documentação do Databasus",
            description:
              "Aprenda a instalar o Databasus com script automatizado, Docker run, Docker Compose, Helm para Kubernetes ou reverse proxy Caddy. Instalação simples, sem configuração, do seu sistema self-hosted de backup PostgreSQL.",
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
            name: "Como instalar o Databasus",
            description:
              "Guia passo a passo para instalar o Databasus, ferramenta de backup PostgreSQL",
            step: [
              {
                "@type": "HowToStep",
                name: "Script de instalação automatizado",
                text: "Execute o script de instalação automatizado para instalar o Docker e configurar o Databasus com inicialização automática.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Execute o comando curl para baixar e executar o script de instalação",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Docker Run",
                text: "Use o comando docker run para iniciar rapidamente o container do Databasus com persistência de dados.",
              },
              {
                "@type": "HowToStep",
                name: "Docker Compose",
                text: "Crie um arquivo docker-compose.yml e use o Docker Compose para uma implantação gerenciada.",
              },
              {
                "@type": "HowToStep",
                name: "Kubernetes com Helm",
                text: "Use o Helm chart oficial para implantar o Databasus no Kubernetes com StatefulSet, armazenamento persistente e ingress opcional.",
              },
              {
                "@type": "HowToStep",
                name: "Execução com reverse proxy Caddy",
                text: "Use o Docker Compose com Caddy para implantações em produção com certificados HTTPS automáticos.",
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
              <h1 id="installation">Instalação</h1>

              <p className="text-lg text-gray-400">
                Há várias formas de instalar o Databasus: script automatizado
                (recomendado), docker run simples, Docker Compose, Helm para
                Kubernetes ou Docker Compose com Caddy para produção.
              </p>

              <h2 id="system-requirements">Requisitos de sistema</h2>

              <p>
                O Databasus precisa dos seguintes recursos mínimos para
                funcionar corretamente:
              </p>

              <ul>
                <li>
                  <strong>CPU</strong>: pelo menos 1 núcleo de CPU
                </li>
                <li>
                  <strong>RAM</strong>: mínimo de 500 MB de RAM
                </li>
                <li>
                  <strong>Armazenamento</strong>: 5 GB para a instalação, mais o
                  espaço de que precisar para os backups
                </li>
                <li>
                  <strong>Docker</strong>: Docker Engine 20.10+ e Docker Compose
                  v2.0+
                </li>
              </ul>

              <h2 id="option-1-automated-script">
                Opção 1: script de instalação (recomendado, apenas Linux)
              </h2>

              <p>O script de instalação vai:</p>

              <ul>
                <li>
                  ✅ Instalar o Docker com Docker Compose (se ainda não
                  estiverem instalados)
                </li>
                <li>✅ Configurar o Databasus</li>
                <li>
                  ✅ Ativar a inicialização automática após reiniciar o sistema
                </li>
              </ul>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{installScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={installScript} />
                </div>
              </div>

              <p>
                Neste caso, o Databasus é instalado no diretório{" "}
                <code>/opt/databasus</code>.
              </p>

              <h2 id="option-2-docker-run">Opção 2: docker run simples</h2>

              <p>A forma mais fácil de executar o Databasus:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRun}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={dockerRun} />
                </div>
              </div>

              <p>Este único comando vai:</p>

              <ul>
                <li>✅ Iniciar o Databasus</li>
                <li>
                  ✅ Salvar todos os dados no diretório{" "}
                  <code>./databasus-data</code>
                </li>
                <li>✅ Reiniciar automaticamente após reiniciar o sistema</li>
              </ul>

              <h2 id="option-3-docker-compose">
                Opção 3: configuração com Docker Compose
              </h2>

              <p>
                Crie um arquivo <code>docker-compose.yml</code> com a seguinte
                configuração:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerCompose}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={dockerCompose} />
                </div>
              </div>

              <p>Depois execute:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text="docker compose up -d" />
                </div>
              </div>

              <p>Observe que a inicialização pode levar até ~2 minutos.</p>

              <h2 id="option-4-helm">Opção 4: Kubernetes com Helm</h2>

              <p>
                Para implantações em Kubernetes, instale diretamente a partir do
                registro OCI. Escolha o método de acesso adequado ao seu
                ambiente.
              </p>

              <h3 id="helm-clusterip">
                Com ClusterIP + port-forward (desenvolvimento)
              </h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallClusterIP}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={helmInstallClusterIP} />
                </div>
              </div>

              <p>Acesso via port-forward:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmPortForward}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={helmPortForward} />
                </div>
              </div>

              <h3 id="helm-loadbalancer">
                Com LoadBalancer (ambientes de nuvem)
              </h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallLoadBalancer}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={helmInstallLoadBalancer} />
                </div>
              </div>

              <p>Obtenha o IP externo e acesse o Databasus:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmGetSvc}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={helmGetSvc} />
                </div>
              </div>

              <h3 id="helm-ingress">Com Ingress (acesso por domínio)</h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallIngress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={helmInstallIngress} />
                </div>
              </div>

              <p>
                Para mais opções (NodePort, TLS, HTTPRoute para Gateway API),
                consulte a{" "}
                <a
                  href="https://github.com/databasus/databasus/tree/main/deploy/helm"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  documentação do Helm chart
                </a>
                .
              </p>

              <h2 id="caddy-reverse-proxy">Execução com reverse proxy Caddy</h2>

              <p>
                Para produção, você pode usar o{" "}
                <a
                  href="https://caddyserver.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Caddy
                </a>{" "}
                como reverse proxy para ter certificados HTTPS automáticos e
                acesso seguro ao Databasus. Abaixo está uma configuração
                completa de Docker Compose com Caddy.
              </p>

              <h3 id="caddy-docker-compose">Docker Compose com Caddy</h3>

              <p>
                Crie um arquivo <code>docker-compose.yml</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeCaddy}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={dockerComposeCaddy} />
                </div>
              </div>

              <p>
                Crie um <code>Caddyfile</code> no mesmo diretório:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{caddyfile}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={caddyfile} />
                </div>
              </div>

              <p>Depois inicie os serviços:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text="docker compose up -d" />
                </div>
              </div>

              <p>Esta configuração oferece:</p>

              <ul>
                <li>✅ HTTPS automático com certificados Let&apos;s Encrypt</li>
                <li>✅ Redirecionamento de HTTP para HTTPS</li>
                <li>✅ Reverse proxy para o Databasus</li>
                <li>
                  ✅ Dados persistentes tanto do Caddy quanto do Databasus
                </li>
              </ul>

              <p>
                Substitua <code>backup.example.com</code> pelo seu domínio real.
                Confirme que o DNS do domínio aponta para o endereço IP do seu
                servidor antes de iniciar os serviços.
              </p>

              <h2 id="getting-started">Primeiros passos</h2>

              <p>Depois da instalação:</p>

              <ol>
                <li>
                  <strong>Inicie e acesse o Databasus</strong>: suba o Databasus
                  e abra <code>http://localhost:4005</code>
                </li>
                <li>
                  <strong>Crie a primeira tarefa de backup</strong>: clique em
                  &quot;New Backup&quot; e configure a conexão com a sua base de
                  dados PostgreSQL
                </li>
                <li>
                  <strong>Configure o agendamento</strong>: defina a frequência
                  dos backups (por hora, dia, semana, mês ou cron)
                </li>
                <li>
                  <strong>Escolha o destino de armazenamento</strong>: indique
                  onde salvar os backups (disco local, S3, Azure Blob Storage, etc.)
                </li>
                <li>
                  <strong>Configure as notificações</strong>: adicione canais
                  (Slack, Telegram, Discord) para receber alertas sobre o estado
                  dos backups
                </li>
                <li>
                  <strong>Comece a fazer backup</strong>: salve a configuração e
                  acompanhe a execução do primeiro backup!
                </li>
              </ol>

              <h2 id="health-checks">Health checks</h2>

              <h3 id="docker-health-check">Health check no Docker</h3>

              <p>
                Um health check embutido é ativado automaticamente para{" "}
                <code>docker run</code> e Docker Compose. O container é marcado
                como <code>healthy</code> assim que o Databasus começa a
                responder a requisições (após um curto período de
                inicialização). Ele verifica apenas se a aplicação responde, por
                isso o container não é reiniciado por condições não críticas,
                como pouco espaço em disco.
              </p>

              <h3 id="monitoring-endpoint">
                Endpoint de monitoramento / status
              </h3>

              <p>Para monitoramento de uptime e painéis de status:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{healthEndpoint}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={healthEndpoint} />
                </div>
              </div>

              <p>
                Retorna <code>200</code> quando está tudo bem, ou{" "}
                <code>503</code> com o motivo quando algo precisa de atenção:
                base de dados interna, uso de disco (acima de 95%), ferramentas
                cliente de base de dados, tarefas em segundo plano e atividade
                do agente de verificação. Sem autenticação, e o CORS
                está aberto para monitores baseados em navegador.
              </p>

              <p>
                <strong>⚠️ Importante:</strong> serve apenas para monitoramento
                e alertas, não como liveness probe de container ou Kubernetes
                &mdash; retorna <code>503</code> em estados degradados mas
                funcionais (por exemplo, um disco quase cheio), o que
                reiniciaria um container que está funcionando.
              </p>

              <h3 id="kubernetes-health-check">Kubernetes</h3>

              <p>
                Use uma liveness/readiness probe que execute{" "}
                <code>databasus healthcheck</code>; reserve o endpoint{" "}
                <a
                  href="#monitoring-endpoint"
                  className="text-blue-400 hover:text-blue-300"
                >
                  /api/v1/system/health
                </a>{" "}
                para monitoramento externo.
              </p>

              <h2 id="how-to-update">Como atualizar o Databasus?</h2>

              <h3 id="update-docker">Atualizar a instalação em Docker</h3>

              <p>
                Para atualizar o Databasus em execução via Docker, é preciso
                pará-lo, limpar o cache do Docker e reiniciar o container.
              </p>

              <ol>
                <li>
                  Vá ao diretório onde o Databasus está instalado (normalmente{" "}
                  <code>/opt/databasus</code>)
                </li>
                <li>
                  Pare o container: <code>docker compose stop</code>
                </li>
                <li>
                  Limpe o cache do Docker: <code>docker system prune -a</code>
                </li>
                <li>
                  Reinicie o container: <code>docker compose up -d</code>
                </li>
              </ol>

              <p>
                Isso baixa a versão mais recente do Databasus do Docker Hub (se
                você não tiver fixado a versão no arquivo{" "}
                <code>docker-compose.yml</code>).
              </p>

              <h3 id="update-helm">Atualizar a instalação via Helm</h3>

              <p>
                Para atualizar o Databasus em Kubernetes via Helm, use o comando
                upgrade:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmUpgrade}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text={helmUpgrade} />
                </div>
              </div>

              <p>
                Se tiver valores personalizados, adicione{" "}
                <code>-f values.yaml</code> ou use flags <code>--set</code> para
                preservar a sua configuração. O Helm faz uma atualização gradual
                (rolling update) para a nova versão.
              </p>

              <h2 id="postgresus-migration">Migrar do Postgresus</h2>

              <p>
                Databasus é o novo nome do Postgresus. Se você usa o Postgresus
                atualmente, pode continuar com ele ou migrar para o Databasus.
              </p>

              <p>
                <strong>Importante:</strong> apenas renomear a imagem Docker não
                basta, porque o Postgresus e o Databasus usam pastas de dados e
                nomes de base de dados interna diferentes.
              </p>

              <p>Para migrar:</p>

              <ol>
                <li>
                  Pare o container do Postgresus:{" "}
                  <code>docker compose stop</code>
                </li>
                <li>
                  Instale o Databasus por qualquer um dos métodos acima (use um
                  caminho de volume diferente, <code>./databasus-data</code>)
                </li>
                <li>
                  Recrie manualmente as suas bases de dados, armazenamentos e
                  notificadores no Databasus
                </li>
              </ol>

              <p>
                Durante a migração, o Postgresus e o Databasus podem funcionar
                lado a lado, com portas e caminhos de volume diferentes.
              </p>

              <h2 id="troubleshooting">Resolução de problemas</h2>

              <h3 id="container-wont-start">O container não inicia</h3>

              <p>Se o container não iniciar, verifique os logs:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker logs databasus</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="pt" text="docker logs databasus" />
                </div>
              </div>

              <h3 id="port-already-in-use">Porta já em uso</h3>

              <p>
                Se a porta 4005 já estiver em uso, você pode mudá-la no
                docker-compose.yml:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>
                    ports:
                    {"\n  "}- &quot;8080:4005&quot; # Change 8080 to any
                    available port
                  </code>
                </pre>
              </div>

              <h3 id="permission-denied">Erros de permissão</h3>

              <p>
                Se você encontrar problemas de permissões no diretório de dados:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>
                    sudo chown -R $USER:$USER ./databasus-data
                    {"\n"}
                    chmod -R 755 ./databasus-data
                  </code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton
                    lang="pt"
                    text={`sudo chown -R $USER:$USER ./databasus-data\nchmod -R 755 ./databasus-data`}
                  />
                </div>
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
