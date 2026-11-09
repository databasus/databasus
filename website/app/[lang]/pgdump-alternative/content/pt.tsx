import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Alternativa ao pg_dump - Databasus, ferramenta de backup PostgreSQL",
  description:
    "O Databasus é construído sobre o pg_dump e amplia suas funcionalidades com gestão de backups, interface web, agendamento automático, armazenamento na nuvem, notificações, colaboração e criptografia.",
  keywords: [
    "alternativa ao pg_dump",
    "pg_dump GUI",
    "automatizar pg_dump",
    "interface web para pg_dump",
    "ferramenta de backup PostgreSQL",
    "agendador pg_dump",
    "pg_dump armazenamento na nuvem",
    "pg_dump criptografia",
    "automatizar backup PostgreSQL",
    "wrapper do pg_dump",
  ],
  openGraph: {
    title:
      "Alternativa ao pg_dump - Databasus, ferramenta de backup PostgreSQL",
    description:
      "O Databasus é construído sobre o pg_dump e amplia suas funcionalidades com gestão de backups, interface web, agendamento automático, armazenamento na nuvem, notificações, colaboração e criptografia.",
    type: "article",
    url: getLocalizedUrl("pt", "pgdump-alternative"),
    locale: OG_LOCALES.pt,
  },
  twitter: {
    card: "summary",
    title:
      "Alternativa ao pg_dump - Databasus, ferramenta de backup PostgreSQL",
    description:
      "O Databasus é construído sobre o pg_dump e amplia suas funcionalidades com gestão de backups, interface web, agendamento automático, armazenamento na nuvem, notificações, colaboração e criptografia.",
  },
  alternates: {
    canonical: getLocalizedUrl("pt", "pgdump-alternative"),
    languages: getLanguageAlternates("pgdump-alternative"),
  },
  robots: "index, follow",
};

export default function PgDumpAlternativePage() {
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
              "Alternativa ao pg_dump - Databasus, ferramenta de backup PostgreSQL",
            description:
              "Guia completo do Databasus como alternativa ao pg_dump: como ele se apoia no pg_dump e amplia suas capacidades com automação, armazenamento na nuvem, notificações e recursos de colaboração.",
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
              <h1 id="pgdump-alternative">Alternativa ao pg_dump</h1>

              <p className="text-lg text-gray-400">
                Para backups lógicos, o Databasus é construído sobre o{" "}
                <code>pg_dump</code>. Em vez de substituir o{" "}
                <code>pg_dump</code>, o Databasus amplia suas capacidades com
                gestão de backups, interface web, agendamento automático,
                integração com armazenamento na nuvem, notificações, recursos de
                colaboração e criptografia integrada. Além dos backups lógicos,
                o Databasus também suporta backups físicos, backups incrementais
                com arquivamento de WAL e Point-in-Time Recovery.
              </p>

              <h2 id="quick-comparison">Comparação rápida</h2>

              <p>
                Veja como o Databasus amplia a funcionalidade base do{" "}
                <code>pg_dump</code>:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Funcionalidade</th>
                    <th>pg_dump</th>
                    <th>Databasus</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Motor de backup</td>
                    <td data-label="pg_dump">pg_dump</td>
                    <td data-label="Databasus">Construído sobre o pg_dump</td>
                  </tr>
                  <tr>
                    <td>Gestão de backups</td>
                    <td data-label="pg_dump">❌ Não</td>
                    <td data-label="Databasus">✅ Sim</td>
                  </tr>
                  <tr>
                    <td>Suporte a outras bases de dados</td>
                    <td data-label="pg_dump">Apenas PostgreSQL</td>
                    <td data-label="Databasus">
                      PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                  </tr>
                  <tr>
                    <td>Interface</td>
                    <td data-label="pg_dump">Linha de comando</td>
                    <td data-label="Databasus">Interface web + API</td>
                  </tr>
                  <tr>
                    <td>Agendamento</td>
                    <td data-label="pg_dump">Manual ou scripts de cron</td>
                    <td data-label="Databasus">✅ Agendador integrado</td>
                  </tr>
                  <tr>
                    <td>Destinos de armazenamento</td>
                    <td data-label="pg_dump">Sistema local</td>
                    <td data-label="Databasus">
                      Local, S3, R2, Azure, NAS, FTP
                    </td>
                  </tr>
                  <tr>
                    <td>Compressão</td>
                    <td data-label="pg_dump">gzip, LZ4, zstd (manual)</td>
                    <td data-label="Databasus">zstd (automática, otimizada)</td>
                  </tr>
                  <tr>
                    <td>Criptografia</td>
                    <td data-label="pg_dump">Requer ferramentas externas</td>
                    <td data-label="Databasus">✅ AES-256-GCM integrada</td>
                  </tr>
                  <tr>
                    <td>Notificações</td>
                    <td data-label="pg_dump">❌ Nenhuma</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, e-mail, Webhooks
                    </td>
                  </tr>
                  <tr>
                    <td>Recursos de colaboração</td>
                    <td data-label="pg_dump">❌ Nenhum</td>
                    <td data-label="Databasus">
                      ✅ Workspaces, RBAC, logs de auditoria
                    </td>
                  </tr>
                  <tr>
                    <td>Políticas de retenção</td>
                    <td data-label="pg_dump">Scripts manuais de limpeza</td>
                    <td data-label="Databasus">✅ Retenção automática</td>
                  </tr>
                  <tr>
                    <td>Monitoramento de saúde</td>
                    <td data-label="pg_dump">❌ Nenhuma</td>
                    <td data-label="Databasus">✅ Health checks integrados</td>
                  </tr>
                  <tr>
                    <td>Backups físicos</td>
                    <td data-label="pg_dump">❌ Não</td>
                    <td data-label="Databasus">✅ Sim</td>
                  </tr>
                  <tr>
                    <td>Backups incrementais</td>
                    <td data-label="pg_dump">❌ Não</td>
                    <td data-label="Databasus">
                      ✅ Em nível de bloco (PG 17+)
                    </td>
                  </tr>
                  <tr>
                    <td>Point-in-Time Recovery</td>
                    <td data-label="pg_dump">❌ Não</td>
                    <td data-label="Databasus">✅ Sim</td>
                  </tr>
                  <tr>
                    <td>Backups remotos</td>
                    <td data-label="pg_dump">✅ Sim (CLI)</td>
                    <td data-label="Databasus">✅ Sim</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="what-is-pgdump">O que é o pg_dump?</h2>

              <p>
                O <code>pg_dump</code> é a ferramenta nativa do PostgreSQL para
                criar backups lógicos. Faz parte do PostgreSQL desde o início e
                é a ferramenta padrão para exportar bases de dados.
              </p>

              <h3 id="pgdump-strengths">Pontos fortes do pg_dump</h3>

              <ul>
                <li>
                  <strong>Backups portáteis</strong>: cria dumps em SQL ou em
                  formato custom que podem ser restaurados em versões diferentes
                  do PostgreSQL.
                </li>
                <li>
                  <strong>Backups seletivos</strong>: pode exportar tabelas
                  específicas, esquemas ou bases de dados inteiras.
                </li>
                <li>
                  <strong>Snapshots consistentes</strong>: usa o MVCC do
                  PostgreSQL para criar backups consistentes sem bloquear
                  escritas.
                </li>
                <li>
                  <strong>Amplamente suportado</strong>: disponível em qualquer
                  instalação do PostgreSQL, bem documentado e testado em
                  produção há décadas.
                </li>
                <li>
                  <strong>Formatos de saída flexíveis</strong>: SQL puro,
                  custom, directory ou tar.
                </li>
              </ul>

              <h3 id="pgdump-limitations">Limitações do pg_dump</h3>

              <p>
                Embora o <code>pg_dump</code> seja poderoso, usá-lo em produção
                normalmente exige scripts adicionais:
              </p>

              <ul>
                <li>
                  <strong>Sem agendamento integrado</strong>: exige cron jobs ou
                  agendadores externos.
                </li>
                <li>
                  <strong>Apenas armazenamento local</strong>: grava no sistema
                  local; enviar para a nuvem exige scripts adicionais.
                </li>
                <li>
                  <strong>Sem criptografia</strong>: os dumps não são
                  criptografados por padrão; é preciso encadear com gpg ou
                  ferramentas semelhantes.
                </li>
                <li>
                  <strong>Sem notificações</strong>: não há como receber alertas
                  de sucesso ou falha do backup sem scripts próprios.
                </li>
                <li>
                  <strong>Sem gestão de retenção</strong>: backups antigos
                  precisam ser limpos manualmente ou via scripts.
                </li>
                <li>
                  <strong>Apenas linha de comando</strong>: nenhuma interface
                  visual para monitoramento ou gestão.
                </li>
              </ul>

              <h2 id="how-databasus-extends">
                Como o Databasus amplia o pg_dump
              </h2>

              <p>
                O Databasus usa o <code>pg_dump</code> como motor de backup,
                preservando todas as vantagens dos backups lógicos e
                acrescentando recursos corporativos por cima.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Por dentro:</strong> quando
                  um backup é iniciado no Databasus, ele executa o{" "}
                  <code className="bg-[#374151] text-gray-200">pg_dump</code>{" "}
                  com parâmetros otimizados e depois trata da compressão, da
                  criptografia e do envio para o destino de armazenamento
                  configurado.
                </p>
              </div>

              <h3 id="web-interface">Interface web</h3>

              <p>
                Em vez de memorizar as opções de linha de comando do{" "}
                <code>pg_dump</code>, o Databasus oferece uma interface web onde
                você pode:
              </p>

              <ul>
                <li>Adicionar bases de dados com um assistente de conexão</li>
                <li>Configurar agendamentos de backup com controles visuais</li>
                <li>
                  Acompanhar o histórico e o status dos backups num relance
                </li>
                <li>Baixar ou restaurar backups com um clique</li>
                <li>
                  Ver gráficos de saúde e disponibilidade das bases de dados
                </li>
              </ul>

              <h3 id="optimized-compression">Compressão otimizada</h3>

              <p>
                O Databasus usa por padrão a compressão zstd (nível 5), que
                oferece:
              </p>

              <ul>
                <li>
                  <strong>Redução de 4 a 8x no tamanho</strong> em comparação
                  com dumps sem compressão
                </li>
                <li>
                  <strong>~20% de custo extra de tempo</strong>, muito mais
                  rápido que gzip
                </li>
                <li>
                  <strong>Tratamento automático</strong>, sem necessidade de
                  encadear ferramentas de compressão
                </li>
              </ul>

              <h2 id="beyond-pgdump">
                Além do pg_dump: backups físicos e PITR
              </h2>

              <p>
                Embora o Databasus se apoie no <code>pg_dump</code> para backups
                lógicos, ele também vai além do que o <code>pg_dump</code>{" "}
                consegue oferecer:
              </p>

              <ul>
                <li>
                  <strong>Backups físicos</strong>: cópias em nível de dados de
                  todo o cluster da base de dados via <code>pg_basebackup</code>
                  . Backup e restauração mais rápidos para bases de dados
                  grandes.
                </li>
                <li>
                  <strong>Backups incrementais e de WAL</strong>: backups
                  incrementais em nível de bloco via{" "}
                  <code>pg_basebackup --incremental</code> (com base nos resumos
                  de WAL do servidor), além de streaming contínuo de WAL via{" "}
                  <code>pg_receivewal</code>, permitindo Point-in-Time Recovery,
                  ou seja, restaurar para qualquer segundo entre backups.
                </li>
                <li>
                  <strong>Recuperação de desastres</strong>: pensado para
                  requisitos de perda de dados quase nula, com backups físicos
                  de base e streaming contínuo de WAL.
                </li>
              </ul>

              <p>
                Esses backups se apoiam no mecanismo nativo de backup do
                PostgreSQL 17, então o Databasus reutiliza as ferramentas já
                comprovadas do próprio PostgreSQL em vez de reinventá-las.
                Exigem PostgreSQL 17 ou mais recente; em versões anteriores
                estão disponíveis apenas os backups lógicos com{" "}
                <code>pg_dump</code>. Tudo é executado remotamente a partir do
                host do Databasus pelo protocolo de replicação, então nada é
                instalado no servidor da base de dados. Redes fechadas são
                alcançadas por um túnel SSH até um host interno ou um bastion,
                então a base de dados nunca precisa ficar exposta publicamente.{" "}
                <a
                  href="/pt/faq#pitr"
                  className="text-blue-400 hover:text-blue-600"
                >
                  Leia como funcionam os backups físicos e PITR
                </a>
                .
              </p>

              <h2 id="backup-automation">Automação de backups</h2>

              <p>
                Um dos desafios mais comuns com o <code>pg_dump</code> é montar
                backups automáticos confiáveis.
              </p>

              <h3 id="automation-pgdump">Automação tradicional do pg_dump</h3>

              <p>
                Um script típico de automação do <code>pg_dump</code> é mais ou
                menos assim:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{`#!/bin/bash
# Backup script for pg_dump
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
DB_NAME="mydb"

# Create backup
pg_dump -Fc -h localhost -U postgres $DB_NAME > $BACKUP_DIR/$DB_NAME_$DATE.dump

# Compress (if not using custom format)
# gzip $BACKUP_DIR/$DB_NAME_$DATE.sql

# Encrypt
gpg --encrypt --recipient backup@company.com $BACKUP_DIR/$DB_NAME_$DATE.dump

# Upload to S3
aws s3 cp $BACKUP_DIR/$DB_NAME_$DATE.dump.gpg s3://my-bucket/backups/

# Cleanup old backups (keep last 7 days)
find $BACKUP_DIR -name "*.dump*" -mtime +7 -delete

# Send notification on failure
if [ $? -ne 0 ]; then
  curl -X POST https://hooks.slack.com/... -d '{"text":"Backup failed!"}'
fi`}</code>
                </pre>
              </div>

              <p>
                Este script precisa ser mantido, testado e monitorado. Cada base
                de dados exige a sua própria entrada no cron.
              </p>

              <h3 id="automation-databasus">Automação no Databasus</h3>

              <p>Com o Databasus, a mesma funcionalidade já vem integrada:</p>

              <ul>
                <li>
                  <strong>Agendador visual</strong>: defina backups por hora,
                  dia, semana, mês ou cron, com horários específicos.
                </li>
                <li>
                  <strong>Compressão automática</strong>: compressão zstd
                  aplicada automaticamente.
                </li>
                <li>
                  <strong>Criptografia integrada</strong>: criptografia
                  AES-256-GCM com chaves únicas por backup.
                </li>
                <li>
                  <strong>Envio para a nuvem</strong>: envio direto para S3,
                  Cloudflare R2, Azure ou outros destinos.
                </li>
                <li>
                  <strong>Políticas de retenção</strong>: limpeza automática de
                  backups antigos segundo as suas regras de retenção.
                </li>
                <li>
                  <strong>Notificações</strong>: alertas no Slack, Teams,
                  Telegram e e-mail em caso de sucesso ou falha.
                </li>
              </ul>

              <h2 id="storage-options">Opções de armazenamento</h2>

              <p>
                O <code>pg_dump</code> grava no sistema local. Levar os backups
                para armazenamento na nuvem exige ferramentas e scripts
                adicionais.
              </p>

              <h3 id="storage-databasus">
                Destinos de armazenamento do Databasus
              </h3>

              <p>
                O Databasus suporta vários destinos de armazenamento sem
                configuração extra:
              </p>

              <ul>
                <li>Armazenamento local</li>
                <li>Amazon S3 e serviços compatíveis com S3</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (armazenamento conectado à rede)</li>
                <li>FTP</li>
              </ul>

              <p>
                Cada base de dados pode ter o seu próprio destino de
                armazenamento, e você pode configurar vários destinos para
                redundância.
              </p>

              <p>
                <a
                  href="/pt/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Veja todas as opções de armazenamento →
                </a>
              </p>

              <h2 id="notifications">Notificações</h2>

              <p>
                Saber quando os backups têm sucesso ou falham é fundamental para
                a proteção dos dados.
              </p>

              <h3 id="notifications-pgdump">Notificações com pg_dump</h3>

              <p>
                O <code>pg_dump</code> não tem sistema de notificações. É
                preciso:
              </p>

              <ul>
                <li>Escrever scripts que verificam códigos de saída</li>
                <li>Integrar com ferramentas externas de monitoramento</li>
                <li>Montar pipelines de alerta próprios</li>
              </ul>

              <h3 id="notifications-databasus">Notificações no Databasus</h3>

              <p>O Databasus inclui notificações integradas para:</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>E-mail</li>
                <li>Webhooks (para integrações próprias)</li>
              </ul>

              <p>
                Configure quais eventos disparam notificações: sucesso do
                backup, falha do backup ou ambos.
              </p>

              <p>
                <a
                  href="/pt/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Veja todos os canais de notificação →
                </a>
              </p>

              <h2 id="team-features">Recursos de colaboração</h2>

              <p>
                O <code>pg_dump</code> é uma ferramenta de linha de comando para
                uma única pessoa. O Databasus adiciona recursos de colaboração
                para equipes:
              </p>

              <h3 id="team-databasus">
                Capacidades de colaboração do Databasus
              </h3>

              <ul>
                <li>
                  <strong>Workspaces</strong>: organize bases de dados,
                  notificadores e armazenamentos por projeto ou equipe. Cada
                  pessoa vê apenas os workspaces para os quais foi convidada.
                </li>
                <li>
                  <strong>Controle de acesso baseado em papéis</strong>: atribua
                  permissões de viewer, editor ou admin para controlar o que
                  cada membro pode fazer.
                </li>
                <li>
                  <strong>Logs de auditoria</strong>: acompanhe todas as
                  atividades e alterações do sistema. Essencial para
                  conformidade de segurança e responsabilização.
                </li>
                <li>
                  <strong>Notificações compartilhadas</strong>: os canais da
                  equipe recebem automaticamente o status dos backups.
                </li>
              </ul>

              <p>
                <a
                  href="/pt/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Saiba mais sobre gestão de acessos →
                </a>
              </p>

              <h2 id="security">Segurança</h2>

              <p>
                A segurança é onde o Databasus acrescenta mais valor em relação
                ao uso direto do <code>pg_dump</code>.
              </p>

              <h3 id="security-pgdump">Segurança do pg_dump</h3>

              <p>
                O <code>pg_dump</code> cria dumps sem criptografia. Para
                protegê-los é preciso:
              </p>

              <ul>
                <li>
                  Encadear a saída com ferramentas de criptografia (gpg,
                  openssl)
                </li>
                <li>Gerenciar as chaves de criptografia separadamente</li>
                <li>Garantir guarda segura e rotação das chaves</li>
                <li>Definir permissões adequadas nos dumps</li>
              </ul>

              <h3 id="security-databasus">Segurança do Databasus</h3>

              <p>O Databasus implementa segurança em vários níveis:</p>

              <ul>
                <li>
                  <strong>Criptografia AES-256-GCM</strong>: todas as senhas,
                  tokens e credenciais são criptografados. A chave de
                  criptografia fica guardada separada da base de dados.
                </li>
                <li>
                  <strong>Criptografia única por backup</strong>: cada backup é
                  criptografado com uma chave única derivada da chave mestra, do
                  ID do backup e de um salt aleatório.
                </li>
                <li>
                  <strong>Acesso somente leitura à base de dados</strong>: impõe
                  apenas permissões de SELECT, prevenindo corrupção de dados
                  mesmo em caso de comprometimento.
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

              <h2 id="restore-process">Processo de restauração</h2>

              <p>
                As duas ferramentas suportam restaurar backups, mas com fluxos
                diferentes.
              </p>

              <h3 id="restore-pgdump">Restaurar backups do pg_dump</h3>

              <p>
                Restaurar um backup do <code>pg_dump</code> exige:
              </p>

              <ol>
                <li>Localizar o backup</li>
                <li>Descriptografar, se estiver criptografado</li>
                <li>Descomprimir, se estiver comprimido</li>
                <li>
                  Executar <code>pg_restore</code> ou <code>psql</code> com os
                  parâmetros corretos
                </li>
              </ol>

              <h3 id="restore-databasus">Restaurar backups do Databasus</h3>

              <p>O Databasus simplifica a restauração:</p>

              <ul>
                <li>
                  <strong>Download com um clique</strong>: baixe qualquer backup
                  diretamente pela interface web.
                </li>
                <li>
                  <strong>Descriptografia automática</strong>: os backups são
                  descriptografados automaticamente ao baixar.
                </li>
                <li>
                  <strong>Comandos de restauração prontos</strong>: o Databasus
                  mostra o comando <code>pg_restore</code> exato para cada
                  backup.
                </li>
                <li>
                  <strong>Suporte a restauração paralela</strong>: use vários
                  núcleos de CPU para restaurar bases de dados grandes mais
                  rápido.
                </li>
              </ul>

              <h2 id="installation">Instalação</h2>

              <h3 id="install-pgdump">Instalação do pg_dump</h3>

              <p>
                O <code>pg_dump</code> vem com o PostgreSQL. Se você tem o
                PostgreSQL instalado, já tem o <code>pg_dump</code>.
              </p>

              <h3 id="install-databasus">Instalação do Databasus</h3>

              <p>O Databasus oferece vários métodos de instalação:</p>

              <ul>
                <li>
                  <strong>Script de uma linha</strong>: instala o Docker (se
                  necessário), configura o Databasus e ativa a inicialização
                  automática.
                </li>
                <li>
                  <strong>Docker run</strong>: um único comando para iniciar com
                  PostgreSQL embutido.
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
                  Veja o guia de instalação →
                </a>
              </p>

              <h2 id="conclusion">Conclusão</h2>

              <p>
                O <code>pg_dump</code> é a ferramenta de cópia de segurança
                consagrada do PostgreSQL, e o Databasus é construído diretamente
                sobre ele. A escolha entre usar o <code>pg_dump</code>{" "}
                diretamente ou através do Databasus depende das suas
                necessidades.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>Use o pg_dump diretamente se:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>Precisa de exportações pontuais ou ad-hoc</li>
                  <li>
                    Você está à vontade escrevendo e mantendo shell scripts
                  </li>
                  <li>
                    Já tem infraestrutura de automação (Ansible, Terraform,
                    etc.)
                  </li>
                  <li>
                    Só precisa de backups locais, sem armazenamento na nuvem
                  </li>
                  <li>Trabalha sozinho e tem necessidades simples</li>
                </ul>
              </div>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">Use o Databasus se:</strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Quer backups automáticos e agendados sem escrever scripts
                  </li>
                  <li>
                    Precisa de guardar backups na nuvem (S3, Azure Blob Storage, etc.)
                  </li>
                  <li>
                    Quer criptografia integrada sem gerenciar chaves manualmente
                  </li>
                  <li>
                    Precisa de notificações quando os backups têm sucesso ou
                    falham
                  </li>
                  <li>
                    Você trabalha em equipe e precisa de recursos de colaboração
                  </li>
                  <li>
                    Prefere uma interface visual a ferramentas de linha de
                    comando
                  </li>
                  <li>Quer políticas de retenção e limpeza automáticas</li>
                  <li>
                    Precisa de backups físicos, backups incrementais ou
                    Point-in-Time Recovery para recuperação de desastres
                  </li>
                </ul>
              </div>

              <p>
                O Databasus se apoia no <code>pg_dump</code> para backups
                lógicos e o amplia com automação, segurança e recursos de
                colaboração. Além disso, o Databasus também suporta backups
                físicos, backups incrementais com arquivamento de WAL e
                Point-in-Time Recovery, capacidades que o <code>pg_dump</code>{" "}
                simplesmente não pode oferecer.
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
