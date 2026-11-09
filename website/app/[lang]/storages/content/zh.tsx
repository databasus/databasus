import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "存储 - Databasus 文档",
  description:
    "Databasus 备份支持的存储目标列表，包括本地存储、S3、Cloudflare R2、Azure Blob Storage、NAS、FTP、SFTP。",
  keywords: [
    "Databasus 存储",
    "备份存储",
    "S3 存储",
    "Cloudflare R2",
    "NAS 备份",
    "本地存储",
    "Azure Blob Storage",
    "FTP 备份",
    "SFTP 备份",
  ],
  openGraph: {
    title: "存储 - Databasus 文档",
    description:
      "Databasus 备份支持的存储目标列表，包括本地存储、S3、Cloudflare R2、Azure Blob Storage、NAS、FTP、SFTP。",
    type: "article",
    url: getLocalizedUrl("zh", "storages"),
    locale: OG_LOCALES.zh,
  },
  twitter: {
    card: "summary",
    title: "存储 - Databasus 文档",
    description:
      "Databasus 备份支持的存储目标列表，包括本地存储、S3、Cloudflare R2、Azure Blob Storage、NAS、FTP、SFTP。",
  },
  alternates: {
    canonical: getLocalizedUrl("zh", "storages"),
    languages: getLanguageAlternates("storages"),
  },
  robots: "index, follow",
};

export default function StoragesPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "存储 - Databasus 文档",
            description:
              "Databasus 备份支持的存储目标列表，包括本地存储、S3、Cloudflare R2、Azure Blob Storage、NAS、FTP、SFTP。",
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

      <DocsNavbarComponent lang="zh" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="zh" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="storages">存储</h1>

              <p className="text-lg text-gray-400">
                Databasus 支持多种 PostgreSQL 备份存储目标。你可以根据
                自己的基础设施和需求选择备份文件的存放位置。
              </p>

              <h2 id="supported-storages">支持的存储</h2>

              <ul>
                <li>
                  <strong>本地存储</strong> - 直接把备份存放在你的服务器或
                  VPS 上
                </li>
                <li>
                  <strong>S3</strong> - Amazon S3 及兼容 S3 协议的存储服务
                </li>
                <li>
                  <a
                    href="/zh/storages/cloudflare-r2"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Cloudflare R2
                  </a>{" "}
                  - Cloudflare 提供的 S3 兼容对象存储
                </li>
                <li>
                  <strong>Azure Blob Storage</strong> - Microsoft Azure
                  提供的云存储
                </li>
                <li>
                  <strong>NAS</strong> - 网络附加存储设备
                </li>
                <li>
                  <strong>FTP</strong> - 文件传输协议（FTP）服务器
                </li>
                <li>
                  <strong>SFTP</strong> - SSH 文件传输协议（SFTP）服务器
                </li>
              </ul>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
