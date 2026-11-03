import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "Резервное копирование MySQL с поддержкой MariaDB",
  description:
    "Бесплатный open source инструмент для регулярных бекапов MySQL и MariaDB. Альтернатива mysqldump с веб-интерфейсом, облачными хранилищами (S3, Google Drive), уведомлениями (Slack, Discord, Telegram) и шифрованием AES-256.",
  keywords:
    "резервное копирование MySQL, резервное копирование MariaDB, альтернатива mysqldump, mysqldump GUI, автоматизация бекапов MySQL, инструмент для бекапа MySQL, инструмент для бекапа MariaDB, бекап MySQL по расписанию, облачный бекап MySQL, бекап MySQL в S3, бекап MySQL в Docker, шифрование бекапов MySQL",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("ru", "mysql-backup"),
    languages: getLanguageAlternates("mysql-backup"),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("ru", "mysql-backup"),
    title: "Резервное копирование MySQL с поддержкой MariaDB",
    description:
      "Бесплатный open source инструмент для регулярных бекапов MySQL и MariaDB. Альтернатива mysqldump с веб-интерфейсом, облачными хранилищами, уведомлениями и шифрованием.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Интерфейс Databasus для управления резервным копированием MySQL",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary_large_image",
    title: "Резервное копирование MySQL с поддержкой MariaDB",
    description:
      "Бесплатный open source инструмент для регулярных бекапов MySQL и MariaDB. Альтернатива mysqldump с веб-интерфейсом, облачными хранилищами, уведомлениями и шифрованием.",
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

export default function MysqlBackupPage() {
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
              "Бесплатный open source инструмент для регулярных бекапов MySQL и MariaDB. Альтернатива mysqldump с веб-интерфейсом, облачными хранилищами, уведомлениями и шифрованием.",
            url: "https://databasus.com/ru/mysql-backup/",
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
              "Бекапы MySQL и MariaDB по расписанию",
              "Разные хранилища (S3, Google Drive, Dropbox, SFTP, rclone и другие)",
              "Мгновенные уведомления (Slack, Telegram, Discord, Webhook, email и другие)",
              "Мониторинг доступности базы данных MySQL",
              "Self-hosted установка через Docker",
              "Открытый исходный код, бесплатно",
              "Поддержка MySQL 5.7, 8.0, 8.4, 9, 26",
              "Поддержка MariaDB 5.5, 10, 11, 12, 13",
              "Сжатие бекапов и шифрование AES-256-GCM",
              "Альтернатива mysqldump с веб-интерфейсом",
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
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "Что такое Databasus и как он делает бекапы MySQL?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus — это self-hosted инструмент резервного копирования с лицензией Apache 2.0, который использует mysqldump для создания консистентных бекапов MySQL. Он дополняет mysqldump современным веб-интерфейсом, расписаниями, интеграцией с облачными хранилищами, мгновенными уведомлениями и шифрованием AES-256-GCM — самописные shell-скрипты и cron-задачи больше не нужны.",
                },
              },
              {
                "@type": "Question",
                name: "Databasus — это альтернатива или замена mysqldump?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus расширяет mysqldump, а не заменяет его. Внутри он запускает mysqldump с оптимальными параметрами (--single-transaction, --routines, --triggers, --events), а затем сжимает дамп, шифрует его и загружает в настроенное хранилище. Вы получаете всю надежность mysqldump плюс расписания, облачные хранилища, уведомления и командную работу через веб-интерфейс.",
                },
              },
              {
                "@type": "Question",
                name: "Какие версии MySQL поддерживает Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus поддерживает MySQL 5.7, 8.0, 8.4, 9 и 26. Для MySQL 8.0 и новее Databasus использует сетевое сжатие zstd для более быстрой передачи. Для MySQL 5.7 автоматически используется классическое сжатие. MySQL 5.7 поддерживается только на x86-64.",
                },
              },
              {
                "@type": "Question",
                name: "Какие версии MariaDB поддерживает Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus поддерживает MariaDB 5.5, 10, 11, 12 и 13. Для бекапов MariaDB Databasus использует родной инструмент mariadb-dump вместо mysqldump — это дает полную совместимость с особенностями и оптимизациями MariaDB.",
                },
              },
              {
                "@type": "Question",
                name: "Можно ли делать бекапы MySQL в AWS RDS, Google Cloud SQL или Azure?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Да, Databasus работает с облачными базами MySQL, включая AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL и PlanetScale. Поскольку Databasus делает логические бекапы через mysqldump, ему достаточно стандартных реквизитов подключения к MySQL — специальные облачные права не нужны.",
                },
              },
              {
                "@type": "Question",
                name: "Как Databasus защищает учетные данные MySQL и бекапы?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus использует многоуровневую защиту: (1) все пароли и учетные данные MySQL шифруются AES-256-GCM перед сохранением; (2) каждый файл бекапа шифруется уникальным ключом, выведенным из мастер-ключа, ID бекапа и случайной соли; (3) реквизиты передаются через временные файлы .my.cnf с правами 0600 и никогда не попадают в командную строку или логи.",
                },
              },
            ],
          }),
        }}
      />

      <LandingHeaderComponent
        lang="ru"
        sectionLinks={[
          { href: "#features", label: "Возможности" },
          { href: "#mysql-backup", label: "MySQL" },
          { href: "#mariadb-backup", label: "MariaDB" },
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

            <h1 className="text-2xl sm:text-4xl sm:max-w-[400px] md:text-4xl leading-tight font-bold mb-4 md:mb-6 mx-auto md:max-w-[500px]">
              Резервное копирование MySQL с поддержкой MariaDB
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus — бесплатный self-hosted инструмент с открытым исходным
              кодом для бекапа баз данных MySQL и MariaDB. Настраивайте
              автоматические дампы по расписанию, храните их в S3, Google Drive
              или локально. Получайте уведомления в Slack, Discord или Telegram
              о завершении бекапов
            </p>

            <div className="flex flex-col sm:flex-row sm:flex-wrap items-center justify-center gap-2 sm:gap-2 max-w-[400px] mx-auto pb-0 sm:pb-[50px] lg:pb-0">
              <a
                href="#installation"
                className="w-full sm:flex-1 inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 bg-white rounded-lg text-black font-medium hover:opacity-70 transition-opacity order-3"
              >
                Установить через Docker
              </a>

              <a
                href="https://github.com/databasus/databasus"
                target="_blank"
                rel="noopener noreferrer"
                className="w-full sm:flex-1 inline-flex items-center justify-center gap-2 px-4 py-2 sm:px-5 sm:py-2.5 rounded-lg font-medium border border-[#ffffff20] bg-[#0C0E13] hover:opacity-70 transition-opacity order-4 sm:order-4"
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
          </div>

          {/* Dashboard Screenshot */}
          <div className="relative mx-auto max-w-[1200px]">
            <div>
              <img
                src="/images/index/dashboard.svg"
                alt="Интерфейс Databasus для управления резервным копированием MySQL"
                width={980}
                height={620}
                className="w-full h-auto"
                loading="eager"
                fetchPriority="high"
              />
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
              Возможности резервного копирования MySQL
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus добавляет к mysqldump все, чего ему не хватает:
              расписания, облачные хранилища, мгновенные уведомления и
              шифрование AES-256-GCM. Подходит разработчикам, DevOps-командам и
              организациям, работающим с базами MySQL
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
                Дампы MySQL по расписанию
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Настройка бекапов MySQL по расписанию"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Запускайте mysqldump в удобное время, когда нагрузка на базу
                минимальна. Выбирайте интервалы: каждый час, день, неделю, месяц
                — или задавайте cron-выражения для точного контроля
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Мониторинг доступности MySQL
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Проверки доступности MySQL"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Следите за доступностью подключения к MySQL с помощью
                настраиваемых проверок. Получайте уведомления, когда база
                становится недоступной или восстанавливается
              </p>

              <p className="text-gray-400 text-sm md:text-base">
                Задавайте интервал проверок (каждую минуту, каждые 5 минут и т.
                д.) и порог сбоев, после которого база считается недоступной
              </p>
            </div>

            {/* Card 3: Many destinations to store */}
            <div className="border-b md:border-r lg:border-r-0 border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                3
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Храните дампы MySQL где угодно
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Держите бекапы MySQL локально, в S3-совместимом хранилище,
                Google Drive, Dropbox, на NAS или в других местах. Ваши данные
                остаются под вашим контролем.{" "}
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
                  alt="Хранилища для бекапов MySQL"
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
                Уведомления о бекапах
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Узнавайте, когда бекап MySQL завершился или упал с ошибкой.
                Отправляйте уведомления в чат DevOps-команды или в личные
                каналы.{" "}
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
                  alt="Уведомления о бекапах MySQL"
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
                Запускайте Databasus на собственной инфраструктуре. Все
                реквизиты MySQL и данные бекапов остаются на серверах, которые
                вы контролируете. Установка занимает около 2-х минут: скрипт,
                Docker или Kubernetes
              </p>

              <div className="flex">
                <img
                  src="/images/index/feature-deploy.svg"
                  alt="Установка через Docker"
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
                Открытый код, бесплатно
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4">
                Databasus полностью открыт под лицензией Apache 2.0. Изучайте
                каждую строку кода и делайте форки. Бесплатен
                для личного и корпоративного использования
              </p>
              <div>
                <img
                  src="/images/index/feature-github.svg"
                  alt="Открытый исходный код на GitHub"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 7: Many MySQL versions - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Many MySQL versions */}
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Версии MySQL и MariaDB
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Поддерживаются MySQL 5.7, 8.0, 8.4, 9 и 26, а также MariaDB 5.5, 10,
                  11, 12 и 13 — последние через родной mariadb-dump для полной
                  совместимости
                </p>

                <div>
                  <img
                    src="/images/index/database-mysql.svg"
                    alt="Версии MySQL"
                    className="w-[75px] h-[75px]"
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
                  Реквизиты MySQL шифруются AES-256-GCM перед сохранением.
                  Каждый файл бекапа шифруется уникальным ключом. Пароли
                  передаются через временные конфигурационные файлы и никогда не
                  попадают в командную строку.{" "}
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
                    alt="Безопасность бекапов MySQL"
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
                  Управление доступом
                </h3>

                <div className="px-2 py-1 rounded border border-[#ffffff20] text-sm font-medium ml-2">
                  для команд
                </div>
              </div>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-access-management.svg"
                  alt="Управление доступом к бекапам MySQL"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Определяйте, кто может просматривать базы MySQL и управлять ими.
                Создавайте рабочие пространства для разных проектов. Назначайте
                роли: просмотр, редактирование или администрирование.{" "}
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
                  alt="Журнал аудита бекапов MySQL"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Отслеживайте все действия: скачивание бекапов, изменение
                расписаний, обновление настроек. Видно, кто что и когда сделал —
                для комплаенса и прозрачности.{" "}
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
                Реквизиты MySQL шифруются AES-256-GCM перед сохранением. Каждый
                файл бекапа шифруется уникальным ключом. Пароли передаются через
                временные конфигурационные файлы и никогда не попадают в
                командную строку.{" "}
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
                  alt="Безопасность бекапов MySQL"
                  loading="lazy"
                />
              </div>
            </div>

            {/* Card 11: Suitable for clouds */}
            <div className="col-span-1 md:col-span-2 lg:col-span-2 p-5 md:p-6 flex flex-col md:flex-row gap-4 md:gap-6">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold border border-[#ffffff20] shrink-0">
                11
              </div>

              <div>
                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Работает с облачными базами MySQL
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus подключается к облачным базам MySQL, включая AWS RDS
                  for MySQL, Google Cloud SQL, Azure Database for MySQL и
                  PlanetScale. Поскольку он делает логические бекапы через
                  mysqldump, достаточно стандартных реквизитов подключения —
                  специальные облачные права и доступ к файловой системе не
                  нужны
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* MYSQL BACKUP SECTION */}
      <section
        id="mysql-backup"
        className="py-12 md:py-20 px-4 md:px-6 lg:px-0"
      >
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            {/* Left side: Info */}
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">На основе mysqldump</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Резервное копирование MySQL
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Внутри Databasus работает <strong>mysqldump</strong> —
                  официальная утилита резервного копирования MySQL. При запуске
                  бекапа Databasus выполняет mysqldump с оптимальными
                  параметрами:
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --single-transaction
                    </code>{" "}
                    для консистентных снимков без блокировки таблиц
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --routines
                    </code>{" "}
                    чтобы включить хранимые процедуры и функции
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --triggers
                    </code>{" "}
                    чтобы включить триггеры базы данных
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --events
                    </code>{" "}
                    чтобы включить события по расписанию
                  </li>
                </ul>

                <p className="text-gray-400">
                  Для MySQL 8.0 и новее Databasus включает{" "}
                  <strong>сетевое сжатие zstd</strong>, ускоряющее передачу по
                  медленным каналам. Для MySQL 5.7 автоматически используется
                  классическое сжатие.
                </p>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Поддерживаемые версии MySQL:
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 5.7
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 8.0
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 8.4
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 9
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MySQL 26
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Right side: Image */}
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mysql.svg"
                  alt="База данных MySQL"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Официальный бекап MySQL через mysqldump со сжатием,
                  шифрованием и облачными хранилищами
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      <div className="border-b border-[#ffffff20] max-w-[calc(100%-2rem)] md:max-w-[calc(100%-3rem)] lg:max-w-[1000px] 2xl:max-w-[1200px] mx-auto" />

      {/* MARIADB BACKUP SECTION */}
      <section
        id="mariadb-backup"
        className="py-12 md:py-20 px-4 md:px-6 lg:px-0"
      >
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row-reverse gap-8 lg:gap-16">
            {/* Right side: Info */}
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">
                  Родной инструмент mariadb-dump
                </span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Резервное копирование MariaDB
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Для баз MariaDB Databasus использует родной инструмент{" "}
                  <strong>mariadb-dump</strong> вместо mysqldump. Это
                  гарантирует полную совместимость с особенностями и
                  оптимизациями MariaDB.
                </p>

                <p className="text-gray-400">
                  mariadb-dump запускается с теми же оптимальными параметрами:
                  режим single-transaction для консистентных бекапов, хранимые
                  процедуры, триггеры и события включаются по умолчанию.
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>Полная совместимость с возможностями MariaDB</li>
                  <li>Поддержка подключений по SSL/TLS</li>
                  <li>Сетевое сжатие для быстрой передачи</li>
                  <li>Те же опции шифрования и хранения, что и для MySQL</li>
                </ul>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Поддерживаемые версии MariaDB:
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 5.5
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 10
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 11
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 12
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MariaDB 13
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Left side: Image */}
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mariadb.svg"
                  alt="База данных MariaDB"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Родной бекап MariaDB через mariadb-dump с полной поддержкой ее
                  особенностей
                </p>
              </div>
            </div>
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

              <p className="text-sm sm:text-base md:text-lg text-gray-200 max-w-[550px] mx-auto">
                Databasus поддерживает несколько способов установки. Разверните
                его на VPS, локальной машине или в кластере Kubernetes примерно
                за 2 минуты. Одна и та же установка работает для бекапов MySQL,
                MariaDB, PostgreSQL и MongoDB
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
              Вопросы о резервном копировании MySQL
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Частые вопросы о бекапах баз данных MySQL и MariaDB с Databasus.
              Если остались другие вопросы, присоединяйтесь к нашему сообществу
              в Telegram
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="Что такое Databasus и как он делает бекапы MySQL?"
              answer="Databasus — это self-hosted инструмент резервного копирования с лицензией Apache 2.0, который использует mysqldump для создания консистентных бекапов MySQL. Он дополняет mysqldump современным веб-интерфейсом, расписаниями, интеграцией с облачными хранилищами (S3, Google Drive, Dropbox), мгновенными уведомлениями (Slack, Discord, Telegram) и шифрованием AES-256-GCM — самописные shell-скрипты и cron-задачи больше не нужны."
            />
            <FaqItem
              number="2"
              question="Databasus — это альтернатива или замена mysqldump?"
              answer="Databasus расширяет mysqldump, а не заменяет его. Внутри он запускает mysqldump с оптимальными параметрами (--single-transaction, --routines, --triggers, --events), а затем сжимает дамп, шифрует его и загружает в настроенное хранилище. Вы получаете всю надежность mysqldump плюс расписания, облачные хранилища, уведомления и командную работу через веб-интерфейс."
            />
            <FaqItem
              number="3"
              question="Какие версии MySQL поддерживает Databasus?"
              answer="Databasus поддерживает MySQL 5.7, 8.0, 8.4, 9 и 26. Для MySQL 8.0 и новее Databasus использует сетевое сжатие zstd, ускоряющее передачу по медленным каналам. Для MySQL 5.7 автоматически используется классическое сжатие. MySQL 5.7 поддерживается только на x86-64. Все версии получают одинаковый набор функций: расписания, шифрование, облачные хранилища и уведомления."
            />
            <FaqItem
              number="4"
              question="Какие версии MariaDB поддерживает Databasus?"
              answer="Databasus поддерживает MariaDB 5.5, 10, 11, 12 и 13. Для бекапов MariaDB Databasus использует родной инструмент mariadb-dump вместо mysqldump — это дает полную совместимость с особенностями и оптимизациями MariaDB. Один и тот же веб-интерфейс, расписания, хранилища и уведомления работают и для MySQL, и для MariaDB."
            />
            <FaqItem
              number="5"
              question="Можно ли делать бекапы MySQL в AWS RDS, Google Cloud SQL или Azure?"
              answer="Да, Databasus работает с облачными базами MySQL, включая AWS RDS for MySQL, Google Cloud SQL, Azure Database for MySQL и PlanetScale. Поскольку Databasus делает логические бекапы через mysqldump, ему достаточно стандартных реквизитов подключения к MySQL (хост, порт, имя пользователя, пароль) — специальные облачные права, IAM-роли и доступ к файловой системе не нужны."
            />
            <FaqItem
              number="6"
              question="Как в Databasus работают расписания бекапов MySQL?"
              answer="Бекапы MySQL можно запускать каждый час, день, неделю или месяц, а для точного контроля — задавать cron-выражения. Выбирайте конкретное время (например, 3 часа ночи, когда трафика мало) и дни. Остальное Databasus берет на себя: подключается к MySQL, запускает mysqldump, сжимает результат, шифрует его и загружает в выбранное хранилище."
            />
            <FaqItem
              number="7"
              question="Где хранятся мои бекапы MySQL?"
              answer="Вы сами выбираете, где хранить файлы бекапов MySQL: локально на сервере, в S3-совместимом хранилище (AWS S3, MinIO, Backblaze B2), Google Drive, Cloudflare R2, Dropbox, на NAS через SFTP или в любом хранилище, которое поддерживает rclone. У каждой базы может быть свое хранилище. Файлы бекапов остаются под вашим контролем."
            />
            <FaqItem
              number="8"
              question="Как Databasus защищает учетные данные MySQL и бекапы?"
              answer={
                <>
                  Databasus использует многоуровневую защиту:
                  <br />
                  <br />
                  <strong>1. Шифрование реквизитов:</strong> все пароли, токены
                  и строки подключения MySQL шифруются AES-256-GCM перед
                  сохранением. Ключ шифрования хранится отдельно от базы данных.
                  <br />
                  <br />
                  <strong>2. Шифрование бекапов:</strong> каждый файл бекапа
                  шифруется уникальным ключом, выведенным из мастер-ключа, ID
                  бекапа и случайной соли.
                  <br />
                  <br />
                  <strong>3. Безопасная передача паролей:</strong> пароли MySQL
                  передаются через временные файлы .my.cnf с правами 0600 и
                  никогда не попадают в аргументы командной строки или логи.
                </>
              }
            />
            <FaqItem
              number="9"
              question="Можно ли восстановить бекап MySQL на другой сервер или в другое облако?"
              answer="Да. Databasus создает стандартные логические бекапы mysqldump, поэтому их можно восстановить на любой сервер MySQL: другой версии, у другого облачного провайдера или на локальной машине разработчика. Скачайте бекап из Databasus (он расшифровывается автоматически), затем восстановите его командой mysql. Databasus показывает точную команду восстановления для каждого бекапа."
            />
            <FaqItem
              number="10"
              question="Поддерживает ли Databasus бекапы реплик MySQL?"
              answer="Да, вы можете делать бекапы реплик MySQL (slave-серверов). Databasus использует --single-transaction, который создает консистентный снимок без остановки репликации. Так бекапы не нагружают основной сервер. Просто укажите Databasus реквизиты подключения к реплике."
            />
            <FaqItem
              number="11"
              question="Чем Databasus отличается от MySQL Enterprise Backup и Percona XtraBackup?"
              answer={
                <>
                  MySQL Enterprise Backup и Percona XtraBackup создают
                  физические (бинарные) бекапы, которым нужен прямой доступ к
                  каталогу данных MySQL на диске. Они быстрее для очень больших
                  баз, но:
                  <br />
                  <br />
                  • не умеют делать бекапы облачных MySQL (RDS, Cloud SQL,
                  Azure)
                  <br />
                  • требуют установки на самом сервере базы данных
                  <br />
                  • сложнее в настройке и восстановлении
                  <br />
                  <br />
                  Databasus делает логические бекапы (mysqldump), которые
                  работают с любым сервером MySQL, к которому можно
                  подключиться, включая всех облачных провайдеров. Для
                  большинства баз до 100 ГБ логические бекапы практичны и
                  заметно проще в обслуживании.
                </>
              }
            />
            <FaqItem
              number="12"
              question="Можно ли делать бекапы MySQL и PostgreSQL одной установкой Databasus?"
              answer="Да, Databasus поддерживает MySQL, MariaDB, PostgreSQL и MongoDB в одной установке. Управляйте бекапами всех баз из одного веб-интерфейса, с отдельными расписаниями, хранилищами и каналами уведомлений для каждой. Это удобно командам с разнородной инфраструктурой баз данных."
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
