import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Установка - документация Databasus",
  description:
    "Как установить Databasus: автоматический скрипт, Docker run, Docker Compose, Helm для Kubernetes или Caddy как reverse proxy. Простая установка self-hosted системы резервного копирования PostgreSQL без ручной настройки.",
  keywords: [
    "установка Databasus",
    "установка Docker",
    "настройка бекапов PostgreSQL",
    "self-hosted бекап",
    "Docker Compose",
    "установка резервного копирования баз данных",
    "настройка pg_dump",
    "Kubernetes",
    "Helm chart",
    "деплой в K8s",
    "Caddy reverse proxy",
    "настройка HTTPS",
    "health check",
    "мониторинг",
    "liveness probe",
  ],
  openGraph: {
    title: "Установка - документация Databasus",
    description:
      "Как установить Databasus: автоматический скрипт, Docker run, Docker Compose, Helm для Kubernetes или Caddy как reverse proxy. Простая установка self-hosted системы резервного копирования PostgreSQL без ручной настройки.",
    type: "article",
    url: getLocalizedUrl("ru", "installation"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Установка - документация Databasus",
    description:
      "Как установить Databasus: автоматический скрипт, Docker run, Docker Compose, Helm для Kubernetes или Caddy как reverse proxy. Простая установка self-hosted системы резервного копирования PostgreSQL без ручной настройки.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "installation"),
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
            headline: "Установка - документация Databasus",
            description:
              "Как установить Databasus: автоматический скрипт, Docker run, Docker Compose, Helm для Kubernetes или Caddy как reverse proxy. Простая установка self-hosted системы резервного копирования PostgreSQL без ручной настройки.",
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
            name: "Как установить Databasus",
            description:
              "Пошаговое руководство по установке Databasus, инструмента резервного копирования PostgreSQL",
            step: [
              {
                "@type": "HowToStep",
                name: "Автоматический скрипт установки",
                text: "Запустите скрипт установки: он поставит Docker и настроит Databasus с автозапуском.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Выполните команду curl, чтобы скачать и запустить скрипт установки",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Docker Run",
                text: "Запустите контейнер Databasus одной командой docker run с сохранением данных.",
              },
              {
                "@type": "HowToStep",
                name: "Docker Compose",
                text: "Создайте файл docker-compose.yml и управляйте развертыванием через Docker Compose.",
              },
              {
                "@type": "HowToStep",
                name: "Kubernetes с Helm",
                text: "Разверните Databasus в Kubernetes официальным Helm-чартом: StatefulSet, персистентное хранилище и опциональный ingress.",
              },
              {
                "@type": "HowToStep",
                name: "Запуск за reverse proxy Caddy",
                text: "Для продакшена используйте Docker Compose с Caddy и автоматическими HTTPS-сертификатами.",
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent lang="ru" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="ru" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="installation">Установка</h1>

              <p className="text-lg text-gray-400">
                Databasus можно установить несколькими способами: автоматическим
                скриптом (рекомендуется), простым docker run, через Docker
                Compose, Helm для Kubernetes или Docker Compose с Caddy для
                продакшена.
              </p>

              <h2 id="system-requirements">Системные требования</h2>

              <p>Минимальные ресурсы, которые нужны Databasus для работы:</p>

              <ul>
                <li>
                  <strong>CPU</strong>: не меньше 1 ядра
                </li>
                <li>
                  <strong>RAM</strong>: минимум 500 МБ
                </li>
                <li>
                  <strong>Диск</strong>: 5 ГБ под установку плюс место под сами
                  бекапы
                </li>
                <li>
                  <strong>Docker</strong>: Docker Engine 20.10+ и Docker Compose
                  v2.0+
                </li>
              </ul>

              <h2 id="option-1-automated-script">
                Способ 1: скрипт установки (рекомендуется, только Linux)
              </h2>

              <p>Скрипт установки:</p>

              <ul>
                <li>
                  ✅ Поставит Docker с Docker Compose (если они еще не
                  установлены)
                </li>
                <li>✅ Настроит Databasus</li>
                <li>✅ Включит автозапуск после перезагрузки системы</li>
              </ul>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{installScript}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={installScript} />
                </div>
              </div>

              <p>
                В этом случае Databasus будет установлен в каталог{" "}
                <code>/opt/databasus</code>.
              </p>

              <h2 id="option-2-docker-run">Способ 2: простой docker run</h2>

              <p>Самый быстрый способ запустить Databasus:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerRun}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={dockerRun} />
                </div>
              </div>

              <p>Эта одна команда:</p>

              <ul>
                <li>✅ Запустит Databasus</li>
                <li>
                  ✅ Сохранит все данные в каталоге{" "}
                  <code>./databasus-data</code>
                </li>
                <li>
                  ✅ Автоматически перезапустит контейнер после перезагрузки
                </li>
              </ul>

              <h2 id="option-3-docker-compose">
                Способ 3: настройка через Docker Compose
              </h2>

              <p>
                Создайте файл <code>docker-compose.yml</code> со следующей
                конфигурацией:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerCompose}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={dockerCompose} />
                </div>
              </div>

              <p>Затем выполните:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text="docker compose up -d" />
                </div>
              </div>

              <p>Учтите, что запуск может занять до 2-х минут.</p>

              <h2 id="option-4-helm">Способ 4: Kubernetes с Helm</h2>

              <p>
                Для Kubernetes устанавливайте напрямую из OCI-реестра. Выберите
                способ доступа под свое окружение.
              </p>

              <h3 id="helm-clusterip">
                ClusterIP + port-forward (для разработки)
              </h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallClusterIP}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={helmInstallClusterIP} />
                </div>
              </div>

              <p>Доступ через port-forward:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmPortForward}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={helmPortForward} />
                </div>
              </div>

              <h3 id="helm-loadbalancer">LoadBalancer (облачные окружения)</h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallLoadBalancer}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={helmInstallLoadBalancer} />
                </div>
              </div>

              <p>Получите внешний IP и откройте Databasus:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmGetSvc}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={helmGetSvc} />
                </div>
              </div>

              <h3 id="helm-ingress">Ingress (доступ по домену)</h3>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmInstallIngress}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={helmInstallIngress} />
                </div>
              </div>

              <p>
                Другие варианты (NodePort, TLS, HTTPRoute для Gateway API)
                описаны в{" "}
                <a
                  href="https://github.com/databasus/databasus/tree/main/deploy/helm"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  документации Helm-чарта
                </a>
                .
              </p>

              <h2 id="caddy-reverse-proxy">Запуск за reverse proxy Caddy</h2>

              <p>
                Для продакшена можно поставить{" "}
                <a
                  href="https://caddyserver.com/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Caddy
                </a>{" "}
                как reverse proxy: он даст автоматические HTTPS-сертификаты и
                безопасный доступ к Databasus. Ниже полная конфигурация Docker
                Compose с Caddy.
              </p>

              <h3 id="caddy-docker-compose">Docker Compose с Caddy</h3>

              <p>
                Создайте файл <code>docker-compose.yml</code>:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{dockerComposeCaddy}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={dockerComposeCaddy} />
                </div>
              </div>

              <p>
                Создайте <code>Caddyfile</code> в том же каталоге:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{caddyfile}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={caddyfile} />
                </div>
              </div>

              <p>Затем запустите сервисы:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker compose up -d</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text="docker compose up -d" />
                </div>
              </div>

              <p>Эта конфигурация дает:</p>

              <ul>
                <li>
                  ✅ Автоматический HTTPS с сертификатами Let&apos;s Encrypt
                </li>
                <li>✅ Редирект с HTTP на HTTPS</li>
                <li>✅ Reverse proxy до Databasus</li>
                <li>✅ Сохранение данных Caddy и Databasus на диске</li>
              </ul>

              <p>
                Замените <code>backup.example.com</code> на свой домен и
                убедитесь, что DNS домена указывает на IP-адрес сервера, прежде
                чем запускать сервисы.
              </p>

              <h2 id="getting-started">Первые шаги</h2>

              <p>После установки:</p>

              <ol>
                <li>
                  <strong>Запустите и откройте Databasus</strong>: стартуйте
                  Databasus и перейдите на <code>http://localhost:4005</code>
                </li>
                <li>
                  <strong>Создайте первую задачу бекапа</strong>: нажмите
                  &quot;New Backup&quot; и настройте подключение к базе
                  PostgreSQL
                </li>
                <li>
                  <strong>Настройте расписание</strong>: задайте график бекапов
                  (каждый час, день, неделю, месяц или cron)
                </li>
                <li>
                  <strong>Выберите хранилище</strong>: укажите, куда складывать
                  бекапы (локальный диск, S3, Azure Blob Storage и т.д.)
                </li>
                <li>
                  <strong>Подключите уведомления</strong>: добавьте каналы
                  (Slack, Telegram, Discord), чтобы получать статусы бекапов
                </li>
                <li>
                  <strong>Запустите резервное копирование</strong>: сохраните
                  настройки и наблюдайте за первым бекапом!
                </li>
              </ol>

              <h2 id="health-checks">Health-чеки</h2>

              <h3 id="docker-health-check">Health-чек в Docker</h3>

              <p>
                Встроенный health-чек включается автоматически для{" "}
                <code>docker run</code> и Docker Compose. Контейнер получает
                статус <code>healthy</code>, как только Databasus начинает
                отвечать на запросы (после короткого периода на старт).
                Проверяется только то, что приложение отвечает, поэтому
                контейнер не перезапускается из-за некритичных состояний вроде
                заканчивающегося места на диске.
              </p>

              <h3 id="monitoring-endpoint">Эндпоинт мониторинга и статуса</h3>

              <p>Для аптайм-мониторинга и статус-дашбордов:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{healthEndpoint}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={healthEndpoint} />
                </div>
              </div>

              <p>
                Возвращает <code>200</code>, когда все в порядке, или{" "}
                <code>503</code> с причиной, когда что-то требует внимания:
                внутренняя база, диск (заполнен больше чем на 95%), клиентские
                утилиты баз данных, фоновые задачи или агент проверки
                восстановления. Аутентификация не нужна, CORS открыт для
                браузерных мониторов.
              </p>

              <p>
                <strong>⚠️ Важно:</strong> эндпоинт предназначен только для
                мониторинга и алертов, не для liveness probe контейнера или
                Kubernetes &mdash; он возвращает <code>503</code> и в состояниях
                «деградировал, но работает» (например, почти полный диск), а это
                перезапустило бы рабочий контейнер.
              </p>

              <h3 id="kubernetes-health-check">Kubernetes</h3>

              <p>
                Используйте liveness/readiness probe с командой{" "}
                <code>databasus healthcheck</code>, а эндпоинт{" "}
                <a
                  href="#monitoring-endpoint"
                  className="text-blue-400 hover:text-blue-300"
                >
                  /api/v1/system/health
                </a>{" "}
                оставьте для внешнего мониторинга.
              </p>

              <h2 id="how-to-update">Как обновить Databasus?</h2>

              <h3 id="update-docker">Обновление установки в Docker</h3>

              <p>
                Чтобы обновить Databasus, запущенный в Docker, остановите его,
                очистите кеш Docker и перезапустите контейнер.
              </p>

              <ol>
                <li>
                  Перейдите в каталог, где установлен Databasus (обычно{" "}
                  <code>/opt/databasus</code>)
                </li>
                <li>
                  Остановите контейнер: <code>docker compose stop</code>
                </li>
                <li>
                  Очистите кеш Docker: <code>docker system prune -a</code>
                </li>
                <li>
                  Перезапустите контейнер: <code>docker compose up -d</code>
                </li>
              </ol>

              <p>
                Docker скачает последнюю версию Databasus с Docker Hub (если вы
                не зафиксировали версию в файле <code>docker-compose.yml</code>
                ).
              </p>

              <h3 id="update-helm">Обновление установки через Helm</h3>

              <p>
                Чтобы обновить Databasus в Kubernetes через Helm, выполните
                команду upgrade:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{helmUpgrade}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text={helmUpgrade} />
                </div>
              </div>

              <p>
                Если у вас кастомные значения, добавьте{" "}
                <code>-f values.yaml</code> или флаги <code>--set</code>, чтобы
                сохранить свою конфигурацию. Helm выполнит rolling-обновление до
                новой версии.
              </p>

              <h2 id="postgresus-migration">Миграция с Postgresus</h2>

              <p>
                Databasus — новое имя Postgresus. Если вы сейчас используете
                Postgresus, можно продолжать им пользоваться или мигрировать на
                Databasus.
              </p>

              <p>
                <strong>Важно:</strong> просто переименовать Docker-образ
                недостаточно: Postgresus и Databasus используют разные каталоги
                данных и разные имена внутренней базы.
              </p>

              <p>Для миграции:</p>

              <ol>
                <li>
                  Остановите контейнер Postgresus:{" "}
                  <code>docker compose stop</code>
                </li>
                <li>
                  Установите Databasus любым способом выше (с другим путем тома,{" "}
                  <code>./databasus-data</code>)
                </li>
                <li>
                  Вручную заново создайте базы, хранилища и уведомления в
                  Databasus
                </li>
              </ol>

              <p>
                На время миграции Postgresus и Databasus могут работать
                параллельно на разных портах и с разными путями томов.
              </p>

              <h2 id="troubleshooting">Устранение неполадок</h2>

              <h3 id="container-wont-start">Контейнер не стартует</h3>

              <p>Если контейнер не запускается, посмотрите логи:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>docker logs databasus</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="ru" text="docker logs databasus" />
                </div>
              </div>

              <h3 id="port-already-in-use">Порт уже занят</h3>

              <p>Если порт 4005 занят, поменяйте его в docker-compose.yml:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>
                    ports:
                    {"\n  "}- &quot;8080:4005&quot; # Change 8080 to any
                    available port
                  </code>
                </pre>
              </div>

              <h3 id="permission-denied">Ошибки прав доступа</h3>

              <p>Если возникают проблемы с правами на каталог данных:</p>

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
                    lang="ru"
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
