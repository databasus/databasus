import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

const PAGE_URL = getLocalizedUrl("es", "storages/neon-object-storage");

export const metadata: Metadata = {
  title: "Copia de seguridad de PostgreSQL en Neon Object Storage: guía paso a paso | Databasus",
  description:
    "Guía paso a paso con capturas: cree un bucket y una credencial en Neon Object Storage, añádalo a Databasus como almacenamiento S3 y guarde las copias de seguridad de sus bases de datos en Neon según una programación.",
  keywords: [
    "Databasus",
    "copia de seguridad en Neon Object Storage",
    "Neon Object Storage",
    "Neon S3",
    "bucket de Neon",
    "copia de seguridad de PostgreSQL en S3",
    "almacenamiento compatible con S3",
    "copia de seguridad de bases de datos",
  ],
  openGraph: {
    title: "Copia de seguridad de PostgreSQL en Neon Object Storage: guía paso a paso | Databasus",
    description:
      "Guía paso a paso con capturas: cree un bucket y una credencial en Neon Object Storage, añádalo a Databasus como almacenamiento S3 y guarde las copias de seguridad de sus bases de datos en Neon según una programación.",
    type: "article",
    url: PAGE_URL,
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Copia de seguridad de PostgreSQL en Neon Object Storage: guía paso a paso | Databasus",
    description:
      "Guía paso a paso con capturas: cree un bucket y una credencial en Neon Object Storage, añádalo a Databasus como almacenamiento S3 y guarde las copias de seguridad de sus bases de datos en Neon según una programación.",
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
            name: "Cómo hacer copias de seguridad en Neon Object Storage con Databasus",
            description:
              "Guía paso a paso para crear un bucket y una credencial en Neon Object Storage y añadirlo a Databasus como almacenamiento de copias de seguridad",
            step: [
              {
                "@type": "HowToStep",
                name: "Cree un bucket en Neon",
                text: "En Neon Console, abra el proyecto para las copias de seguridad, haga clic en Object storage en la barra lateral y luego en Create bucket.",
                url: `${PAGE_URL}#create-bucket`,
              },
              {
                "@type": "HowToStep",
                name: "Copie las credenciales del almacenamiento",
                text: "Haga clic en Connect, abra la pestaña Storage, elija Parameters only y haga clic en Copy credentials.",
                url: `${PAGE_URL}#copy-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "Elija Neon Object Storage en Databasus",
                text: "En Databasus, abra Storages, haga clic en Add storage y elija Neon Object Storage en la lista de tipos.",
                url: `${PAGE_URL}#choose-neon-object-storage`,
              },
              {
                "@type": "HowToStep",
                name: "Pegue las credenciales y escriba el nombre del bucket",
                text: "Haga clic en Paste from clipboard para rellenar el endpoint, la región, la access key y la secret key, y luego escriba el nombre del bucket.",
                url: `${PAGE_URL}#paste-credentials`,
              },
              {
                "@type": "HowToStep",
                name: "Pruebe la conexión y guarde",
                text: "Haga clic en Test connection y, cuando la prueba pase, en Save.",
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
              <h1 id="neon-object-storage-backup">Cómo hacer copias de seguridad en Neon Object Storage</h1>

              <p className="text-lg text-gray-400">
                <a href="https://neon.com/docs/storage/overview">Neon Object Storage</a> es un
                almacenamiento de objetos compatible con S3 integrado en cada proyecto de Neon.
                Para guardar ahí sus copias de seguridad, cree un bucket y una credencial en
                Neon Console y luego añada el bucket a Databasus como almacenamiento. Databasus
                subirá allí cada respaldo de las bases de datos que usen este almacenamiento,
                estén en Neon o en cualquier otro lugar.
              </p>

              <p>
                Solo necesita un Databasus en funcionamiento. Si aún no lo tiene, siga
                la <a href={getLocalizedHref("es", "installation")}>guía de instalación</a>.
                La <a href="https://neon.com/docs/storage/s3-compatibility">página de
                compatibilidad con S3</a> de la documentación de Neon indica qué admite.
              </p>

              <h2 id="create-bucket">Paso 1. Cree un bucket en Neon</h2>

              <p>
                Abra en Neon Console el proyecto que guardará las copias de seguridad. Mejor que
                sea un proyecto distinto del de las bases de datos que respalda: al eliminar un
                proyecto también se eliminan sus buckets. Haga clic en <strong>Object
                storage</strong> en la barra lateral, luego en <strong>Create bucket</strong>, y
                escriba un nombre para el bucket. Un bucket nuevo es privado y pertenece a la
                rama actual del proyecto.
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-object-storage-buckets.png"
                alt="Página Object storage de un proyecto de Neon con un bucket y el botón Create bucket"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
              />

              <h2 id="copy-credentials">Paso 2. Copie las credenciales del almacenamiento</h2>

              <p>
                Haga clic en <strong>Connect</strong> y abra la
                pestaña <strong>Storage</strong>. Con <strong>Parameters
                only</strong> seleccionado, Neon muestra el endpoint, la access key, la secret
                key y la región. Si el bloque muestra marcadores en lugar de claves, genere
                primero una credencial, como pide su primera línea. Después haga clic
                en <strong>Copy credentials</strong>.
              </p>

              <img
                src="/images/storages/neon-object-storage/neon-connect-storage-credentials.png"
                alt="Pestaña Storage de la ventana Connect de Neon con el endpoint, las claves, la región y el botón Copy credentials"
                width={1446}
                height={901}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                El bloque tiene este aspecto. No incluye el nombre del bucket, así que recuerde
                el nombre del paso 1.
              </p>

              <pre>
                <code>{`AWS_ENDPOINT_URL_S3="https://br-<branch>.storage.c-9.us-east-2.aws.neon.tech"
AWS_ACCESS_KEY_ID="nak_live_..."
AWS_SECRET_ACCESS_KEY="..."
AWS_REGION="us-east-2"`}</code>
              </pre>

              <h2 id="add-storage">Paso 3. Añada el almacenamiento a Databasus</h2>

              <h3 id="choose-neon-object-storage">Elija Neon Object Storage en Databasus</h3>

              <p>
                En Databasus, abra <strong>Storages</strong> y haga clic en <strong>Add
                storage</strong>. Póngale un nombre al almacenamiento y elija <strong>Neon
                Object Storage</strong> en la lista de tipos. Databasus lo guarda como un
                almacenamiento S3 normal, por eso la lista de almacenamientos muestra su tipo
                como S3.
              </p>

              <h3 id="paste-credentials">Pegue las credenciales y escriba el nombre del bucket</h3>

              <p>
                Haga clic en <strong>Paste from clipboard</strong>. Databasus lee el bloque que
                copió en Neon y rellena el endpoint, la región, la access key y la secret key.
                El bloque no sale de su navegador: las claves solo salen de él cuando prueba o
                guarda el almacenamiento. Escriba usted mismo el nombre del bucket en <strong>S3
                Bucket</strong>. También puede rellenar todos los campos a mano debajo del
                separador <strong>or enter the details manually</strong>.
              </p>

              <img
                src="/images/storages/neon-object-storage/databasus-neon-object-storage-form.png"
                alt="Formulario vacío de Neon Object Storage en Databasus, con el bloque para pegar las credenciales y los campos para completar a mano"
                width={1440}
                height={898}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="test-connection">Pruebe la conexión y guarde</h3>

              <p>
                Haga clic en <strong>Test connection</strong>. Cuando la prueba pase, haga clic
                en <strong>Save</strong>. Ahora puede elegir este almacenamiento en la
                configuración de copias de seguridad de cualquier base de datos. Si la propia
                base de datos está en Neon, la <a href={getLocalizedHref("es", "faq/neon")}>guía
                de copias de seguridad de Neon</a> explica el resto de la configuración.
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("es", "storages")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Volver a almacenamientos
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
