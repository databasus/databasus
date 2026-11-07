import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import HeroStatsComponent from "@/app/components/HeroStatsComponent";
import InstallationComponent from "@/app/components/InstallationComponent";
import LiteYouTubeEmbed from "@/app/components/LiteYouTubeEmbed";

export const metadata: Metadata = {
  title: "Резервное копирование PostgreSQL | Databasus",
  description:
    "Бесплатный open source инструмент для резервного копирования PostgreSQL по расписанию (поддерживаются также MySQL и MongoDB). Храните бекапы локально и в облаках. Уведомления в Slack, Discord, Telegram, на почту, через вебхук и другие каналы.",
  keywords:
    "PostgreSQL, резервное копирование, бекап, мониторинг, база данных, бекапы по расписанию, Docker, self-hosted, open source, S3, Google Drive, уведомления в Slack, Discord, DevOps, мониторинг баз данных, pg_dump, восстановление базы данных, шифрование, AES-256, шифрование бекапов",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("ru", ""),
    languages: getLanguageAlternates(""),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("ru", ""),
    title: "Резервное копирование PostgreSQL | Databasus",
    description:
      "Бесплатный open source инструмент для резервного копирования PostgreSQL по расписанию (поддерживаются также MySQL и MongoDB). Храните бекапы локально и в облаках. Уведомления в Slack, Discord, Telegram, на почту, через вебхук и другие каналы.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Интерфейс панели Databasus для управления бекапами",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary_large_image",
    title: "Резервное копирование PostgreSQL | Databasus",
    description:
      "Бесплатный open source инструмент для резервного копирования PostgreSQL по расписанию (поддерживаются также MySQL и MongoDB). Храните бекапы локально и в облаках. Уведомления в Slack, Discord, Telegram, на почту, через вебхук и другие каналы.",
    images: ["https://databasus.com/images/index/dashboard.png"],
  },
  applicationName: "Databasus",
  appleWebApp: {
    title: "Databasus",
    capable: true,
  },
  icons: {
    icon: [
      { url: "/favicon.ico", type: "image/x-icon" },
      { url: "/favicon.svg", type: "image/svg+xml" },
    ],
    apple: "/favicon.svg",
    shortcut: "/favicon.ico",
  },
};

export default function Index() {
  return (
    <div className="overflow-x-hidden">
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "SoftwareApplication",
            name: "Databasus",
            description:
              "Бесплатный open source инструмент для резервного копирования PostgreSQL по расписанию (поддерживаются также MySQL и MongoDB). Храните бекапы локально и в облаках. Уведомления в Slack, Discord, Telegram, на почту, через вебхук и другие каналы.",
            url: "https://databasus.com",
            image: "https://databasus.com/images/index/dashboard.png",
            logo: "https://databasus.com/logo.svg",
            publisher: {
              "@type": "Organization",
              name: "Databasus",
              logo: {
                "@type": "ImageObject",
                url: "https://databasus.com/logo.svg",
              },
            },
            featureList: [
              "Резервное копирование PostgreSQL по расписанию",
              "Много хранилищ для бекапов (S3, Google Drive, Dropbox, SFTP, rclone и другие)",
              "Мгновенные уведомления (Slack, Telegram, Discord, вебхук, почта и другие)",
              "Мониторинг состояния баз данных",
              "Self-hosted развертывание через Docker",
              "Open source и бесплатно",
              "Поддержка PostgreSQL 12-18",
              "Сжатие бекапов и шифрование AES-256-GCM",
              "Поддержка PostgreSQL, MySQL, MariaDB и MongoDB",
              "Политики хранения: по сроку, по количеству, GFS и по объему",
              "Point-in-Time Recovery (PITR) с архивированием WAL",
              "Проверка восстановления: автоматическое тестовое восстановление в реальных Docker-контейнерах с базой данных",
            ],
            screenshot: "https://databasus.com/images/index/dashboard.png",
            softwareVersion: "latest",
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "Organization",
            name: "Databasus",
            url: "https://databasus.com/",
            alternateName: ["databasus", "Databasus"],
            logo: "https://databasus.com/logo.svg",
            sameAs: ["https://github.com/databasus/databasus"],
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "WebSite",
            name: "Databasus",
            alternateName: ["databasus", "Databasus"],
            url: "https://databasus.com/",
            description: "Инструмент резервного копирования PostgreSQL",
            publisher: { "@type": "Organization", name: "Databasus" },
          }),
        }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "Что такое Databasus и почему он лучше самописных скриптов?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus — это self-hosted сервис под лицензией Apache 2.0 для резервного копирования PostgreSQL с 13 по 18 версию. От shell-скриптов он отличается веб-интерфейсом: запуск по расписанию, сжатие, сохранение архивов сразу в несколько хранилищ (локальный диск, S3, Google Drive, Dropbox, SFTP, rclone и другие), автоудаление старых бекапов по политикам хранения и уведомления об успехе или сбое — и все это без единой строчки самописного кода",
                },
              },
              {
                "@type": "Question",
                name: "Как установить Databasus быстрее всего?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Самый короткий путь — установочный скрипт в одну строку через cURL. Он скачивает актуальный Docker-образ, поднимает один контейнер с PostgreSQL, создает docker-compose.yml и запускает сервис с автоперезапуском после перезагрузки. На типичном VPS все занимает меньше 2-х минут.",
                },
              },
              {
                "@type": "Question",
                name: "Как работает проверка восстановления?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus запускает небольшой агент проверки на вашем сервере. При каждом запуске по расписанию агент скачивает последний бекап, восстанавливает его во временный контейнер с базой данных и сверяет восстановленную базу с исходной. Результат приходит вам: код завершения восстановления и число строк по каждой таблице. Расписания: после бекапа, каждый час, день, неделю, месяц или cron-выражение в UTC. О сбоях можно узнавать через любой канал уведомлений, подключенный к базе — Slack, Teams, Discord, почту и другие.",
                },
              },
              {
                "@type": "Question",
                name: "Где хранятся мои бекапы и сколько места они займут?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Архивы можно сохранять на локальные диски, в S3-совместимые бакеты, Google Drive, Dropbox и другие облачные хранилища. Databasus применяет сбалансированное сжатие: обычно оно уменьшает размер дампа в 4-8 раз при накладных расходах всего около 20% времени, так что вы экономите и место, и трафик.",
                },
              },
              {
                "@type": "Question",
                name: "Как я узнаю, что бекап прошел успешно — или, хуже, что он упал?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus умеет мгновенно уведомлять через почту, Slack, Telegram, вебхуки, Mattermost, Discord и другие каналы. Вы сами выбираете, куда отправлять сообщения, чтобы ваша DevOps-команда узнавала об успехах и сбоях в реальном времени — это упрощает процедуры восстановления и аудиты соответствия.",
                },
              },
              {
                "@type": "Question",
                name: "Как Databasus обеспечивает безопасность?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus обеспечивает безопасность на трех уровнях: (1) шифрование чувствительных данных — все пароли, токены и учетные данные шифруются AES-256-GCM и хранятся отдельно от базы данных; (2) шифрование бекапов — каждый файл бекапа шифруется уникальным ключом, выведенным из мастер-ключа, ID бекапа и случайной соли, так что без вашего ключа бекапы бесполезны, даже если кто-то получит доступ к хранилищу; (3) доступ к базе только на чтение — Databasus требует лишь права SELECT и тщательно проверяет отсутствие прав на запись, что защищает данные от порчи даже при компрометации инструмента. Сам вход тоже можно закрыть вторым фактором: когда он включен, после верного пароля на почту аккаунта приходит шестизначный код, а вход через Google или GitHub по-прежнему опирается на проверки самих провайдеров. Помимо этого, безопасность и надежность встроены в каждый релиз: статический анализ CodeQL, поиск секретов через gitleaks, проверка правил безопасности через semgrep, мониторинг CVE через Dependabot, сканирование образов и Dockerfile через Trivy и периодические аудиты Codex Security от OpenAI. Интеграционные тесты гоняются на реальных контейнерах PostgreSQL, MySQL, MariaDB и MongoDB и проверяют полный цикл «бекап, затем восстановление» перед каждым релизом. GitHub Actions закреплены на SHA коммитов, а воркфлоу следуют принципу минимальных привилегий. Все операции выполняются в контейнерах под вашим контролем на ваших серверах, и поскольку это open source, ваша команда безопасности может проверить каждую строку кода до развертывания.",
                },
              },
              {
                "@type": "Question",
                name: "Правда, что Anthropic и OpenAI поддержали Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Да, в марте 2026 года Databasus был принят и в Claude for Open Source от Anthropic, и в Codex for Open Source от OpenAI. Проект прошел независимую оценку и был признан лидерами индустрии критически важной open source инфраструктурой, которую стоит поддерживать.",
                },
              },
              {
                "@type": "Question",
                name: "Чем Databasus отличается от pgBackRest, Barman и pg_dump?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus делает ставку на простоту: вместо конфигов и консольных утилит — современный веб-интерфейс для управления бекапами сразу многих баз данных со встроенными расписаниями, сжатием, разными хранилищами, мониторингом состояния и мгновенными уведомлениями. При этом, в отличие от pgBackRest и WAL-G, физические, инкрементальные и WAL-бекапы Databasus строит на нативном механизме PostgreSQL 17, а не изобретает собственный движок резервного копирования. Он подключается к базам удаленно и дотягивается до закрытых сетей через SSH-туннель к серверу или бастиону, так что базы без публичного доступа тоже можно бекапить и администрировать из одной панели.",
                },
              },
              {
                "@type": "Question",
                name: "Какие базы данных поддерживает Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus поддерживает PostgreSQL, MySQL, MariaDB и MongoDB. При этом изначально Databasus создавался именно для PostgreSQL и сохраняет на нем основной фокус — со стопроцентной поддержкой и максимальной эффективностью бекапов PostgreSQL. MySQL, MariaDB и MongoDB поддерживаются, но приоритетом остается PostgreSQL: самые проработанные возможности и активная разработка именно там. Например, Databasus нативно поддерживает физические и WAL-бекапы для аварийного восстановления PostgreSQL. По сути Databasus — это инструмент резервного копирования PostgreSQL, а остальные СУБД идут бонусом.",
                },
              },
              {
                "@type": "Question",
                name: "Насколько широко используется Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Сегодня Databasus — самый распространенный open source инструмент резервного копирования PostgreSQL. На 17 июня 2026 года его скачали из Docker больше 1 800 000 раз — DBA, DevOps-инженеры, разработчики и команды по всему миру. У проекта 8 500+ звезд на GitHub: больше, чем у pgBackRest (~4 200 звезд, существует с 2014 года) и WAL-G (~4 100 звезд, с 2017 года). Databasus вышел в 2025 году и обогнал оба проекта за первый же год.",
                },
              },
              {
                "@type": "Question",
                name: "Какие типы бекапов поддерживает Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus поддерживает физические, полные, инкрементальные, WAL и логические бекапы. Физический бекап — это копия всего кластера базы данных на уровне файлов: для больших объемов данных он быстрее логических дампов и при копировании, и при восстановлении, а построен он на нативном механизме бекапов PostgreSQL 17, то есть мы опираемся на проверенный инструментарий самого PostgreSQL вместо того, чтобы изобретать свой. Полный бекап — это самодостаточная копия кластера, основа, с которой начинается каждая цепочка бекапов. Инкрементальный бекап хранит только изменения с прошлого бекапа, поэтому бекапы остаются маленькими и быстрыми. Потоковая передача WAL непрерывно захватывает поток записи базы, обеспечивая Point-in-Time Recovery (PITR) для аварийного восстановления с почти нулевой потерей данных. Логический бекап — нативный дамп базы в бинарном формате конкретного движка, сжимаемый и передаваемый в хранилище потоком без промежуточных файлов. Все эти бекапы могут работать через SSH-туннель, если вам нельзя открывать публичные подключения — базу вообще не нужно выставлять наружу. SSH-туннелирование встроено.",
                },
              },
              {
                "@type": "Question",
                name: "Я забыл email или пароль администратора",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: 'Первая учетная запись, созданная на инстансе, им и управляет. Если вы не помните, какой это адрес, выполните docker exec -it databasus ./main --list-admins на сервере, где запущен Databasus — команда перечислит все учетные записи администраторов с их email, именем, датой создания и состоянием активности и не выведет ни одного пароля. Чтобы задать новый пароль этой учетной записи, выполните docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com". Инстанс, созданный до того, как первая учетная запись стала им управлять, до сих пор носит адрес-заглушку admin, пока владелец его не заменит. Если инстанс еще и требует код при входе, а почта перестала его доставлять, выполните docker exec -it databasus ./main --disable-2fa, чтобы инстанс перестал запрашивать код. Все три команды описаны на странице про пароль.',
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        lang="ru"
        sectionLinks={[
          { href: "#how-to-use", label: "Как пользоваться" },
        ]}
      />

      {/* MAIN SECTION */}
      <main className="relative overflow-hidden pt-[60px] md:pt-[68px]">
        <div className="relative mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px] px-4 md:px-6 lg:px-0 pt-12 md:pt-[100px] pb-12 md:pb-[100px]">
          {/* Background ellipse */}
          <div className="relative">
            <div className="absolute left-1/2 -translate-x-1/2 -translate-y-1/4 w-[400px] h-[400px] md:w-[900px] md:h-[900px] bg-[#155dfc]/4 top-0 rounded-full blur-3xl -z-10" />
          </div>

          {/* Content */}
          <div className="text-center mb-8 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Databasus</span>
            </div>

            <h1 className="text-2xl sm:text-4xl sm:max-w-[300px] md:text-4xl leading-tight font-bold mb-4 md:mb-6 mx-auto md:max-w-[650px]">
              Резервное копирование PostgreSQL с Point-in-Time Recovery и
              проверкой восстановления
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus — открытый self-hosted инструмент для резервного
              копирования PostgreSQL. Делайте бекапы в разные хранилища (S3,
              Google Drive, FTP и другие) и получайте уведомления о результате
              (Slack, Discord, Telegram и другие). С фокусом на Point-in-Time
              Recovery{" "}
              <span className="underline decoration-2 underline-offset-2 decoration-blue-600">
                с низкими RPO/RTO
              </span>
            </p>

            <div>
              <div className="flex flex-col items-center justify-center gap-2 max-w-[370px] sm:max-w-[340px] mx-auto">
                <a
                  href="#installation"
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-1"
                >
                  Развернуть через Docker
                </a>

                <a
                  href="https://github.com/databasus/databasus"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="w-full inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 rounded-lg font-medium border border-[#ffffff20] bg-[#0C0E13] hover:opacity-70 transition-opacity order-2 sm:order-2"
                >
                  <svg
                    aria-hidden={true}
                    width="24"
                    height="24"
                    viewBox="0 0 20 20"
                    fill="none"
                    xmlns="http://www.w3.org/2000/svg"
                  >
                    <g clipPath="url(#clip0_1_2459)">
                      <path
                        fillRule="evenodd"
                        clipRule="evenodd"
                        d="M9.9702 0C4.45694 0 0 4.4898 0 10.0443C0 14.4843 2.85571 18.2427 6.81735 19.5729C7.31265 19.6729 7.49408 19.3567 7.49408 19.0908C7.49408 18.858 7.47775 18.0598 7.47775 17.2282C4.70429 17.8269 4.12673 16.0308 4.12673 16.0308C3.68102 14.8667 3.02061 14.5676 3.02061 14.5676C2.11286 13.9522 3.08673 13.9522 3.08673 13.9522C4.09367 14.0188 4.62204 14.9833 4.62204 14.9833C5.51327 16.5131 6.94939 16.0808 7.52714 15.8147C7.60959 15.1661 7.87388 14.7171 8.15449 14.4678C5.94245 14.2349 3.6151 13.3702 3.6151 9.51204C3.6151 8.41449 4.01102 7.51653 4.63837 6.81816C4.53939 6.56878 4.19265 5.53755 4.73755 4.15735C4.73755 4.15735 5.57939 3.89122 7.47755 5.18837C8.29022 4.9685 9.12832 4.85666 9.9702 4.85571C10.812 4.85571 11.6702 4.97225 12.4627 5.18837C14.361 3.89122 15.2029 4.15735 15.2029 4.15735C15.7478 5.53755 15.4008 6.56878 15.3018 6.81816C15.9457 7.51653 16.3253 8.41449 16.3253 9.51204C16.3253 13.3702 13.998 14.2182 11.7694 14.4678C12.1327 14.7837 12.4461 15.3822 12.4461 16.3302C12.4461 17.6771 12.4298 18.7582 12.4298 19.0906C12.4298 19.3567 12.6114 19.6729 13.1065 19.5731C17.0682 18.2424 19.9239 14.4843 19.9239 10.0443C19.9402 4.4898 15.4669 0 9.9702 0Z"
                        fill="white"
                      />
                    </g>
                    <defs>
                      <clipPath id="clip0_1_2459">
                        <rect width="20" height="20" fill="white" />
                      </clipPath>
                    </defs>
                  </svg>

                  <span>GitHub</span>
                </a>
              </div>

              <div className="mt-2 max-w-[370px] sm:max-w-[340px] mx-auto pb-0 sm:pb-[50px] lg:pb-0">
                <HeroStatsComponent />
              </div>
            </div>
          </div>

          {/* Dashboard Screenshot */}
          <div className="relative mx-auto max-w-[1200px]">
            <div>
              <img
                src="/images/index/dashboard.svg"
                alt="Интерфейс панели Databasus"
                width={980}
                height={620}
                className="w-full h-auto"
                loading="eager"
                fetchPriority="high"
              />
            </div>
          </div>

          <div className="mt-10 md:mt-15 mb-12 md:mb-20 flex justify-center px-4 md:px-0">
            <div className="flex flex-col md:flex-row items-center">
              <img
                className="h-[45px] md:h-[55px]"
                src="/images/index/ais.svg"
                alt="Поддержка open source программ Anthropic и OpenAI"
              />

              <div className="flex justify-center text-base md:text-xl mt-4 md:mt-0 md:ml-10">
                <div className="max-w-[370px] text-gray-400 text-center md:text-left">
                  Проект поддержан open source программами Anthropic и OpenAI.{" "}
                  <a
                    href="/ru/faq#oss-programs"
                    target="_blank"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Подробнее&nbsp;→
                  </a>
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>

      {/* FEATURES OVERVIEW SECTION */}
      <section id="features" className="pb-12 md:pb-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Обзор</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Возможности
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              В Databasus есть все для надежного управления бекапами в
              продакшене: от запуска по расписанию до шифрования. Подходит и
              соло-разработчикам с личными проектами, и DevOps-командам, и
              крупным компаниям
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          {/* Feature Cards Grid */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 border border-[#ffffff20] rounded-xl">
            {/* Card 1: Scheduled backups */}
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                1
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Запуск по расписанию
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Бекапы по расписанию"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Бекапы нужно делать регулярно и в заданное время. Поэтому вы
                можете выбрать любой из вариантов: каждый час, день, неделю,
                месяц, cron и другие
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Проверка состояния базы
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Проверки состояния"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Каждую минуту (или с любым другим интервалом) система будет
                пинговать вашу базу данных и показывать историю попыток
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                Базу можно считать недоступной, например, после 3-х неудачных
                попыток. Когда база снова заработает, вы тоже получите
                уведомление
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Разные хранилища для бекапов
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Файлы хранятся на VPS, в облачных хранилищах и облачных дисках.
                Выбирайте любое хранилище. Файлы всегда принадлежат только вам.{" "}
                <a
                  href="/ru/storages"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Смотреть все →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-destinations.svg"
                  alt="Хранилища"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 4: Notifications */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                4
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Уведомления
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Вы можете получать уведомления об успехе или сбое процесса.
                Лично вам в мессенджер или в Slack DevOps-командам.{" "}
                <a
                  href="/ru/notifiers"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Смотреть все →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-notifications.svg"
                  alt="Уведомления"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 5: Self hosted via Docker */}
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                5
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Self-hosted через Docker
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Databasus работает на вашем компьютере или VPS, поэтому все
                данные принадлежат вам и защищены. Развертывание занимает около
                2-х минут через скрипт, Docker или k8s
              </p>

              <div className="flex">
                <img
                  src="/images/index/feature-deploy.svg"
                  alt="Docker"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 6: Open source and free */}
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                6
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Open source и бесплатно
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Проект полностью открыт, бесплатен и распространяется под
                лицензией Apache 2.0. Вы можете копировать и форкать код.
                Подходит крупным компаниям с комплаенсом
              </p>
              <div>
                <img
                  src="/images/index/feature-github.svg"
                  alt="GitHub"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 7: Restore verification - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Restore verification */}
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Проверка восстановления
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Бекап, завершившийся без ошибок, и бекап, из которого реально
                  можно восстановиться — не одно и то же. Databasus периодически
                  берет последний бекап, восстанавливает его во временный
                  контейнер с базой данных и сообщает результат.{" "}
                  <a
                    href="/ru/restore-verification"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Подробнее →
                  </a>
                </p>

                <div>
                  <img
                    src="/images/index/feature-postgresql.svg"
                    alt="PostgreSQL"
                    loading="lazy"
                  />
                </div>
              </div>

              {/* Card 10: Security - Only visible on desktop, merged with card 7 */}
              <div className="hidden lg:block p-5 md:p-6">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  10
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Безопасность
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Шифрование корпоративного уровня защищает чувствительные
                  данные и бекапы. Доступ к базе только на чтение исключает
                  порчу данных. Все это работает из коробки и не требует
                  специальных знаний.{" "}
                  <a
                    href="/ru/security"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Подробнее →
                  </a>
                </p>

                <div>
                  <img
                    src="/images/index/feature-encryption.svg"
                    alt="Безопасность"
                    loading="lazy"
                  />
                </div>
              </div>
            </div>

            {/* Card 8: Access management */}
            <div className="border-b md:border-r lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20]">
                  8
                </div>
              </div>

              <div className="flex flex-wrap items-center mb-4 md:mb-5">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold">
                  Разные роли
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  для команд
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-access-management.svg"
                  alt="Управление доступом"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Выдавайте пользователям доступ на просмотр или управление
                базами. Разделяйте команды и проекты. Подходит DevOps-командам и
                компаниям.{" "}
                <a
                  href="/ru/access-management#settings"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Подробнее →
                </a>
              </p>
            </div>

            {/* Card 9: Audit logs */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20]">
                  9
                </div>
              </div>

              <div className="flex flex-wrap items-center mb-4 md:mb-5">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold">
                  Журнал аудита
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  для команд
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-audit-logs.svg"
                  alt="Журнал аудита"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Отслеживайте все действия в системе через подробный журнал
                аудита. По каждому пользователю видна история доступа и
                изменений (скачивания бекапов, изменения расписаний, обновления
                настроек и т.д.).{" "}
                <a
                  href="/ru/access-management#audit-logs"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Подробнее →
                </a>
              </p>
            </div>

            {/* Card 10: Security - Mobile/Tablet only */}
            <div className="border-b border-[#ffffff20] p-5 md:p-6 col-span-1 lg:hidden">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                10
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Безопасность
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Шифрование корпоративного уровня защищает чувствительные данные
                и бекапы. Доступ к базе только на чтение исключает порчу данных.
                Все это работает из коробки и не требует специальных знаний.{" "}
                <a
                  href="/ru/security"
                  className="text-blue-500 hover:text-blue-600 font-medium"
                >
                  Подробнее →
                </a>
              </p>

              <div>
                <img
                  src="/images/index/feature-encryption.svg"
                  alt="Безопасность"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 11: Backup types and modes */}
            <div className="col-span-1 md:col-span-2 lg:col-span-2 p-5 md:p-6 flex flex-col md:flex-row gap-4 md:gap-6">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20] shrink-0">
                11
              </div>

              <div>
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Логические, физические, инкрементальные и WAL-бекапы
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus поддерживает логические и физические (полные и
                  инкрементальные) бекапы со стримингом WAL для Point-in-Time
                  Recovery. Это делает Databasus подходящим для аварийного
                  восстановления. Бонусом: это одинаково хорошо работает и с
                  self-hosted, и с облачными базами — для облачных и публично
                  доступных баз используется удаленный режим. Физические бекапы
                  построены на нативных бекапах PG 17.{" "}
                  <a
                    href="/ru/faq/#pitr"
                    className="text-blue-500 hover:text-blue-600 font-medium"
                  >
                    Подробнее →
                  </a>
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* VIDEO SECTION */}
      <section className="pb-12 md:pb-20 px-4 md:px-6 lg:px-0" id="how-to-use">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            {/* Left side: Info */}
            <div className="w-full lg:w-[450px] lg:shrink-0">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Обзор за 4 минуты</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Как пользоваться Databasus?
              </h2>

              <p className="text-gray-200 max-w-[450px] leading-relaxed mb-6 md:mb-8 text-sm sm:text-base">
                Посмотрите в видео, как подключить базу данных, настроить бекапы
                по расписанию, скачать и восстановить бекап, добавить участников
                команды и что такое журнал аудита пользователей
              </p>

              <a
                href="https://rostislav-dugin.com"
                target="_blank"
                className="flex items-center gap-3 md:gap-4 hover:opacity-70 transition-colors"
              >
                <img
                  src="/images/index/rostislav.png"
                  alt="Ростислав Дугин"
                  className="w-10 h-10 md:w-12 md:h-12 rounded-full object-cover"
                  loading="lazy"
                />

                <div>
                  <p className="font-medium text-base md:text-lg">
                    Ростислав Дугин
                  </p>
                  <p className="text-sm text-gray-400">Разработчик Databasus</p>
                </div>
              </a>
            </div>

            {/* Right side: Video */}
            <div className="flex-1 relative">
              <div className="rounded-lg overflow-hidden shadow-lg border border-[#ffffff20]">
                <LiteYouTubeEmbed
                  videoId="1qsAnijJfJE"
                  title="Как пользоваться Databasus (обзор)?"
                  thumbnailSrc="/images/index/how-to-use-preview.svg"
                />
              </div>
            </div>
          </div>
        </div>
      </section>

      <div className="border-b border-[#ffffff20] max-w-[calc(100%-2rem)] md:max-w-[calc(100%-3rem)] lg:max-w-[1000px] 2xl:max-w-[1200px] mx-auto" />

      {/* Databases section */}
      <section className="pt-12 md:pt-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-10 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Базы данных</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Поддерживаемые базы данных
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              Databasus умеет бекапить и восстанавливать PostgreSQL, MySQL,
              MariaDB и MongoDB — все одним инструментом. Основной фокус —
              PostgreSQL, остальные базы идут бонусом
            </p>
          </div>

          {/* Databases list */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 border border-[#ffffff20] rounded-xl">
            {/* PostgreSQL */}
            <div className="border-b md:border-r lg:border-b-0 border-[#ffffff20] p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-postgresql.svg"
                    alt="PostgreSQL"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                PostgreSQL
              </h3>

              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                PostgreSQL — основная база данных Databasus. Поддерживаются все
                версии с 12 по 18
              </p>
            </div>

            {/* MySQL */}
            <div className="border-b lg:border-r lg:border-b-0 border-[#ffffff20] p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-mysql.svg"
                    alt="MySQL"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                MySQL
              </h3>

              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                MySQL — вторая по популярности база данных в мире. Бекапить и
                восстанавливать базы MySQL можно так же просто
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/ru/mysql-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Подробнее →
                </a>
              </div>
            </div>

            {/* MariaDB */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-mariadb.svg"
                    alt="MariaDB"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                MariaDB
              </h3>

              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                MariaDB поддерживается с теми же возможностями, что и MySQL.
                Бекапы и восстановление баз MariaDB работают без лишних настроек
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/ru/mysql-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Подробнее →
                </a>
              </div>
            </div>

            {/* MongoDB */}
            <div className="p-5 md:py-6 md:px-5 flex flex-col">
              <div className="flex items-center justify-center mb-4 md:mb-6">
                <div className="text-5xl md:text-6xl">
                  <img
                    src="/images/index/database-mongodb.svg"
                    alt="MongoDB"
                    className="w-[75px] h-[75px]"
                    loading="lazy"
                  />
                </div>
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3 md:mb-4 text-center">
                MongoDB
              </h3>

              <p className="text-gray-400 text-sm md:text-base text-center mb-4">
                MongoDB — самая популярная NoSQL база данных. Бекапить и
                восстанавливать базы MongoDB можно через тот же удобный
                интерфейс
              </p>

              <div className="text-center mt-auto">
                <a
                  href="/ru/mongodb-backup"
                  className="text-blue-500 hover:text-blue-600 font-medium text-sm md:text-base"
                >
                  Подробнее →
                </a>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* PROCESS SECTION */}
      <section className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-10 md:mb-16">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">Процесс</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Как делать бекапы?
            </h2>

            <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
              Главный приоритет Databasus — простота: сейчас это самый простой
              инструмент резервного копирования PostgreSQL в мире. Чтобы делать
              бекапы, нужно пройти 4 шага. После этого восстановиться можно
              будет в один клик
            </p>
          </div>

          {/* Steps */}
          <div className="space-y-6 md:space-y-10 max-w-[1000px] mx-auto">
            {/* Step 1 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Шаг 1
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Выберите нужное расписание
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Выбирайте любое время: каждый день, неделю, месяц,
                    конкретное время (например, 4 утра) или cron-выражения
                  </p>
                  <p>
                    Для недельного интервала нужно указать конкретный день
                    недели, для месячного — конкретное число
                  </p>
                  <p>
                    Если база большая, рекомендуем выбирать время, когда трафик
                    спадает
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Шаг 1"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 2 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Шаг 2
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Введите данные вашей базы
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Введите учетные данные вашей базы PostgreSQL, выберите
                    версию и целевую базу. Также укажите, нужен ли SSL
                  </p>
                  <p>
                    По умолчанию Databasus сжимает бекапы на сбалансированном
                    уровне: бекап идет примерно на 20% дольше, зато файл выходит
                    в 4-8 раз меньше (заодно меньше сетевой трафик)
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-2.svg"
                  alt="Шаг 2"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 3 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Шаг 3
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Выберите хранилище для бекапов
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Файлы бекапов можно хранить локально, в S3, Google Drive,
                    NAS, Dropbox и других сервисах
                  </p>
                  <p>
                    Учитывайте, что в хранилище должно быть достаточно места
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-3.svg"
                  alt="Шаг 3"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Step 4 */}
            <div className="flex flex-col lg:flex-row gap-4 md:gap-8 items-start rounded-lg border border-[#ffffff20] p-4 md:p-6">
              <span className="px-3 py-1 rounded-lg bg-white text-black font-medium text-sm shrink-0">
                Шаг 4
              </span>

              <div className="w-full lg:w-[400px] lg:shrink-0">
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-3">
                  Выберите, куда получать уведомления (необязательно)
                </h3>

                <div className="space-y-3 max-w-[370px] text-gray-400 text-sm md:text-base">
                  <p>
                    Когда бекап завершился успешно или упал, Databasus может
                    отправить вам уведомление: в чат с DevOps, на почту или даже
                    в вебхук вашей команды
                  </p>
                  <p>
                    Мы планируем поддержать большинство популярных мессенджеров
                    и платформ
                  </p>
                </div>
              </div>

              <div className="flex-1 w-full lg:pl-10">
                <img
                  src="/images/index/backup-step-4.svg"
                  alt="Шаг 4"
                  className="w-full"
                  loading="lazy"
                />
              </div>
            </div>
          </div>

          {/* CTA Button */}
          <div className="text-center mt-8 md:mt-12">
            <a
              href="#installation"
              className="inline-flex items-center gap-2 px-6 py-3 bg-white text-black rounded-lg text-[15px] font-medium hover:opacity-70 transition-colors"
            >
              Начать
              <svg
                aria-hidden={true}
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
              >
                <path d="M5 12h14M12 5l7 7-7 7" />
              </svg>
            </a>
          </div>
        </div>
      </section>

      {/* INSTALLATION SECTION */}
      <section id="installation" className="px-4 md:px-6 lg:px-0">
        <div className="max-w-[1000px] 2xl:max-w-[1200px] mx-auto border border-[#ffffff20] rounded-xl py-10 md:py-20 px-4 md:px-6">
          <div className="max-w-[1100px] mx-auto">
            <div className="text-center mb-8 md:mb-10">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">Начало работы</span>
              </div>

              <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
                Как установить?
              </h2>

              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
                Databasus поддерживает разные способы установки. Все варианты
                предельно просты даже для тех, у кого нет опыта в
                администрировании или DevOps. Поддерживаются и локальные, и
                облачные базы данных.
              </p>
            </div>

            <InstallationComponent lang="ru" />
          </div>
        </div>
      </section>

      {/* FAQ SECTION */}
      <section id="faq" className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="text-center mb-8 md:mb-12">
            <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
              <span className="text-sm font-medium">FAQ</span>
            </div>

            <h2 className="text-3xl md:text-4xl lg:text-5xl font-bold mb-4 md:mb-6">
              Частые вопросы
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Цель Databasus — сделать резервное копирование максимально простым
              и для самостоятельных разработчиков (или DevOps), и для целых
              команд. Интерфейс позволяет легко создавать бекапы, наглядно
              показывает прогресс и восстанавливает что угодно в пару кликов
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="Что такое Databasus и почему он лучше самописных скриптов?"
              answer="Databasus — это self-hosted сервис для резервного копирования баз данных под лицензией Apache 2.0. От shell-скриптов он отличается веб-интерфейсом: запуск по расписанию, сжатие, сохранение архивов сразу в несколько хранилищ (локальный диск, S3, Google Drive, NAS, Dropbox, SFTP, rclone и другие), автоудаление старых бекапов по политикам хранения и уведомления об успехе или сбое — и все это без единой строчки самописного кода"
            />
            <FaqItem
              number="2"
              question="Как установить Databasus быстрее всего?"
              answer="Databasus поддерживает несколько способов установки: автоматический скрипт, Docker, Docker Compose и Kubernetes с Helm. Самый короткий путь — установочный скрипт в одну строку через cURL: он скачивает актуальный Docker-образ, создает docker-compose.yml и запускает сервис с автоперезапуском после перезагрузки. Для Kubernetes есть официальный Helm-чарт для продакшен-развертываний. На типичном VPS все занимает меньше 2-х минут."
            />
            <FaqItem
              number="3"
              question="Как работает проверка восстановления?"
              answer="Databasus запускает небольшой агент проверки на вашем сервере. При каждом запуске по расписанию агент скачивает последний бекап, восстанавливает его во временный контейнер с базой данных и сверяет восстановленную базу с исходной. Результат приходит вам: код завершения восстановления и число строк по каждой таблице. Расписания: после бекапа, каждый час, день, неделю, месяц или cron-выражение в UTC. О сбоях можно узнавать через любой канал уведомлений, подключенный к базе — Slack, Teams, Discord, почту и другие."
            />
            <FaqItem
              number="4"
              question="Как Databasus обеспечивает безопасность?"
              answer={
                <>
                  Databasus обеспечивает безопасность на трех уровнях: (1)
                  шифрование чувствительных данных — все пароли, токены и
                  учетные данные шифруются AES-256-GCM и хранятся отдельно от
                  базы данных; (2) шифрование бекапов — каждый файл бекапа
                  шифруется уникальным ключом, выведенным из мастер-ключа, ID
                  бекапа и случайной соли, так что без вашего ключа бекапы
                  бесполезны, даже если кто-то получит доступ к хранилищу; (3)
                  доступ к базе только на чтение — Databasus требует лишь права
                  SELECT и тщательно проверяет отсутствие прав на запись, что
                  защищает данные от порчи даже при компрометации инструмента.
                  <br />
                  <br />
                  Сам вход тоже можно закрыть вторым фактором: когда он включен,
                  после верного пароля на почту аккаунта приходит шестизначный
                  код, а вход через Google или GitHub по-прежнему опирается на
                  проверки самих провайдеров.
                  <br />
                  <br />
                  Помимо этого, безопасность и надежность встроены в каждый
                  релиз: статический анализ CodeQL, поиск секретов через
                  gitleaks, проверка правил безопасности через semgrep,
                  мониторинг CVE через Dependabot, сканирование образов и
                  Dockerfile через Trivy и периодические аудиты Codex Security
                  от OpenAI. Интеграционные тесты гоняются на реальных
                  контейнерах PostgreSQL, MySQL, MariaDB и MongoDB и проверяют
                  полный цикл «бекап, затем восстановление» перед каждым
                  релизом. GitHub Actions закреплены на SHA коммитов, а
                  воркфлоу следуют принципу минимальных привилегий.
                  <br />
                  <br />
                  Весь конвейер описан в разделе{" "}
                  <a
                    href="/ru/security#security-and-reliability-engineering"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Проектирование безопасности и надежности
                  </a>
                  .
                  <br />
                  <br />
                  Кроме того, все логи можно выгружать в любую внешнюю систему
                  (VictoriaLogs, SigNoz, Graylog и другие) по стандарту
                  OpenTelemetry. По умолчанию логи (включая журнал аудита) также
                  пишутся в локальные файлы, поэтому записи аудита не теряются.
                  Подробности —{" "}
                  <a
                    href="/ru/advanced-config#logging"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    в расширенной настройке
                  </a>
                  .
                </>
              }
            />
            <FaqItem
              number="5"
              question="Как настроить и запустить первый бекап в Databasus?"
              answer={
                <>
                  Чтобы сделать первый бекап, войдите в панель, нажмите New
                  Backup и выберите интервал — каждый час, день, неделю, месяц
                  или cron. Затем укажите точное время запуска (например, 02:30
                  — часы минимальной нагрузки).
                  <br />
                  <br />
                  Дальше введите хост PostgreSQL, порт, имя базы, учетные данные
                  и настройки SSL. Выберите, куда отправлять архив (локальный
                  путь, бакет S3, папка Google Drive, Dropbox и так далее).{" "}
                  <br />
                  <br />
                  При необходимости добавьте каналы уведомлений — почту, Slack,
                  Telegram или вебхук — и нажмите Save. Databasus сразу проверит
                  подключение, включит расписание, сделает первый бекап и
                  пришлет статус. Из готового бекапа восстановиться можно в один
                  клик.
                </>
              }
            />
            <FaqItem
              number="6"
              question="Насколько широко используется Databasus?"
              answer="Сегодня Databasus — самый распространенный open source инструмент резервного копирования PostgreSQL. На 17 июня 2026 года его скачали из Docker больше 1 800 000 раз — DBA, DevOps-инженеры, разработчики и команды по всему миру. У проекта 8 500+ звезд на GitHub: больше, чем у pgBackRest (~4 200 звезд, существует с 2014 года) и WAL-G (~4 100 звезд, с 2017 года). Databasus вышел в 2025 году и обогнал оба проекта за первый же год."
            />
            <FaqItem
              number="7"
              question="Чем Databasus отличается от pgBackRest, Barman и pg_dump? Где почитать сравнения?"
              answer={
                <>
                  Databasus делает ставку на простоту: вместо конфигов и
                  консольных утилит — современный веб-интерфейс, где под
                  управлением сразу все ваши базы. В отличие от голых скриптов
                  на pg_dump, тут из коробки расписания, сжатие, разные
                  хранилища, мониторинг состояния и мгновенные уведомления.
                  <br />
                  <br />
                  При этом, в отличие от pgBackRest и WAL-G, физические,
                  инкрементальные и WAL-бекапы Databasus строит на нативном
                  механизме PostgreSQL 17, а не изобретает собственный движок
                  резервного копирования. Он подключается к базам удаленно и
                  дотягивается до закрытых сетей через SSH-туннель к серверу или
                  бастиону, так что базы без публичного доступа тоже можно
                  бекапить и администрировать из одной панели.{" "}
                  <a
                    href="/ru/faq/#pitr"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Читайте, как устроены физические и PITR-бекапы
                  </a>
                  .
                  <br />
                  <br />У нас есть подробные страницы сравнения с популярными
                  инструментами резервного копирования:{" "}
                  <a
                    href="/ru/pgdump-alternative"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus против pg_dump
                  </a>
                  ,{" "}
                  <a
                    href="/ru/databasus-vs-pgbackrest"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus против pgBackRest
                  </a>
                  ,{" "}
                  <a
                    href="/ru/databasus-vs-barman"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus против Barman
                  </a>
                  ,{" "}
                  <a
                    href="/ru/databasus-vs-wal-g"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus против WAL-G
                  </a>{" "}
                  и{" "}
                  <a
                    href="/ru/databasus-vs-pgbackweb"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Databasus против pgBackWeb
                  </a>
                  . В каждом — ключевые отличия, плюсы и минусы, чтобы было
                  проще выбрать инструмент под ваши задачи.
                </>
              }
            />
            <FaqItem
              number="8"
              question="Правда, что Anthropic и OpenAI поддержали Databasus?"
              answer={
                <>
                  Да. В марте 2026 года Databasus приняли и в{" "}
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Claude for Open Source
                  </a>{" "}
                  от Anthropic, и в{" "}
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Codex for Open Source
                  </a>{" "}
                  от OpenAI. Для нас это независимая оценка: проект признали
                  критически важной инфраструктурой, которую стоит поддерживать.{" "}
                  <a
                    href="/ru/faq#oss-programs"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    Подробнее →
                  </a>
                </>
              }
            />
            <FaqItem
              number="9"
              question="Является ли Databasus альтернативой pg_dump?"
              answer="Не совсем. Databasus сосредоточен на аварийном восстановлении с низкими RTO и RPO, поэтому он скорее альтернатива pgBackRest или WAL-G — только с простотой уровня pg_dump. При этом логические бекапы он действительно делает через pg_dump под капотом, добавляя веб-интерфейс, запуск по расписанию, разные хранилища, уведомления, мониторинг состояния и шифрование. Логические бекапы доступны также для MySQL, MariaDB и MongoDB."
            />
            <FaqItem
              number="10"
              question="Какие базы данных поддерживает Databasus?"
              answer={
                <>
                  Databasus поддерживает PostgreSQL, MySQL, MariaDB и MongoDB.
                  При этом изначально Databasus создавался именно для PostgreSQL
                  и держит на нем основной фокус: PostgreSQL поддержан
                  полностью, и бекапы для него самые эффективные.
                  <br />
                  <br />
                  MySQL, MariaDB и MongoDB поддерживаются, но приоритетом
                  остается PostgreSQL: самые проработанные возможности и
                  активная разработка именно там.
                  <br />
                  <br />
                  Например, Databasus нативно поддерживает физические и
                  WAL-бекапы для аварийного восстановления PostgreSQL. По сути
                  Databasus — это инструмент резервного копирования PostgreSQL,
                  а остальные СУБД идут бонусом.
                </>
              }
            />
            <FaqItem
              number="11"
              question="Какие типы бекапов поддерживает Databasus?"
              answer={
                <>
                  Databasus поддерживает физические, полные, инкрементальные,
                  WAL и логические бекапы — так что он подходит и тем, кому
                  нужны простые логические дампы, и тем, кому нужен серьезный
                  инструмент аварийного восстановления.
                  <ul className="list-disc list-inside mt-3 space-y-2">
                    <li>
                      <strong>Физический</strong> — копия всего кластера базы
                      данных на уровне файлов. Для больших объемов данных
                      быстрее логических дампов и при копировании, и при
                      восстановлении. Построен на нативном механизме бекапов
                      PostgreSQL 17: мы опираемся на проверенный инструментарий
                      самого PostgreSQL, а не изобретаем свой
                    </li>
                    <li>
                      <strong>Полный</strong> — самодостаточная копия кластера,
                      основа, с которой начинается каждая цепочка бекапов
                    </li>
                    <li>
                      <strong>Инкрементальный</strong> — хранит только изменения
                      с прошлого бекапа, поэтому бекапы остаются маленькими и
                      быстрыми
                    </li>
                    <li>
                      <strong>Стриминг WAL</strong> — непрерывно пишет поток
                      изменений базы и дает Point-in-Time Recovery (PITR). Это
                      про аварийное восстановление с почти нулевой потерей
                      данных
                    </li>
                    <li>
                      <strong>Логический</strong> — нативный дамп базы в
                      бинарном формате конкретного движка. Сжимается и
                      передается в хранилище потоком без промежуточных файлов
                    </li>
                  </ul>
                  <br />
                  Физические, инкрементальные и WAL-бекапы построены на нативном
                  механизме PostgreSQL 17, поэтому требуют PostgreSQL 17 или
                  новее; на старых версиях доступны только логические бекапы.
                  Это осознанное решение: большинство продакшен-баз уже работают
                  на PostgreSQL 17 и выше, а старые версии за пару лет доживают
                  до конца поддержки. Databasus стремится стать стандартным
                  инструментом резервного копирования для баз данных начиная с
                  PostgreSQL 17.
                  <br />
                  <br />
                  Все типы бекапов умеют работать через встроенный SSH-туннель —
                  базу вообще не нужно выставлять наружу.
                </>
              }
            />
            <FaqItem
              number="12"
              question="Как присоединиться к сообществу Databasus?"
              answer={
                <>
                  Присоединяйтесь к нашему большому сообществу разработчиков,
                  DBA и DevOps-инженеров:{" "}
                  <a
                    href="https://t.me/databasus_community"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    t.me/databasus_community
                  </a>
                  . Там можно задать вопрос, поделиться опытом, получить помощь
                  с настройкой и следить за релизами.
                </>
              }
            />
            <FaqItem
              number="13"
              question="Насколько распространен Databasus?"
              answer={
                <>
                  У Databasus более 1,8 миллиона скачиваний Docker и 8,5 тысяч
                  звезд на GitHub. Для сравнения: у pgBackRest и WAL-G около 4,2
                  тысячи звезд у каждого, у Barman — около 3,1 тысячи, что
                  делает Databasus самым популярным инструментом резервного
                  копирования баз данных на GitHub.
                  <br />
                  <br />
                  Он принят в open source программы Anthropic и OpenAI как
                  критически важный проект. Сегодня Databasus используют
                  компании, команды и DevOps-инженеры, а за проектом стоит
                  большое активное сообщество.
                  <br />
                  <br />
                  Databasus разрабатывается и используется с 2023 года, а как
                  open source широко разошелся с начала 2025-го. Он давно живет
                  в реальном продакшене и обкатан на множестве граничных
                  случаев. Важно и то, что Databasus не придумывает собственных
                  способов копировать ваши данные — он опирается на нативную,
                  проверенную реализацию PostgreSQL.
                  <br />
                  <br />
                  Наша цель — стать стандартным инструментом резервного
                  копирования PostgreSQL начиная с 17-й версии: Databasus первым
                  построен на нативном и теперь уже стандартном протоколе
                  бекапов PostgreSQL, а не на собственной реализации.
                </>
              }
            />
            <FaqItem
              number="14"
              question="Я забыл email или пароль администратора"
              answer={
                <>
                  Первая учетная запись, созданная на инстансе, им и управляет.
                  Если вы не помните, какой это адрес, выполните docker exec -it
                  databasus ./main --list-admins на сервере, где запущен
                  Databasus — команда перечислит все учетные записи
                  администраторов с их email, именем, датой создания и
                  состоянием активности и не выведет ни одного пароля. Чтобы
                  задать новый пароль этой учетной записи, выполните docker exec
                  -it databasus ./main
                  --new-password=&quot;YourNewSecurePassword123&quot;
                  --email=&quot;owner@example.com&quot;. Инстанс, созданный до
                  того, как первая учетная запись стала им управлять, до сих пор
                  носит адрес-заглушку admin, пока владелец его не заменит. Если
                  инстанс еще и требует код при входе, а почта перестала его
                  доставлять, выполните docker exec -it databasus ./main
                  --disable-2fa, чтобы инстанс перестал запрашивать код. Все три
                  команды описаны на
                  <a
                    href="/ru/password"
                    className="text-blue-400 hover:text-blue-600"
                  >
                    странице про пароль
                  </a>
                  .
                </>
              }
            />
          </div>
        </div>
      </section>

      <LandingFooterComponent lang="ru" />
    </div>
  );
}

function FaqItem({
  number,
  question,
  answer,
}: {
  number: string;
  question: string;
  answer: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-[#ffffff20] p-4 md:p-6">
      <div className="flex items-center justify-center w-6 h-6 rounded border border-[#ffffff20] text-sm font-semibold mb-3 md:mb-4">
        {number}
      </div>

      <h3 className="text-base md:text-lg font-bold mb-2 md:mb-3">
        {question}
      </h3>

      <div className="text-gray-400 text-sm md:text-base">{answer}</div>
    </div>
  );
}
