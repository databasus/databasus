import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

const PAGE_URL = getLocalizedUrl("ru", "storages/neon-object-storage");

export const metadata: Metadata = {
  title: "Резервное копирование PostgreSQL в Neon Object Storage: пошаговая инструкция | Databasus",
  description:
    "Пошаговая инструкция со скриншотами: как создать бакет и ключи в Neon Object Storage, подключить их в Databasus как S3-хранилище и хранить бекапы баз данных в Neon по расписанию.",
  keywords: [
    "Databasus",
    "бекап в Neon Object Storage",
    "Neon Object Storage",
    "Neon S3",
    "бакет Neon",
    "резервное копирование PostgreSQL в S3",
    "S3-совместимое хранилище",
    "бекап базы данных",
  ],
  openGraph: {
    title: "Резервное копирование PostgreSQL в Neon Object Storage: пошаговая инструкция | Databasus",
    description:
      "Пошаговая инструкция со скриншотами: как создать бакет и ключи в Neon Object Storage, подключить их в Databasus как S3-хранилище и хранить бекапы баз данных в Neon по расписанию.",
    type: "article",
    url: PAGE_URL,
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Резервное копирование PostgreSQL в Neon Object Storage: пошаговая инструкция | Databasus",
    description:
      "Пошаговая инструкция со скриншотами: как создать бакет и ключи в Neon Object Storage, подключить их в Databasus как S3-хранилище и хранить бекапы баз данных в Neon по расписанию.",
  },
  alternates: {
    canonical: PAGE_URL,
    languages: getLanguageAlternates("storages/neon-object-storage"),
  },
  robots: "index, follow",
};

export default function NeonObjectStoragePage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Как настроить резервное копирование в Neon Object Storage с помощью Databasus",
            description:
              "Пошаговая инструкция: создать бакет и ключи в Neon Object Storage и добавить его в Databasus как хранилище бекапов",
            step: [
              {
                "@type": "HowToStep",
                name: "Создайте бакет в Neon",
                text: "В Neon Console откройте проект для бекапов, нажмите Object storage в боковом меню, а затем Create bucket.",
                url: `${PAGE_URL}#create-bucket`,
              },
              {
                "@type": "HowToStep",
                name: "Скопируйте ключи доступа к хранилищу",
                text: "Нажмите Connect, откройте вкладку Storage, выберите Parameters only и нажмите Copy credentials.",
                url: `${PAGE_URL}#copy-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "Выберите Neon Object Storage в Databasus",
                text: "В Databasus откройте Storages, нажмите Add storage и выберите Neon Object Storage в списке типов.",
                url: `${PAGE_URL}#choose-neon-object-storage`,
              },
              {
                "@type": "HowToStep",
                name: "Вставьте ключи и введите имя бакета",
                text: "Нажмите Paste from clipboard, чтобы заполнить endpoint, регион, access key и secret key, затем введите имя бакета.",
                url: `${PAGE_URL}#paste-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "Проверьте подключение и сохраните",
                text: "Нажмите Test connection и, когда проверка пройдет, нажмите Save.",
                url: `${PAGE_URL}#test-connection`,
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="neon-object-storage-backup">Как настроить резервное копирование в Neon Object Storage</h1>

              <p className="text-lg text-gray-400">
                <a href="https://neon.com/docs/storage/overview">Neon Object Storage</a> —
                S3-совместимое объектное хранилище, встроенное в каждый проект Neon. Чтобы
                хранить там бекапы, создайте бакет и ключи в Neon Console, а затем добавьте
                бакет в Databasus как хранилище. Databasus будет загружать туда бекапы всех баз,
                которые используют это хранилище, где бы они ни работали: в Neon или в другом
                месте.
              </p>

              <p>
                Нужен только запущенный Databasus. Если его еще нет, начните
                с <a href={getLocalizedHref("ru", "installation")}>инструкции по установке</a>.
                Neon Object Storage пока в бете; что оно поддерживает, описано
                на <a href="https://neon.com/docs/storage/s3-compatibility">странице о
                совместимости с S3</a> в документации Neon.
              </p>

              <h2 id="create-bucket">Шаг 1. Создайте бакет в Neon</h2>

              <p>
                Откройте в Neon Console проект, где будут лежать бекапы. Лучше взять другой
                проект, а не тот, где живут базы: при удалении проекта удаляются и его бакеты.
                Нажмите <strong>Object storage</strong> в боковом меню, затем <strong>Create
                bucket</strong> и введите имя бакета. Новый бакет приватный и относится к
                текущей ветке проекта.
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-object-storage-buckets.png"
                alt="Страница Object storage проекта Neon с бакетом и кнопкой Create bucket"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
              />

              <h2 id="copy-credentials">Шаг 2. Скопируйте ключи доступа к хранилищу</h2>

              <p>
                Нажмите <strong>Connect</strong> и откройте вкладку <strong>Storage</strong>. В
                режиме <strong>Parameters only</strong> Neon показывает endpoint, access key,
                secret key и регион. Если вместо ключей стоят заглушки, сначала сгенерируйте
                ключ, как просит первая строка блока. Затем нажмите <strong>Copy
                credentials</strong>.
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-connect-storage-credentials.png"
                alt="Вкладка Storage окна Connect в Neon с endpoint, ключами, регионом и кнопкой Copy credentials"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Блок выглядит так. Имени бакета в нем нет, поэтому запомните имя из шага 1.
              </p>

              <pre>
                <code>{`AWS_ENDPOINT_URL_S3="https://br-<branch>.storage.c-9.us-east-2.aws.neon.tech"
AWS_ACCESS_KEY_ID="nak_live_..."
AWS_SECRET_ACCESS_KEY="..."
AWS_REGION="us-east-2"`}</code>
              </pre>

              <h2 id="add-storage">Шаг 3. Добавьте хранилище в Databasus</h2>

              <h3 id="choose-neon-object-storage">Выберите Neon Object Storage в Databasus</h3>

              <p>
                В Databasus откройте <strong>Storages</strong> и нажмите <strong>Add
                storage</strong>. Дайте хранилищу имя и выберите <strong>Neon Object
                Storage</strong> в списке типов. Databasus сохраняет его как обычное
                S3-хранилище, поэтому в списке хранилищ у него тип S3.
              </p>

              <h3 id="paste-credentials">Вставьте ключи и введите имя бакета</h3>

              <p>
                Нажмите <strong>Paste from clipboard</strong>. Databasus прочитает блок,
                скопированный в Neon, и заполнит endpoint, регион, access key и secret key. Блок
                не покидает браузер: ключи уходят из него, только когда вы проверяете или
                сохраняете хранилище. Имя бакета впишите в <strong>S3 Bucket</strong> сами. Все
                поля можно заполнить и вручную, под разделителем <strong>or enter the details
                manually</strong>.
              </p>

              <img
                src="/images/storages/neon-object-storage/databasus-neon-object-storage-form.png"
                alt="Форма Neon Object Storage в Databasus со вставленными endpoint, регионом и ключами и введенным именем bucket"
                width={1435}
                height={897}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="test-connection">Проверьте подключение и сохраните</h3>

              <p>
                Нажмите <strong>Test connection</strong>. Когда проверка пройдет,
                нажмите <strong>Save</strong>. Теперь это хранилище можно выбрать в настройках
                бекапов любой базы. Если сама база работает в Neon, остальную настройку
                описывает <a href={getLocalizedHref("ru", "faq/neon")}>инструкция по бекапу
                Neon</a>.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("ru", "storages")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Назад к хранилищам
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
