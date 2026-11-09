import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Databasus vs WAL-G: сравнение инструментов резервного копирования PostgreSQL",
  description:
    "Сравнение инструментов резервного копирования PostgreSQL Databasus и WAL-G: подход к бекапам, поддержка нескольких СУБД, удобство, командные функции и когда выбирать каждый инструмент.",
  keywords: [
    "Databasus vs WAL-G",
    "сравнение резервного копирования PostgreSQL",
    "альтернатива WAL-G",
    "инструменты резервного копирования PostgreSQL",
    "сравнение бекапов баз данных",
    "pg_dump или WAL-архивирование",
    "self-hosted бекап",
    "PITR PostgreSQL",
    "архивирование WAL",
    "бекап нескольких баз данных",
  ],
  openGraph: {
    title:
      "Databasus vs WAL-G: сравнение инструментов резервного копирования PostgreSQL",
    description:
      "Сравнение инструментов резервного копирования PostgreSQL Databasus и WAL-G: подход к бекапам, поддержка нескольких СУБД, удобство, командные функции и когда выбирать каждый инструмент.",
    type: "article",
    url: getLocalizedUrl("ru", "databasus-vs-wal-g"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs WAL-G: сравнение инструментов резервного копирования PostgreSQL",
    description:
      "Сравнение инструментов резервного копирования PostgreSQL Databasus и WAL-G: подход к бекапам, поддержка нескольких СУБД, удобство, командные функции и когда выбирать каждый инструмент.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "databasus-vs-wal-g"),
    languages: getLanguageAlternates("databasus-vs-wal-g"),
  },
  robots: "index, follow",
};

export default function DatabasusVsWalGPage() {
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
              "Databasus vs WAL-G: сравнение инструментов резервного копирования PostgreSQL",
            description:
              "Подробное сравнение инструментов резервного копирования PostgreSQL Databasus и WAL-G: подход к бекапам, поддержка нескольких СУБД, удобство, командные функции и когда выбирать каждый инструмент.",
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
              <h1 id="databasus-vs-wal-g">Databasus vs WAL-G</h1>

              <p className="text-lg text-gray-400">
                И Databasus, и WAL-G созданы для аварийного восстановления с
                минимальными RTO и RPO, и оба поддерживают физические бекапы
                PostgreSQL, архивирование WAL и восстановление на момент времени
                (PITR). Databasus выполняет эти бекапы удаленно на нативном
                стеке PostgreSQL 17, то есть опирается на проверенный
                инструментарий самого PostgreSQL, а не изобретает свой движок, и
                все это в понятном веб-интерфейсе. Он подходит для баз любого
                размера и сложности. Физические бекапы требуют PostgreSQL 17 или
                новее; на старых версиях доступны только логические бекапы через{" "}
                <code>pg_dump</code>. WAL-G — инструмент командной строки с
                собственным движком, поэтому он делает физические бекапы на
                гораздо более старых версиях PostgreSQL, использует свой
                потоковый протокол ради чуть большей производительности,
                поддерживает дельта-бекапы (только измененные страницы) и
                охватывает больше СУБД, включая MS SQL, FoundationDB и
                Greenplum.
              </p>

              <h2 id="quick-comparison">Быстрое сравнение</h2>

              <p>Краткий обзор ключевых различий между Databasus и WAL-G:</p>

              <table>
                <thead>
                  <tr>
                    <th>Возможность</th>
                    <th>Databasus</th>
                    <th>WAL-G</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Управление бекапами</td>
                    <td data-label="Databasus">✅ Да (несколько БД)</td>
                    <td data-label="WAL-G">❌ Нет (только одна БД)</td>
                  </tr>
                  <tr>
                    <td>Поддержка других СУБД</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="WAL-G">✅ PostgreSQL, MySQL, MS SQL</td>
                  </tr>
                  <tr>
                    <td>Интерфейс</td>
                    <td data-label="Databasus">Веб-интерфейс</td>
                    <td data-label="WAL-G">Только командная строка</td>
                  </tr>
                  <tr>
                    <td>Тип бекапа</td>
                    <td data-label="Databasus">Логический + физический</td>
                    <td data-label="WAL-G">Физический (архивирование WAL)</td>
                  </tr>
                  <tr>
                    <td>Версия PostgreSQL для физических бекапов</td>
                    <td data-label="Databasus">17+ (нативный стек)</td>
                    <td data-label="WAL-G">9.x+ (собственный движок)</td>
                  </tr>
                  <tr>
                    <td>Расписание бекапов</td>
                    <td data-label="Databasus">✅ Встроенный планировщик</td>
                    <td data-label="WAL-G">Нужен внешний (cron)</td>
                  </tr>
                  <tr>
                    <td>Варианты восстановления</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="WAL-G">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Инкрементальные бекапы</td>
                    <td data-label="Databasus">✅ На уровне блоков (PG 17+)</td>
                    <td data-label="WAL-G">
                      Дельта-бекапы (только измененные страницы)
                    </td>
                  </tr>
                  <tr>
                    <td>Удаленные бекапы</td>
                    <td data-label="Databasus">✅ Да</td>
                    <td data-label="WAL-G">❌ Нет (работает локально)</td>
                  </tr>
                  <tr>
                    <td>Командные функции</td>
                    <td data-label="Databasus">
                      ✅ Рабочие пространства, RBAC, журнал аудита
                    </td>
                    <td data-label="WAL-G">❌ Только права на уровне ОС</td>
                  </tr>
                  <tr>
                    <td>Уведомления</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, Email
                    </td>
                    <td data-label="WAL-G">❌ Нужны собственные скрипты</td>
                  </tr>
                  <tr>
                    <td>Шифрование</td>
                    <td data-label="Databasus">Встроенное AES-256-GCM</td>
                    <td data-label="WAL-G">GPG или libsodium</td>
                  </tr>
                  <tr>
                    <td>Порог входа</td>
                    <td data-label="Databasus">Минимальный</td>
                    <td data-label="WAL-G">Нужно владеть CLI</td>
                  </tr>
                  <tr>
                    <td>Установка</td>
                    <td data-label="Databasus">
                      Скрипт в одну строку или Docker
                    </td>
                    <td data-label="WAL-G">Скачивание бинарника + настройка</td>
                  </tr>
                  <tr>
                    <td>Подходит для self-hosted БД</td>
                    <td data-label="Databasus">✅ Да</td>
                    <td data-label="WAL-G">✅ Да</td>
                  </tr>
                  <tr>
                    <td>Подходит для облачных БД</td>
                    <td data-label="Databasus">
                      ✅ Да (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="WAL-G">
                      ❌ Только бекап (без восстановления в облако)
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="database-focus">Фокус по базам данных</h2>

              <p>
                Одно из самых заметных различий между инструментами — охват баз
                данных:
              </p>

              <h3 id="focus-databasus">
                Databasus: комплексное управление бекапами
              </h3>

              <p>
                Databasus создан для комплексного управления бекапами нескольких
                СУБД с упором на простоту:
              </p>

              <ul>
                <li>
                  <strong>Несколько СУБД</strong>: управляйте бекапами
                  PostgreSQL, MySQL, MariaDB и MongoDB из одного интерфейса.
                </li>
                <li>
                  <strong>Единообразие</strong>: интерфейс, сценарии работы и
                  функции одинаковы для всех поддерживаемых баз.
                </li>
                <li>
                  <strong>Поддержка версий</strong>: PostgreSQL с 12 по 18
                  версию с оптимизациями под конкретные версии.
                </li>
                <li>
                  <strong>Сфокусированная разработка</strong>: все усилия
                  команды идут на улучшение управления бекапами.
                </li>
              </ul>

              <h3 id="focus-wal-g">WAL-G: поддержка нескольких СУБД</h3>

              <p>
                WAL-G начинался как инструмент бекапов PostgreSQL, но со
                временем стал поддерживать и другие системы:
              </p>

              <ul>
                <li>
                  <strong>PostgreSQL</strong>: изначальная и самая зрелая
                  реализация.
                </li>
                <li>
                  <strong>MySQL/MariaDB</strong>: бекапы на основе binlog.
                </li>
                <li>
                  <strong>MS SQL Server</strong>: бекапы SQL Server под Windows.
                </li>
                <li>
                  <strong>MongoDB</strong>: поддержка документной СУБД.
                </li>
                <li>
                  <strong>FoundationDB</strong>: поддержка распределенной СУБД.
                </li>
                <li>
                  <strong>Greenplum</strong>: бекапы хранилища данных.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">
                    Когда важно комплексное управление:
                  </strong>{" "}
                  если вам нужно управлять бекапами нескольких баз из единого
                  интерфейса, Databasus для этого и сделан: централизованное
                  управление бекапами и командные функции без жонглирования
                  разными инструментами для разных СУБД.
                </p>
              </div>

              <h2 id="target-audience">Целевая аудитория</h2>

              <p>
                Инструменты рассчитаны на разных пользователей — философия у них
                разная:
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
                  <strong>Окружения с несколькими СУБД</strong>: организации,
                  где работают PostgreSQL, MySQL, MariaDB или MongoDB, получают
                  централизованное управление бекапами.
                </li>
                <li>
                  <strong>DBA и аварийное восстановление</strong>: физические
                  бекапы, архивирование WAL и PITR для критически важных систем,
                  где потерю данных нужно свести почти к нулю.
                </li>
                <li>
                  <strong>DevOps-инженеры</strong>: режим агента встраивается в
                  существующую инфраструктуру, а веб-интерфейс и API дают
                  наблюдаемость и контроль без собственных скриптов.
                </li>
              </ul>

              <h3 id="audience-wal-g">Аудитория WAL-G</h3>

              <p>
                WAL-G рассчитан на пользователей, которым комфортно в командной
                строке:
              </p>

              <ul>
                <li>
                  <strong>DevOps-инженеры</strong>: те, кто предпочитает
                  infrastructure-as-code и работу через CLI.
                </li>
                <li>
                  <strong>Окружения с несколькими СУБД</strong>: организации,
                  где PostgreSQL работает рядом с MySQL, MongoDB и другими
                  поддерживаемыми базами.
                </li>
                <li>
                  <strong>Cloud-native развертывания</strong>: команды с
                  Kubernetes или контейнерными окружениями, куда CLI-инструменты
                  хорошо встраиваются.
                </li>
                <li>
                  <strong>Расширенная поддержка СУБД</strong>: команды, которым
                  помимо PostgreSQL нужны бекапы MS SQL, FoundationDB или
                  Greenplum.
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
                  обоих типов бекапов уменьшает размер в 4-8 раз.
                </li>
                <li>
                  <strong>Доступ только на чтение</strong>: логическим бекапам
                  достаточно прав SELECT, что снижает риски безопасности.
                </li>
              </ul>

              <h3 id="backup-wal-g">
                WAL-G: физические бекапы с архивированием WAL
              </h3>

              <p>
                WAL-G делает файловые (физические) бекапы с непрерывным
                архивированием WAL:
              </p>

              <ul>
                <li>
                  <strong>Базовые бекапы</strong>: полные файловые копии
                  каталога данных PostgreSQL.
                </li>
                <li>
                  <strong>Дельта-бекапы</strong>: копируются только измененные
                  страницы, что экономит хранилище и время передачи.
                </li>
                <li>
                  <strong>Архивирование WAL</strong>: журналы предзаписи
                  архивируются непрерывно — это дает восстановление на любой
                  момент времени.
                </li>
                <li>
                  <strong>Оптимизация copy-on-write</strong>: эффективная
                  обработка неизменившихся блоков данных.
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

              <h3 id="recovery-wal-g">Восстановление в WAL-G</h3>

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
                  <strong>Дельта-восстановление</strong>: быстрее за счет
                  загрузки только измененных страниц.
                </li>
                <li>
                  <strong>Создание реплик</strong>: создание standby-реплик
                  PostgreSQL из бекапов для отказоустойчивых схем.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Примечание:</strong> оба
                  инструмента поддерживают PITR. WAL-G дополнительно предлагает
                  дельта-восстановление (загрузку только измененных страниц) и
                  собственный потоковый протокол ради чуть большей
                  производительности на больших объемах.{" "}
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

              <h3 id="ease-wal-g">Работа с WAL-G</h3>

              <ul>
                <li>
                  <strong>Интерфейс командной строки</strong>: все операции
                  выполняются командами в терминале, например{" "}
                  <code>wal-g backup-push</code>,{" "}
                  <code>wal-g backup-fetch</code>.
                </li>
                <li>
                  <strong>Переменные окружения</strong>: конфигурация в основном
                  через переменные окружения, а не конфиг-файлы.
                </li>
                <li>
                  <strong>Внешнее расписание</strong>: для автоматических
                  бекапов нужен cron или внешняя оркестрация.
                </li>
                <li>
                  <strong>Настройка архивирования WAL</strong>: нужно настроить{" "}
                  <code>archive_command</code> PostgreSQL для интеграции с
                  WAL-G.
                </li>
                <li>
                  <strong>Без CLI никак</strong>: документация предполагает
                  знание командной строки и shell-скриптов.
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

              <h3 id="team-wal-g">Командные возможности WAL-G</h3>

              <p>
                WAL-G — инструмент командной строки без встроенных командных
                функций:
              </p>

              <ul>
                <li>Нет управления пользователями и контроля доступа</li>
                <li>Нет журнала аудита операций</li>
                <li>
                  Координация в команде требует внешних инструментов и процессов
                </li>
                <li>
                  Доступ контролируется правами на уровне ОС и IAM-политиками
                  облака
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

              <h3 id="security-wal-g">Безопасность WAL-G</h3>

              <ul>
                <li>
                  <strong>Шифрование GPG</strong>: поддержка шифрования файлов
                  бекапов через GPG.
                </li>
                <li>
                  <strong>Шифрование libsodium</strong>: альтернативное
                  шифрование библиотекой libsodium.
                </li>
                <li>
                  <strong>Интеграция с облачным IAM</strong>: доступ к хранилищу
                  контролируется через IAM облачного провайдера.
                </li>
                <li>
                  <strong>Нет встроенного управления секретами</strong>:
                  полагается на переменные окружения или внешние менеджеры
                  секретов.
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
                Оба инструмента поддерживают облачные хранилища, но с разным
                фокусом:
              </p>

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

              <h3 id="storage-wal-g">Хранилища WAL-G</h3>

              <p>Cloud-native варианты:</p>

              <ul>
                <li>Amazon S3</li>
                <li>Google Cloud Storage (GCS)</li>
                <li>Azure Blob Storage</li>
                <li>Swift (OpenStack)</li>
                <li>Локальная файловая система</li>
                <li>SSH/SFTP</li>
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

              <h3 id="notifications-wal-g">Уведомления WAL-G</h3>

              <p>
                У WAL-G нет встроенных уведомлений. Чтобы их получить,
                понадобятся:
              </p>

              <ul>
                <li>Собственные скрипты вокруг команд бекапа</li>
                <li>Интеграция с внешними системами мониторинга</li>
                <li>Ручной разбор логов и настройка алертов</li>
                <li>
                  Интеграция с инструментами вроде Prometheus, Grafana или
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

              <h2 id="compression">Сжатие</h2>

              <p>
                Оба инструмента сжимают бекапы, чтобы они занимали меньше места:
              </p>

              <h3 id="compression-databasus">Сжатие в Databasus</h3>

              <ul>
                <li>
                  <strong>Сжатие zstd</strong>: используется zstd уровня 5 как
                  баланс скорости и степени сжатия.
                </li>
                <li>
                  <strong>Уменьшение размера в 4-8 раз</strong>: типичные
                  коэффициенты сжатия при накладных расходах всего ~20% времени.
                </li>
                <li>
                  <strong>Автоматически</strong>: сжатие включено по умолчанию и
                  не требует настройки.
                </li>
              </ul>

              <h3 id="compression-wal-g">Сжатие в WAL-G</h3>

              <ul>
                <li>
                  <strong>Несколько алгоритмов</strong>: поддержка LZ4, LZMA,
                  Brotli и zstd.
                </li>
                <li>
                  <strong>Настраиваемые уровни</strong>: тонкая настройка
                  баланса между степенью сжатия и скоростью.
                </li>
                <li>
                  <strong>Раздельное сжатие</strong>: для WAL-файлов и базовых
                  бекапов можно задать разные настройки.
                </li>
              </ul>

              <h2 id="conclusion">Вывод</h2>

              <p>
                Databasus и WAL-G закрывают разные потребности в экосистеме
                резервного копирования PostgreSQL. Выбор зависит от вашего
                окружения, команды и того, как вам удобнее работать.
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Выбирайте Databasus, если:
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Вам нужно комплексное управление бекапами PostgreSQL из
                    одного интерфейса
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
                    Вы хотите встроенное расписание без внешней настройки cron
                  </li>
                  <li>
                    Вы хотите управлять бекапами нескольких баз с одного
                    дашборда с расписанием, уведомлениями и командными функциями
                  </li>
                  <li>
                    Вам важна быстрая настройка без глубоких знаний баз данных
                  </li>
                  <li>Вам важно встроенное шифрование бекапов</li>
                  <li>
                    Вы используете управляемые облачные базы (AWS RDS, Google
                    Cloud SQL, Azure) или self-hosted базы
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Выбирайте WAL-G, если:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Вам нужны физические или инкрементальные бекапы на
                    PostgreSQL старше 17 (у WAL-G собственный движок бекапов)
                  </li>
                  <li>
                    Вам нужны дельта-бекапы (только измененные страницы) для
                    экономии хранилища и времени передачи
                  </li>
                  <li>
                    Вам нужна поддержка MS SQL, FoundationDB или Greenplum
                  </li>
                  <li>
                    Вы предпочитаете командную строку и infrastructure-as-code
                  </li>
                  <li>
                    Вам нужны несколько алгоритмов сжатия (LZ4, LZMA, Brotli,
                    zstd) с тонкой настройкой
                  </li>
                  <li>
                    У вашей команды есть DevOps-экспертиза для работы с
                    CLI-инструментами
                  </li>
                </ul>
              </div>

              <p>
                Оба инструмента поддерживают физические бекапы, архивирование
                WAL и PITR, и оба созданы для аварийного восстановления с
                минимальными RTO и RPO. Databasus подходит для баз любого
                размера и сложности и дает веб-интерфейс, командные функции и
                оба типа бекапов — логические и физические — как для
                self-hosted, так и для управляемых облачных баз.
                <br />
                <br />
                WAL-G остается отличным выбором для команд, которые предпочитают
                CLI и которым нужны его уникальные преимущества: дельта-бекапы
                (только измененные страницы), собственный потоковый протокол
                ради чуть большей производительности и поддержка дополнительных
                СУБД помимо PostgreSQL.
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
