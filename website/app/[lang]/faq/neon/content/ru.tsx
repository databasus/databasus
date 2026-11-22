import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

export const metadata: Metadata = {
  title: "Резервное копирование Neon PostgreSQL: пошаговая инструкция | Databasus",
  description:
    "Пошаговая инструкция со скриншотами: как настроить резервное копирование PostgreSQL в Neon с помощью Databasus, хранить бекапы в своем хранилище и восстанавливать их в Neon.",
  keywords: [
    "Databasus",
    "бекап Neon",
    "резервное копирование PostgreSQL в Neon",
    "бекап базы данных Neon",
    "восстановление в Neon",
    "пользователь только для чтения Neon",
    "резервное копирование PostgreSQL",
    "бекап базы данных",
  ],
  openGraph: {
    title: "Резервное копирование Neon PostgreSQL: пошаговая инструкция | Databasus",
    description:
      "Пошаговая инструкция со скриншотами: как настроить резервное копирование PostgreSQL в Neon с помощью Databasus, хранить бекапы в своем хранилище и восстанавливать их в Neon.",
    type: "article",
    url: getLocalizedUrl("ru", "faq/neon"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Резервное копирование Neon PostgreSQL: пошаговая инструкция | Databasus",
    description:
      "Пошаговая инструкция со скриншотами: как настроить резервное копирование PostgreSQL в Neon с помощью Databasus, хранить бекапы в своем хранилище и восстанавливать их в Neon.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "faq/neon"),
    languages: getLanguageAlternates("faq/neon"),
  },
  robots: "index, follow",
};

export default function NeonPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Как сделать бекап Neon PostgreSQL с Databasus",
            description:
              "Пошаговая инструкция: резервное копирование базы PostgreSQL в Neon с помощью Databasus и восстановление в Neon",
            step: [
              {
                "@type": "HowToStep",
                name: "Скопируйте строку подключения в Neon",
                text: "В консоли Neon откройте проект, нажмите Connect, затем Copy snippet.",
                url: `${getLocalizedUrl("ru", "faq/neon")}#copy-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "Добавьте базу Neon в Databasus",
                text: "Нажмите Add database, выберите Neon, вставьте строку подключения кнопкой Paste from clipboard и нажмите Test connection.",
                url: `${getLocalizedUrl("ru", "faq/neon")}#paste-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "Создайте пользователя только для чтения",
                text: "Нажмите Yes, create read-only user: тогда Databasus хранит учетные данные, которыми нельзя изменить ваши данные.",
                url: `${getLocalizedUrl("ru", "faq/neon")}#read-only-user`,
              },
              {
                "@type": "HowToStep",
                name: "Задайте расписание, хранилище и срок хранения",
                text: "Выберите, как часто делать бекап, куда его сохранять и сколько хранить.",
                url: `${getLocalizedUrl("ru", "faq/neon")}#schedule-storage-retention`,
              },
              {
                "@type": "HowToStep",
                name: "Выберите каналы уведомлений",
                text: "Отметьте каналы, куда придут уведомления о бекапах, и нажмите Complete. Databasus запустит первый бекап.",
                url: `${getLocalizedUrl("ru", "faq/neon")}#notifiers`,
              },
              {
                "@type": "HowToStep",
                name: "Восстановите бекап в Neon",
                text: "Нажмите иконку восстановления у бекапа, вставьте строку подключения целевой базы Neon и нажмите Restore to this DB.",
                url: `${getLocalizedUrl("ru", "faq/neon")}#restore`,
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
              <h1 id="neon-backup">Как сделать бекап Neon PostgreSQL</h1>

              <p className="text-lg text-gray-400">
                Neon — это serverless-сервис управляемого PostgreSQL. Чтобы
                делать его бекапы через Databasus, скопируйте строку
                подключения в консоли Neon, добавьте базу в Databasus с типом
                Neon и выберите, где хранить бекапы. Дальше Databasus сам
                делает бекапы по расписанию и умеет восстанавливать их в любую
                базу Neon.
              </p>

              <p>
                Понадобится только запущенный Databasus. Если его еще нет, поставьте его по{" "}
                <a href={getLocalizedHref("ru", "installation")}>инструкции по установке</a>
                . Хранилище и каналы уведомлений можно создать по ходу настройки.
              </p>

              <h2 id="copy-connection-string">
                Шаг 1. Скопируйте строку подключения в Neon
              </h2>

              <p>
                Откройте проект в консоли Neon и нажмите{" "}
                <strong>Connect</strong>.
              </p>

              <img
                src="/images/faq/neon/neon-connect-button.png"
                alt="Кнопка Connect на странице Computes проекта Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
              />

              <p>
                В окне <strong>Connection details</strong> выберите ветку,
                базу и роль, которые нужно бекапить, и нажмите{" "}
                <strong>Copy snippet</strong>.
              </p>

              <img
                src="/images/faq/neon/neon-copy-connection-string.png"
                alt="Окно Connection details в Neon с кнопкой Copy snippet"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="add-database">Шаг 2. Добавьте базу Neon в Databasus</h2>

              <h3 id="paste-connection-string">
                Выберите Neon и вставьте строку подключения
              </h3>

              <p>
                В Databasus нажмите <strong>Add database</strong>, задайте
                имя базы, выберите <strong>Neon</strong> и нажмите{" "}
                <strong>Continue</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-choose-neon.png"
                alt="Окно добавления базы в Databasus с выбранным типом Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Нажмите <strong>Paste from clipboard</strong>. Databasus
                прочитает строку, которую вы скопировали в Neon, и заполнит
                хост, порт, пользователя, пароль, имя базы и режим SSL.
              </p>

              <img
                src="/images/faq/neon/databasus-paste-connection-string.png"
                alt="Форма подключения к Neon в Databasus с кнопкой Paste from clipboard"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Нажмите <strong>Test connection</strong>. Когда проверка
                пройдет, нажмите <strong>Continue</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-test-connection.png"
                alt="Заполненная форма подключения к Neon в Databasus после успешной проверки"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="read-only-user">Создайте пользователя только для чтения</h3>

              <p>
                Databasus предложит завести для бекапов{" "}
                <a
                  href={`${getLocalizedHref("ru", "security")}#level-3-read-only-access`}
                >
                  пользователя только для чтения
                </a>
                . Нажмите <strong>Yes, create read-only user</strong>. Тогда
                Databasus хранит только учетные данные, которыми можно читать
                данные, но нельзя их изменить.
              </p>

              <img
                src="/images/faq/neon/databasus-create-read-only-user.png"
                alt="Databasus предлагает создать пользователя только для чтения для бекапов Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="schedule-storage-retention">
                Задайте расписание, хранилище и срок хранения
              </h3>

              <p>Выберите:</p>

              <ul>
                <li>
                  <strong>Backup interval</strong> и время: как часто
                  Databasus делает бекап, например каждый день в 03:00.
                </li>
                <li>
                  <strong>Storage</strong>: куда сохранять бекапы.{" "}
                  <a href={getLocalizedHref("ru", "storages")}>
                    Новое хранилище
                  </a>{" "}
                  можно добавить прямо из этого списка. Хотите хранить бекапы тоже в
                  Neon? Смотрите,{" "}
                  <a href={getLocalizedHref("ru", "storages/neon-object-storage")}>
                    как настроить резервное копирование в Neon Object Storage
                  </a>
                  .
                </li>
                <li>
                  <strong>Encryption</strong>:{" "}
                  <a
                    href={`${getLocalizedHref("ru", "security")}#level-2-backups-encryption`}
                  >
                    шифровать ли файлы бекапов
                  </a>{" "}
                  перед загрузкой.
                </li>
                <li>
                  <strong>Retention policy</strong>: сколько хранить старые
                  бекапы, например 3 месяца.
                </li>
                <li>
                  <strong>Notifications</strong>: о каких событиях сообщать,
                  только о неудачном бекапе или об успешном тоже.
                </li>
              </ul>

              <img
                src="/images/faq/neon/databasus-schedule-storage-retention.png"
                alt="Настройки расписания, хранилища, шифрования и срока хранения бекапов в Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="notifiers">Выберите каналы уведомлений</h3>

              <p>
                Отметьте{" "}
                <a href={getLocalizedHref("ru", "notifiers")}>
                  каналы уведомлений
                </a>
                , куда придут эти сообщения, например{" "}
                <a href={getLocalizedHref("ru", "notifiers/slack")}>Slack</a>{" "}
                или{" "}
                <a href={getLocalizedHref("ru", "notifiers/teams")}>
                  Microsoft Teams
                </a>
                . Можно выбрать несколько каналов или ни одного и добавить их
                позже. Нажмите <strong>Complete</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-choose-notifiers.png"
                alt="Выбор каналов уведомлений в Databasus перед завершением настройки базы"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="first-backup">Шаг 3. Проверьте первый бекап</h2>

              <p>
                Databasus запускает первый бекап сразу после нажатия{" "}
                <strong>Complete</strong>. Он появится на вкладке{" "}
                <strong>Backups</strong> вместе с размером и длительностью.
                Следующие бекапы пойдут по расписанию, а кнопка{" "}
                <strong>Make backup right now</strong> запускает бекап в любой
                момент.
              </p>

              <img
                src="/images/faq/neon/databasus-first-neon-backup.png"
                alt="Первый успешный бекап Neon на вкладке Backups в Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Успешный бекап еще не гарантирует, что из него получится
                восстановиться. Чтобы проверять это для каждого бекапа,
                настройте{" "}
                <a href={getLocalizedHref("ru", "restore-verification")}>
                  проверку восстановления
                </a>
                .
              </p>

              <h2 id="restore">Как восстановить бекап Neon</h2>

              <p>
                Нажмите иконку восстановления (облако со стрелкой) рядом с
                бекапом. Скопируйте в консоли Neon строку подключения целевой
                базы так же, как в <a href="#copy-connection-string">шаге 1</a>
                , нажмите <strong>Paste from clipboard</strong>, а затем{" "}
                <strong>Restore to this DB</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-restore-neon-backup.png"
                alt="Окно восстановления из бекапа в Databasus с целевой базой Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Перед восстановлением Databasus очищает целевую базу. Целевой
                базой может быть:
              </p>

              <ul>
                <li>новая база в том же проекте Neon</li>
                <li>база на ветке Neon</li>
                <li>существующая база, в том числе исходная</li>
              </ul>

              <p>
                Надежнее всего восстанавливать в новую ветку Neon. Там можно
                спокойно проверить данные, а рабочая база останется нетронутой,
                пока вы не решите переключиться.
              </p>

              <p>Когда восстановление закончится, его статус сменится на Successful.</p>

              <img
                src="/images/faq/neon/databasus-restore-done.png"
                alt="Завершенное восстановление бекапа Neon в Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="health-check">
                Не будет ли проверка доступности будить compute в Neon?
              </h2>

              <p>
                По умолчанию нет. Neon усыпляет compute, когда к базе никто не
                подключен, и не берет за него деньги, пока тот спит. Проверка
                доступности подключается раз в минуту, поэтому compute не
                засыпал бы и оплачивался круглые сутки. Так что для баз Neon
                Databasus выключает проверку доступности. Если включить ее
                вручную, Databasus сначала попросит подтверждение. Бекап по
                расписанию будит compute только на время работы, а потом Neon
                снова его усыпляет.
              </p>

              <p>
                Бекапите еще и Supabase? Смотрите,{" "}
                <a href={getLocalizedHref("ru", "faq/supabase")}>
                  как делать бекапы Supabase
                </a>
                .
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("ru", "faq")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Назад к FAQ
                </a>
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
