import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Проверка восстановления бекапа - документация Databasus",
  description:
    "Докажите, что ваши бекапы действительно восстанавливаются. Databasus берет свежий бекап, восстанавливает его во временный контейнер с базой, сверяет восстановленную базу с исходной и при каждом запуске отчитывается о числе строк по таблицам.",
  keywords: [
    "проверка восстановления бекапа",
    "восстановление базы данных",
    "проверка бекапов",
    "аварийное восстановление",
    "тестирование бекапов баз данных",
    "агент проверки Databasus",
    "целостность бекапа",
    "автоматический тест восстановления",
  ],
  openGraph: {
    title: "Проверка восстановления бекапа - документация Databasus",
    description:
      "Докажите, что ваши бекапы действительно восстанавливаются. Databasus берет свежий бекап, восстанавливает его во временный контейнер с базой, сверяет восстановленную базу с исходной и при каждом запуске отчитывается о числе строк по таблицам.",
    type: "article",
    url: getLocalizedUrl("ru", "restore-verification"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Проверка восстановления бекапа - документация Databasus",
    description:
      "Докажите, что ваши бекапы действительно восстанавливаются. Databasus берет свежий бекап, восстанавливает его во временный контейнер с базой, сверяет восстановленную базу с исходной и при каждом запуске отчитывается о числе строк по таблицам.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "restore-verification"),
    languages: getLanguageAlternates("restore-verification"),
  },
  robots: "index, follow",
};

export default function RestoreVerificationPage() {
  const downloadAgent = `curl -L -o verification-agent "https://your-databasus-host/api/v1/system/verification-agent?arch=amd64" \\
  && chmod +x verification-agent`;

  const startAgent = `./verification-agent start \\
  --databasus-host=https://your-databasus-host \\
  --agent-id=<AGENT_ID> \\
  --token=<TOKEN> \\
  --max-cpu=2 \\
  --max-ram-mb=2048 \\
  --max-disk-gb=20 \\
  --max-concurrent-jobs=1`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Проверка восстановления бекапа - документация Databasus",
            description:
              "Докажите, что ваши бекапы действительно восстанавливаются. Databasus берет свежий бекап, восстанавливает его во временный контейнер с базой, сверяет восстановленную базу с исходной и при каждом запуске отчитывается о числе строк по таблицам.",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Как настроить проверку восстановления в Databasus",
            description:
              "Пошаговое руководство: зарегистрировать агента проверки, запустить его на своем сервере и настроить проверку восстановления по расписанию.",
            step: [
              {
                "@type": "HowToStep",
                name: "Создайте агента проверки в интерфейсе",
                text: "Откройте Settings → Verification agents, нажмите Create verification agent, задайте имя и скопируйте токен и ID агента из диалога.",
              },
              {
                "@type": "HowToStep",
                name: "Скачайте бинарник агента",
                text: "Выполните команду curl на хосте, где будут идти проверки, выбрав amd64 или arm64 под вашу архитектуру.",
              },
              {
                "@type": "HowToStep",
                name: "Запустите агента",
                text: "Стартуйте агента с --agent-id, --token и лимитами ресурсов (--max-cpu, --max-ram-mb, --max-disk-gb, --max-concurrent-jobs).",
              },
              {
                "@type": "HowToStep",
                name: "Настройте расписание проверок",
                text: "Откройте настройки проверки у базы данных, включите Scheduled verification и выберите интервал (After backup, Hourly, Daily, Weekly, Monthly или Cron).",
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
              <h1 id="restore-verification">Проверка восстановления бекапа</h1>

              <p className="text-lg text-gray-400">
                Бекап, который завершился без ошибок, &mdash; еще не бекап,
                который получится восстановить. Единственное настоящее
                доказательство &mdash; восстановить его. Databasus делает это за
                вас по расписанию:
              </p>

              <ul>
                <li>берет последний бекап</li>
                <li>восстанавливает его во временный контейнер с базой</li>
                <li>сверяет восстановленную базу с исходной</li>
                <li>удаляет контейнер</li>
                <li>сообщает результат</li>
              </ul>

              <img
                src="/images/restore-verification/verified-backups.png"
                alt="Вкладка проверенных бекапов"
                loading="lazy"
              />

              <img
                className="mt-5"
                src="/images/restore-verification/verifications.png"
                alt="Вкладка проверок"
                loading="lazy"
              />

              <h2 id="what-is-verification-agent">Что такое агент проверки?</h2>

              <p>
                Агент проверки &mdash; это небольшой бинарник на Go, который вы
                запускаете на своей машине: подойдет любая со свободными CPU,
                RAM и диском. Агент регистрируется в Databasus, забирает задачи
                проверки из очереди, выполняет их локально и отправляет
                результаты обратно.
              </p>

              <h3 id="what-you-need">Что понадобится</h3>

              <ul>
                <li>
                  Хост с исходящим HTTPS-доступом к вашему адресу Databasus.
                </li>
                <li>
                  Docker на этом хосте &mdash; для каждой задачи агент поднимает
                  временный контейнер базы данных подходящей мажорной версии.
                </li>
                <li>
                  Место на диске под каждую задачу проверки:{" "}
                  <strong>размер файла бекапа</strong> плюс{" "}
                  <strong>размер базы в несжатом виде</strong> плюс{" "}
                  запас и <strong>самые большие индексы</strong>, которые строятся одновременно.
                </li>
                <li>
                  Не меньше 1 ядра CPU и 512 МБ RAM на каждую параллельную
                  задачу.
                </li>
              </ul>

              <h3 id="why-not-just-checksums">
                Почему не хватает контрольных сумм?
              </h3>

              <p>
                Контрольные суммы и коды возврата ловят часть проблем, но другие
                пропускают полностью:
              </p>

              <ul>
                <li>
                  <strong>Контрольные суммы</strong> ловят повреждение битов в
                  файле архива, но ничего не говорят о том, полон ли сам дамп и
                  корректен ли он по смыслу.
                </li>
                <li>
                  <strong>Код возврата дампа</strong> говорит лишь о том, что
                  команда дампа отработала. Он не поймает роль без права чтения
                  на отдельных объектах, отсутствующее расширение на источнике
                  или несовпадение tablespace &mdash; а из-за них объекты молча
                  пропускаются или урезаются.
                </li>
                <li>
                  <strong>Проверка восстановления</strong> реально прогоняет
                  архив через штатный инструмент восстановления СУБД и считает
                  строки в каждой таблице. Это единственная проверка, которая
                  ловит все перечисленное: если бекап не восстановится, вы
                  узнаете об этом заранее, а не во время аварии.
                </li>
              </ul>

              <h2 id="configuration">Настройка</h2>

              <h3 id="create-on-ui">Создайте агента в интерфейсе</h3>

              <p>
                Откройте <strong>Settings → Verification agents</strong> и
                нажмите <strong>Create verification agent</strong>. Выберите
                говорящее имя вроде <code>staging-verifier</code> или{" "}
                <code>eu-west-host-1</code>. В следующем диалоге будут показаны{" "}
                <strong>токен</strong> и <strong>ID</strong> агента.
              </p>

              <p>
                Токен показывается <strong>ровно один раз</strong> &mdash;
                скопируйте его до закрытия диалога. Если позже вы его потеряете,
                воспользуйтесь действием <strong>Rotate token</strong> в строке
                агента, чтобы выпустить новый; старый токен перестанет работать
                на следующем heartbeat агента. Следующий диалог показывает
                команды установки под архитектуру вашего сервера &mdash; те же
                команды описаны ниже.
              </p>

              <h3 id="launch">Запустите агента на своем сервере</h3>

              <p>
                Зайдите по SSH на машину, которая будет выполнять проверки.
                Сначала скачайте бинарник агента. Замените{" "}
                <code>https://your-databasus-host</code> на адрес вашего
                Databasus и поменяйте <code>amd64</code> на <code>arm64</code>,
                если у вас ARM-сервер:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={downloadAgent} lang="ru" />
                </div>
              </div>

              <p>
                Затем запустите агента. ID агента и токен возьмите из диалога с
                предыдущего шага:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={startAgent} lang="ru" />
                </div>
              </div>

              <p>
                Команда <code>start</code> демонизирует агента и записывает его
                флаги в файл <code>databasus-verification.json</code> в рабочем
                каталоге, так что при последующих перезапусках можно выполнять{" "}
                <code>./verification-agent start</code> вообще без флагов. Логи
                пишутся в <code>databasus-verification.log</code> рядом с
                бинарником.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] mb-3 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    Адрес Databasus должен начинаться с <code>https://</code>.
                    Обычный HTTP разрешен только с флагом{" "}
                    <code>--allow-insecure-http</code> и предназначен для
                    локальных тестов &mdash; никогда не выставляйте боевого
                    агента наружу по незашифрованному HTTP.
                  </p>
                </div>
              </div>

              <p>
                Четыре флага <code>--max-*</code> &mdash; это{" "}
                <strong>общие лимиты</strong>, а не выделение ресурсов на одну
                задачу. Агент сообщает их Databasus в каждом heartbeat, и
                Databasus делит их между разрешенными параллельными задачами.
                При{" "}
                <code>
                  --max-cpu=2 --max-ram-mb=2048 --max-concurrent-jobs=1
                </code>{" "}
                единственная задача получает все 2 CPU и 2 ГБ RAM. При{" "}
                <code>--max-concurrent-jobs=2</code> каждая задача получает 1
                CPU и 1 ГБ. Минимум &mdash; 1 CPU и 512 МБ на задачу: если
                лимитов на этот минимум не хватает, агент сам снизит
                параллельность. С лимитом диска ошибиться проще всего: каждой
                задаче нужно место под <strong>размер файла бекапа</strong>,{" "}
                <strong>размер базы в несжатом виде</strong>,{" "}
                <strong>запас около 1,5 ГБ</strong> и{" "}
                <strong>самые большие индексы</strong>, по одному на каждый
                параллельный поток восстановления. Пока индекс строится, его
                сортировка занимает на диске примерно столько же, сколько сам
                индекс. Databasus уменьшает число параллельных потоков, пока все
                это не поместится на свободный диск агента. Поэтому с большим{" "}
                <code>--max-cpu</code> нужно больше диска, а больший{" "}
                <code>--max-disk-gb</code> позволяет восстанавливать большие базы
                быстрее.
              </p>

              <h3 id="manage">Управление агентом</h3>

              <p>Тот же бинарник дает четыре подкоманды:</p>

              <ul>
                <li>
                  <code>./verification-agent status</code> &mdash; показать,
                  запущен ли демон и какие задачи он сейчас выполняет.
                </li>
                <li>
                  <code>./verification-agent stop</code> &mdash; остановить
                  демона. Незавершенные проверки отправляются в Databasus как
                  неуспешные и ставятся в очередь заново.
                </li>
                <li>
                  <code>./verification-agent start</code> &mdash; перезапустить
                  демона. Флаги запоминаются с первого запуска; после ротации
                  токена передайте <code>--token=&lt;NEW&gt;</code>, чтобы
                  обновить сохраненный токен.
                </li>
                <li>
                  <code>./verification-agent run</code> &mdash; работать на
                  переднем плане, а не как демон. Используйте этот режим, когда
                  оборачиваете агента в systemd-юнит или Docker-контейнер: такие
                  супервизоры ожидают, что процесс не уйдет в фон.
                </li>
              </ul>

              <p>
                На странице Settings у каждой строки агента есть три
                действия-иконки: снова посмотреть команды установки (без
                раскрытия токена), ротировать токен и удалить агента. Удаление
                безопасно: проверки, назначенные этому агенту, возвращаются в
                очередь, и их подхватит другой агент, если он доступен.
              </p>

              <h2 id="schedules-and-notifications">Расписания и уведомления</h2>

              <p>
                Проверка восстановления настраивается для каждой базы отдельно.
                Откройте настройки проверки у базы данных, включите{" "}
                <strong>Scheduled verification</strong> и выберите интервал.
              </p>

              <h3 id="interval-options">Варианты интервалов</h3>

              <ul>
                <li>
                  <strong>After backup</strong> &mdash; самая сильная гарантия:
                  каждый успешный бекап проверяется сразу после завершения.
                </li>
                <li>
                  <strong>Hourly, daily, weekly, monthly</strong> &mdash;
                  выберите периодичность и время суток.
                </li>
                <li>
                  <strong>Cron</strong> &mdash; cron-выражение в UTC для всего,
                  что не покрывают пресеты. Примеры: <code>0 4 * * 0</code>{" "}
                  (каждое воскресенье в 4:00 UTC) и <code>0 */6 * * *</code>{" "}
                  (каждые шесть часов).
                </li>
              </ul>

              <h3 id="how-the-queue-works">
                Как очередь обрабатывает &quot;After backup&quot;
              </h3>

              <p>
                Проверка обычно идет дольше, чем породивший ее бекап, и если
                бекапы приходят чаще, чем успевают проверяться, очередь росла бы
                бесконечно. Databasus избегает этого,{" "}
                <strong>
                  отменяя все ожидающие проверки той же базы, как только
                  приходит свежий бекап
                </strong>
                : в очереди ждет только самый последний бекап. Этот компромисс
                сознательный: лучше пропустить проверку устаревшего бекапа, чем
                часами проверять то, из чего вы все равно никогда не станете
                восстанавливаться.
              </p>

              <h3 id="manual-runs">Ручные запуски</h3>

              <p>
                Разовую проверку можно запустить со вкладки{" "}
                <strong>Restore verifications</strong> у базы данных, не меняя
                расписание. Это удобно, чтобы выборочно проверить конкретный
                бекап или прогнать нового агента от начала до конца, прежде чем
                доверить ему нагрузку по расписанию.
              </p>

              <h3 id="notifications">Уведомления</h3>

              <p>
                Об успехе и провале можно сообщать через любой канал
                уведомлений, уже подключенный к базе. Два чекбокса &mdash;{" "}
                <strong>Verification success</strong> и{" "}
                <strong>Verification failed</strong> &mdash; независимы.
                Большинство команд включает только уведомления о провале, чтобы
                не утонуть в оповещениях. Как подключить Slack, Microsoft Teams,
                Discord, почту и другие каналы, описано в{" "}
                <a
                  href="/ru/notifiers"
                  className="text-blue-400 hover:text-blue-300"
                >
                  документации по уведомлениям
                </a>
                .
              </p>

              <h3 id="results">Чтение результатов</h3>

              <p>
                Каждая попытка проверки отображается отдельной строкой на
                вкладке <strong>Restore verifications</strong> у базы данных.
                Статус &mdash; один из <strong>Pending</strong>,{" "}
                <strong>Running</strong>, <strong>Successful</strong>,{" "}
                <strong>Failed</strong> или <strong>Canceled</strong>. Клик по
                строке открывает панель с полным таймлайном, кодом возврата
                восстановления, размером восстановленной базы, числом схем и
                таблиц и разбивкой числа строк по таблицам. У неуспешных
                запусков сообщение об ошибке показано в верхней части панели.
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
