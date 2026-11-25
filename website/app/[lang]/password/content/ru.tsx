import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Сброс пароля - документация Databasus",
  description:
    "Как сбросить пароль пользователя, вывести список администраторов и создать учетную запись в Databasus встроенной консольной утилитой. Быстрый и безопасный сброс пароля в вашей системе резервного копирования PostgreSQL.",
  keywords: [
    "сброс пароля Databasus",
    "сбросить пароль пользователя",
    "пароль системы бекапов PostgreSQL",
    "восстановление пароля в Docker",
    "восстановление пароля",
    "аутентификация Databasus",
  ],
  openGraph: {
    title: "Сброс пароля - документация Databasus",
    description:
      "Как сбросить пароль пользователя, вывести список администраторов и создать учетную запись в Databasus встроенной консольной утилитой. Быстрый и безопасный сброс пароля в вашей системе резервного копирования PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("ru", "password"),
    locale: OG_LOCALES.ru,
  },
  twitter: {
    card: "summary",
    title: "Сброс пароля - документация Databasus",
    description:
      "Как сбросить пароль пользователя, вывести список администраторов и создать учетную запись в Databasus встроенной консольной утилитой. Быстрый и безопасный сброс пароля в вашей системе резервного копирования PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("ru", "password"),
    languages: getLanguageAlternates("password"),
  },
  robots: "index, follow",
};

export default function PasswordResetPage() {
  const resetPasswordCommand = `docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com"`;
  const listAdminsCommand = `docker exec -it databasus ./main --list-admins`;
  const createUserCommand = `docker exec -it databasus ./main --create-user --email="ops@example.com" --password="YourSecurePassword123" --role=admin`;
  const disableTwoFactorCommand = `docker exec -it databasus ./main --disable-2fa`;
  const resetTwoFactorCommand = `docker exec -it databasus ./main --reset-2fa --email="owner@example.com"`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Сброс пароля - документация Databasus",
            description:
              "Как сбросить пароль пользователя в Databasus встроенной консольной утилитой.",
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
            name: "Как сбросить пароль пользователя Databasus",
            description:
              "Пошаговое руководство по сбросу пароля пользователя в Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Выполните команду сброса пароля",
                text: "Выполните команду docker exec с новым паролем и email пользователя.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Запустите команду сброса пароля внутри контейнера Databasus через docker exec",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Проверьте смену пароля",
                text: "Войдите в Databasus с новым паролем и убедитесь, что смена прошла успешно.",
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
              <h1 id="reset-password">Сброс пароля пользователя</h1>

              <h2 id="reset-password-command">Команда сброса пароля</h2>

              <p>
                Чтобы сбросить пароль пользователя, выполните следующую команду
                на сервере, где запущен Databasus:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetPasswordCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={resetPasswordCommand} lang="ru" />
                </div>
              </div>

              <h2 id="parameters">Параметры</h2>

              <p>Команда принимает следующие параметры:</p>

              <ul>
                <li>
                  <strong>--new-password</strong>: новый пароль, не короче 8
                  символов. Убедитесь, что он надежный и содержит буквы, цифры и
                  специальные символы.
                </li>
                <li>
                  <strong>--email</strong>: email учетной записи, чей пароль вы
                  хотите сбросить (например, <code>owner@example.com</code>).
                  Инстанс, созданный до того, как первая учетная запись стала им
                  управлять, до сих пор носит адрес-заглушку <code>admin</code>,
                  пока владелец его не заменит — там передавайте{" "}
                  <code>admin</code>.
                </li>
              </ul>

              <h2 id="list-admins">Список администраторов</h2>

              <p>
                Если вы не помните, какой адрес управляет инстансом, выведите
                список администраторов на сервере, где запущен Databasus:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{listAdminsCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={listAdminsCommand} lang="ru" />
                </div>
              </div>

              <p>
                В выводе перечислены все учетные записи администраторов с их
                email, именем, датой создания и состоянием активности, а та,
                которую инстанс считает своим администратором, отмечена
                отдельно. Пароли, их хеши и токены не выводятся. На инстансе,
                где еще никто не завел учетную запись, команда сообщит, что
                администратора нет.
              </p>

              <h2 id="create-user">Создание учетной записи из консоли</h2>

              <p>
                Учетную запись можно завести без регистрации в браузере,
                например из скрипта деплоя. Выполните команду на сервере, где
                запущен Databasus:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createUserCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={createUserCommand} lang="ru" />
                </div>
              </div>

              <p>Команда принимает следующие параметры:</p>

              <ul>
                <li>
                  <strong>--email</strong>: email новой учетной записи. Адрес
                  должен быть корректным.
                </li>
                <li>
                  <strong>--password</strong>: пароль новой учетной записи, не
                  короче 8 символов.
                </li>
                <li>
                  <strong>--role</strong>: <code>admin</code> или{" "}
                  <code>member</code>.
                </li>
                <li>
                  <strong>--name</strong>: отображаемое имя, необязательно. Без
                  него учетная запись получит имя по части адреса до{" "}
                  <code>@</code>.
                </li>
              </ul>

              <p>
                Учетная запись сразу активна, и в нее можно войти с этим адресом
                и паролем, даже если открытая регистрация выключена. На
                инстансе, где еще нет ни одной учетной записи, созданная
                учетная запись станет администратором инстанса, поэтому первой
                нужно передать <code>--role=admin</code>. Администратор,
                созданный так позже, будет обычным администратором.
              </p>

              <p>
                Повторный запуск для существующей учетной записи с той же ролью
                завершится успешно и ничего не изменит, так что команду можно
                выполнять при каждом деплое. Пароль при этом тоже не меняется:
                для смены пароля используйте команду сброса с{" "}
                <code>--new-password</code> выше. Если адрес принадлежит
                учетной записи с другой ролью, деактивированной или
                приглашенной учетной записи либо той, что входит через Google
                или GitHub, команда завершится ошибкой и ничего не изменит.
              </p>

              <h2 id="disable-two-factor">Отключить требование 2FA</h2>

              <p>
                Инстанс может требовать при входе шестизначный код, присланный
                по почте, в дополнение к паролю. Если почтовый сервер перестал
                доставлять такие письма, по паролю не войдет никто, и сброс
                пароля сам по себе не поможет. Выключите второй фактор на
                сервере, где запущен Databasus:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{disableTwoFactorCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={disableTwoFactorCommand} />
                </div>
              </div>

              <p>
                Команда сообщает, изменила ли она что-нибудь, спокойно
                отрабатывает, если второй фактор уже выключен, и записывает
                изменение в журнал аудита. Следующий вход по паролю пройдет без
                кода, а когда почта заработает, администратор может снова
                включить настройку.
              </p>

              <h2 id="reset-authenticator-app">Сбросить приложение-аутентификатор аккаунта</h2>

              <p>
                Если человек потерял телефон с приложением-аутентификатором, удалите приложение из его аккаунта на сервере, где запущен Databasus:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetTwoFactorCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={resetTwoFactorCommand} />
                </div>
              </div>

              <p>
                Следующий вход в этот аккаунт пойдет по настройке 2FA инстанса. Администратор может сбросить приложение пользователя и из списка пользователей.
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
