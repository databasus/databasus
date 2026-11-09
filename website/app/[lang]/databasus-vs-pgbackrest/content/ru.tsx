import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Databasus vs pgBackRest: сравнение инструментов резервного копирования PostgreSQL",
  description:
    "Сравнение инструментов резервного копирования PostgreSQL Databasus и pgBackRest: подход к бекапам, целевая аудитория, удобство, варианты восстановления и когда выбирать каждый инструмент.",
  keywords: [
    "Databasus vs pgBackRest",
    "сравнение резервного копирования PostgreSQL",
    "альтернатива pgBackRest",
    "инструменты резервного копирования PostgreSQL",
    "сравнение бекапов баз данных",
    "pg_dump или физический бекап",
    "self-hosted бекап",
    "PITR PostgreSQL",
    "бекап больших баз данных",
    "инструменты бекапа для DBA",
  ],
  openGraph: {
    title:
      "Databasus vs pgBackRest: сравнение инструментов резервного копирования PostgreSQL",
    description:
      "Сравнение инструментов резервного копирования PostgreSQL Databasus и pgBackRest: подход к бекапам, целевая аудитория, удобство, варианты восстановления и когда выбирать каждый инструмент.",
    type: "article",
    url: getLocalizedUrl("ru", "databasus-vs-pgbackrest"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs pgBackRest: сравнение инструментов резервного копирования PostgreSQL",
    description:
      "Сравнение инструментов резервного копирования PostgreSQL Databasus и pgBackRest: подход к бекапам, целевая аудитория, удобство, варианты восстановления и когда выбирать каждый инструмент.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "databasus-vs-pgbackrest"),
    languages: getLanguageAlternates("databasus-vs-pgbackrest"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackRestPage() {
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
              "Databasus vs pgBackRest: сравнение инструментов резервного копирования PostgreSQL",
            description:
              "Подробное сравнение инструментов резервного копирования PostgreSQL Databasus и pgBackRest: подход к бекапам, целевая аудитория, удобство, варианты восстановления и когда выбирать каждый инструмент.",
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
              <h1 id="databasus-vs-pgbackrest">Databasus vs pgBackRest</h1>

              <p className="text-lg text-gray-400">
                И Databasus, и pgBackRest созданы для аварийного восстановления
                с минимальными RTO и RPO, и оба поддерживают физические бекапы,
                архивирование WAL и восстановление на момент времени (PITR).
                Databasus выполняет эти бекапы удаленно на нативном стеке
                PostgreSQL 17, то есть опирается на проверенный инструментарий
                самого PostgreSQL, а не изобретает свой — и все это за понятным
                веб-интерфейсом. Он подходит для баз любого размера и сложности.
                Физические бекапы требуют PostgreSQL 17 или новее; на старых
                версиях доступны только логические бекапы через{" "}
                <code>pg_dump</code>. pgBackRest поставляется с собственным
                движком бекапов, поэтому он делает физические и инкрементальные
                бекапы и на гораздо более старых версиях PostgreSQL, а еще
                добавляет продвинутые функции вроде дифференциальных бекапов и
                дельта-восстановления.
              </p>

              <h2 id="quick-comparison">Быстрое сравнение</h2>

              <p>
                Краткий обзор ключевых различий между Databasus и pgBackRest:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Возможность</th>
                    <th>Databasus</th>
                    <th>pgBackRest</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Целевая аудитория</td>
                    <td data-label="Databasus">
                      Разработчики, команды, DBA, крупные компании
                    </td>
                    <td data-label="pgBackRest">
                      DBA, командная строка и IaC-процессы
                    </td>
                  </tr>
                  <tr>
                    <td>Управление бекапами</td>
                    <td data-label="Databasus">✅ Несколько БД</td>
                    <td data-label="pgBackRest">❌ Только одна БД</td>
                  </tr>
                  <tr>
                    <td>Поддержка других СУБД</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="pgBackRest">❌ Только PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>Интерфейс</td>
                    <td data-label="Databasus">Веб-интерфейс</td>
                    <td data-label="pgBackRest">
                      Командная строка, файлы конфигурации
                    </td>
                  </tr>
                  <tr>
                    <td>Тип бекапа</td>
                    <td data-label="Databasus">Логический + физический</td>
                    <td data-label="pgBackRest">
                      Физический (на уровне файлов)
                    </td>
                  </tr>
                  <tr>
                    <td>Версия PostgreSQL для физических бекапов</td>
                    <td data-label="Databasus">17+ (нативный стек)</td>
                    <td data-label="pgBackRest">9.4+ (собственный движок)</td>
                  </tr>
                  <tr>
                    <td>Варианты восстановления</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="pgBackRest">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Параллельные операции</td>
                    <td data-label="Databasus">✅ Да</td>
                    <td data-label="pgBackRest">✅ Да</td>
                  </tr>
                  <tr>
                    <td>Инкрементальные бекапы</td>
                    <td data-label="Databasus">✅ На уровне блоков (PG 17+)</td>
                    <td data-label="pgBackRest">
                      Инкрементальные на уровне блоков
                    </td>
                  </tr>
                  <tr>
                    <td>Дифференциальные бекапы</td>
                    <td data-label="Databasus">❌ Нет</td>
                    <td data-label="pgBackRest">✅ Да</td>
                  </tr>
                  <tr>
                    <td>Дельта-восстановление</td>
                    <td data-label="Databasus">❌ Нет</td>
                    <td data-label="pgBackRest">
                      ✅ Да (только измененные файлы)
                    </td>
                  </tr>
                  <tr>
                    <td>Удаленные бекапы</td>
                    <td data-label="Databasus">✅ Да</td>
                    <td data-label="pgBackRest">
                      ❌ Нет (нужен доступ к файловой системе)
                    </td>
                  </tr>
                  <tr>
                    <td>Командные функции</td>
                    <td data-label="Databasus">
                      ✅ Рабочие пространства, RBAC, журнал аудита
                    </td>
                    <td data-label="pgBackRest">❌ Один пользователь</td>
                  </tr>
                  <tr>
                    <td>Порог входа</td>
                    <td data-label="Databasus">Минимальный</td>
                    <td data-label="pgBackRest">Нужна экспертиза DBA</td>
                  </tr>
                  <tr>
                    <td>Установка</td>
                    <td data-label="Databasus">
                      Скрипт в одну строку или Docker
                    </td>
                    <td data-label="pgBackRest">Ручная настройка</td>
                  </tr>
                  <tr>
                    <td>Подходит для self-hosted БД</td>
                    <td data-label="Databasus">✅ Да</td>
                    <td data-label="pgBackRest">✅ Да</td>
                  </tr>
                  <tr>
                    <td>Подходит для облачных БД</td>
                    <td data-label="Databasus">
                      ✅ Да (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="pgBackRest">
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
                  ролевой доступ и журнал аудита дают безопасную совместную
                  работу.
                </li>
                <li>
                  <strong>Крупные компании</strong>: масштабируется под
                  корпоративные нужды с полноценной безопасностью, множеством
                  хранилищ и каналов уведомлений.
                </li>
                <li>
                  <strong>DBA и аварийное восстановление</strong>: физические
                  бекапы, архивирование WAL и PITR для критически важных систем,
                  где потеря данных почти недопустима.
                </li>
              </ul>

              <h3 id="audience-pgbackrest">Аудитория pgBackRest</h3>

              <p>
                pgBackRest — инструмент командной строки для команд, которые
                хотят запускать движок бекапов самостоятельно:
              </p>

              <ul>
                <li>
                  <strong>Командная строка и IaC-процессы</strong>: команды,
                  которым удобнее настраивать бекапы файлами и скриптами, а не
                  через веб-интерфейс.
                </li>
                <li>
                  <strong>Старые версии PostgreSQL</strong>: у pgBackRest свой
                  движок, поэтому он делает физические и инкрементальные бекапы
                  на PostgreSQL старше 17.
                </li>
                <li>
                  <strong>Продвинутые функции на больших объемах</strong>: там,
                  где дифференциальные бекапы, дельта-восстановление и создание
                  standby-реплик оправдывают дополнительную настройку.
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
                Databasus поддерживает обе стратегии, логическую и физическую:
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
                  других версиях PostgreSQL, с выборочным восстановлением
                  отдельных таблиц или схем. Это же единственный тип бекапа на
                  PostgreSQL старше 17 и путь для MySQL, MariaDB и MongoDB.
                </li>
                <li>
                  <strong>Ничего не ставится на сервер БД</strong>: бекапы
                  подключаются удаленно; в закрытые сети доступ идет через
                  SSH-туннель к внутреннему хосту или бастиону, так что базу не
                  нужно открывать наружу.
                </li>
                <li>
                  <strong>Эффективное сжатие</strong>: zstd (уровень 5) для
                  обоих типов бекапов уменьшает размер в 4-8 раз.
                </li>
                <li>
                  <strong>Доступ только на чтение</strong>: логическим бекапам
                  достаточно прав SELECT, что снижает риски безопасности.
                </li>
              </ul>

              <h3 id="backup-pgbackrest">pgBackRest: физические бекапы</h3>

              <p>
                pgBackRest делает файловые (физические) бекапы каталога данных
                PostgreSQL:
              </p>

              <ul>
                <li>
                  <strong>Инкрементальность на уровне блоков</strong>:
                  копируются только измененные блоки, что сокращает время бекапа
                  и объем хранилища для очень больших баз.
                </li>
                <li>
                  <strong>Архивирование WAL</strong>: WAL архивируется
                  непрерывно, что дает точное восстановление на момент времени.
                </li>
                <li>
                  <strong>Полные, дифференциальные, инкрементальные</strong>:
                  несколько стратегий бекапа под разные сценарии восстановления.
                </li>
                <li>
                  <strong>Оптимизирован под масштаб</strong>: рассчитан на базы,
                  для которых логические бекапы шли бы слишком долго.
                </li>
              </ul>

              <h2 id="recovery-options">Варианты восстановления</h2>

              <p>
                Оба инструмента дают гибкие варианты восстановления, но с разной
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

              <h3 id="recovery-pgbackrest">Восстановление в pgBackRest</h3>

              <ul>
                <li>
                  <strong>Восстановление на момент времени (PITR)</strong>:
                  откат к любой конкретной секунде через проигрывание WAL.
                </li>
                <li>
                  <strong>Полное восстановление кластера</strong>: весь кластер
                  базы данных восстанавливается на нужный момент времени из
                  физических бекапов.
                </li>
                <li>
                  <strong>Параллельное восстановление</strong>: многопоточное
                  восстановление ускоряет работу с большими базами.
                </li>
                <li>
                  <strong>Дельта-восстановление</strong>: восстанавливаются
                  только измененные файлы, что сокращает время.
                </li>
                <li>
                  <strong>Создание реплик</strong>: создание реплик PostgreSQL
                  из бекапов.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Примечание:</strong> оба
                  инструмента поддерживают PITR. pgBackRest дополнительно
                  предлагает дельта-восстановление (загрузку только измененных
                  файлов), дифференциальные бекапы и создание standby-реплик из
                  бекапов.{" "}
                  <a
                    href="/ru/faq#pitr"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Как Databasus поддерживает PITR →
                  </a>
                </p>
              </div>

              <h2 id="ease-of-use">Удобство использования</h2>

              <p>Здесь инструменты различаются кардинально:</p>

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
                  <strong>Визуальный мониторинг</strong>: дашборд показывает
                  статус бекапов, health-чеки и историю с одного взгляда.
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

              <h3 id="ease-pgbackrest">Работа с pgBackRest</h3>

              <ul>
                <li>
                  <strong>Интерфейс командной строки</strong>: все операции
                  выполняются командами в терминале.
                </li>
                <li>
                  <strong>Файлы конфигурации</strong>: нужно вручную править
                  конфигурационные файлы в стиле INI.
                </li>
                <li>
                  <strong>Настройка архивирования WAL</strong>: нужно настроить{" "}
                  <code>archive_command</code> PostgreSQL и связанные параметры.
                </li>
                <li>
                  <strong>Высокий порог входа</strong>: требуется понимание
                  внутреннего устройства PostgreSQL, механики WAL и стратегий
                  бекапа.
                </li>
                <li>
                  <strong>Ожидается экспертиза DBA</strong>: документация
                  предполагает знакомство с администрированием баз данных.
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
                  видят только те рабочие пространства, куда их пригласили.
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

              <h3 id="team-pgbackrest">Командные возможности pgBackRest</h3>

              <p>
                pgBackRest — инструмент командной строки без встроенных
                командных функций:
              </p>

              <ul>
                <li>Нет управления пользователями и контроля доступа</li>
                <li>Нет журнала аудита операций</li>
                <li>
                  Координация в команде требует внешних инструментов и процессов
                </li>
                <li>
                  Доступ контролируется правами на уровне ОС на файлы
                  конфигурации
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

              <p>Оба инструмента дают серьезные средства безопасности:</p>

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

              <h3 id="security-pgbackrest">Безопасность pgBackRest</h3>

              <ul>
                <li>
                  <strong>Шифрование репозитория</strong>: репозитории бекапов
                  можно шифровать AES-256.
                </li>
                <li>
                  <strong>Транспорт TLS/SSH</strong>: безопасная связь для
                  удаленных операций.
                </li>
                <li>
                  <strong>Проверка контрольных сумм</strong>: целостность
                  бекапов проверяется при создании и восстановлении.
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

              <p>
                Оба инструмента поддерживают несколько хранилищ, но у Databasus
                больше вариантов для обычных пользователей:
              </p>

              <h3 id="storage-databasus">Хранилища Databasus</h3>

              <ul>
                <li>Локальное хранилище</li>
                <li>Amazon S3 и S3-совместимые сервисы</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (сетевые хранилища)</li>
                <li>FTP</li>
              </ul>

              <h3 id="storage-pgbackrest">Хранилища pgBackRest</h3>

              <ul>
                <li>Локальное хранилище (POSIX, CIFS)</li>
                <li>Amazon S3 и S3-совместимые сервисы</li>
                <li>Cloudflare R2 (S3-совместимое)</li>
                <li>Azure Blob Storage</li>
                <li>NAS (сетевые хранилища)</li>
                <li>Google Cloud Storage</li>
                <li>SFTP</li>
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

              <h3 id="notifications-pgbackrest">Уведомления pgBackRest</h3>

              <p>
                У pgBackRest нет встроенных уведомлений. Чтобы их получить,
                понадобятся:
              </p>

              <ul>
                <li>Собственные скрипты вокруг команд бекапа</li>
                <li>Интеграция с внешними системами мониторинга</li>
                <li>Ручной разбор логов и настройка алертов</li>
              </ul>

              <p>
                <a
                  href="/ru/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Все каналы уведомлений Databasus →
                </a>
              </p>

              <h2 id="conclusion">Вывод</h2>

              <p>
                Databasus и pgBackRest закрывают разные потребности в экосистеме
                резервного копирования PostgreSQL. Правильный выбор зависит от
                размера базы, структуры команды и технических требований.
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
                    дашборда
                  </li>
                  <li>
                    Вам важна быстрая настройка без глубоких знаний PostgreSQL
                  </li>
                  <li>
                    Вы используете облачные управляемые базы (AWS RDS, Google
                    Cloud SQL, Azure) или self-hosted PostgreSQL
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Выбирайте pgBackRest, если:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Вам нужны физические или инкрементальные бекапы на
                    PostgreSQL старше 17 (у pgBackRest собственный движок
                    бекапов)
                  </li>
                  <li>
                    Вам нужны дифференциальные бекапы или дельта-восстановление
                    (только измененные файлы)
                  </li>
                  <li>
                    Вам нужно создание standby-реплик из бекапов для
                    отказоустойчивости
                  </li>
                  <li>
                    Вы предпочитаете командную строку и infrastructure-as-code
                  </li>
                  <li>
                    У вашей команды хватает экспертизы PostgreSQL, чтобы его
                    запускать и настраивать
                  </li>
                </ul>
              </div>

              <p>
                Оба инструмента поддерживают физические бекапы, архивирование
                WAL и PITR, и оба созданы для аварийного восстановления с
                минимальными RTO и RPO. Databasus подходит для баз любого
                размера и сложности и дает веб-интерфейс, командные функции и
                оба типа бекапов, логические и физические, для self-hosted и
                облачных управляемых баз.
                <br />
                <br />
                pgBackRest лучше подойдет, когда вы хотите запускать движок
                бекапов самостоятельно, вам нужны физические бекапы на
                PostgreSQL старше 17 или вы полагаетесь на его дифференциальные
                бекапы и дельта-восстановление.
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
