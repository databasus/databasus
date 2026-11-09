import type { Metadata } from "next";
import LandingFooterComponent from "@/app/components/LandingFooterComponent";
import LandingHeaderComponent from "@/app/components/LandingHeaderComponent";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import InstallationComponent from "@/app/components/InstallationComponent";

export const metadata: Metadata = {
  title: "Резервное копирование MongoDB",
  description:
    "Бесплатный open source инструмент для регулярных бекапов MongoDB. Автоматизируйте mongodump через веб-интерфейс, храните архивы в S3, Azure Blob Storage или локально. Уведомления в Slack, Discord, Telegram. Шифрование BSON-данных AES-256.",
  keywords:
    "резервное копирование MongoDB, альтернатива mongodump, автоматизация бекапов MongoDB, инструмент для бекапа MongoDB, бекап MongoDB по расписанию, облачный бекап MongoDB, бекап MongoDB в S3, бекап MongoDB в Docker, шифрование бекапов MongoDB, бекап MongoDB Atlas, бекап replica set, бекап документной базы данных, бекап BSON, бекап NoSQL",
  robots: "index, follow",
  alternates: {
    canonical: getLocalizedUrl("ru", "mongodb-backup"),
    languages: getLanguageAlternates("mongodb-backup"),
  },
  openGraph: {
    type: "website",
    url: getLocalizedUrl("ru", "mongodb-backup"),
    title: "Резервное копирование MongoDB",
    description:
      "Бесплатный open source инструмент для регулярных бекапов MongoDB. Автоматизируйте mongodump с веб-интерфейсом, облачными хранилищами, уведомлениями и шифрованием.",
    images: [
      {
        url: "https://databasus.com/images/index/dashboard.png",
        alt: "Интерфейс Databasus для управления резервным копированием MongoDB",
        width: 980,
        height: 573,
      },
    ],
    siteName: "Databasus",
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary_large_image",
    title: "Резервное копирование MongoDB",
    description:
      "Бесплатный open source инструмент для регулярных бекапов MongoDB. Автоматизируйте mongodump с веб-интерфейсом, облачными хранилищами, уведомлениями и шифрованием.",
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

export default function MongodbBackupPage() {
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
              "Бесплатный open source инструмент для регулярных бекапов MongoDB. Автоматизируйте mongodump с веб-интерфейсом, облачными хранилищами, уведомлениями и шифрованием.",
            url: "https://databasus.com/ru/mongodb-backup/",
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
              "Бекапы MongoDB по расписанию через mongodump",
              "Разные хранилища (S3, Azure Blob Storage, FTP, SFTP, NAS и другие)",
              "Мгновенные уведомления (Slack, Telegram, Discord, Webhook, email и другие)",
              "Мониторинг доступности подключения к MongoDB",
              "Self-hosted установка через Docker",
              "Открытый исходный код, бесплатно",
              "Поддержка MongoDB 4, 5, 6, 7, 8 и 9",
              "Сжатие BSON-архивов через gzip",
              "Шифрование файлов бекапов AES-256-GCM",
              "Поддержка MongoDB Atlas и replica set",
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
                name: "Что такое Databasus и как он делает бекапы MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus — это self-hosted инструмент резервного копирования с лицензией Apache 2.0, который использует mongodump для создания консистентных бекапов MongoDB. Он дополняет mongodump современным веб-интерфейсом, расписаниями, интеграцией с облачными хранилищами, мгновенными уведомлениями и шифрованием AES-256-GCM — самописные shell-скрипты и cron-задачи больше не нужны.",
                },
              },
              {
                "@type": "Question",
                name: "Поддерживает ли Databasus реплика-сеты MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Да, Databasus полностью поддерживает реплика-сеты MongoDB. Вы можете подключиться к любому члену реплика-сета через стандартный формат строки подключения MongoDB. Databasus будет читать с указанного узла, так что бекапы можно снимать с secondary-узлов и не нагружать primary.",
                },
              },
              {
                "@type": "Question",
                name: "Можно ли делать бекапы баз MongoDB Atlas через Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Да, Databasus работает с MongoDB Atlas. Поскольку Databasus делает логические бекапы через mongodump, ему достаточно стандартных реквизитов подключения к MongoDB — специальные права и роли Atlas не нужны. Просто укажите строку подключения Atlas, остальное Databasus сделает сам.",
                },
              },
              {
                "@type": "Question",
                name: "Какие версии MongoDB поддерживает Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus поддерживает MongoDB версий 4, 5, 6, 7, 8 и 9. Все бекапы делаются родным инструментом mongodump с флагами --archive и --gzip: получаются компактные сжатые BSON-архивы, которые восстанавливаются через mongorestore.",
                },
              },
              {
                "@type": "Question",
                name: "Как Databasus защищает учетные данные MongoDB и бекапы?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus использует многоуровневую защиту: (1) все пароли и строки подключения MongoDB шифруются AES-256-GCM перед сохранением; (2) каждый файл бекапа шифруется уникальным ключом, выведенным из мастер-ключа, ID бекапа и случайной соли; (3) строки подключения передаются mongodump безопасно и никогда не попадают в логи или вывод командной строки.",
                },
              },
              {
                "@type": "Question",
                name: "Поддерживает ли Databasus инкрементальные бекапы MongoDB?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus делает полные логические бекапы через mongodump, а не инкрементальные. В большинстве сценариев полных бекапов по расписанию — каждый час, день или неделю — хватает как точек восстановления. В MongoDB Atlas уже есть встроенное восстановление на момент времени, а внешние инкрементальные бекапы в кластеры Atlas просто так не восстановить.",
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
              Резервное копирование MongoDB
            </h1>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[720px] mx-auto mb-6 md:mb-10 px-2">
              Databasus — бесплатный self-hosted инструмент с открытым исходным
              кодом для бекапа документных баз данных MongoDB. Автоматизируйте
              mongodump по расписанию, храните BSON-архивы в S3, Azure Blob Storage
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
                alt="Интерфейс Databasus для управления резервным копированием MongoDB"
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
              Возможности резервного копирования MongoDB
            </h2>

            <p className="text-sm sm:text-lg text-gray-200 max-w-[650px] mx-auto mb-8 md:mb-10">
              Databasus добавляет к mongodump все, чего ему не хватает:
              расписания, облачные хранилища, мгновенные уведомления и
              шифрование AES-256-GCM. Подходит разработчикам и DevOps-командам,
              работающим с документными базами и коллекциями MongoDB
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
                Дампы MongoDB по расписанию
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/backup-step-1.svg"
                  alt="Настройка бекапов MongoDB по расписанию"
                  className="w-full h-full object-contain rounded-lg"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Запускайте mongodump в удобное время, когда нагрузка на
                приложение минимальна. Выбирайте интервалы: каждый час, день,
                неделю, месяц — или задавайте cron-выражения для точного
                контроля
              </p>
            </div>

            {/* Card 2: Configurable health checks */}
            <div className="border-b lg:border-r border-[#ffffff20] p-5 md:p-6 col-span-1">
              <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                2
              </div>

              <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                Мониторинг доступности MongoDB
              </h3>

              <div className="mb-4 md:mb-5">
                <img
                  src="/images/index/feature-healthcheck.svg"
                  alt="Проверки доступности MongoDB"
                  className="w-full h-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base mb-3">
                Следите за доступностью подключения к MongoDB с помощью
                настраиваемых проверок. Получайте уведомления, когда база или
                реплика-сет становятся недоступны
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
                Храните BSON-архивы где угодно
              </h3>

              <p className="text-gray-400 text-sm md:text-base mb-4 md:mb-5">
                Держите архивы бекапов MongoDB локально, в S3-совместимом
                хранилище, Azure Blob Storage, FTP, на NAS или в других местах.
                Ваши документы остаются под вашим контролем.{" "}
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
                  alt="Хранилища для бекапов MongoDB"
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
                Узнавайте, когда бекап MongoDB завершился или упал с ошибкой.
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
                  alt="Уведомления о бекапах MongoDB"
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
                Запускайте Databasus на собственной инфраструктуре. Все строки
                подключения MongoDB и данные бекапов остаются на серверах,
                которые вы контролируете. Установка занимает около 2-х минут:
                скрипт, Docker или Kubernetes
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

            {/* Card 7: Many MongoDB versions - Mobile/Tablet separate, Desktop merged with card 10 */}
            <div className="border-b md:border-r lg:border-r lg:border-b-0 border-[#ffffff20] col-span-1 lg:row-span-2 lg:flex lg:flex-col">
              {/* Card 7: Many MongoDB versions */}
              <div className="p-5 md:p-6 lg:border-b lg:border-[#ffffff20]">
                <div className="flex items-center justify-center w-6 h-6 rounded text-sm font-semibold mb-4 border border-[#ffffff20]">
                  7
                </div>

                <h3 className="text-lg md:text-xl 2xl:text-2xl font-bold mb-4 md:mb-5">
                  Поддерживаемые версии MongoDB
                </h3>

                <p className="text-gray-400 text-sm md:text-base mb-4">
                  Поддерживаются MongoDB 4, 5, 6, 7, 8 и 9. Для каждой версии
                  Databasus использует родной mongodump — это дает полную
                  совместимость с вашей документной базой
                </p>

                <div>
                  <img
                    src="/images/index/database-mongodb.svg"
                    alt="Версии MongoDB"
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
                  Строки подключения MongoDB шифруются AES-256-GCM перед
                  сохранением. Каждый BSON-архив шифруется уникальным ключом.
                  Реквизиты передаются mongodump безопасно и никогда не попадают
                  в логи.{" "}
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
                    alt="Безопасность бекапов MongoDB"
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
                  alt="Управление доступом к бекапам MongoDB"
                  className="w-full"
                  loading="lazy"
                />
              </div>

              <p className="text-gray-400 text-sm md:text-base">
                Определяйте, кто может просматривать базы MongoDB и управлять
                ими. Создавайте рабочие пространства для разных проектов.
                Назначайте роли: просмотр, редактирование или администрирование.{" "}
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
                  alt="Журнал аудита бекапов MongoDB"
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
                Строки подключения MongoDB шифруются AES-256-GCM перед
                сохранением. Каждый BSON-архив шифруется уникальным ключом.
                Реквизиты передаются mongodump безопасно и никогда не попадают в
                логи.{" "}
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
                  alt="Безопасность бекапов MongoDB"
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
                  Работает с MongoDB Atlas и self-hosted базами
                </h3>

                <p className="text-gray-400 text-sm md:text-base">
                  Databasus подключается к облачным базам MongoDB, включая
                  MongoDB Atlas и AWS DocumentDB, а также к self-hosted
                  установкам. Поскольку он делает логические бекапы через
                  mongodump, достаточно стандартных реквизитов подключения —
                  специальные облачные права и доступ к файловой системе не
                  нужны
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* MONGODUMP SECTION */}
      <section id="mongodump" className="py-12 md:py-20 px-4 md:px-6 lg:px-0">
        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="flex flex-col lg:flex-row gap-8 lg:gap-16">
            {/* Left side: Info */}
            <div className="w-full lg:w-[50%]">
              <div className="inline-flex items-center justify-center px-3 md:px-4 py-1 md:py-1.5 rounded-lg border border-[#ffffff20] mb-4 md:mb-6">
                <span className="text-sm font-medium">На основе mongodump</span>
              </div>

              <h2 className="text-2xl md:text-3xl lg:text-4xl font-bold mb-4 md:mb-6">
                Как устроен бекап MongoDB
              </h2>

              <div className="space-y-4 text-gray-200 text-sm sm:text-base">
                <p>
                  Внутри Databasus работает <strong>mongodump</strong> —
                  официальная утилита резервного копирования MongoDB. При
                  запуске бекапа Databasus выполняет mongodump с оптимальными
                  параметрами:
                </p>

                <ul className="list-disc list-inside space-y-2 text-gray-400">
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --archive
                    </code>{" "}
                    для вывода BSON одним файлом вместо структуры каталогов
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --gzip
                    </code>{" "}
                    для сжатых архивов, экономящих место и трафик
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --db
                    </code>{" "}
                    чтобы копировать конкретные базы вашего экземпляра MongoDB
                  </li>
                  <li>
                    <code className="bg-[#1f2937] px-1.5 py-0.5 rounded text-sm">
                      --uri
                    </code>{" "}
                    для безопасной передачи строки подключения с аутентификацией
                  </li>
                </ul>

                <p className="text-gray-400">
                  Поток бекапа отправляется напрямую в настроенное хранилище и
                  при необходимости шифруется AES-256-GCM перед записью. Такой
                  подход сводит дисковый ввод-вывод к минимуму и хорошо работает
                  с большими коллекциями.
                </p>

                <div className="pt-2">
                  <p className="text-white font-medium mb-2">
                    Поддерживаемые версии MongoDB:
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 4
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 5
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 6
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 7
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 8
                    </span>
                    <span className="px-3 py-1 rounded border border-[#ffffff20] text-sm">
                      MongoDB 9
                    </span>
                  </div>
                </div>
              </div>
            </div>

            {/* Right side: Image */}
            <div className="w-full lg:w-[50%] flex items-center">
              <div className="w-full rounded-lg border border-[#ffffff20] p-6 md:p-8 flex flex-col items-center justify-center">
                <img
                  src="/images/index/database-mongodb.svg"
                  alt="База данных MongoDB"
                  className="w-[120px] h-[120px] md:w-[150px] md:h-[150px] mb-4"
                  loading="lazy"
                />
                <p className="text-center text-gray-400 text-sm md:text-base">
                  Официальный бекап MongoDB через mongodump со сжатием gzip,
                  шифрованием и облачными хранилищами
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
                за 2 минуты. Одна и та же установка работает для бекапов
                MongoDB, PostgreSQL, MySQL и MariaDB
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
              Вопросы о резервном копировании MongoDB
            </h2>

            <p className="text-base md:text-lg text-gray-200 max-w-[600px] mx-auto">
              Частые вопросы о бекапах документных баз данных MongoDB с
              Databasus. Если остались другие вопросы, присоединяйтесь к нашему
              сообществу в Telegram
            </p>
          </div>
        </div>

        <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 md:gap-8">
            <FaqItem
              number="1"
              question="Что такое Databasus и как он делает бекапы MongoDB?"
              answer="Databasus — это self-hosted инструмент резервного копирования с лицензией Apache 2.0, который использует mongodump для создания консистентных бекапов MongoDB. Он дополняет mongodump современным веб-интерфейсом, расписаниями, интеграцией с облачными хранилищами (S3, Azure Blob Storage, FTP), мгновенными уведомлениями (Slack, Discord, Telegram) и шифрованием AES-256-GCM — самописные shell-скрипты и cron-задачи больше не нужны."
            />
            <FaqItem
              number="2"
              question="Поддерживает ли Databasus реплика-сеты MongoDB?"
              answer="Да, Databasus полностью поддерживает реплика-сеты MongoDB. Вы можете подключиться к любому члену реплика-сета через стандартную строку подключения MongoDB с опциями replica set. Databasus будет читать с указанного узла, поэтому бекапы можно снимать с secondary-узлов и разгружать primary. Это особенно полезно в продакшене, где важно не влиять на производительность основного узла."
            />
            <FaqItem
              number="3"
              question="Можно ли делать бекапы баз MongoDB Atlas через Databasus?"
              answer="Да, Databasus работает с MongoDB Atlas. Поскольку Databasus делает логические бекапы через mongodump, ему достаточно стандартных реквизитов подключения к MongoDB — специальные права Atlas и административные роли не нужны, а в список разрешенных IP достаточно добавить только сервер Databasus. Просто укажите строку подключения Atlas (она есть в панели Atlas), остальное Databasus сделает сам."
            />
            <FaqItem
              number="4"
              question="Какие версии MongoDB поддерживает Databasus?"
              answer="Databasus поддерживает MongoDB версий 4, 5, 6, 7, 8 и 9. Все бекапы делаются родным инструментом mongodump с флагами --archive и --gzip: получаются компактные сжатые BSON-архивы. Их можно восстановить через mongorestore на любую совместимую версию MongoDB, что упрощает миграции между версиями."
            />
            <FaqItem
              number="5"
              question="Как Databasus работает с большими коллекциями MongoDB?"
              answer="Databasus передает вывод mongodump напрямую в хранилище, при необходимости шифруя поток на лету. Такой подход обходится без временных файлов на диске, поэтому хорошо работает с базами, где есть большие коллекции. Флаг --archive создает один сжатый файл вместо структуры каталогов, что снижает нагрузку на ввод-вывод и упрощает хранение."
            />
            <FaqItem
              number="6"
              question="Можно ли делать бекапы шардированных кластеров MongoDB через Databasus?"
              answer={
                <>
                  Сейчас Databasus ориентирован на бекапы отдельных баз MongoDB,
                  а не на координированные бекапы шардированных кластеров.
                  <br />
                  <br />
                  Для шардированных кластеров можно:
                  <br />
                  <br />
                  • копировать каждый шард отдельно, подключаясь к его
                  реплика-сету
                  <br />
                  • делать бекап через роутер mongos (это может сказаться на
                  производительности)
                  <br />
                  <br />
                  Для продакшен-кластеров с шардированием рассмотрите встроенные
                  бекапы MongoDB Atlas или mongodump с --oplog, чтобы получить
                  консистентный снимок всех шардов на один момент времени.
                </>
              }
            />
            <FaqItem
              number="7"
              question="Как Databasus защищает учетные данные MongoDB и бекапы?"
              answer={
                <>
                  Databasus использует многоуровневую защиту:
                  <br />
                  <br />
                  <strong>1. Шифрование реквизитов:</strong> все строки
                  подключения, пароли и данные аутентификации MongoDB шифруются
                  AES-256-GCM перед сохранением.
                  <br />
                  <br />
                  <strong>2. Шифрование бекапов:</strong> каждый BSON-архив
                  шифруется уникальным ключом, выведенным из мастер-ключа, ID
                  бекапа и случайной соли.
                  <br />
                  <br />
                  <strong>3. Безопасная передача реквизитов:</strong> строки
                  подключения передаются mongodump напрямую через безопасные
                  параметры и никогда не попадают в логи или список процессов.
                </>
              }
            />
            <FaqItem
              number="8"
              question="Поддерживает ли Databasus инкрементальные бекапы MongoDB или чтение oplog?"
              answer="Databasus делает полные логические бекапы через mongodump, без инкрементальных бекапов и восстановления на момент времени по oplog. В большинстве сценариев полных бекапов по расписанию — каждый час, день или неделю — хватает как точек восстановления, и возиться с oplog не приходится. В MongoDB Atlas уже есть встроенные непрерывные бекапы с восстановлением на момент времени, а внешние инкрементальные бекапы в кластеры Atlas просто так не восстановить."
            />
            <FaqItem
              number="9"
              question="Можно ли восстановить бекап MongoDB на другую версию или в другой кластер?"
              answer="Да. Databasus создает стандартные архивы mongodump в формате BSON, поэтому их можно восстановить на любой совместимый сервер MongoDB: другой версии, у другого облачного провайдера или на локальной машине разработчика. Скачайте бекап из Databasus (он расшифровывается автоматически), затем запустите mongorestore с флагами --archive и --gzip. Databasus показывает точную команду восстановления для каждого бекапа."
            />
            <FaqItem
              number="10"
              question="Как в Databasus работает сжатие mongodump?"
              answer="Databasus использует встроенный флаг --gzip у mongodump, который сжимает BSON-данные прямо во время дампа. Обычно это уменьшает размер архива на 60-80% по сравнению с несжатым BSON. Сжатие происходит в потоке mongodump до опционального шифрования, поэтому и сжатые, и зашифрованные архивы остаются компактными. При восстановлении через mongorestore с флагом --gzip распаковка выполняется автоматически."
            />
            <FaqItem
              number="11"
              question="Можно ли копировать отдельные коллекции MongoDB вместо целых баз?"
              answer="Сейчас Databasus копирует базы MongoDB целиком, а не отдельные коллекции. Так вы получаете полные консистентные бекапы со всеми коллекциями, индексами и метаданными. Если нужны бекапы на уровне коллекций, разнесите данные по отдельным базам для разных доменов — у каждой будет свое расписание бекапов в Databasus."
            />
            <FaqItem
              number="12"
              question="Работает ли Databasus с MongoDB в Docker или Kubernetes?"
              answer="Да, Databasus подключается к MongoDB по сети через стандартные строки подключения, поэтому работает с MongoDB где угодно: в контейнерах Docker, подах Kubernetes, на виртуальных машинах или голом железе. Достаточно сетевой связности между Databasus и вашим экземпляром MongoDB. В Kubernetes можно использовать внутренние DNS-имена сервисов или внешние адреса балансировщика."
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
