import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Расширенная конфигурация - документация Databasus",
  description:
    "Необязательные переменные окружения для self-hosted Databasus: вход через Google и GitHub, почта по SMTP, капча Cloudflare Turnstile, телеметрия, экспорт логов по OpenTelemetry и свой скрипт аналитики. Для обычной установки не нужны.",
  keywords: [
    "переменные окружения Databasus",
    "расширенная конфигурация Databasus",
    "конфигурация self-hosted",
    "GitHub OAuth",
    "Google OAuth",
    "настройка SMTP",
    "Cloudflare Turnstile",
    "переменные окружения Docker",
    "логи OpenTelemetry",
  ],
  openGraph: {
    title: "Расширенная конфигурация - документация Databasus",
    description:
      "Необязательные переменные окружения для self-hosted Databasus: вход через Google и GitHub, почта по SMTP, капча Cloudflare Turnstile, телеметрия, экспорт логов по OpenTelemetry и свой скрипт аналитики. Для обычной установки не нужны.",
    type: "article",
    url: getLocalizedUrl("ru", "advanced-config"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Расширенная конфигурация - документация Databasus",
    description:
      "Необязательные переменные окружения для self-hosted Databasus: вход через Google и GitHub, почта по SMTP, капча Cloudflare Turnstile, телеметрия, экспорт логов по OpenTelemetry и свой скрипт аналитики. Для обычной установки не нужны.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "advanced-config"),
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
            headline: "Расширенная конфигурация - документация Databasus",
            description:
              "Необязательные переменные окружения для self-hosted Databasus: вход через Google и GitHub, почта по SMTP, капча Cloudflare Turnstile, телеметрия, экспорт логов по OpenTelemetry и свой скрипт аналитики. Для обычной установки не нужны.",
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

      <DocsNavbarComponent lang="ru" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="ru" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="advanced-config">Расширенная конфигурация</h1>

              <p className="text-lg text-gray-400">
                Databasus из коробки работает с разумными настройками по
                умолчанию — стандартной установке в один контейнер конфигурация
                не нужна вовсе. Каждая переменная на этой странице{" "}
                <strong>необязательна</strong> и не требуется в 99%
                продакшен-установок
              </p>

              <h2 id="oauth">OAuth</h2>

              <p>
                По умолчанию в Databasus вход по email и паролю. Дополнительно
                можно разрешить вход через аккаунт Google или GitHub. Кнопка
                провайдера появляется, как только задан его client ID, но вход
                срабатывает, только когда заданы <strong>оба</strong> значения:
                и client ID, и client secret.
              </p>

              <p>
                При регистрации OAuth-приложения укажите redirect-адрес
                (callback){" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>. Из-за
                этого редиректа для входа через OAuth ваш инстанс должен быть
                доступен по HTTPS на публичном домене — см. заметку ниже.
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
                    <strong>Для входа и почты нужен HTTPS.</strong> И
                    OAuth-вход, и почта требуют, чтобы инстанс был доступен по
                    HTTPS на публичном домене: OAuth-провайдеры возвращают
                    браузер на{" "}
                    <code>https://&lt;your-domain&gt;/auth/callback</code>, а
                    ссылки в письмах должны открываться у получателей. Инстанс
                    только на localhost или на голом HTTP этими функциями
                    пользоваться не сможет. Проще всего получить HTTPS через{" "}
                    <a
                      href="/ru/installation/#caddy-reverse-proxy"
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
                Создайте OAuth-клиент в{" "}
                <a
                  href="https://console.cloud.google.com/apis/credentials"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Google Cloud Console
                </a>{" "}
                (APIs &amp; Services → Credentials → Create credentials → OAuth
                client ID, тип приложения <em>Web application</em>) и добавьте{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code> как
                разрешенный redirect URI.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Переменная</th>
                    <th>Описание</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_ID</code>
                    </td>
                    <td data-label="Описание">
                      Client ID вашего OAuth-клиента Google. Если задан,
                      появляется кнопка &quot;Sign in with Google&quot;.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Описание">
                      Client secret вашего OAuth-клиента Google. Чтобы вход
                      работал, нужен вместе с ID.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h3 id="oauth-github">GitHub</h3>

              <p>
                Создайте OAuth-приложение в{" "}
                <a
                  href="https://github.com/settings/developers"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  настройках разработчика GitHub
                </a>{" "}
                (Settings → Developer settings → OAuth Apps → New OAuth App) и
                укажите authorization callback URL{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Переменная</th>
                    <th>Описание</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_ID</code>
                    </td>
                    <td data-label="Описание">
                      Client ID вашего OAuth-приложения GitHub. Если задан,
                      появляется кнопка &quot;Sign in with GitHub&quot;.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Описание">
                      Client secret вашего OAuth-приложения GitHub. Чтобы вход
                      работал, нужен вместе с ID.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="email-smtp">Почта (SMTP)</h2>

              <p>
                Подключите почтовый сервер, чтобы Databasus мог отправлять приглашения в рабочие пространства, коды сброса пароля и коды входа. Почтовый сервер считается настроенным, <strong>как только задан <code>SMTP_HOST</code></strong>. Без него экран настроек показывает, что почтовый сервер не настроен, а на экране входа нет ссылки для сброса пароля.
              </p>

              <p>
                Это почтовый сервер инстанса. У каналов уведомлений по email, которые сообщают о бекапах, свои настройки SMTP в форме каждого канала. Поэтому если письма от канала уведомлений доходят, это ничего не говорит о почтовом сервере инстанса. Чтобы проверить его, откройте <strong>Databasus settings → Mail server</strong> и нажмите <strong>Send test email</strong>. Письмо придет на ваш адрес, а если доставка не удалась, вы увидите ответ почтового сервера.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Переменная</th>
                    <th>Описание</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>SMTP_HOST</code>
                    </td>
                    <td data-label="Описание">
                      Хост почтового сервера (например, <code>smtp.gmail.com</code>). Если переменная задана, почта инстанса включена.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PORT</code>
                    </td>
                    <td data-label="Описание">
                      Порт почтового сервера (например, <code>587</code>). Должен быть положительным целым числом, если задан <code>SMTP_HOST</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_SECURITY</code>
                    </td>
                    <td data-label="Описание">
                      Защита соединения: <code>tls</code>, <code>starttls</code> или <code>none</code>, подробнее — ниже. По умолчанию <code>tls</code> на порту 465 и <code>starttls</code> на всех остальных. С любым другим значением инстанс не запустится.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_USER</code>
                    </td>
                    <td data-label="Описание">
                      Имя пользователя для аутентификации на SMTP.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PASSWORD</code>
                    </td>
                    <td data-label="Описание">
                      Пароль для аутентификации на SMTP. Для Gmail используйте App Password, а не пароль аккаунта.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_FROM</code>
                    </td>
                    <td data-label="Описание">
                      Адрес отправителя: просто адрес (<code>noreply@example.com</code>) или адрес с именем (<code>Acme Backups &lt;noreply@example.com&gt;</code>). Если имени нет, получатели видят <code>Databasus</code>. Если переменная не задана, используется <code>SMTP_USER</code>, когда это адрес почты, а иначе <code>noreply@</code> и <code>SMTP_HOST</code>, так что логин вроде <code>apikey</code> никогда не станет отправителем. Если значение нельзя разобрать как адрес, инстанс не запустится.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_HELO_NAME</code>
                    </td>
                    <td data-label="Описание">
                      Имя, которым инстанс представляется почтовому серверу. По умолчанию берется хост из <code>DATABASUS_URL</code>, а если его нет — имя машины. Должно быть именем хоста или адресным литералом вроде <code>[192.0.2.1]</code> или <code>[IPv6:2001:db8::1]</code>, иначе инстанс не запустится.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_INSECURE_SKIP_VERIFY</code>
                    </td>
                    <td data-label="Описание">
                      Установите <code>true</code>, чтобы пропускать проверку TLS-сертификата в режимах <code>tls</code> и <code>starttls</code>. По умолчанию <code>false</code>. Используйте только для серверов с самоподписанным сертификатом в доверенной сети: эта настройка отключает защиту от атак «человек посередине».
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>DATABASUS_URL</code>
                    </td>
                    <td data-label="Описание">
                      Публичный базовый URL инстанса (например, <code>https://backup.example.com</code>). Необязателен: добавляет ссылку в письма-приглашения и задает имя в приветствии по умолчанию. Почта работает и без него.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                <code>SMTP_SECURITY</code> выбирает один из трех режимов. У каналов уведомлений по email такой же выбор есть в разделе <strong>Advanced settings</strong>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Режим</th>
                    <th>Что происходит</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>tls</code>
                    </td>
                    <td data-label="Что происходит">
                      Соединение зашифровано с первого байта. Обычно порт 465.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>starttls</code>
                    </td>
                    <td data-label="Что происходит">
                      Соединение начинается без шифрования и переводится в зашифрованное до отправки пароля и писем. Если сервер этого не поддерживает, доставка завершается ошибкой, а не продолжается без шифрования. Обычно порт 587.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>none</code>
                    </td>
                    <td data-label="Что происходит">
                      Соединение никогда не шифруется. Подходит для релея без STARTTLS, например локального Postfix на порту 25.
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
                    В режиме <code>none</code> пароль SMTP и все письма передаются без шифрования, и их может прочитать любой, кто находится между Databasus и релеем. Выбирайте его только для релея на том же хосте или в доверенной частной сети.
                  </p>
                </div>
              </div>

              <h2 id="signup-captcha">
                Капча при регистрации (Cloudflare Turnstile)
              </h2>

              <p>
                Если ваш инстанс доступен из публичного интернета, на формы
                регистрации и входа можно поставить проверку{" "}
                <a
                  href="https://www.cloudflare.com/products/turnstile/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Cloudflare Turnstile
                </a>
                , чтобы отсечь ботов. Оба ключа берутся в панели Turnstile, а
                проверка включается, только когда заданы оба.
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
                    Если нужно вообще закрыть регистрацию, а не просто прикрыть
                    ее капчей, капча не понадобится — откройте в интерфейсе{" "}
                    <strong>Databasus settings → Allow sign up</strong> и
                    выключите эту опцию. Форма регистрации исчезнет совсем.
                  </p>
                </div>
              </div>

              <table>
                <thead>
                  <tr>
                    <th>Переменная</th>
                    <th>Описание</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SITE_KEY</code>
                    </td>
                    <td data-label="Описание">
                      Публичный site key Turnstile — им рендерится виджет в
                      браузере.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SECRET_KEY</code>
                    </td>
                    <td data-label="Описание">
                      Секретный ключ Turnstile — им бэкенд проверяет ответы на
                      капчу.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="docker-storage-permissions">Права Docker-хранилища</h2>

              <p>
                Обычно Databasus берет числовые ID пользователя и группы у
                существующего каталога pgdata, затем проверяет mount с бекапами
                и корневой каталог данных. Если подходящего владельца нет,
                используется <code>999</code>. Задавайте <code>PUID</code> или{" "}
                <code>PGID</code>, только когда автоматический выбор не подходит
                для вашего bind mount, CIFS или NFS. Допустимы десятичные значения
                от <code>1</code> до <code>4294967294</code>.
              </p>

              <table>
                <thead><tr><th>Переменная</th><th>Автоматическое значение</th><th>Учетная запись</th></tr></thead>
                <tbody>
                  <tr><td><code>PUID</code></td><td>Владелец mount или <code>999</code></td><td>Пользователь <code>databasus</code></td></tr>
                  <tr><td><code>PGID</code></td><td>Группа mount или <code>999</code></td><td>Основная группа <code>databasus</code></td></tr>
                </tbody>
              </table>

              <p>
                Databasus и PostgreSQL работают под одной учетной записью
                ОС без прав root с именем <code>databasus</code>.
              </p>

              <p>
                Entrypoint запускается с правами root внутри контейнера, выбирает
                ID и пробует выполнить <code>chown</code> и <code>chmod</code>.
                После этого он проверяет полный цикл работы с файлами от имени{" "}
                <code>databasus</code>. Mount может работать, даже если файловая
                система запрещает менять владельца или режим. Произвольное
                значение Docker <code>user:</code> не поддерживается.
              </p>

              <pre><code>{`ERROR: Databasus cannot write to local storage paths /databasus-data/temp and /databasus-data/backups as UID 999 and GID 999.
Required operation: save a file through local storage.
Set PUID and PGID or fix the mounted directory permissions: https://databasus.com/advanced-config/#docker-storage-permissions
Details: permission denied`}</code></pre>

              <p>
                Четыре прежние переменные для отдельных сервисов удалены
                сознательно. При обновлении уберите их из конфигурации. Если
                автоматически выбранным ID не хватает прав, контейнер остановится
                и укажет неудавшуюся операцию вместе со ссылкой на эту инструкцию.
              </p>

              <h2 id="telemetry">Телеметрия</h2>

              <p>
                По умолчанию Databasus отправляет анонимную телеметрию
                использования. Она не содержит персональных данных и помогает
                нам понять, как используется проект. Что именно собирается,
                написано в{" "}
                <a
                  href="/privacy"
                  className="text-blue-400 hover:text-blue-300"
                >
                  политике конфиденциальности
                </a>
                , и телеметрию можно полностью выключить.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Переменная</th>
                    <th>По умолчанию</th>
                    <th>Описание</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>IS_DISABLE_ANONYMOUS_TELEMETRY</code>
                    </td>
                    <td data-label="По умолчанию">
                      <code>false</code>
                    </td>
                    <td data-label="Описание">
                      Установите <code>true</code>, чтобы отключить анонимную
                      телеметрию использования.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="logging">Логирование</h2>

              <p>
                Databasus пишет логи в stdout и дублирует их в формате JSON в
                файл <code>databasus.log</code> на томе с данными. Если задать{" "}
                <code>OPEN_TELEMETRY_URL</code>, он также экспортирует их по
                OpenTelemetry в бэкенд вроде VictoriaLogs, Graylog, SigNoz,
                Grafana Loki, Datadog или Honeycomb, либо в OpenTelemetry
                Collector, который сам является OTLP-приемником.
              </p>

              <ul>
                <li>
                  <strong>Транспорт</strong> определяется схемой.{" "}
                  <code>http://</code> и <code>https://</code> отправляют
                  OTLP/HTTP и используют URL как есть, включая путь;{" "}
                  <code>grpc://</code> и <code>grpcs://</code> отправляют
                  OTLP/gRPC и используют только хост и порт.
                </li>
                <li>
                  <strong>Аутентификация</strong> задается в{" "}
                  <code>OPEN_TELEMETRY_HEADERS</code> или прямо в URL в виде{" "}
                  <code>user:password@host</code>.
                </li>
                <li>
                  <strong>Секреты</strong> (пароли, токены, учетные данные) в
                  URL вычищаются до того, как запись покидает процесс.
                </li>
                <li>
                  <strong>Записи аудита</strong> уходят вместе с логами
                  приложения с меткой <code>log_type=audit</code> и игнорируют{" "}
                  <code>LOG_LEVEL</code>, поэтому повышение уровня никогда не
                  обрежет аудиторский след.
                </li>
              </ul>

              <table>
                <thead>
                  <tr>
                    <th>Переменная</th>
                    <th>По умолчанию</th>
                    <th>Описание</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_URL</code>
                    </td>
                    <td data-label="По умолчанию">—</td>
                    <td data-label="Описание">
                      Полный URL OTLP-эндпоинта, включая путь. Не задавайте,
                      чтобы логи оставались в контейнере. Query-строка,
                      отсутствующий хост или неизвестная схема останавливают
                      контейнер при старте, вместо того чтобы экспортировать в
                      никуда.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </td>
                    <td data-label="По умолчанию">—</td>
                    <td data-label="Описание">
                      Пары <code>key=value</code> через запятую, отправляемые с
                      каждым экспортом, обычно API-ключ. Значения
                      percent-декодируются, как в стандартном формате{" "}
                      <code>OTEL_EXPORTER_OTLP_HEADERS</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_LEVEL</code>
                    </td>
                    <td data-label="По умолчанию">
                      <code>info</code>
                    </td>
                    <td data-label="Описание">
                      Одно из <code>debug</code>, <code>info</code>,{" "}
                      <code>warn</code> или <code>error</code>. Нераспознанное
                      значение откатывается к <code>info</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_FILE_IS_ENABLED</code>
                    </td>
                    <td data-label="По умолчанию">
                      <code>true</code>
                    </td>
                    <td data-label="Описание">
                      Пишет <code>databasus.log</code> рядом с остальными
                      данными, с ротацией на 5 МБ и хранением 3 старых файлов.
                      Установите <code>false</code>, если ваша платформа уже
                      собирает stdout.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Значения для популярных бэкендов, каждое с заголовком
                аутентификации. Замените хосты, регионы и ключи на свои:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Бэкенд</th>
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
                      учетные данные, которые ждет ваш <code>vmauth</code> или
                      reverse proxy: у самого VictoriaLogs на пути приема
                      аутентификации нет.
                    </td>
                  </tr>
                  <tr>
                    <td>OpenTelemetry Collector</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://otel-collector:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> —
                      соответствует расширению <code>bearertokenauth</code> или{" "}
                      <code>basicauth</code> на приемнике. Collector, доступный
                      только внутри вашей сети, обычно не требует ничего.
                    </td>
                  </tr>
                  <tr>
                    <td>Graylog 6.2+</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://graylog:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> — токен,
                      заданный на входе OpenTelemetry (gRPC). Вход также
                      принимает mTLS вместо токена.
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
                      от <code>instance-id:api-token</code>
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
                      Не нужны — API-ключ хранит сам Agent и пересылает все от
                      вашего имени.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Значения заголовков percent-декодируются, поэтому пробел после{" "}
                <code>Basic</code> или <code>Bearer</code> записывается как{" "}
                <code>%20</code>, а запятая внутри значения — как{" "}
                <code>%2C</code>. Basic-аутентификацию можно указать и прямо в
                URL в виде <code>https://user:password@host/path</code> —
                Databasus превратит ее в тот же заголовок и не пустит в логи.
                Через <code>http://</code> и <code>grpc://</code> ключи и пароли
                идут открытым текстом, так что вне доверенной сети используйте{" "}
                <code>https://</code> или <code>grpcs://</code>.
              </p>

              <h2 id="analytics-script">Скрипт аналитики</h2>

              <p>
                Databasus может встроить в приложение ваш собственный скрипт
                аналитики или трекинга — Google Analytics, Plausible, Umami и
                похожие. Когда задан <code>ANALYTICS_SCRIPT</code>, его значение
                вставляется в <code>&lt;head&gt;</code> страницы при старте.
              </p>

              <p>
                <strong>Предупреждение о безопасности:</strong> значение
                вставляется как есть, как сырой HTML и JavaScript, и получает
                полный доступ к интерфейсу Databasus в браузере каждого
                посетителя. Задавайте только тот сниппет, который вы полностью
                контролируете и которому доверяете.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Переменная</th>
                    <th>Описание</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>ANALYTICS_SCRIPT</code>
                    </td>
                    <td data-label="Описание">
                      Свой <code>&lt;script&gt;</code>, вставляемый перед
                      закрывающим тегом <code>&lt;/head&gt;</code>. Не
                      задавайте, чтобы аналитики не было.
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
