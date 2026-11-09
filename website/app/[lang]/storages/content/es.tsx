import type { Metadata } from "next";
import {
  OG_LOCALES,
  getLanguageAlternates,
  getLocalizedUrl,
} from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Almacenamientos - Documentación de Databasus",
  description:
    "Lista de destinos de almacenamiento compatibles con las copias de seguridad de Databasus: almacenamiento local, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
  keywords: [
    "almacenamientos Databasus",
    "almacenamiento de copias de seguridad",
    "almacenamiento S3",
    "Cloudflare R2",
    "copia de seguridad en NAS",
    "almacenamiento local",
    "Azure Blob Storage",
    "copia de seguridad por FTP",
    "copia de seguridad por SFTP",
  ],
  openGraph: {
    title: "Almacenamientos - Documentación de Databasus",
    description:
      "Lista de destinos de almacenamiento compatibles con las copias de seguridad de Databasus: almacenamiento local, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
    type: "article",
    url: getLocalizedUrl("es", "storages"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Almacenamientos - Documentación de Databasus",
    description:
      "Lista de destinos de almacenamiento compatibles con las copias de seguridad de Databasus: almacenamiento local, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "storages"),
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
            headline: "Almacenamientos - Documentación de Databasus",
            description:
              "Lista de destinos de almacenamiento compatibles con las copias de seguridad de Databasus: almacenamiento local, S3, Cloudflare R2, Azure Blob Storage, NAS, FTP, SFTP.",
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

      <DocsNavbarComponent lang="es" />

      <div className="flex min-h-screen bg-[#0F1115]">
        {/* Sidebar */}
        <DocsSidebarComponent lang="es" />

        {/* Main Content */}
        <main className="flex-1 min-w-0 px-4 py-6 sm:px-6 sm:py-8 lg:px-12">
          <div className="mx-auto max-w-4xl">
            <article className="prose prose-blue max-w-none">
              <h1 id="storages">Almacenamientos</h1>

              <p className="text-lg text-gray-400">
                Databasus admite varios destinos de almacenamiento para sus
                copias de seguridad de PostgreSQL. Elija dónde guardar los
                archivos de respaldo según su infraestructura y sus requisitos.
              </p>

              <h2 id="supported-storages">Almacenamientos compatibles</h2>

              <ul>
                <li>
                  <strong>Almacenamiento local</strong> - Guarde los respaldos
                  directamente en su servidor o VPS
                </li>
                <li>
                  <strong>S3</strong> - Amazon S3 y servicios de almacenamiento
                  compatibles con S3
                </li>
                <li>
                  <a
                    href="/es/storages/cloudflare-r2"
                    className="font-semibold! text-blue-600 hover:text-blue-800"
                  >
                    Cloudflare R2
                  </a>{" "}
                  - Almacenamiento de objetos de Cloudflare compatible con S3
                </li>
                <li>
                  <strong>Azure Blob Storage</strong> - Almacenamiento en la
                  nube de Microsoft Azure
                </li>
                <li>
                  <strong>NAS</strong> - Dispositivos de almacenamiento
                  conectados a la red
                </li>
                <li>
                  <strong>FTP</strong> - Servidores del protocolo de
                  transferencia de archivos
                </li>
                <li>
                  <strong>SFTP</strong> - Servidores de transferencia de
                  archivos sobre SSH
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
