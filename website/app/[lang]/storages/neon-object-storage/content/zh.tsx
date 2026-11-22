import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

const PAGE_URL = getLocalizedUrl("zh", "storages/neon-object-storage");

export const metadata: Metadata = {
  title: "PostgreSQL 备份到 Neon Object Storage：图文分步指南 | Databasus",
  description:
    "图文分步指南：在 Neon Object Storage 中创建存储桶和凭据，在 Databasus 中把它添加为 S3 存储，按计划把数据库备份保存到 Neon。",
  keywords: [
    "Databasus",
    "Neon Object Storage 备份",
    "Neon Object Storage",
    "Neon S3",
    "Neon 存储桶",
    "PostgreSQL 备份到 S3",
    "S3 兼容存储",
    "数据库备份",
  ],
  openGraph: {
    title: "PostgreSQL 备份到 Neon Object Storage：图文分步指南 | Databasus",
    description:
      "图文分步指南：在 Neon Object Storage 中创建存储桶和凭据，在 Databasus 中把它添加为 S3 存储，按计划把数据库备份保存到 Neon。",
    type: "article",
    url: PAGE_URL,
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "PostgreSQL 备份到 Neon Object Storage：图文分步指南 | Databasus",
    description:
      "图文分步指南：在 Neon Object Storage 中创建存储桶和凭据，在 Databasus 中把它添加为 S3 存储，按计划把数据库备份保存到 Neon。",
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
            name: "如何用 Databasus 把备份存到 Neon Object Storage",
            description:
              "在 Neon Object Storage 中创建存储桶和凭据，并在 Databasus 中添加为备份存储的分步指南",
            step: [
              {
                "@type": "HowToStep",
                name: "在 Neon 中创建存储桶",
                text: "在 Neon Console 中打开用于备份的项目，点击侧边栏中的 Object storage，再点击 Create bucket。",
                url: `${PAGE_URL}#create-bucket`,
              },
              {
                "@type": "HowToStep",
                name: "复制存储凭据",
                text: "点击 Connect，打开 Storage 标签页，选择 Parameters only，然后点击 Copy credentials。",
                url: `${PAGE_URL}#copy-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "在 Databasus 中选择 Neon Object Storage",
                text: "在 Databasus 中打开 Storages，点击 Add storage，在类型列表中选择 Neon Object Storage。",
                url: `${PAGE_URL}#choose-neon-object-storage`,
              },
              {
                "@type": "HowToStep",
                name: "粘贴凭据并填写存储桶名称",
                text: "点击 Paste from clipboard 填入 endpoint、区域、access key 和 secret key，然后填写存储桶名称。",
                url: `${PAGE_URL}#paste-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "测试连接并保存",
                text: "点击 Test connection，测试通过后点击 Save。",
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
              <h1 id="neon-object-storage-backup">如何将 PostgreSQL 备份存到 Neon Object Storage</h1>

              <p className="text-lg text-gray-400">
                <a href="https://neon.com/docs/storage/overview">Neon Object Storage</a> 是每个
                Neon 项目内置的 S3 兼容对象存储。要把备份存到这里，先在 Neon Console 中创建存储桶和凭据，再把存储桶作为存储添加到
                Databasus。之后，使用这个存储的数据库每次备份都会上传到这里，无论数据库运行在 Neon 还是别处。
              </p>

              <p>
                你只需要一个正在运行的
                Databasus。如果还没有，请先按照<a href={getLocalizedHref("zh", "installation")}>安装指南</a>进行安装。Neon
                Object Storage 目前处于 beta 阶段，Neon
                文档中的 <a href="https://neon.com/docs/storage/s3-compatibility">S3
                兼容性页面</a>列出了它支持的功能。
              </p>

              <h2 id="create-bucket">第 1 步：在 Neon 中创建存储桶</h2>

              <p>
                在 Neon Console
                中打开用来存放备份的项目。最好不要用数据库所在的项目：删除项目时，其中的存储桶也会被删除。点击侧边栏中的 <strong>Object
                storage</strong>，再点击 <strong>Create
                bucket</strong>，输入存储桶名称。新建的存储桶是私有的，属于项目的当前分支。
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-object-storage-buckets.png"
                alt="Neon 项目的 Object storage 页面，显示一个存储桶和 Create bucket 按钮"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
              />

              <h2 id="copy-credentials">第 2 步：复制存储凭据</h2>

              <p>
                点击 <strong>Connect</strong>，打开 <strong>Storage</strong> 标签页。选中 <strong>Parameters
                only</strong> 时，Neon 会显示 endpoint、access key、secret key
                和区域。如果显示的是占位符而不是密钥，请先按代码块第一行的提示生成凭据。然后点击 <strong>Copy credentials</strong>。
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-connect-storage-credentials.png"
                alt="Neon Connect 窗口的 Storage 标签页，显示 endpoint、密钥、区域和 Copy credentials 按钮"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                代码块如下。其中不包含存储桶名称，请记住第 1 步中的名称。
              </p>

              <pre>
                <code>{`AWS_ENDPOINT_URL_S3="https://br-<branch>.storage.c-9.us-east-2.aws.neon.tech"
AWS_ACCESS_KEY_ID="nak_live_..."
AWS_SECRET_ACCESS_KEY="..."
AWS_REGION="us-east-2"`}</code>
              </pre>

              <h2 id="add-storage">第 3 步：将存储添加到 Databasus</h2>

              <h3 id="choose-neon-object-storage">在 Databasus 中选择 Neon Object Storage</h3>

              <p>
                在 Databasus 中打开 <strong>Storages</strong>，点击 <strong>Add
                storage</strong>。为存储命名，并在类型列表中选择 <strong>Neon Object Storage</strong>。Databasus
                把它保存为普通的 S3 存储，所以存储列表中显示的类型是 S3。
              </p>

              <h3 id="paste-credentials">粘贴凭据并填写存储桶名称</h3>

              <p>
                点击 <strong>Paste from clipboard</strong>。Databasus 读取你在 Neon 中复制的代码块，填入
                endpoint、区域、access key 和 secret
                key。代码块只在你的浏览器中处理，只有在你测试或保存存储时，密钥才会发送出去。存储桶名称需要你自己填写到 <strong>S3
                Bucket</strong> 中。你也可以在 <strong>or enter the details
                manually</strong> 分隔线下方手动填写所有字段。
              </p>

              <img
                src="/images/storages/neon-object-storage/databasus-neon-object-storage-form.png"
                alt="Databasus 的 Neon Object Storage 表单，已粘贴 endpoint、区域和密钥，并填写了 bucket 名称"
                width={1435}
                height={897}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="test-connection">测试连接并保存</h3>

              <p>
                点击 <strong>Test
                connection</strong>。测试通过后，点击 <strong>Save</strong>。现在可以在任意数据库的备份设置中选择这个存储。如果数据库本身也运行在
                Neon 上，<a href={getLocalizedHref("zh", "faq/neon")}>Neon 备份指南</a>介绍了其余的设置步骤。
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("zh", "storages")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← 返回存储列表
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
