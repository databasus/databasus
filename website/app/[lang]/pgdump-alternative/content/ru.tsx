import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Альтернатива pg_dump - Databasus для резервного копирования PostgreSQL",
  description:
    "Databasus построен на pg_dump и расширяет его: управление бекапами, веб-интерфейс, автоматическое расписание, облачные хранилища, уведомления, командная работа и шифрование.",
  keywords: [
    "альтернатива pg_dump",
    "pg_dump GUI",
    "автоматизация pg_dump",
    "веб-интерфейс pg_dump",
    "инструмент резервного копирования PostgreSQL",
    "планировщик pg_dump",
    "pg_dump облачное хранилище",
    "шифрование pg_dump",
    "автоматизация бекапов PostgreSQL",
    "обертка pg_dump",
  ],
  openGraph: {
    title:
      "Альтернатива pg_dump - Databasus для резервного копирования PostgreSQL",
    description:
      "Databasus построен на pg_dump и расширяет его: управление бекапами, веб-интерфейс, автоматическое расписание, облачные хранилища, уведомления, командная работа и шифрование.",
    type: "article",
    url: getLocalizedUrl("ru", "pgdump-alternative"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title:
      "Альтернатива pg_dump - Databasus для резервного копирования PostgreSQL",
    description:
      "Databasus построен на pg_dump и расширяет его: управление бекапами, веб-интерфейс, автоматическое расписание, облачные хранилища, уведомления, командная работа и шифрование.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "pgdump-alternative"),
    languages: getLanguageAlternates("pgdump-alternative"),
  },
  robots: "index, follow",
};

export default function PgDumpAlternativePage() {
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
              "Альтернатива pg_dump - Databasus для резервного копирования PostgreSQL",
            description:
              "Подробный разбор Databasus как альтернативы pg_dump: как он строится на pg_dump и расширяет его автоматизацией, облачными хранилищами, уведомлениями и командными функциями.",
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
              <h1 id="pgdump-alternative">Альтернатива pg_dump</h1>

              <p className="text-lg text-gray-400">
                Для логических бекапов Databasus построен поверх{" "}
                <code>pg_dump</code>. Databasus не заменяет <code>pg_dump</code>
                , а надстраивается над ним: управление бекапами, веб-интерфейс,
                автоматическое расписание, интеграция с облачными хранилищами,
                уведомления, командная работа и встроенное шифрование. Помимо
                логических бекапов Databasus также поддерживает физические
                бекапы, инкрементальные бекапы с архивированием WAL и
                Point-in-Time Recovery.
              </p>

              <h2 id="quick-comparison">Краткое сравнение</h2>

              <p>
                Вот как Databasus расширяет базовую функциональность{" "}
                <code>pg_dump</code>:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Возможность</th>
                    <th>pg_dump</th>
                    <th>Databasus</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Движок бекапов</td>
                    <td data-label="pg_dump">pg_dump</td>
                    <td data-label="Databasus">Построен на pg_dump</td>
                  </tr>
                  <tr>
                    <td>Управление бекапами</td>
                    <td data-label="pg_dump">❌ Нет</td>
                    <td data-label="Databasus">✅ Да</td>
                  </tr>
                  <tr>
                    <td>Поддержка других СУБД</td>
                    <td data-label="pg_dump">Только PostgreSQL</td>
                    <td data-label="Databasus">
                      PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                  </tr>
                  <tr>
                    <td>Интерфейс</td>
                    <td data-label="pg_dump">Командная строка</td>
                    <td data-label="Databasus">Веб-интерфейс + API</td>
                  </tr>
                  <tr>
                    <td>Расписание</td>
                    <td data-label="pg_dump">Вручную или cron-скрипты</td>
                    <td data-label="Databasus">✅ Встроенный планировщик</td>
                  </tr>
                  <tr>
                    <td>Хранилища</td>
                    <td data-label="pg_dump">Локальная файловая система</td>
                    <td data-label="Databasus">
                      Локальный диск, S3, R2, Azure, NAS, FTP
                    </td>
                  </tr>
                  <tr>
                    <td>Сжатие</td>
                    <td data-label="pg_dump">gzip, LZ4, zstd (вручную)</td>
                    <td data-label="Databasus">
                      zstd (автоматически, оптимизировано)
                    </td>
                  </tr>
                  <tr>
                    <td>Шифрование</td>
                    <td data-label="pg_dump">Нужны внешние инструменты</td>
                    <td data-label="Databasus">✅ Встроенное AES-256-GCM</td>
                  </tr>
                  <tr>
                    <td>Уведомления</td>
                    <td data-label="pg_dump">❌ Нет</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, Email, Webhooks
                    </td>
                  </tr>
                  <tr>
                    <td>Командные функции</td>
                    <td data-label="pg_dump">❌ Нет</td>
                    <td data-label="Databasus">
                      ✅ Рабочие пространства, RBAC, журналы аудита
                    </td>
                  </tr>
                  <tr>
                    <td>Политики хранения</td>
                    <td data-label="pg_dump">Скрипты очистки вручную</td>
                    <td data-label="Databasus">✅ Автоматическая ротация</td>
                  </tr>
                  <tr>
                    <td>Мониторинг состояния</td>
                    <td data-label="pg_dump">❌ Нет</td>
                    <td data-label="Databasus">✅ Встроенные health-чеки</td>
                  </tr>
                  <tr>
                    <td>Физические бекапы</td>
                    <td data-label="pg_dump">❌ Нет</td>
                    <td data-label="Databasus">✅ Да</td>
                  </tr>
                  <tr>
                    <td>Инкрементальные бекапы</td>
                    <td data-label="pg_dump">❌ Нет</td>
                    <td data-label="Databasus">✅ На уровне блоков (PG 17+)</td>
                  </tr>
                  <tr>
                    <td>Point-in-Time Recovery</td>
                    <td data-label="pg_dump">❌ Нет</td>
                    <td data-label="Databasus">✅ Да</td>
                  </tr>
                  <tr>
                    <td>Удаленные бекапы</td>
                    <td data-label="pg_dump">✅ Да (CLI)</td>
                    <td data-label="Databasus">✅ Да</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="what-is-pgdump">Что такое pg_dump?</h2>

              <p>
                <code>pg_dump</code> — родная утилита PostgreSQL для создания
                логических бекапов. Она входит в PostgreSQL с самого начала и
                остается стандартным инструментом экспорта баз данных.
              </p>

              <h3 id="pgdump-strengths">Сильные стороны pg_dump</h3>

              <ul>
                <li>
                  <strong>Переносимые бекапы</strong>: создает дампы в формате
                  SQL или custom, которые можно восстановить на других версиях
                  PostgreSQL.
                </li>
                <li>
                  <strong>Выборочные бекапы</strong>: умеет выгружать отдельные
                  таблицы, схемы или базы целиком.
                </li>
                <li>
                  <strong>Консистентные снимки</strong>: использует MVCC
                  PostgreSQL, чтобы создавать консистентные бекапы, не блокируя
                  запись.
                </li>
                <li>
                  <strong>Повсеместная поддержка</strong>: доступен в каждой
                  установке PostgreSQL, хорошо документирован и проверен
                  временем.
                </li>
                <li>
                  <strong>Гибкие форматы вывода</strong>: обычный SQL, custom,
                  directory или tar.
                </li>
              </ul>

              <h3 id="pgdump-limitations">Ограничения pg_dump</h3>

              <p>
                <code>pg_dump</code> — мощный инструмент, но его использование в
                продакшене обычно требует дополнительных скриптов:
              </p>

              <ul>
                <li>
                  <strong>Нет встроенного расписания</strong>: нужны cron-задачи
                  или внешние планировщики.
                </li>
                <li>
                  <strong>Только локальное хранилище</strong>: пишет в локальную
                  файловую систему; загрузка в облако требует отдельных
                  скриптов.
                </li>
                <li>
                  <strong>Нет шифрования</strong>: файлы бекапов по умолчанию не
                  зашифрованы; нужно пропускать вывод через gpg или похожие
                  инструменты.
                </li>
                <li>
                  <strong>Нет уведомлений</strong>: без своих скриптов не
                  узнать, что бекап прошел успешно или упал.
                </li>
                <li>
                  <strong>Нет управления хранением</strong>: старые бекапы
                  приходится удалять вручную или скриптами.
                </li>
                <li>
                  <strong>Только командная строка</strong>: нет визуального
                  интерфейса для мониторинга и управления.
                </li>
              </ul>

              <h2 id="how-databasus-extends">
                Как Databasus расширяет pg_dump
              </h2>

              <p>
                Databasus использует <code>pg_dump</code> как движок бекапов:
                все преимущества логических бекапов сохраняются, а сверху
                добавляются корпоративные функции.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Под капотом:</strong> когда
                  вы запускаете бекап в Databasus, он выполняет{" "}
                  <code className="bg-[#374151] text-gray-200">pg_dump</code> с
                  оптимизированными параметрами, а затем берет на себя сжатие,
                  шифрование и загрузку в настроенное хранилище.
                </p>
              </div>

              <h3 id="web-interface">Веб-интерфейс</h3>

              <p>
                Вместо запоминания опций командной строки <code>pg_dump</code>{" "}
                Databasus дает веб-интерфейс, где можно:
              </p>

              <ul>
                <li>Добавлять базы через пошаговый мастер подключения</li>
                <li>Настраивать расписание бекапов визуально</li>
                <li>Смотреть историю и статусы бекапов с одного экрана</li>
                <li>Скачивать или восстанавливать бекапы в один клик</li>
                <li>Видеть графики состояния и доступности баз</li>
              </ul>

              <h3 id="optimized-compression">Оптимизированное сжатие</h3>

              <p>
                По умолчанию Databasus использует сжатие zstd (уровень 5),
                которое дает:
              </p>

              <ul>
                <li>
                  <strong>Уменьшение размера в 4-8 раз</strong> по сравнению с
                  несжатыми дампами
                </li>
                <li>
                  <strong>~20% накладных расходов по времени</strong> — намного
                  быстрее gzip
                </li>
                <li>
                  <strong>Полная автоматика</strong> — не нужно пропускать вывод
                  через утилиты сжатия
                </li>
              </ul>

              <h2 id="beyond-pgdump">
                За пределами pg_dump: физические бекапы и PITR
              </h2>

              <p>
                Логические бекапы Databasus строит на <code>pg_dump</code>, но
                выходит и за пределы того, что <code>pg_dump</code> вообще
                умеет:
              </p>

              <ul>
                <li>
                  <strong>Физические бекапы</strong>: копия всего кластера базы
                  данных на уровне файлов через <code>pg_basebackup</code>.
                  Быстрее и бекап, и восстановление для больших баз.
                </li>
                <li>
                  <strong>Инкрементальные и WAL-бекапы</strong>: инкрементальные
                  бекапы на уровне блоков через{" "}
                  <code>pg_basebackup --incremental</code> (на основе серверных
                  WAL-сводок) плюс непрерывный стриминг WAL через{" "}
                  <code>pg_receivewal</code>. Это дает Point-in-Time Recovery —
                  восстановление на любую секунду между бекапами.
                </li>
                <li>
                  <strong>Аварийное восстановление</strong>: физические базовые
                  бекапы и непрерывный стриминг WAL позволяют восстановиться
                  почти без потери данных.
                </li>
              </ul>

              <p>
                Эти бекапы построены на родном механизме резервного копирования
                PostgreSQL 17, так что Databasus переиспользует проверенный
                инструментарий самого PostgreSQL, а не изобретает свой. Нужен
                PostgreSQL 17 или новее; на более старых версиях доступны только
                логические бекапы через <code>pg_dump</code>. Все выполняется
                удаленно с хоста Databasus по протоколу репликации, поэтому на
                сервер базы данных ничего не устанавливается. До закрытых сетей
                Databasus добирается через SSH-туннель к внутреннему хосту или
                бастиону, так что базу не нужно открывать наружу.{" "}
                <a
                  href="/ru/faq#pitr"
                  className="text-blue-400 hover:text-blue-600"
                >
                  Как работают физические и PITR-бекапы
                </a>
                .
              </p>

              <h2 id="backup-automation">Автоматизация бекапов</h2>

              <p>
                Одна из самых частых задач при работе с <code>pg_dump</code> —
                настроить надежные автоматические бекапы.
              </p>

              <h3 id="automation-pgdump">Традиционная автоматизация pg_dump</h3>

              <p>
                Типичный скрипт автоматизации <code>pg_dump</code> выглядит
                примерно так:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{`#!/bin/bash
# Backup script for pg_dump
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
DB_NAME="mydb"

# Create backup
pg_dump -Fc -h localhost -U postgres $DB_NAME > $BACKUP_DIR/$DB_NAME_$DATE.dump

# Compress (if not using custom format)
# gzip $BACKUP_DIR/$DB_NAME_$DATE.sql

# Encrypt
gpg --encrypt --recipient backup@company.com $BACKUP_DIR/$DB_NAME_$DATE.dump

# Upload to S3
aws s3 cp $BACKUP_DIR/$DB_NAME_$DATE.dump.gpg s3://my-bucket/backups/

# Cleanup old backups (keep last 7 days)
find $BACKUP_DIR -name "*.dump*" -mtime +7 -delete

# Send notification on failure
if [ $? -ne 0 ]; then
  curl -X POST https://hooks.slack.com/... -d '{"text":"Backup failed!"}'
fi`}</code>
                </pre>
              </div>

              <p>
                Этот скрипт нужно поддерживать, тестировать и мониторить. Для
                каждой базы — своя запись в cron.
              </p>

              <h3 id="automation-databasus">Автоматизация в Databasus</h3>

              <p>В Databasus то же самое уже встроено:</p>

              <ul>
                <li>
                  <strong>Визуальный планировщик</strong>: бекапы каждый час,
                  день, неделю, месяц или по cron с точным временем.
                </li>
                <li>
                  <strong>Автоматическое сжатие</strong>: zstd применяется сам,
                  без настройки.
                </li>
                <li>
                  <strong>Встроенное шифрование</strong>: AES-256-GCM с
                  уникальным ключом для каждого бекапа.
                </li>
                <li>
                  <strong>Загрузка в облако</strong>: напрямую в S3, Cloudflare R2, Azure и другие хранилища.
                </li>
                <li>
                  <strong>Политики хранения</strong>: автоматическое удаление
                  старых бекапов по вашим настройкам ротации.
                </li>
                <li>
                  <strong>Уведомления</strong>: алерты в Slack, Teams, Telegram,
                  Email об успехе или ошибке.
                </li>
              </ul>

              <h2 id="storage-options">Варианты хранилищ</h2>

              <p>
                <code>pg_dump</code> пишет в локальную файловую систему. Чтобы
                доставить бекапы в облачное хранилище, нужны дополнительные
                инструменты и скрипты.
              </p>

              <h3 id="storage-databasus">Хранилища в Databasus</h3>

              <p>Databasus поддерживает несколько типов хранилищ из коробки:</p>

              <ul>
                <li>Локальное хранилище</li>
                <li>Amazon S3 и S3-совместимые сервисы</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (сетевые хранилища)</li>
                <li>FTP</li>
              </ul>

              <p>
                У каждой базы может быть свое хранилище, а для избыточности
                можно настроить несколько хранилищ сразу.
              </p>

              <p>
                <a
                  href="/ru/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Все варианты хранилищ →
                </a>
              </p>

              <h2 id="notifications">Уведомления</h2>

              <p>
                Знать, что бекап прошел или упал, критически важно для защиты
                данных.
              </p>

              <h3 id="notifications-pgdump">Уведомления в pg_dump</h3>

              <p>
                У <code>pg_dump</code> нет системы уведомлений. Придется:
              </p>

              <ul>
                <li>Писать скрипты-обертки, проверяющие коды выхода</li>
                <li>Интегрироваться с внешними системами мониторинга</li>
                <li>Настраивать собственные цепочки алертов</li>
              </ul>

              <h3 id="notifications-databasus">Уведомления в Databasus</h3>

              <p>В Databasus уведомления встроены и отправляются в:</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>Email</li>
                <li>Webhooks (для собственных интеграций)</li>
              </ul>

              <p>
                Вы сами выбираете, какие события присылать: успешный бекап,
                ошибка или и то и другое.
              </p>

              <p>
                <a
                  href="/ru/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Все каналы уведомлений →
                </a>
              </p>

              <h2 id="team-features">Командные функции</h2>

              <p>
                <code>pg_dump</code> — однопользовательский инструмент командной
                строки. Databasus добавляет функции для совместной работы
                команд:
              </p>

              <h3 id="team-databasus">Командные возможности Databasus</h3>

              <ul>
                <li>
                  <strong>Рабочие пространства</strong>: базы, уведомления и
                  хранилища группируются по проектам или командам. Пользователи
                  видят только те рабочие пространства, куда их пригласили.
                </li>
                <li>
                  <strong>Ролевая модель доступа</strong>: раздавайте права
                  наблюдателя, редактора или администратора, чтобы управлять
                  тем, что может делать каждый участник.
                </li>
                <li>
                  <strong>Журналы аудита</strong>: фиксируют все действия и
                  изменения в системе. Необходимы для комплаенса и
                  прослеживаемости.
                </li>
                <li>
                  <strong>Общие уведомления</strong>: командные каналы
                  автоматически получают статусы бекапов.
                </li>
              </ul>

              <p>
                <a
                  href="/ru/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Подробнее об управлении доступом →
                </a>
              </p>

              <h2 id="security">Безопасность</h2>

              <p>
                Безопасность — то место, где Databasus добавляет больше всего по
                сравнению с «голым» <code>pg_dump</code>.
              </p>

              <h3 id="security-pgdump">Безопасность pg_dump</h3>

              <p>
                <code>pg_dump</code> создает незашифрованные файлы бекапов.
                Чтобы их защитить, нужно:
              </p>

              <ul>
                <li>
                  Пропускать вывод через инструменты шифрования (gpg, openssl)
                </li>
                <li>Отдельно управлять ключами шифрования</li>
                <li>Обеспечивать безопасное хранение и ротацию ключей</li>
                <li>Настраивать корректные права на файлы</li>
              </ul>

              <h3 id="security-databasus">Безопасность Databasus</h3>

              <p>Databasus обеспечивает безопасность на нескольких уровнях:</p>

              <ul>
                <li>
                  <strong>Шифрование AES-256-GCM</strong>: все пароли, токены и
                  учетные данные зашифрованы. Ключ шифрования хранится отдельно
                  от базы.
                </li>
                <li>
                  <strong>Уникальное шифрование бекапов</strong>: каждый файл
                  бекапа шифруется своим ключом, выведенным из мастер-ключа, ID
                  бекапа и случайной соли.
                </li>
                <li>
                  <strong>Доступ к базе только на чтение</strong>: требуются
                  только права SELECT, поэтому данные не пострадают даже при
                  компрометации.
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

              <h2 id="restore-process">Восстановление</h2>

              <p>
                Оба инструмента умеют восстанавливать бекапы, но процесс
                различается.
              </p>

              <h3 id="restore-pgdump">Восстановление бекапов pg_dump</h3>

              <p>
                Чтобы восстановить бекап <code>pg_dump</code>, нужно:
              </p>

              <ol>
                <li>Найти файл бекапа</li>
                <li>Расшифровать, если он зашифрован</li>
                <li>Распаковать, если он сжат</li>
                <li>
                  Запустить <code>pg_restore</code> или <code>psql</code> с
                  правильными параметрами
                </li>
              </ol>

              <h3 id="restore-databasus">Восстановление бекапов Databasus</h3>

              <p>Databasus упрощает восстановление:</p>

              <ul>
                <li>
                  <strong>Скачивание в один клик</strong>: любой бекап можно
                  скачать прямо из веб-интерфейса.
                </li>
                <li>
                  <strong>Автоматическая расшифровка</strong>: бекапы
                  расшифровываются сами при скачивании.
                </li>
                <li>
                  <strong>Готовые команды восстановления</strong>: Databasus
                  показывает точную команду <code>pg_restore</code> для каждого
                  бекапа.
                </li>
                <li>
                  <strong>Параллельное восстановление</strong>: задействуйте
                  несколько ядер CPU, чтобы быстрее восстанавливать большие
                  базы.
                </li>
              </ul>

              <h2 id="installation">Установка</h2>

              <h3 id="install-pgdump">Установка pg_dump</h3>

              <p>
                <code>pg_dump</code> поставляется вместе с PostgreSQL. Если у
                вас установлен PostgreSQL, у вас есть и <code>pg_dump</code>.
              </p>

              <h3 id="install-databasus">Установка Databasus</h3>

              <p>Databasus можно установить несколькими способами:</p>

              <ul>
                <li>
                  <strong>Скрипт в одну строку</strong>: поставит Docker (если
                  нужно), настроит Databasus и включит автозапуск.
                </li>
                <li>
                  <strong>Docker run</strong>: одна команда для запуска со
                  встроенным PostgreSQL.
                </li>
                <li>
                  <strong>Docker Compose</strong>: больше контроля над
                  развертыванием.
                </li>
              </ul>

              <p>
                <a
                  href="/ru/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Руководство по установке →
                </a>
              </p>

              <h2 id="conclusion">Вывод</h2>

              <p>
                <code>pg_dump</code> — проверенная утилита резервного
                копирования PostgreSQL, и Databasus построен прямо на ней. Что
                выбрать — <code>pg_dump</code> напрямую или через Databasus —
                зависит от ваших задач.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>Используйте pg_dump напрямую, если:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>Вам нужны разовые выгрузки базы</li>
                  <li>Вам комфортно писать и поддерживать shell-скрипты</li>
                  <li>
                    У вас уже есть инфраструктура автоматизации (Ansible,
                    Terraform и т.д.)
                  </li>
                  <li>
                    Вам нужны только локальные бекапы без облачных хранилищ
                  </li>
                  <li>Вы соло-разработчик с простыми задачами</li>
                </ul>
              </div>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Используйте Databasus, если:
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Вы хотите автоматические бекапы по расписанию без написания
                    скриптов
                  </li>
                  <li>
                    Вам нужно складывать бекапы в облако (S3, Azure Blob Storage и
                    т.д.)
                  </li>
                  <li>
                    Вы хотите встроенное шифрование без ручного управления
                    ключами
                  </li>
                  <li>Вам нужны уведомления об успехе или ошибке бекапа</li>
                  <li>
                    Вы работаете в команде и вам нужны функции совместной работы
                  </li>
                  <li>
                    Вам удобнее визуальный интерфейс, чем командная строка
                  </li>
                  <li>Вы хотите автоматические политики хранения и очистку</li>
                  <li>
                    Вам нужны физические бекапы, инкрементальные бекапы или
                    Point-in-Time Recovery для аварийного восстановления
                  </li>
                </ul>
              </div>

              <p>
                Databasus строит логические бекапы на <code>pg_dump</code> и
                расширяет его автоматизацией, безопасностью и командными
                функциями. Кроме того, Databasus поддерживает физические бекапы,
                инкрементальные бекапы с архивированием WAL и Point-in-Time
                Recovery — то, чего <code>pg_dump</code> не умеет в принципе.
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
