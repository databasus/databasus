import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "FAQ - частые вопросы | Databasus",
  description:
    "Частые вопросы о Databasus — инструменте резервного копирования PostgreSQL с поддержкой MySQL, MariaDB и MongoDB. Как бекапить базы на localhost, форматы бекапов, методы сжатия и другое.",
  keywords: [
    "Databasus FAQ",
    "вопросы о бекапах PostgreSQL",
    "бекап базы на localhost",
    "форматы бекапов",
    "сжатие pg_dump",
    "сжатие zstd",
    "помощь с бекапом PostgreSQL",
    "руководство по резервному копированию баз данных",
  ],
  openGraph: {
    title: "FAQ - частые вопросы | Databasus",
    description:
      "Частые вопросы о Databasus — инструменте резервного копирования PostgreSQL с поддержкой MySQL, MariaDB и MongoDB. Как бекапить базы на localhost, форматы бекапов, методы сжатия и другое.",
    type: "article",
    url: getLocalizedUrl("ru", "faq"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "FAQ - частые вопросы | Databasus",
    description:
      "Частые вопросы о Databasus — инструменте резервного копирования PostgreSQL с поддержкой MySQL, MariaDB и MongoDB. Как бекапить базы на localhost, форматы бекапов, методы сжатия и другое.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "faq"),
    languages: getLanguageAlternates("faq"),
  },
  robots: "index, follow",
};

export default function FAQPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "Почему Databasus не использует формат сырого SQL-дампа для логических бекапов PostgreSQL?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Для логических бекапов Databasus использует формат custom утилиты pg_dump со сжатием zstd, потому что после обширного тестирования он показал лучшую скорость бекапа и восстановления. Формат custom со сжатием zstd уровня 5 дает оптимальный баланс между скоростью создания бекапа, скоростью восстановления и размером файла.",
                },
              },
              {
                "@type": "Question",
                name: "Куда устанавливается Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus устанавливается в /opt/databasus/",
                },
              },
              {
                "@type": "Question",
                name: "Как работают физические и PITR-бекапы (Point-in-Time Recovery)?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus выполняет физические бекапы удаленно со своего хоста, подключаясь к вашему PostgreSQL по стандартному протоколу репликации, поэтому на сервер базы данных ничего устанавливать не нужно. До баз в закрытых сетях можно добраться через SSH-туннель. Физические бекапы используют родной стек PostgreSQL 17: полные бекапы через pg_basebackup, блочные инкременты через pg_basebackup --incremental на основе серверных WAL-сводок (summarize_wal = on) и непрерывную передачу WAL через pg_receivewal. Физические бекапы требуют PostgreSQL 17 или новее; на более старых версиях используются логические бекапы pg_dump. Для восстановления на момент времени pg_combinebackup собирает работоспособный каталог данных из полного бекапа и цепочки инкрементов, после чего PostgreSQL проигрывает WAL до выбранного вами целевого времени, восстанавливаясь на любую секунду между бекапами. Интерфейс Databasus дает пошаговые инструкции для восстановления на хост или в Docker-базу: либо готовым скриптом, который сводит восстановление к одной команде, либо скачиванием бекапов и сборкой цепочки полных, инкрементальных и WAL-частей вручную. Инкременты и WAL опциональны: можно снимать только полный бекап, и WAL не обязателен. Мы используем родные бекапы PostgreSQL 17, потому что они переиспользуют проверенный механизм резервного копирования самого PostgreSQL вместо его переизобретения, работают с удаленными базами, включая управляемые сервисы вроде RDS и Cloud SQL, и дают почти нулевую потерю данных.",
                },
              },
              {
                "@type": "Question",
                name: "Почему Databasus отказался от бекапов через агент?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Ранняя версия Databasus поставлялась с агентом бекапов: бинарником, который работал на хосте базы, передавал WAL и создавал физические бекапы локально. Та первая реализация оказалась ошибкой и была удалена. Это была наивная реализация, которая лишь копировала WAL поверх полных бекапов, что приводило к долгому RTO. Пользователям приходилось настраивать и Databasus, и отдельный агент, хотя делать все удаленно из одного места намного проще. Поскольку агент жил вне основной системы, было трудно покрыть все тестовые случаи. По сути агент решает только одну задачу — доступ к базе, недоступной снаружи, а для 99% пользователей это уже решается запуском Databasus внутри приватной сети или подключением по SSH, так что агент изобретал колесо и делал простую задачу сильно сложнее, чем нужно. Он также не мог работать на управляемых базах вроде RDS и Cloud SQL, которые запрещают установку на хост, но уже открывают протокол репликации, поэтому удаленный путь был нужен в любом случае. Вдобавок он тянул за собой много граничных случаев с разрывами соединений, обновлениями агента и сбором логов из отдельного процесса, а чем меньше движущихся частей в системе, тем надежнее она в повседневной работе. Теперь физические бекапы выполняются удаленно с хоста Databasus. Существующие бекапы в безопасности: при обновлении с версии, где еще есть агентские бекапы, Databasus не удалит их молча — он предупредит об изменении и даст выбор: остаться на поддерживаемой версии 3.42.0 или удалить старые агентские бекапы самостоятельно перед обновлением. Реализация на агенте остается доступной до версии 3.42.0 и будет работать еще долго.",
                },
              },
              {
                "@type": "Question",
                name: "Как забекапить сам Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Чтобы забекапить Databasus, зайдите в /opt/databasus (или папку, куда вы его установили), затем в каталог databasus-data. Забекапить нужно файл secret.key (ключ шифрования учетных данных) и папку /pgdata (внутренняя база с конфигурациями и метаданными бекапов). Есть два сценария восстановления: 1) восстановить бекапы баз можно с одним secret.key без интерфейса Databasus (см. руководство по ручному восстановлению), 2) чтобы восстановить интерфейс Databasus со всеми конфигурациями и историей, нужны и secret.key, и папка /pgdata. Для восстановления воссоздайте эту структуру папок на другом сервере.",
                },
              },
              {
                "@type": "Question",
                name: "Как Databasus поддерживается open-source-программами Anthropic и OpenAI?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "В марте 2026 года Databasus был принят и в Claude for Open Source от Anthropic, и в Codex for Open Source от OpenAI. Поддержка этих программ — сигнал надежности: проект прошел независимую оценку и признан лидерами индустрии критичной open-source-инфраструктурой, которую стоит поддерживать.",
                },
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
              <h1 id="faq">Частые вопросы</h1>

              <p className="text-lg text-gray-400">
                Ответы на самые частые вопросы о Databasus: установка,
                конфигурация и стратегии резервного копирования.
              </p>

              <h2 id="why-no-raw-sql-dump">
                Почему Databasus не использует формат сырого SQL-дампа для
                логических бекапов PostgreSQL?
              </h2>

              <p>
                Для логических бекапов Databasus использует{" "}
                <strong>формат custom</strong> утилиты <code>pg_dump</code> со{" "}
                <strong>сжатием zstd уровня 5</strong> вместо обычного
                SQL-формата, потому что он дает лучший баланс между:
              </p>

              <ul>
                <li>Скоростью создания бекапа</li>
                <li>Скоростью восстановления</li>
                <li>
                  Размером файла (до 20 раз меньше, чем обычный SQL-формат)
                </li>
              </ul>

              <p>
                Это решение принято после обширного тестирования и бенчмарков
                разных форматов бекапов PostgreSQL и методов сжатия. Подробнее о
                тестировании можно прочитать здесь:{" "}
                <a
                  href="https://dev.to/rostislav_dugin/postgresql-backups-comparing-pgdump-speed-in-different-formats-and-with-different-compression-4pmd"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  PostgreSQL backups: comparing pg_dump speed in different
                  formats and with different compression
                </a>
                .
              </p>

              <p>
                Databasus не будет добавлять формат сырого SQL-дампа, потому
                что:
              </p>

              <ul>
                <li>лишнее разнообразие вредит UX;</li>
                <li>дополнительный формат усложняет поддержку кода;</li>
                <li>текущий формат дампа покрывает 99% случаев</li>
              </ul>

              <h2 id="installation-directory">
                Куда ставится Databasus при установке через .sh-скрипт?
              </h2>

              <p>
                Databasus устанавливается в каталог <code>/opt/databasus/</code>
                .
              </p>

              <h2 id="pitr">
                Как работают физические и PITR-бекапы (Point-in-Time Recovery)?
              </h2>

              <p>
                Databasus выполняет физические бекапы{" "}
                <strong>удаленно со своего хоста</strong>, подключаясь к вашему
                PostgreSQL по стандартному <strong>протоколу репликации</strong>
                , поэтому на сервер базы данных ничего устанавливать не нужно.
                Если база живет в закрытой сети, Databasus может добраться до
                нее через SSH-туннель к внутреннему хосту или бастиону, так что
                базу не придется открывать наружу.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>Почему это стало возможным:</strong> годами
                    инструменты вроде pgBackRest и WAL-G строили собственные
                    движки инкрементальных блочных бекапов, потому что у
                    PostgreSQL не было родного. Все изменилось с PostgreSQL 17,
                    где эту функциональность разработал{" "}
                    <strong>Robert Haas</strong> при участии{" "}
                    <strong>David Steele</strong>, автора pgBackRest. Теперь
                    PostgreSQL сам поставляет серверные блочные инкрементальные
                    бекапы (<code>pg_basebackup --incremental</code> и{" "}
                    <code>summarize_wal</code>), и Databasus строится на них, а
                    не изобретает свои.
                  </p>
                </div>
              </div>

              <p>
                <strong>Как работают бекапы:</strong>
              </p>

              <ul>
                <li>
                  Полные бекапы создаются через <code>pg_basebackup</code> и
                  передаются напрямую в Databasus
                </li>
                <li>
                  Блочные инкременты используют{" "}
                  <code>pg_basebackup --incremental</code>: серверные WAL-сводки
                  PostgreSQL 17 (<code>summarize_wal = on</code>) отслеживают
                  изменения, поэтому передаются только измененные блоки
                </li>
                <li>
                  WAL передается непрерывно через <code>pg_receivewal</code>,
                  чтобы цепочка восстановления между бекапами оставалась полной
                </li>
                <li>
                  Физические бекапы требуют{" "}
                  <strong>PostgreSQL 17 или новее</strong>; на более старых
                  версиях используются логические бекапы через{" "}
                  <code>pg_dump</code>
                </li>
              </ul>

              <p>
                <strong>Как работает восстановление:</strong>
              </p>

              <ul>
                <li>
                  <code>pg_combinebackup</code> собирает работоспособный каталог
                  данных из полного бекапа и цепочки инкрементов
                </li>
                <li>
                  Затем PostgreSQL проигрывает WAL до выбранного вами целевого
                  времени, восстанавливаясь на любую секунду между бекапами
                </li>
                <li>
                  После запуска PostgreSQL завершает восстановление, становится
                  первичным сервером и возвращается к обычной работе
                </li>
              </ul>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-blue-500 mt-0.5 shrink-0"
                >
                  <circle cx="12" cy="12" r="10" />
                  <path d="M12 16v-4M12 8h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <strong>Делать это вручную не обязательно.</strong>{" "}
                    Интерфейс Databasus дает пошаговые инструкции для
                    восстановления на хост или в Docker-базу: либо готовым
                    скриптом, либо ручным скачиванием бекапов. Мы подготовили
                    скрипт так, что восстановление — это одна команда, но при
                    желании вы можете собрать цепочку из полных, инкрементальных
                    и WAL-частей сами. Инкременты и WAL тоже опциональны: можно
                    снимать только полный бекап, без инкрементов, и WAL не
                    обязателен.
                  </p>
                </div>
              </div>

              <p>
                <strong>Почему мы используем родные бекапы PG 17:</strong>
              </p>

              <ul>
                <li>
                  Они переиспользуют собственный механизм резервного копирования
                  PostgreSQL вместо его переизобретения — вы получаете
                  проверенные временем внутренности с тысячами тестов и учтенных
                  граничных случаев
                </li>
                <li>
                  Они работают с удаленными базами, включая управляемые сервисы
                  вроде Amazon RDS и Google Cloud SQL, которые открывают
                  протокол репликации, но запрещают установку ПО на хост
                </li>
                <li>
                  Они дают почти нулевую потерю данных: восстановиться можно на
                  любую секунду между бекапами
                </li>
              </ul>

              <h2 id="why-no-agent">
                Почему Databasus отказался от бекапов через агент?
              </h2>

              <p>
                Ранняя версия Databasus поставлялась с <strong>агентом</strong>{" "}
                бекапов: бинарником, который работал на хосте базы, передавал
                WAL и создавал физические бекапы локально. Та первая реализация
                оказалась ошибкой, и мы ее удалили. Физические бекапы теперь
                выполняются удаленно с хоста Databasus, как описано выше.
              </p>

              <p>
                <strong>Почему агент был неверным подходом:</strong>
              </p>

              <ul>
                <li>
                  Это была наивная реализация, которая лишь копировала WAL
                  поверх полных бекапов, что приводило к долгому RTO
                </li>
                <li>
                  Пользователям приходилось настраивать и Databasus, и отдельный
                  агент, хотя делать все удаленно из одного места намного проще
                </li>
                <li>
                  Поскольку агент жил вне основной системы, было трудно покрыть
                  все тестовые случаи
                </li>
                <li>
                  По сути агент решает только одну задачу: доступ к базе,
                  недоступной снаружи. Для 99% пользователей это уже решается
                  запуском Databasus внутри приватной сети или подключением по
                  SSH, так что агент изобретал колесо и делал простую задачу
                  сильно сложнее, чем нужно
                </li>
                <li>
                  Он не мог работать на управляемых базах вроде RDS и Cloud SQL,
                  которые запрещают установку на хост, но уже открывают протокол
                  репликации, поэтому удаленный путь был нужен в любом случае
                </li>
                <li>
                  Еще он тянул за собой много граничных случаев. Разрывы
                  соединений, обновления агента и сбор логов из отдельного
                  процесса были болезненными, а чем меньше движущихся частей в
                  системе, тем надежнее она в повседневной работе
                </li>
              </ul>

              <p>
                <strong>
                  Мы позаботились о сохранности существующих бекапов.
                </strong>{" "}
                Если вы обновляетесь с версии, где еще есть агентские бекапы,
                Databasus не удалит их молча: он предупредит об изменении и даст
                выбор — остаться на поддерживаемой{" "}
                <strong>версии 3.42.0</strong> или удалить старые агентские
                бекапы самостоятельно перед обновлением. Реализация на агенте
                остается доступной до версии 3.42.0 и будет работать еще долго,
                так что ничего не сломается.
              </p>

              <h2 id="backup-databasus">Как забекапить сам Databasus?</h2>

              <p>
                Если вы хотите забекапить свой инстанс Databasus (включая все
                конфигурации, базы и учетные данные), сделайте следующее:
              </p>

              <ol>
                <li>
                  Зайдите в <code>/opt/databasus</code> (или папку, куда вы
                  установили Databasus)
                </li>
                <li>
                  Перейдите в каталог <code>databasus-data</code>
                </li>
              </ol>

              <p>
                <strong>Забекапить нужно:</strong>
              </p>

              <ul>
                <li>
                  <code>secret.key</code> — ключ шифрования ваших учетных данных
                </li>
                <li>
                  <code>/pgdata</code> — внутренняя база PostgreSQL самого
                  Databasus со всеми конфигурациями и метаданными бекапов
                </li>
              </ul>

              <p>
                Если вы храните бекапы в локальном хранилище, можно забекапить и
                папку <code>backups</code>.
              </p>

              <p>
                <strong>Важно:</strong> есть два разных сценария восстановления:
              </p>

              <ul>
                <li>
                  <strong>Восстановить бекапы без интерфейса Databasus:</strong>{" "}
                  бекапы баз можно восстановить с одним лишь файлом{" "}
                  <code>secret.key</code>, без самого Databasus и его внутренних
                  данных. Подробные инструкции — в{" "}
                  <a href="/ru/how-to-recover-without-databasus">
                    руководстве по ручному восстановлению
                  </a>
                  .
                </li>
                <li>
                  <strong>
                    Восстановить интерфейс Databasus со всеми конфигурациями:
                  </strong>{" "}
                  чтобы вернуть интерфейс Databasus со всеми конфигурациями,
                  расписаниями и историей бекапов, нужно забекапить и{" "}
                  <code>secret.key</code>, и папку <code>/pgdata</code> (в ней
                  лежат метаданные шифрования и все конфигурации Databasus).
                </li>
              </ul>

              <p>
                <strong>Чтобы восстановить Databasus на другом сервере:</strong>{" "}
                просто воссоздайте структуру папки <code>databasus-data</code>{" "}
                из сохраненных файлов и запустите Databasus.
              </p>

              <h2 id="oss-programs">
                Как Databasus поддерживается open-source-программами Anthropic и
                OpenAI?
              </h2>

              <p>
                В марте 2026 года Databasus был принят и в{" "}
                <strong>
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Claude for Open Source
                  </a>
                </strong>{" "}
                от Anthropic, и в{" "}
                <strong>
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Codex for Open Source
                  </a>
                </strong>{" "}
                от OpenAI. Для нас очень ценно, что две ведущие ИИ-компании мира
                признали проект важным open-source-софтом для индустрии —
                особенно с учетом высоких требований обеих программ к
                участникам.
              </p>

              <p>
                Что это значит для пользователей? Это еще одно подтверждение
                надежности: проект прошел независимую оценку и признан лидерами
                индустрии критичной инфраструктурой, которую стоит поддерживать.
                Так что качество кода становится еще выше, ревью безопасности —
                быстрее, а разработка остается активной благодаря доступу к
                новейшим ИИ без ограничений.
              </p>

              <img
                src="/images/faq/anthropic-email.png"
                alt="Databasus принят в программу Claude for Open Source от Anthropic"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <img
                src="/images/faq/openai-email.png"
                alt="Databasus принят в программу Codex for Open Source от OpenAI"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
