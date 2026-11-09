import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Databasus vs PgBackWeb: сравнение инструментов резервного копирования PostgreSQL",
  description:
    "Сравнение инструментов резервного копирования PostgreSQL Databasus и PgBackWeb: различия в функциях, безопасности, командной работе, хранилищах, уведомлениях и удобстве.",
  keywords: [
    "Databasus vs PgBackWeb",
    "сравнение резервного копирования PostgreSQL",
    "альтернатива PgBackWeb",
    "инструменты резервного копирования PostgreSQL",
    "сравнение бекапов баз данных",
    "pg_dump с интерфейсом",
    "self-hosted бекап",
    "безопасность бекапов PostgreSQL",
  ],
  openGraph: {
    title:
      "Databasus vs PgBackWeb: сравнение инструментов резервного копирования PostgreSQL",
    description:
      "Сравнение инструментов резервного копирования PostgreSQL Databasus и PgBackWeb: различия в функциях, безопасности, командной работе, хранилищах, уведомлениях и удобстве.",
    type: "article",
    url: getLocalizedUrl("ru", "databasus-vs-pgbackweb"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs PgBackWeb: сравнение инструментов резервного копирования PostgreSQL",
    description:
      "Сравнение инструментов резервного копирования PostgreSQL Databasus и PgBackWeb: различия в функциях, безопасности, командной работе, хранилищах, уведомлениях и удобстве.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "databasus-vs-pgbackweb"),
    languages: getLanguageAlternates("databasus-vs-pgbackweb"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackWebPage() {
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
              "Databasus vs PgBackWeb: сравнение инструментов резервного копирования PostgreSQL",
            description:
              "Подробное сравнение инструментов резервного копирования PostgreSQL Databasus и PgBackWeb: функции, безопасность, командная работа, хранилища и удобство.",
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
              <h1 id="databasus-vs-pgbackweb">Databasus vs PgBackWeb</h1>

              <p className="text-lg text-gray-400">
                И Databasus, и PgBackWeb — open-source инструменты, которые
                упрощают управление бекапами PostgreSQL через веб-интерфейс.
                Цель у них общая: сделать резервное копирование доступнее, но
                они заметно различаются функциями, безопасностью, поддержкой
                командной работы и удобством.
              </p>

              <h2 id="quick-comparison">Быстрое сравнение</h2>

              <p>
                Краткий обзор ключевых различий между Databasus и PgBackWeb:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Возможность</th>
                    <th>Databasus</th>
                    <th>PgBackWeb</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Лицензия</td>
                    <td data-label="Databasus">Apache 2.0</td>
                    <td data-label="PgBackWeb">AGPL-3.0</td>
                  </tr>
                  <tr>
                    <td>Управление бекапами</td>
                    <td data-label="Databasus">✅ Несколько БД</td>
                    <td data-label="PgBackWeb">✅ Несколько БД</td>
                  </tr>
                  <tr>
                    <td>Поддержка других СУБД</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="PgBackWeb">❌ Только PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>Варианты хранилищ</td>
                    <td data-label="Databasus">
                      Локальное, S3, Cloudflare R2, Azure, NAS,
                      FTP
                    </td>
                    <td data-label="PgBackWeb">
                      Только локальное и S3-совместимое
                    </td>
                  </tr>
                  <tr>
                    <td>Уведомления</td>
                    <td data-label="Databasus">
                      Slack, Discord, Telegram, Teams, Email, вебхуки
                    </td>
                    <td data-label="PgBackWeb">Только вебхуки</td>
                  </tr>
                  <tr>
                    <td>Безопасность</td>
                    <td data-label="Databasus">
                      ✅ AES-256-GCM, уникальные ключи бекапов, контроль доступа
                      только на чтение
                    </td>
                    <td data-label="PgBackWeb">✅ Шифрование PGP</td>
                  </tr>
                  <tr>
                    <td>Командные функции</td>
                    <td data-label="Databasus">
                      ✅ Рабочие пространства, ролевой доступ, журнал аудита
                    </td>
                    <td data-label="PgBackWeb">❌ Нет</td>
                  </tr>
                  <tr>
                    <td>Мониторинг доступности</td>
                    <td data-label="Databasus">✅ Встроенный</td>
                    <td data-label="PgBackWeb">❌ Нет</td>
                  </tr>
                  <tr>
                    <td>Установка</td>
                    <td data-label="Databasus">
                      Скрипт в одну строку, Docker или Helm
                    </td>
                    <td data-label="PgBackWeb">Ручная настройка Docker</td>
                  </tr>
                  <tr>
                    <td>Физические бекапы</td>
                    <td data-label="Databasus">✅ Да</td>
                    <td data-label="PgBackWeb">❌ Нет</td>
                  </tr>
                  <tr>
                    <td>Инкрементальные бекапы</td>
                    <td data-label="Databasus">✅ На уровне блоков (PG 17+)</td>
                    <td data-label="PgBackWeb">❌ Нет</td>
                  </tr>
                  <tr>
                    <td>Архивирование WAL</td>
                    <td data-label="Databasus">✅ Непрерывный стриминг</td>
                    <td data-label="PgBackWeb">❌ Нет</td>
                  </tr>
                  <tr>
                    <td>Восстановление на момент времени</td>
                    <td data-label="Databasus">✅ Да</td>
                    <td data-label="PgBackWeb">❌ Нет</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="backup-features">Возможности бекапов</h2>

              <p>Оба инструмента делают бекапы по гибкому расписанию:</p>

              <ul>
                <li>
                  <strong>Databasus</strong>: расписание каждый час, день,
                  неделю, месяц или по cron с точным временем (например, 4 часа
                  утра). Использует{" "}
                  <strong>сбалансированное сжатие zstd (уровень 5)</strong>,
                  уменьшая размер бекапов в 4-8 раз при накладных расходах всего
                  ~20% времени. Это заметно эффективнее gzip.
                </li>
                <li>
                  <strong>PgBackWeb</strong>: поддерживает расписание на основе
                  cron. Сжимает бекапы gzip, который медленнее и менее
                  эффективен, чем zstd.
                </li>
              </ul>

              <p>
                Помимо логических бекапов, Databasus поддерживает физические,
                инкрементальные и WAL-бекапы. Они построены на нативном стеке
                бекапов PostgreSQL 17 и выполняются удаленно: на сервер базы
                ничего не ставится, а до закрытых сетей можно добраться через
                SSH-туннель. Это дает инкрементальные бекапы на уровне блоков,
                непрерывный стриминг WAL и Point-in-Time Recovery —
                восстановление почти без потери данных, с откатом к любой
                секунде между бекапами. В PgBackWeb ничего из этого нет.
              </p>

              <h2 id="storage-options">Варианты хранилищ</h2>

              <p>Гибкость хранения — важная часть стратегии бекапов:</p>

              <ul>
                <li>
                  <strong>Databasus</strong>: поддерживает широкий набор
                  хранилищ:
                  <ul>
                    <li>Локальное хранилище</li>
                    <li>Amazon S3 и S3-совместимые сервисы</li>
                    <li>Cloudflare R2</li>
                    <li>Azure Blob Storage</li>
                    <li>NAS (сетевые хранилища)</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>: ограничен локальным хранилищем и
                  S3-совместимыми сервисами.
                </li>
              </ul>

              <p>
                <a
                  href="/ru/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Все варианты хранилищ Databasus →
                </a>
              </p>

              <h2 id="security">Безопасность</h2>

              <p>
                Безопасность — критичная часть управления бекапами. Databasus
                выстраивает защиту на трех уровнях:
              </p>

              <h3 id="security-databasus">Модель безопасности Databasus</h3>

              <ol>
                <li>
                  <strong>Шифрование чувствительных данных</strong>: все пароли,
                  токены и учетные данные шифруются AES-256-GCM. Ключ шифрования
                  хранится отдельно от базы, поэтому даже при компрометации базы
                  чувствительные данные остаются защищены.
                </li>
                <li>
                  <strong>Шифрование бекапов</strong>: каждый файл бекапа
                  шифруется уникальным ключом, выведенным из мастер-ключа, ID
                  бекапа и случайной соли. Даже получив доступ к вашему
                  облачному хранилищу, злоумышленник не прочитает бекапы без
                  вашего ключа шифрования.
                </li>
                <li>
                  <strong>Доступ к базе только на чтение</strong>: Databasus
                  контролирует доступ только на чтение, проверяя права на уровне
                  роли, базы и таблиц. Ему достаточно прав SELECT, и он
                  предупредит, если обнаружит права на запись. Это защищает
                  данные от порчи даже при компрометации Databasus.
                </li>
              </ol>

              <h3 id="security-pgbackweb">Модель безопасности PgBackWeb</h3>

              <ul>
                <li>
                  <strong>Шифрование PGP</strong>: PgBackWeb предлагает
                  PGP-шифрование файлов бекапов.
                </li>
                <li>
                  <strong>Нет контроля доступа только на чтение</strong>:
                  PgBackWeb не проверяет и не требует доступа только на чтение,
                  поэтому бекапы могут создаваться пользователями с правами на
                  запись.
                </li>
              </ul>

              <p>
                <a
                  href="/ru/security"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Подробнее о безопасности Databasus →
                </a>
              </p>

              <h2 id="notifications">Уведомления</h2>

              <p>
                Чтобы бекапы были надежными, важно вовремя узнавать об их
                статусе:
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong>: уведомления в реальном времени по
                  нескольким каналам:
                  <ul>
                    <li>Slack</li>
                    <li>Discord</li>
                    <li>Telegram</li>
                    <li>Microsoft Teams</li>
                    <li>Email</li>
                    <li>Вебхуки</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>: поддерживает только вебхуки. Чтобы
                  получать алерты в Slack, Telegram или другие платформы,
                  придется поднимать дополнительную прослойку или сервисы.
                </li>
              </ul>

              <p>
                <a
                  href="/ru/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Все каналы уведомлений Databasus →
                </a>
              </p>

              <h2 id="team-features">Командные функции</h2>

              <p>
                Для организаций и DevOps-команд функции совместной работы
                необходимы. Именно здесь Databasus заметно превосходит
                PgBackWeb:
              </p>

              <h3 id="team-databasus">Командные возможности Databasus</h3>

              <ul>
                <li>
                  <strong>Рабочие пространства</strong>: группируйте базы,
                  уведомления и хранилища по проектам или командам. Пользователи
                  видят только те пространства, куда их пригласили.
                </li>
                <li>
                  <strong>Ролевой доступ</strong>: уровни прав контролируют, что
                  каждый участник может делать внутри рабочих пространств.
                </li>
                <li>
                  <strong>Журнал аудита</strong>: фиксируются все действия в
                  системе и изменения, сделанные пользователями. Необходим для
                  комплаенса и подотчетности команды.
                </li>
              </ul>

              <h3 id="team-pgbackweb">Командные возможности PgBackWeb</h3>

              <p>
                В PgBackWeb нет встроенного управления пользователями, рабочих
                пространств и журнала аудита. Он рассчитан прежде всего на
                сценарии с одним пользователем.
              </p>

              <p>
                <a
                  href="/ru/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Подробнее об управлении доступом в Databasus →
                </a>
              </p>

              <h2 id="ease-of-use">Удобство использования</h2>

              <p>
                <strong>Databasus заметно проще в использовании</strong>, чем
                PgBackWeb: упор сделан на понятный UX и минимальное время
                настройки.
              </p>

              <h3 id="ease-databasus">Работа с Databasus</h3>

              <ul>
                <li>
                  <strong>Простая установка</strong>: используйте Docker
                  напрямую или запустите скрипт в одну строку, который поставит
                  Docker (если нужно), настроит Databasus и включит автозапуск.
                  Всего ~2 минуты.
                </li>
                <li>
                  <strong>Понятный веб-интерфейс</strong>: вылизанный дизайн,
                  который шаг за шагом ведет вас по настройке бекапов.
                  Экспертиза PostgreSQL не нужна.
                </li>
                <li>
                  <strong>Темная и светлая темы</strong>: выберите вид, который
                  вам удобнее.
                </li>
                <li>
                  <strong>Адаптивность под мобильные</strong>: проверяйте бекапы
                  с любого устройства, где бы вы ни были.
                </li>
                <li>
                  <strong>Встроенный мониторинг доступности</strong>:
                  настраиваемые health-чеки с наглядными графиками доступности.
                </li>
                <li>
                  <strong>Восстановление в один клик</strong>: скачивайте и
                  восстанавливайте любой бекап одним кликом.
                </li>
              </ul>

              <h3 id="ease-pgbackweb">Работа с PgBackWeb</h3>

              <ul>
                <li>
                  <strong>Ручная настройка Docker</strong>: нужно настроить
                  переменные окружения и поднять внешнюю базу PostgreSQL для
                  хранения конфигурации.
                </li>
                <li>
                  <strong>Базовый веб-интерфейс</strong>: функциональный, но
                  менее проработанный, чем у Databasus. Есть темная тема.
                </li>
                <li>
                  <strong>Нет мониторинга доступности</strong>: мониторинг
                  доступности базы придется настраивать отдельно.
                </li>
              </ul>

              <h2 id="installation">Установка и развертывание</h2>

              <h3 id="install-databasus">Установка Databasus</h3>

              <p>
                У Databasus три способа установки, самый быстрый —
                автоматический скрипт:
              </p>

              <ul>
                <li>
                  <strong>Автоматический скрипт (рекомендуется)</strong>:
                  cURL-команда в одну строку, которая поставит Docker, настроит
                  Databasus и включит автозапуск.
                </li>
                <li>
                  <strong>Docker run</strong>: одна команда запускает Databasus
                  со встроенной PostgreSQL.
                </li>
                <li>
                  <strong>Docker Compose</strong>: для большего контроля над
                  развертыванием.
                </li>
              </ul>

              <p>
                <a
                  href="/ru/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Руководство по установке Databasus →
                </a>
              </p>

              <h3 id="install-pgbackweb">Установка PgBackWeb</h3>

              <p>
                PgBackWeb требует Docker и ручной настройки переменных
                окружения. Также нужно поднять внешнюю базу PostgreSQL для
                хранения конфигурации PgBackWeb.
              </p>

              <h2 id="licensing">Лицензирование</h2>

              <p>
                Модель лицензирования сильно влияет на то, как вы можете
                использовать и изменять софт:
              </p>

              <ul>
                <li>
                  <strong>Databasus (Apache 2.0)</strong>: пермиссивная
                  лицензия, разрешающая неограниченное коммерческое
                  использование, изменение и распространение. Databasus можно
                  использовать в проприетарных проектах без лицензионных
                  вопросов.
                </li>
                <li>
                  <strong>PgBackWeb (AGPL-3.0)</strong>: копилефт-лицензия,
                  требующая, чтобы производные работы и изменения тоже были
                  открыты под AGPL-3.0. Если вы изменили PgBackWeb и
                  предоставляете его как сервис, вы обязаны опубликовать свои
                  изменения.
                </li>
              </ul>

              <h2 id="conclusion">Вывод</h2>

              <p>
                И Databasus, и PgBackWeb — достойные инструменты резервного
                копирования PostgreSQL, но задачи у них разные:
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Выбирайте Databasus, если вам нужны:
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Безопасность корпоративного уровня с трехуровневой защитой
                  </li>
                  <li>
                    Командная работа с рабочими пространствами и журналом аудита
                  </li>
                  <li>
                    Несколько вариантов хранилищ (FTP, SFTP, Azure и т.д.)
                  </li>
                  <li>
                    Встроенные уведомления в Slack, Teams, Telegram и т.д.
                  </li>
                  <li>
                    Быстрая установка скриптом в одну строку или через Docker
                  </li>
                  <li>
                    Понятный современный интерфейс с минимальным порогом входа
                  </li>
                  <li>
                    Пермиссивная лицензия Apache 2.0 для коммерческого
                    использования
                  </li>
                  <li>
                    Физические и инкрементальные бекапы, архивирование WAL и
                    PITR для аварийного восстановления
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>Выбирайте PgBackWeb, если:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Вам нужно простое решение для бекапов в сценарии с одним
                    пользователем
                  </li>
                  <li>Вам хватает локального или S3-хранилища</li>
                  <li>Вам достаточно уведомлений через вебхуки</li>
                  <li>Лицензия AGPL-3.0 вам подходит</li>
                </ul>
              </div>

              <p>
                Большинству пользователей, особенно командам и организациям,
                которым нужны серьезная безопасность, несколько вариантов
                хранилищ и полноценные каналы уведомлений,{" "}
                <strong>мы рекомендуем Databasus</strong>.
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
