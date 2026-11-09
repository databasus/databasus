import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Databasus vs Barman: сравнение инструментов резервного копирования PostgreSQL",
  description:
    "Сравнение инструментов резервного копирования PostgreSQL Databasus и Barman: подход к бекапам, возможности PITR, удобство, командные функции и когда выбирать каждый инструмент.",
  keywords: [
    "Databasus vs Barman",
    "сравнение резервного копирования PostgreSQL",
    "альтернатива Barman",
    "инструменты резервного копирования PostgreSQL",
    "сравнение бекапов баз данных",
    "pg_dump или физический бекап",
    "self-hosted бекап",
    "PITR PostgreSQL",
    "архивирование WAL",
    "аварийное восстановление PostgreSQL",
  ],
  openGraph: {
    title:
      "Databasus vs Barman: сравнение инструментов резервного копирования PostgreSQL",
    description:
      "Сравнение инструментов резервного копирования PostgreSQL Databasus и Barman: подход к бекапам, возможности PITR, удобство, командные функции и когда выбирать каждый инструмент.",
    type: "article",
    url: getLocalizedUrl("ru", "databasus-vs-barman"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs Barman: сравнение инструментов резервного копирования PostgreSQL",
    description:
      "Сравнение инструментов резервного копирования PostgreSQL Databasus и Barman: подход к бекапам, возможности PITR, удобство, командные функции и когда выбирать каждый инструмент.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "databasus-vs-barman"),
    languages: getLanguageAlternates("databasus-vs-barman"),
  },
  robots: "index, follow",
};

export default function DatabasusVsBarmanPage() {
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
              "Databasus vs Barman: сравнение инструментов резервного копирования PostgreSQL",
            description:
              "Подробное сравнение инструментов резервного копирования PostgreSQL Databasus и Barman: подход к бекапам, возможности PITR, удобство, командные функции и когда выбирать каждый инструмент.",
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
              <h1 id="databasus-vs-barman">Databasus vs Barman</h1>

              <p className="text-lg text-gray-400">
                И Databasus, и Barman созданы для аварийного восстановления с
                минимальными RTO и RPO, и оба поддерживают физические бекапы,
                архивирование WAL и восстановление на момент времени (PITR).
                Databasus выполняет эти бекапы удаленно на нативном стеке
                PostgreSQL 17, то есть опирается на проверенный инструментарий
                самого PostgreSQL, а не изобретает свой движок, и все это в
                понятном веб-интерфейсе с командными функциями и поддержкой
                нескольких СУБД. Он подходит для баз любого размера и сложности.
                Физические бекапы требуют PostgreSQL 17 или новее; на старых
                версиях доступны только логические бекапы через{" "}
                <code>pg_dump</code>. Barman (Backup and Recovery Manager)
                поставляется с собственным движком бекапов, поэтому он делает
                физические бекапы на гораздо более старых версиях PostgreSQL и
                добавляет продвинутые функции: инкрементальные бекапы на rsync,
                интеграцию с потоковой репликацией и георезервирование
                Barman-to-Barman.
              </p>

              <h2 id="quick-comparison">Быстрое сравнение</h2>

              <p>Краткий обзор ключевых различий между Databasus и Barman:</p>

              <table>
                <thead>
                  <tr>
                    <th>Возможность</th>
                    <th>Databasus</th>
                    <th>Barman</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Целевая аудитория</td>
                    <td data-label="Databasus">
                      Разработчики, команды, DBA, крупные компании
                    </td>
                    <td data-label="Barman">DBA, крупные компании</td>
                  </tr>
                  <tr>
                    <td>Поддержка других СУБД</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="Barman">❌ Только PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>Интерфейс</td>
                    <td data-label="Databasus">Веб-интерфейс</td>
                    <td data-label="Barman">Только командная строка</td>
                  </tr>
                  <tr>
                    <td>Тип бекапа</td>
                    <td data-label="Databasus">Логический + физический</td>
                    <td data-label="Barman">Физический (на уровне файлов)</td>
                  </tr>
                  <tr>
                    <td>Версия PostgreSQL для физических бекапов</td>
                    <td data-label="Databasus">17+ (нативный стек)</td>
                    <td data-label="Barman">9.x+ (собственный движок)</td>
                  </tr>
                  <tr>
                    <td>Варианты восстановления</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="Barman">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Инкрементальные бекапы</td>
                    <td data-label="Databasus">✅ На уровне блоков (PG 17+)</td>
                    <td data-label="Barman">Инкрементальные на rsync</td>
                  </tr>
                  <tr>
                    <td>Удаленные бекапы</td>
                    <td data-label="Databasus">✅ Да</td>
                    <td data-label="Barman">
                      ❌ Нет (нужен доступ к файловой системе)
                    </td>
                  </tr>
                  <tr>
                    <td>Управление несколькими серверами</td>
                    <td data-label="Databasus">Расписание на каждую базу</td>
                    <td data-label="Barman">Центральный сервер бекапов</td>
                  </tr>
                  <tr>
                    <td>Командные функции</td>
                    <td data-label="Databasus">
                      ✅ Рабочие пространства, RBAC, журнал аудита
                    </td>
                    <td data-label="Barman">❌ Только права на уровне ОС</td>
                  </tr>
                  <tr>
                    <td>Уведомления</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, Email
                    </td>
                    <td data-label="Barman">❌ Нужны собственные скрипты</td>
                  </tr>
                  <tr>
                    <td>Порог входа</td>
                    <td data-label="Databasus">Минимальный</td>
                    <td data-label="Barman">Нужна экспертиза DBA</td>
                  </tr>
                  <tr>
                    <td>Установка</td>
                    <td data-label="Databasus">
                      Скрипт в одну строку или Docker
                    </td>
                    <td data-label="Barman">Ручная настройка</td>
                  </tr>
                  <tr>
                    <td>Управление бекапами</td>
                    <td data-label="Databasus">✅ Да</td>
                    <td data-label="Barman">❌ Нет</td>
                  </tr>
                  <tr>
                    <td>Подходит для self-hosted БД</td>
                    <td data-label="Databasus">✅ Да</td>
                    <td data-label="Barman">✅ Да</td>
                  </tr>
                  <tr>
                    <td>Подходит для облачных БД</td>
                    <td data-label="Databasus">
                      ✅ Да (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="Barman">
                      ❌ Нет (нужен доступ к файловой системе)
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="target-audience">Целевая аудитория</h2>

              <p>
                Самое существенное различие между инструментами — для кого они
                сделаны:
              </p>

              <h3 id="audience-databasus">Аудитория Databasus</h3>

              <p>
                Databasus сделан для широкой аудитории, от соло-разработчиков до
                крупных компаний:
              </p>

              <ul>
                <li>
                  <strong>Соло-разработчики</strong>: простая установка и
                  понятный интерфейс позволяют защитить личные проекты без
                  глубоких знаний PostgreSQL.
                </li>
                <li>
                  <strong>Команды разработки</strong>: рабочие пространства,
                  ролевой доступ и журнал аудита позволяют безопасно работать
                  вместе.
                </li>
                <li>
                  <strong>Крупные компании</strong>: масштабируется под
                  корпоративные нужды с полноценной безопасностью, множеством
                  хранилищ и каналов уведомлений.
                </li>
                <li>
                  <strong>DBA и аварийное восстановление</strong>: физические
                  бекапы, архивирование WAL и PITR для критически важных систем,
                  где потерю данных нужно свести почти к нулю.
                </li>
              </ul>

              <h3 id="audience-barman">Аудитория Barman</h3>

              <p>
                Barman создан специально для администраторов баз данных (DBA),
                которые управляют корпоративной инфраструктурой PostgreSQL:
              </p>

              <ul>
                <li>
                  <strong>Корпоративные DBA</strong>: специалисты, которым нужно
                  централизованное управление бекапами нескольких серверов
                  PostgreSQL с выделенного сервера бекапов.
                </li>
                <li>
                  <strong>
                    Команды, которым нужны инкрементальные бекапы на rsync
                  </strong>
                  : сравнение на уровне файлов сокращает время бекапа и сетевой
                  трафик для больших кластеров.
                </li>
                <li>
                  <strong>Кому нужно георезервирование</strong>: репликация
                  Barman-to-Barman для географической избыточности между
                  дата-центрами.
                </li>
              </ul>

              <h2 id="backup-approach">Подход к бекапам</h2>

              <p>
                Инструменты используют принципиально разные стратегии резервного
                копирования, у каждой свои плюсы:
              </p>

              <h3 id="backup-databasus">
                Databasus: логические + физические бекапы
              </h3>

              <p>
                Databasus поддерживает обе стратегии — и логическую, и
                физическую:
              </p>

              <ul>
                <li>
                  <strong>Физические, инкрементальные и WAL-бекапы</strong>:
                  выполняются удаленно по протоколу репликации PostgreSQL на
                  нативном стеке PostgreSQL 17 — <code>pg_basebackup</code>,
                  поблочный <code>pg_basebackup --incremental</code> на основе
                  серверных сводок WAL, <code>pg_receivewal</code> и{" "}
                  <code>pg_combinebackup</code>. Databasus опирается на
                  проверенный инструментарий самого PostgreSQL, а не изобретает
                  свой. Требуется PostgreSQL 17 или новее.
                </li>
                <li>
                  <strong>Логические бекапы</strong>: через <code>pg_dump</code>{" "}
                  создаются переносимые дампы, которые можно восстановить на
                  других версиях PostgreSQL. Это же единственный тип бекапа на
                  PostgreSQL старше 17 и путь для MySQL, MariaDB и MongoDB.
                </li>
                <li>
                  <strong>Ничего не ставится на сервер БД</strong>: Databasus
                  подключается удаленно; в закрытые сети доступ идет через
                  SSH-туннель к внутреннему хосту или бастиону, так что базу не
                  нужно открывать наружу.
                </li>
                <li>
                  <strong>Эффективное сжатие</strong>: zstd (уровень 5) для
                  логических и физических бекапов.
                </li>
                <li>
                  <strong>Доступ только на чтение</strong>: логическим бекапам
                  достаточно прав SELECT, что снижает риски безопасности.
                </li>
              </ul>

              <h3 id="backup-barman">Barman: физические бекапы</h3>

              <p>
                Barman делает файловые (физические) бекапы каталога данных
                PostgreSQL:
              </p>

              <ul>
                <li>
                  <strong>Полный бекап кластера</strong>: снимает весь кластер
                  базы данных на уровне файловой системы через rsync или
                  pg_basebackup.
                </li>
                <li>
                  <strong>Архивирование WAL</strong>: журналы предзаписи
                  архивируются непрерывно для восстановления на любой момент
                  времени.
                </li>
                <li>
                  <strong>Инкрементальность через rsync</strong>: rsync передает
                  только измененные файлы, сокращая время бекапа и сетевой
                  трафик.
                </li>
                <li>
                  <strong>Интеграция с потоковой репликацией</strong>: может
                  получать WAL-файлы по протоколу репликации и архивировать их в
                  реальном времени.
                </li>
              </ul>

              <h2 id="recovery-options">Варианты восстановления</h2>

              <p>
                Оба инструмента умеют восстанавливать данные, но с разной
                гранулярностью:
              </p>

              <h3 id="recovery-databasus">Восстановление в Databasus</h3>

              <ul>
                <li>
                  <strong>Восстановление на момент времени</strong>: откат к
                  любой конкретной секунде через проигрывание WAL.
                </li>
                <li>
                  <strong>Полное восстановление кластера</strong>: весь кластер
                  базы данных восстанавливается на нужный момент времени из
                  физических бекапов.
                </li>
                <li>
                  <strong>Логическое восстановление</strong>: восстановление из
                  логических бекапов по расписанию на любую точку бекапа.
                </li>
                <li>
                  <strong>Восстановление в один клик</strong>: скачивайте и
                  восстанавливайте логические бекапы прямо из веб-интерфейса.
                </li>
                <li>
                  <strong>Совместимость между версиями</strong>: логические
                  бекапы можно восстановить на других версиях PostgreSQL.
                </li>
              </ul>

              <h3 id="recovery-barman">Восстановление в Barman</h3>

              <ul>
                <li>
                  <strong>Восстановление на момент времени (PITR)</strong>:
                  откат к любой конкретной секунде через проигрывание WAL с
                  минимальной потерей данных.
                </li>
                <li>
                  <strong>Полное восстановление кластера</strong>: весь кластер
                  базы данных восстанавливается на нужный момент времени.
                </li>
                <li>
                  <strong>Удаленное восстановление</strong>: восстановление баз
                  на удаленные серверы по SSH.
                </li>
                <li>
                  <strong>Создание реплик</strong>: создание standby-реплик
                  PostgreSQL из бекапов для отказоустойчивых схем.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Примечание:</strong> оба
                  инструмента поддерживают PITR. Barman дополнительно умеет
                  создавать standby-реплики из бекапов и восстанавливать на
                  другие серверы по SSH, что ценно для отказоустойчивых схем.{" "}
                  <a
                    href="/ru/faq#pitr"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Как Databasus поддерживает PITR →
                  </a>
                </p>
              </div>

              <h2 id="ease-of-use">Удобство использования</h2>

              <p>
                В повседневной работе инструменты ощущаются совсем по-разному:
              </p>

              <h3 id="ease-databasus">Работа с Databasus</h3>

              <ul>
                <li>
                  <strong>Веб-интерфейс</strong>: все настройки бекапов задаются
                  мышью, командная строка не нужна.
                </li>
                <li>
                  <strong>Установка за 2 минуты</strong>: cURL-скрипт в одну
                  строку или простая Docker-команда, и все работает.
                </li>
                <li>
                  <strong>Визуальный мониторинг</strong>: статус бекапов,
                  health-чеки и история — все видно на одном дашборде.
                </li>
                <li>
                  <strong>Встроенные уведомления</strong>: Slack, Teams,
                  Telegram, Email или вебхуки настраиваются прямо в интерфейсе.
                </li>
                <li>
                  <strong>Экспертиза PostgreSQL не нужна</strong>: сделан для
                  разработчиков, которым нужны надежные бекапы без глубокого
                  погружения в базы данных.
                </li>
              </ul>

              <h3 id="ease-barman">Работа с Barman</h3>

              <ul>
                <li>
                  <strong>Интерфейс командной строки</strong>: все операции
                  выполняются командами в терминале, например{" "}
                  <code>barman backup</code>, <code>barman recover</code>.
                </li>
                <li>
                  <strong>Файлы конфигурации</strong>: для каждого сервера нужно
                  вручную править конфигурационные файлы в стиле INI.
                </li>
                <li>
                  <strong>Настройка архивирования WAL</strong>: нужно настроить{" "}
                  <code>archive_command</code> PostgreSQL или параметры
                  потоковой репликации.
                </li>
                <li>
                  <strong>Управление SSH-ключами</strong>: нужно настроить
                  SSH-ключи между сервером Barman и серверами PostgreSQL.
                </li>
                <li>
                  <strong>Без экспертизы DBA никак</strong>: документация
                  предполагает знание внутреннего устройства PostgreSQL и
                  механики WAL.
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

              <h2 id="team-features">Командные функции</h2>

              <p>Для организаций, где бекапами занимаются несколько человек:</p>

              <h3 id="team-databasus">Командные возможности Databasus</h3>

              <ul>
                <li>
                  <strong>Рабочие пространства</strong>: группируйте базы,
                  уведомления и хранилища по проектам или командам. Пользователи
                  видят только те пространства, куда их пригласили.
                </li>
                <li>
                  <strong>Ролевой доступ</strong>: назначайте права viewer,
                  editor или admin, чтобы контролировать, что может делать
                  каждый участник.
                </li>
                <li>
                  <strong>Журнал аудита</strong>: фиксируются все действия и
                  изменения в системе. Необходим для комплаенса и подотчетности.
                </li>
                <li>
                  <strong>Общие уведомления</strong>: командные каналы
                  автоматически получают статусы бекапов.
                </li>
              </ul>

              <h3 id="team-barman">Командные возможности Barman</h3>

              <p>
                Barman — инструмент командной строки без встроенных командных
                функций:
              </p>

              <ul>
                <li>Нет управления пользователями и контроля доступа</li>
                <li>Нет журнала аудита операций</li>
                <li>
                  Координация в команде требует внешних инструментов и процессов
                </li>
                <li>
                  Доступ контролируется правами на уровне ОС и SSH-ключами
                </li>
              </ul>

              <p>
                <a
                  href="/ru/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Подробнее об управлении доступом в Databasus →
                </a>
              </p>

              <h2 id="security">Безопасность</h2>

              <p>Оба инструмента заботятся о безопасности, но по-разному:</p>

              <h3 id="security-databasus">Безопасность Databasus</h3>

              <ul>
                <li>
                  <strong>Шифрование AES-256-GCM</strong>: все пароли, токены и
                  учетные данные шифруются. Ключ шифрования хранится отдельно от
                  базы данных.
                </li>
                <li>
                  <strong>Уникальное шифрование бекапов</strong>: каждый файл
                  бекапа шифруется уникальным ключом, выведенным из
                  мастер-ключа, ID бекапа и случайной соли.
                </li>
                <li>
                  <strong>Доступ к базе только на чтение</strong>: используются
                  только права SELECT, что защищает данные от порчи даже при
                  компрометации.
                </li>
              </ul>

              <h3 id="security-barman">Безопасность Barman</h3>

              <ul>
                <li>
                  <strong>Связь по SSH</strong>: безопасное взаимодействие между
                  сервером Barman и серверами PostgreSQL идет по SSH.
                </li>
                <li>
                  <strong>Нет встроенного шифрования</strong>: Barman не шифрует
                  бекапы сам, нужны внешние инструменты или зашифрованное
                  хранилище.
                </li>
                <li>
                  <strong>Безопасность на уровне ОС</strong>: полагается на
                  права файловой системы и управление SSH-ключами.
                </li>
                <li>
                  <strong>Проверка контрольных сумм</strong>: целостность
                  бекапов проверяется контрольными суммами.
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

              <h2 id="storage-options">Варианты хранилищ</h2>

              <p>Инструменты поддерживают разные хранилища:</p>

              <h3 id="storage-databasus">Хранилища Databasus</h3>

              <p>Понятные варианты под разные сценарии:</p>

              <ul>
                <li>Локальное хранилище</li>
                <li>Amazon S3 и S3-совместимые сервисы</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (сетевые хранилища)</li>
                <li>FTP</li>
              </ul>

              <h3 id="storage-barman">Хранилища Barman</h3>

              <p>Варианты с корпоративным фокусом:</p>

              <ul>
                <li>Локальное хранилище (POSIX-файловые системы)</li>
                <li>Amazon S3 и S3-совместимые объектные хранилища</li>
                <li>
                  Географическая избыточность через репликацию Barman-to-Barman
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

              <h2 id="notifications">Уведомления</h2>

              <p>Как быть в курсе статуса бекапов:</p>

              <h3 id="notifications-databasus">Уведомления Databasus</h3>

              <p>Встроенная поддержка нескольких каналов уведомлений:</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>Email</li>
                <li>Вебхуки</li>
              </ul>

              <h3 id="notifications-barman">Уведомления Barman</h3>

              <p>
                У Barman нет встроенных уведомлений. Чтобы их получить,
                понадобятся:
              </p>

              <ul>
                <li>Собственные скрипты вокруг команд бекапа</li>
                <li>Интеграция с внешними системами мониторинга</li>
                <li>Ручной разбор логов и настройка алертов</li>
                <li>
                  Интеграция с инструментами вроде Nagios, Zabbix или
                  самописными решениями
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

              <h2 id="multi-server-management">
                Управление несколькими серверами
              </h2>

              <p>
                Оба инструмента умеют делать бекапы нескольких серверов
                PostgreSQL, но по-разному:
              </p>

              <h3 id="multi-databasus">Подход Databasus</h3>

              <ul>
                <li>
                  <strong>Расписание на каждую базу</strong>: у каждой базы
                  может быть свое расписание бекапов и свое хранилище.
                </li>
                <li>
                  <strong>Организация в рабочие пространства</strong>:
                  группируйте связанные базы в пространства для удобного
                  управления.
                </li>
                <li>
                  <strong>Единый дашборд</strong>: все бекапы баз и их статусы
                  видны в одном веб-интерфейсе.
                </li>
              </ul>

              <h3 id="multi-barman">Подход Barman</h3>

              <ul>
                <li>
                  <strong>Центральный сервер бекапов</strong>: выделенный сервер
                  Barman управляет бекапами нескольких экземпляров PostgreSQL.
                </li>
                <li>
                  <strong>Конфигурация на каждый сервер</strong>: для каждого
                  сервера PostgreSQL нужен свой файл конфигурации на сервере
                  Barman.
                </li>
                <li>
                  <strong>Георезервирование</strong>: серверы Barman могут
                  реплицироваться на другие серверы Barman для географической
                  избыточности.
                </li>
              </ul>

              <h2 id="conclusion">Вывод</h2>

              <p>
                Databasus и Barman закрывают разные потребности в экосистеме
                резервного копирования PostgreSQL. Выбор зависит от требований к
                восстановлению, команды и ее технической экспертизы.
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Выбирайте Databasus, если:
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Вы соло-разработчик, команда или компания и ищете понятное
                    решение для бекапов
                  </li>
                  <li>
                    Вы предпочитаете веб-интерфейс инструментам командной строки
                  </li>
                  <li>
                    Вам нужны командные функции (рабочие пространства, RBAC,
                    журнал аудита)
                  </li>
                  <li>
                    Вы хотите встроенные уведомления в Slack, Teams, Telegram и
                    т.д.
                  </li>
                  <li>
                    Вы хотите управлять бекапами нескольких баз с одного
                    дашборда с расписанием, уведомлениями и командными функциями
                  </li>
                  <li>
                    Вам важна быстрая настройка без глубоких знаний PostgreSQL
                  </li>
                  <li>Вам важно встроенное шифрование бекапов</li>
                  <li>
                    Вы используете управляемые облачные базы (AWS RDS, Google
                    Cloud SQL, Azure) или self-hosted PostgreSQL
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Выбирайте Barman, если:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Вам нужны физические или инкрементальные бекапы на
                    PostgreSQL старше 17 (у Barman собственный движок бекапов)
                  </li>
                  <li>
                    Вам нужны инкрементальные бекапы на rsync (сравнение на
                    уровне файлов) для сокращения времени передачи
                  </li>
                  <li>
                    Вам нужна интеграция с потоковой репликацией для
                    архивирования WAL в реальном времени
                  </li>
                  <li>Вам нужно георезервирование Barman-to-Barman</li>
                  <li>
                    Вам нужно создание standby-реплик из бекапов для
                    отказоустойчивых схем
                  </li>
                  <li>
                    Вам комфортно с командной строкой и внутренним устройством
                    PostgreSQL
                  </li>
                  <li>В вашей организации есть выделенная экспертиза DBA</li>
                </ul>
              </div>

              <p>
                Оба инструмента поддерживают физические бекапы, архивирование
                WAL и PITR, и оба созданы для аварийного восстановления с
                минимальными RTO и RPO. Databasus подходит для баз любого
                размера и сложности и дает веб-интерфейс, командные функции и
                оба типа бекапов — логические и физические — как для
                self-hosted, так и для управляемых облачных баз. Barman лучше
                подойдет, когда нужны инкрементальные бекапы на rsync,
                интеграция с потоковой репликацией, георезервирование
                Barman-to-Barman или создание standby-реплик из бекапов.
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
