import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "重置密码 - Databasus 文档",
  description:
    "了解如何使用内置命令行工具在 Databasus 中重置用户密码、列出管理员和创建账户。为你的 PostgreSQL 备份系统提供快速安全的密码找回方式。",
  keywords: [
    "Databasus 重置密码",
    "重置用户密码",
    "PostgreSQL 备份密码",
    "Docker 密码找回",
    "密码找回",
    "Databasus 认证",
  ],
  openGraph: {
    title: "重置密码 - Databasus 文档",
    description:
      "了解如何使用内置命令行工具在 Databasus 中重置用户密码、列出管理员和创建账户。为你的 PostgreSQL 备份系统提供快速安全的密码找回方式。",
    type: "article",
    url: getLocalizedUrl("zh", "password"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "重置密码 - Databasus 文档",
    description:
      "了解如何使用内置命令行工具在 Databasus 中重置用户密码、列出管理员和创建账户。为你的 PostgreSQL 备份系统提供快速安全的密码找回方式。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "password"),
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
            headline: "重置密码 - Databasus 文档",
            description:
              "了解如何使用内置命令行工具在 Databasus 中重置用户密码。",
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
            name: "如何重置 Databasus 用户密码",
            description: "在 Databasus 中重置用户密码的分步指南",
            step: [
              {
                "@type": "HowToStep",
                name: "运行重置密码命令",
                text: "执行 docker exec 命令，传入新密码和用户邮箱。",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "使用 docker exec 在 Databasus 容器内运行重置密码命令",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "验证密码修改",
                text: "用新密码登录 Databasus，确认修改已生效。",
              },
            ],
          }),
        }}
      />

      <DocsNavbarComponent lang="zh" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="zh" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="reset-password">重置用户密码</h1>

              <h2 id="reset-password-command">重置密码命令</h2>

              <p>要重置用户密码，在运行 Databasus 的服务器上执行以下命令：</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetPasswordCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="zh" text={resetPasswordCommand} />
                </div>
              </div>

              <h2 id="parameters">参数</h2>

              <p>命令接受以下参数：</p>

              <ul>
                <li>
                  <strong>--new-password</strong>：新密码，至少 8
                  个字符。请确保密码足够安全，包含字母、数字和特殊字符。
                </li>
                <li>
                  <strong>--email</strong>：要重置密码的账户邮箱（例如{" "}
                  <code>owner@example.com</code>
                  ）。在第一个账户成为管理员之前创建的实例，
                  在其所有者替换之前仍使用占位地址 <code>admin</code>
                  ，此时请传入 <code>admin</code>。
                </li>
              </ul>

              <h2 id="list-admins">列出管理员账户</h2>

              <p>
                如果你不记得哪个地址是该实例的管理员，可以在运行 Databasus
                的服务器上列出管理员账户：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{listAdminsCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={listAdminsCommand} lang="zh" />
                </div>
              </div>

              <p>
                输出会列出每个管理员账户的邮箱、显示名称、创建日期和活动状态，并标记实例所认可的管理员。不会输出任何密码、密码哈希或令牌。如果实例上还没有人创建账户，命令会提示该实例没有管理员。
              </p>

              <h2 id="create-user">通过控制台创建账户</h2>

              <p>
                如果不想在浏览器里注册，比如要用部署脚本开通账户，可以在运行
                Databasus 的服务器上直接创建：
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createUserCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={createUserCommand} lang="zh" />
                </div>
              </div>

              <p>命令接受以下参数：</p>

              <ul>
                <li>
                  <strong>--email</strong>：新账户的邮箱，必须是有效的邮箱地址。
                </li>
                <li>
                  <strong>--password</strong>：新账户的密码，至少 8 个字符。
                </li>
                <li>
                  <strong>--role</strong>：<code>admin</code> 或{" "}
                  <code>member</code>。
                </li>
                <li>
                  <strong>--name</strong>：可选的显示名称。不传时，账户名取邮箱中{" "}
                  <code>@</code> 之前的部分。
                </li>
              </ul>

              <p>
                账户创建后立即生效，无论是否开放注册，都可以用这个邮箱和密码登录。如果实例上还没有任何账户，新建的账户就是该实例的管理员，所以第一个账户必须使用{" "}
                <code>--role=admin</code>
                。之后用这种方式创建的管理员只是普通管理员。
              </p>

              <p>
                对同一角色的已有账户再次运行命令会成功结束，但不做任何改动，所以部署流程可以在每次上线时都运行它。这时密码也不会改变：要更换密码，请使用上面的{" "}
                <code>--new-password</code>{" "}
                重置命令。如果该邮箱属于另一角色的账户、已停用或仍在邀请中的账户，或者通过
                Google 或 GitHub 登录的账户，命令会失败，也不做任何改动。
              </p>

              <h2 id="disable-two-factor">停止要求 2FA</h2>

              <p>
                实例可以在密码之外，要求输入通过邮件发送的六位登录验证码。如果邮件服务器不再送达这些验证码，谁都无法用密码登录，单靠重置密码也不管用。请在运行
                Databasus 的服务器上关掉第二重验证：
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
                命令会说明它是否改动了设置，第二重验证本来就关着时也会正常结束，并把这次改动写入审计日志。之后的密码登录不再需要验证码；等邮件恢复正常，管理员可以重新打开该设置。
              </p>

              <h2 id="reset-authenticator-app">重置账号的身份验证器应用</h2>

              <p>
                如果有人丢了装有身份验证器应用的手机，请在运行 Databasus 的服务器上把该应用从其账号中移除：
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
                该账号下次登录时会按照实例的 2FA 设置进行。管理员也可以在用户列表中重置用户的应用。
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
