<div align="center">
  <img src="../logo.svg" alt="Databasus Logo" width="250"/>

  <h3>Ferramenta de backup PostgreSQL</h3>
  <p>O Databasus é uma ferramenta gratuita, de código aberto e self-hosted para fazer backup de PostgreSQL. Crie backups em diferentes armazenamentos (S3, Google Drive, FTP, etc.) com notificações sobre o progresso (Slack, Discord, Telegram, etc.). Com foco em Point-in-Time Recovery com RPO/RTO baixos</p>
  
  <!-- Badges -->
   [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
  [![MySQL](https://img.shields.io/badge/MySQL-4479A1?logo=mysql&logoColor=white)](https://www.mysql.com/)
  [![MariaDB](https://img.shields.io/badge/MariaDB-003545?logo=mariadb&logoColor=white)](https://mariadb.org/)
  [![MongoDB](https://img.shields.io/badge/MongoDB-47A248?logo=mongodb&logoColor=white)](https://www.mongodb.com/)
  <br />
  [![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../../LICENSE)
  [![Docker Pulls](https://img.shields.io/docker/pulls/databasus/databasus?color=brightgreen)](https://hub.docker.com/r/databasus/databasus)
  [![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey)](https://github.com/databasus/databasus)
  [![Self Hosted](https://img.shields.io/badge/self--hosted-yes-brightgreen)](https://github.com/databasus/databasus)
  [![Open Source](https://img.shields.io/badge/open%20source-❤️-red)](https://github.com/databasus/databasus)

  <p>
    <a href="../../README.md">English</a> •
    <a href="README.ru.md">Русский</a> •
    <a href="README.es.md">Español</a> •
    <b>Português</b> •
    <a href="README.zh.md">中文</a> •
    <a href="README.fr.md">Français</a>
  </p>

  <p>
    <a href="#-recursos">Recursos</a> •
    <a href="#-instalação">Instalação</a> •
    <a href="#-como-usar">Como usar</a> •
    <a href="#-licença">Licença</a> •
    <a href="#-contribuindo">Contribuindo</a>
  </p>

  <p style="margin-top: 20px; margin-bottom: 20px; font-size: 1.2em;">
    <a href="https://databasus.com/pt/" target="_blank"><strong>🌐 Site do Databasus</strong></a>
  </p>
  
  <img src="../dashboard-dark.svg" alt="Databasus Dark Dashboard" width="800" style="margin-bottom: 10px;"/>

  <img src="../dashboard.svg" alt="Databasus Dashboard" width="800"/>
</div>

---

## ✨ Recursos

### 📦 **Tipos de backup**

- **Físico**: cópia em nível de arquivos de todo o cluster da base de dados, feita sobre o mecanismo nativo de backups incrementais do PostgreSQL (leia mais)
  - **Completo**: uma cópia integral e autossuficiente do cluster
  - **Incremental**: guarda apenas o que mudou desde o backup completo anterior, então os backups continuam pequenos e rápidos
  - **Streaming de WAL**: captura continuamente o fluxo de escrita da base, o que permite Point-in-time recovery (PITR). Feito para recuperação de desastres e perda de dados próxima de zero
- **Lógico**: dump nativo da base no formato binário do próprio motor (comprimido, adequado para restauração em paralelo)

### 🔄 **Backups agendados**

- **Agendamento flexível**: de hora em hora, diário, semanal, mensal ou cron
- **Horário exato**: rode os backups no horário que quiser (por exemplo, às 4h da manhã, quando o tráfego é menor)
- **Compressão inteligente**: arquivos de 4-8x menores com compressão balanceada (cerca de ~20% de tempo a mais)

### 🧪 **Verificação de restauração** <a href="https://databasus.com/pt/restore-verification/">(docs)</a>

O Databasus faz uma restauração de verdade para confirmar que os backups servem, em vez de apenas checar se o arquivo está inteiro no disco ou conferir o checksum.

- **Gatilhos**: após cada backup ou em um agendamento flexível (de hora em hora, diário, semanal, mensal ou cron)
- **Restauração real**: sobe um contêiner de base de dados, executa a restauração e compara o tamanho restaurado com o do backup
- **Relatório**: lista cada tabela com a contagem de linhas
- **Notificações opcionais**: envie o relatório ou apenas os alertas de falha por qualquer notificador configurado

### 🗑️ **Políticas de retenção**

- **Período de tempo**: mantenha os backups por um prazo fixo (por exemplo, 7 dias, 3 meses, 1 ano)
- **Quantidade**: mantenha um número fixo dos backups mais recentes (por exemplo, os últimos 30)
- **GFS (Grandfather-Father-Son)**: retenção em camadas. Guarde backups de hora em hora, diários, semanais, mensais e anuais de forma independente, para um histórico de longo prazo detalhado (requisito de empresas)
- **Limites de tamanho**: defina um teto por backup e um teto total para controlar o uso do armazenamento

### 🗄️ **Vários destinos de armazenamento** <a href="https://databasus.com/pt/storages/">(ver suportados)</a>

- **Armazenamento local**: mantenha os backups no seu VPS ou servidor
- **Armazenamento na nuvem**: S3, Cloudflare R2, Google Drive, NAS, Dropbox, SFTP, Rclone e outros
- **Seguro**: todos os dados continuam sob o seu controle

### 📱 **Notificações** <a href="https://databasus.com/pt/notifiers/">(ver suportados)</a>

- **Vários canais**: e-mail, Telegram, Slack, Discord, Teams, Mattermost, webhooks
- **Em tempo real**: notificações de sucesso e de falha
- **Integração com a equipe**: feito para o dia a dia de DevOps

### 🔒 **Segurança de nível empresarial** <a href="https://databasus.com/pt/security/">(docs)</a>

- **Criptografia AES-256-GCM**: proteção de nível empresarial para os arquivos de backup
- **Armazenamento zero-trust**: os backups são criptografados e continuam inúteis para um atacante, então você pode guardá-los com segurança em armazenamentos compartilhados como S3, Azure Blob Storage, etc.
- **Criptografia de segredos**: qualquer dado sensível é criptografado e nunca fica exposto, nem em logs nem em mensagens de erro
- **Usuário somente leitura**: por padrão, o Databasus usa um usuário somente leitura para os backups e nunca guarda nada capaz de alterar os seus dados
- **Autenticação de dois fatores**: o login por senha pode exigir um código de seis dígitos enviado ao e-mail da conta. Entrar pelo Google ou pelo GitHub continua contando com as verificações desses provedores

### 👥 **Adequado para equipes** <a href="https://databasus.com/pt/access-management/">(docs)</a>

- **Espaços de trabalho**: agrupe bases de dados, notificadores e armazenamentos por projeto ou por equipe
- **Gestão de acessos**: controle quem pode ver ou gerenciar cada base de dados, com permissões por papel
- **Logs de auditoria**: acompanhe todas as atividades do sistema e as alterações feitas pelos usuários
- **Papéis de usuário**: atribua os papéis de viewer, member, admin ou owner dentro dos espaços de trabalho
- **Logs em OpenTelemetry**: exporte os logs da aplicação e de auditoria para um sistema externo (por padrão eles também são gravados em um arquivo local)

### 🎨 **Interface agradável**

- **UI cuidada por designer**: interface limpa e intuitiva, feita com atenção aos detalhes
- **Tema claro e escuro**: escolha o visual que combina com o seu trabalho
- **Adaptada para celular**: confira os seus backups de onde estiver, em qualquer dispositivo

### 💾 **Bases de dados suportadas**

- **PostgreSQL**: 14, 15, 16, 17 e 18 (físico e lógico)
- **MySQL**: 5.7, 8.0, 8.4, 9 e 26 (apenas lógico)
- **MariaDB**: 5.5, 10, 11, 12 e 13 (apenas lógico)
- **MongoDB**: 4.2+, 5, 6, 7, 8 e 9 (apenas lógico)

### 🐳 **Self-hosted e seguro**

- **Baseado em Docker**: fácil de implantar e de manter
- **Privacidade em primeiro lugar**: todos os seus dados ficam na sua infraestrutura
- **Código aberto**: licença Apache 2.0, você pode inspecionar cada linha do código
- **SSH integrado**: conecte-se ao seu Databasus por um túnel SSH

### 📦 Instalação <a href="https://databasus.com/pt/installation/">(docs)</a>

Você tem quatro formas de instalar o Databasus:

- script automatizado (recomendado)
- um `docker run` simples
- Docker Compose
- Kubernetes com Helm

<img src="../healthchecks.svg" alt="Databasus Dashboard" width="800"/>

---

## 📦 Instalação

Você tem quatro formas de instalar o Databasus: script automatizado (recomendado), um `docker run` simples ou Docker Compose.

### Opção 1: script de instalação automatizada (recomendado, só Linux)

O script de instalação vai:

- ✅ instalar o Docker com Docker Compose (se ainda não estiverem instalados)
- ✅ configurar o Databasus
- ✅ deixar a inicialização automática pronta para quando o sistema reiniciar

```bash
sudo apt-get install -y curl && \
sudo curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh \
| sudo bash
```

### Opção 2: um `docker run` simples

A forma mais fácil de rodar o Databasus:

```bash
docker run -d \
  --name databasus \
  -p 4005:4005 \
  -v ./databasus-data:/databasus-data \
  --restart unless-stopped \
  databasus/databasus:latest
```

_A mesma imagem está no registry do GitHub: use `ghcr.io/databasus/databasus:latest` se o Docker Hub limitar o seu download._

Esse único comando vai:

- ✅ iniciar o Databasus
- ✅ guardar todos os dados no diretório `./databasus-data`
- ✅ reiniciar automaticamente quando o sistema for reiniciado

### Opção 3: Docker Compose

Crie um arquivo `docker-compose.yml` com esta configuração:

```yaml
services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "databasus", "healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 60s
```

Depois rode:

```bash
docker compose up -d
```

### Opção 4: Kubernetes com Helm

Para implantações em Kubernetes, instale direto do registry OCI.

_Adicione `--set image.repository=ghcr.io/databasus/databasus` a qualquer um dos comandos abaixo para baixar a imagem do GHCR em vez do Docker Hub._

**Com ClusterIP + port-forward (desenvolvimento e testes):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace
```

```bash
kubectl port-forward svc/databasus-service 4005:4005 -n databasus
# Access at http://localhost:4005
```

**Com LoadBalancer (ambientes na nuvem):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set service.type=LoadBalancer
```

```bash
kubectl get svc databasus-service -n databasus
# Access at http://<EXTERNAL-IP>:4005
```

**Com Ingress (acesso por domínio):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=backup.example.com
```

Para mais opções (NodePort, TLS, HTTPRoute do Gateway API), veja o [README do chart Helm](../../deploy/helm/README.md).

---

## 🚀 Como usar

1. **Crie a primeira conta**: acesse `http://localhost:4005` e cadastre-se. A primeira conta criada em uma instância é quem a administra
2. **Adicione a sua primeira base de dados para backup**: clique em "New Database" e siga o assistente de configuração
3. **Configure o agendamento**: escolha entre intervalos de hora em hora, diário, semanal, mensal ou cron
4. **Informe a conexão com a base**: preencha as credenciais e os dados de conexão da sua base de dados
5. **Escolha o armazenamento**: selecione onde guardar os backups (local, S3, Google Drive, etc.)
6. **Configure a política de retenção**: escolha período de tempo, quantidade ou GFS para definir por quanto tempo os backups ficam guardados
7. **Adicione notificações** (opcional): configure notificações por e-mail, Telegram, Slack, Mattermost ou webhook
8. **Salve e inicie**: o Databasus valida as configurações e começa a seguir o agendamento dos backups

### 🔑 Redefinir a senha <a href="https://databasus.com/pt/password/">(docs)</a>

Se precisar redefinir a senha, use o comando de redefinição já incluído:

```bash
docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com"
```

Troque `owner@example.com` pelo e-mail da conta cuja senha você quer redefinir.

Se você não lembra qual endereço administra a instância, liste as contas de administrador:

```bash
docker exec -it databasus ./main --list-admins
```

A saída nomeia cada conta de administrador com seu e-mail, nome de exibição, data de criação e estado de atividade, e marca aquela que a instância reconhece como seu administrador. Nenhuma senha ou hash de senha é impresso.

Para criar uma conta sem passar pelo cadastro no navegador, por exemplo a partir de um script de deploy, use o console:

```bash
docker exec -it databasus ./main --create-user --email="ops@example.com" --password="YourSecurePassword123" --role=admin
```

`--role` aceita `admin` ou `member`, a senha precisa ter pelo menos 8 caracteres e `--name` define, se quiser, o nome de exibição. Em uma instância que ainda não tem nenhuma conta, a primeira precisa usar `--role=admin` e passa a administrar a instância. Rodar o comando de novo para a mesma conta não altera nada, então um deploy pode executá-lo a cada rollout; para trocar a senha, use `--new-password`.

### 💾 Fazer backup do próprio Databasus

Depois de instalar, também é recomendável <a href="https://databasus.com/pt/faq/#backup-databasus">fazer backup do próprio Databasus</a> ou, no mínimo, copiar a chave secreta usada na criptografia (leva 30 segundos). Assim você consegue restaurar a partir dos seus backups criptografados se perder o acesso ao servidor com o Databasus ou se ele for corrompido.

---

## 🛡️ Engenharia de segurança e confiabilidade

O Databasus lida com dados sensíveis, então evitar vulnerabilidades, acessos não autorizados e vazamentos é a preocupação central. Investimos nisso dos dois lados do sistema: no próprio código (verificação de permissões, criptografia, cuidado no tratamento de segredos) e na infraestrutura em volta dele (análise de dependências, resposta a CVEs, boas práticas de DevSecOps). O pipeline descrito abaixo roda automaticamente a cada commit e a cada PR. Nenhuma camada basta sozinha, mas juntas elas reduzem a chance de que código vulnerável, dependências inseguras, imagens quebradas ou backups impossíveis de restaurar cheguem a uma release.

Na análise estática combinamos várias passagens independentes. O CodeQL varre todo o código em busca de problemas de segurança. Em cada PR, o gitleaks procura segredos vazados e o semgrep verifica o código com regras de segurança. Dockerfiles e workflows de CI ganham regras próprias (referências de action fixadas, permissões de privilégio mínimo, imagens base suspeitas), então padrões inseguros aparecem antes do merge. Além dessas checagens por PR, o Codex Security da OpenAI faz auditorias periódicas e mais profundas de toda a base de código. É um programa à parte, que pega problemas arquiteturais e transversais que uma varredura restrita ao PR deixa passar.

Do lado das dependências, o Dependabot acompanha todas as nossas dependências no GitHub Advisory Database e mostra CVEs poucos minutos depois da publicação. As atualizações passam por um período de espera, para que versões recém-lançadas amadureçam antes de serem adotadas. Essa é uma defesa deliberada contra incidentes com pacotes comprometidos, como os ataques à cadeia de suprimentos. O Dependency Review Action bloqueia de imediato qualquer PR que introduza um CVE novo de severidade HIGH ou CRITICAL. No código Go, o govulncheck reprova um PR quando o nosso código chama uma função com vulnerabilidade conhecida, e o `go mod verify` confere cada módulo baixado com o checksum registrado.

As imagens de contêiner passam pelo Trivy a cada build. Uma passagem separada do Trivy sobre o Dockerfile pega configurações erradas antes que elas entrem em uma imagem. Todas as GitHub Actions são fixadas em SHAs completos de commit, e não em tags móveis como `@v4` ou `@main`, que foram um vetor de ataque ativo em 2025. Os workflows usam permissões de privilégio mínimo por padrão e só elevam por job quando é realmente necessário.

Os caminhos críticos têm testes unitários e de integração, executados contra contêineres reais de base de dados para cada motor e versão principal suportados. A restauração é o caminho que mais importa em uma ferramenta de backup, então testamos isso explicitamente: cada PR roda ciclos completos de backup e restauração contra esses mesmos contêineres reais, comprovando que os backups podem mesmo ser restaurados de ponta a ponta, e não apenas gravados sem erro. O resto do pipeline de CI/CD roda lint, verificação de tipos, a suíte completa de testes, smoke tests de imagem e builds multiarquitetura em cada PR. Uma release só sai se tudo isso passar.

Encontrou uma vulnerabilidade? Reporte pela aba Security do GitHub. Veja o [SECURITY.md](https://github.com/databasus/databasus?tab=security-ov-file#readme). Relatos de segurança são a fila de trabalho de maior prioridade. Sobre a segurança da aplicação em execução (AES-256-GCM em repouso, armazenamento zero-trust, segredos criptografados, usuário de base de dados somente leitura por padrão), veja [Segurança de nível empresarial](#-segurança-de-nível-empresarial-docs) na seção de recursos acima.

---

## 📝 Licença

Este projeto está sob a licença Apache 2.0 - veja o arquivo [LICENSE](../../LICENSE) para mais detalhes

## 🤝 Contribuindo

Contribuições são bem-vindas! Leia o <a href="https://databasus.com/contribute">guia de contribuição</a> para conhecer os detalhes, as prioridades e as regras. Se você quer contribuir mas não sabe por onde começar, me escreva no Telegram [@rostislav_dugin](https://t.me/rostislav_dugin)

Você também pode entrar na nossa grande comunidade de desenvolvedores, DBAs e engenheiros de DevOps no Telegram [@databasus_community](https://t.me/databasus_community).
