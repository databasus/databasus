import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

const SCREENSHOT_CLASS_NAME =
  "my-6 h-auto w-full max-w-full rounded-lg border border-gray-700";

export const metadata: Metadata = {
  title: "Copia de seguridad de PostgreSQL en Neon: guía paso a paso | Databasus",
  description:
    "Guía paso a paso con capturas de pantalla: programe copias de seguridad de una base de datos PostgreSQL de Neon con Databasus, guárdelas en su propio almacenamiento y restáurelas en Neon.",
  keywords: [
    "Databasus",
    "copia de seguridad de Neon",
    "copia de seguridad de PostgreSQL en Neon",
    "respaldo de base de datos Neon",
    "restauración en Neon",
    "usuario de solo lectura de Neon",
    "copia de seguridad de PostgreSQL",
    "copia de seguridad de base de datos",
  ],
  openGraph: {
    title: "Copia de seguridad de PostgreSQL en Neon: guía paso a paso | Databasus",
    description:
      "Guía paso a paso con capturas de pantalla: programe copias de seguridad de una base de datos PostgreSQL de Neon con Databasus, guárdelas en su propio almacenamiento y restáurelas en Neon.",
    type: "article",
    url: getLocalizedUrl("es", "faq/neon"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Copia de seguridad de PostgreSQL en Neon: guía paso a paso | Databasus",
    description:
      "Guía paso a paso con capturas de pantalla: programe copias de seguridad de una base de datos PostgreSQL de Neon con Databasus, guárdelas en su propio almacenamiento y restáurelas en Neon.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "faq/neon"),
    languages: getLanguageAlternates("faq/neon"),
  },
  robots: "index, follow",
};

export default function NeonPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cómo hacer una copia de seguridad de PostgreSQL en Neon con Databasus",
            description:
              "Guía paso a paso para respaldar una base de datos PostgreSQL de Neon con Databasus y restaurarla en Neon",
            step: [
              {
                "@type": "HowToStep",
                name: "Copie la cadena de conexión de Neon",
                text: "En la consola de Neon, abra su proyecto, haga clic en Connect y luego en Copy snippet.",
                url: `${getLocalizedUrl("es", "faq/neon")}#copy-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "Añada la base de datos de Neon a Databasus",
                text: "Haga clic en Add database, elija Neon, pegue la cadena de conexión con Paste from clipboard y haga clic en Test connection.",
                url: `${getLocalizedUrl("es", "faq/neon")}#paste-connection-string`,
              },
              {
                "@type": "HowToStep",
                name: "Cree un usuario de solo lectura",
                text: "Haga clic en Yes, create read-only user para que Databasus guarde credenciales que no pueden modificar sus datos.",
                url: `${getLocalizedUrl("es", "faq/neon")}#read-only-user`,
              },
              {
                "@type": "HowToStep",
                name: "Configure la programación, el almacenamiento y la retención",
                text: "Elija cada cuánto se hace la copia de seguridad, dónde se guarda y cuánto tiempo se conserva.",
                url: `${getLocalizedUrl("es", "faq/neon")}#schedule-storage-retention`,
              },
              {
                "@type": "HowToStep",
                name: "Elija los canales de notificación",
                text: "Marque los canales que recibirán los avisos sobre las copias de seguridad y haga clic en Complete. Databasus inicia la primera copia.",
                url: `${getLocalizedUrl("es", "faq/neon")}#notifiers`,
              },
              {
                "@type": "HowToStep",
                name: "Restaure una copia de seguridad en Neon",
                text: "Haga clic en el icono de restauración junto a una copia, pegue la cadena de conexión de la base de datos de Neon de destino y haga clic en Restore to this DB.",
                url: `${getLocalizedUrl("es", "faq/neon")}#restore`,
              },
            ],
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
              <h1 id="neon-backup">
                Cómo hacer una copia de seguridad de PostgreSQL en Neon
              </h1>

              <p className="text-lg text-gray-400">
                Neon es un servicio de PostgreSQL gestionado y serverless. Para
                respaldarlo con Databasus, copie la cadena de conexión en la
                consola de Neon, añada la base de datos en Databasus con el
                tipo Neon y elija dónde guardar las copias. A partir de ahí,
                Databasus hace copias de seguridad según la
                programación que defina y puede restaurarlas en cualquier base
                de datos de Neon.
              </p>

              <p>
                Solo necesita una instancia de Databasus en funcionamiento. Si aún no la tiene, siga la{" "}
                <a href={getLocalizedHref("es", "installation")}>guía de instalación</a>
                . El almacenamiento y los canales de notificación se pueden crear durante la configuración.
              </p>

              <h2 id="copy-connection-string">
                Paso 1. Copie la cadena de conexión de Neon
              </h2>

              <p>
                Abra su proyecto en la consola de Neon y haga clic en{" "}
                <strong>Connect</strong>.
              </p>

              <img
                src="/images/faq/neon/neon-connect-button.png"
                alt="Botón Connect en la página Computes de un proyecto de Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
              />

              <p>
                En la ventana <strong>Connection details</strong>, elija la
                rama, la base de datos y el rol que quiere respaldar, y haga
                clic en <strong>Copy snippet</strong>.
              </p>

              <img
                src="/images/faq/neon/neon-copy-connection-string.png"
                alt="Ventana Connection details de Neon con el botón Copy snippet"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="add-database">
                Paso 2. Añada la base de datos de Neon a Databasus
              </h2>

              <h3 id="paste-connection-string">
                Elija Neon y pegue la cadena de conexión
              </h3>

              <p>
                En Databasus, haga clic en <strong>Add database</strong>,
                póngale un nombre a la base de datos, elija{" "}
                <strong>Neon</strong> y haga clic en <strong>Continue</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-choose-neon.png"
                alt="Ventana Add database de Databasus con el tipo Neon seleccionado"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Haga clic en <strong>Paste from clipboard</strong>. Databasus
                lee la cadena de conexión que copió en Neon y rellena el host,
                el puerto, el usuario, la contraseña, el nombre de la base de
                datos y el modo SSL.
              </p>

              <img
                src="/images/faq/neon/databasus-paste-connection-string.png"
                alt="Formulario de conexión a Neon en Databasus con el botón Paste from clipboard"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Haga clic en <strong>Test connection</strong>. Cuando la prueba
                salga bien, haga clic en <strong>Continue</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-test-connection.png"
                alt="Formulario de conexión a Neon completado en Databasus tras una prueba de conexión correcta"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="read-only-user">Cree un usuario de solo lectura</h3>

              <p>
                Databasus le propone crear un{" "}
                <a
                  href={`${getLocalizedHref("es", "security")}#level-3-read-only-access`}
                >
                  usuario de solo lectura
                </a>{" "}
                para las copias de seguridad. Haga clic en{" "}
                <strong>Yes, create read-only user</strong>. Así Databasus solo
                guarda credenciales que pueden leer sus datos, pero no
                modificarlos.
              </p>

              <img
                src="/images/faq/neon/databasus-create-read-only-user.png"
                alt="Databasus propone crear un usuario de solo lectura para las copias de seguridad de Neon"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="schedule-storage-retention">
                Configure la programación, el almacenamiento y la retención
              </h3>

              <p>Elija:</p>

              <ul>
                <li>
                  <strong>Backup interval</strong> y hora: cada cuánto hace
                  Databasus una copia de seguridad, por ejemplo, todos los días
                  a las 03:00.
                </li>
                <li>
                  <strong>Storage</strong>: dónde se guardan las copias. Puede{" "}
                  <a href={getLocalizedHref("es", "storages")}>
                    añadir un almacenamiento
                  </a>{" "}
                  directamente desde esta lista. Si también quiere guardar los
                  respaldos en Neon, consulte{" "}
                  <a href={getLocalizedHref("es", "storages/neon-object-storage")}>
                    cómo hacer copias de seguridad en Neon Object Storage
                  </a>
                  .
                </li>
                <li>
                  <strong>Encryption</strong>: si Databasus{" "}
                  <a
                    href={`${getLocalizedHref("es", "security")}#level-2-backups-encryption`}
                  >
                    cifra los archivos de respaldo
                  </a>{" "}
                  antes de subirlos.
                </li>
                <li>
                  <strong>Retention policy</strong>: cuánto tiempo se conservan
                  las copias antiguas, por ejemplo, 3 meses.
                </li>
                <li>
                  <strong>Notifications</strong>: qué eventos se notifican, solo
                  las copias fallidas o también las correctas.
                </li>
              </ul>

              <img
                src="/images/faq/neon/databasus-schedule-storage-retention.png"
                alt="Ajustes de programación, almacenamiento, cifrado y retención de las copias de seguridad en Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h3 id="notifiers">Elija los canales de notificación</h3>

              <p>
                Marque los{" "}
                <a href={getLocalizedHref("es", "notifiers")}>
                  canales de notificación
                </a>{" "}
                que recibirán estos avisos, por ejemplo,{" "}
                <a href={getLocalizedHref("es", "notifiers/slack")}>Slack</a>{" "}
                o{" "}
                <a href={getLocalizedHref("es", "notifiers/teams")}>
                  Microsoft Teams
                </a>
                . Puede elegir varios, o ninguno y añadirlos más tarde. Haga
                clic en <strong>Complete</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-choose-notifiers.png"
                alt="Selección de canales de notificación en Databasus antes de terminar la configuración de la base de datos"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="first-backup">
                Paso 3. Compruebe la primera copia de seguridad
              </h2>

              <p>
                Databasus inicia la primera copia de seguridad en cuanto hace
                clic en <strong>Complete</strong>. La copia aparece en la
                pestaña <strong>Backups</strong> con su tamaño y su duración.
                Las siguientes se hacen según la programación, y{" "}
                <strong>Make backup right now</strong> lanza una en cualquier
                momento.
              </p>

              <img
                src="/images/faq/neon/databasus-first-neon-backup.png"
                alt="Primera copia de seguridad correcta de Neon en la pestaña Backups de Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Que una copia de seguridad termine bien no demuestra que se
                pueda restaurar. Para comprobar que cada respaldo se restaura de
                verdad, configure la{" "}
                <a href={getLocalizedHref("es", "restore-verification")}>
                  verificación de restauración
                </a>
                .
              </p>

              <h2 id="restore">Cómo restaurar una copia de seguridad de Neon</h2>

              <p>
                Haga clic en el icono de restauración (una nube con una flecha)
                junto a la copia de seguridad. Copie en la consola de Neon la
                cadena de conexión de la base de datos de destino igual que en
                el <a href="#copy-connection-string">paso 1</a>, haga clic en{" "}
                <strong>Paste from clipboard</strong> y luego en{" "}
                <strong>Restore to this DB</strong>.
              </p>

              <img
                src="/images/faq/neon/databasus-restore-neon-backup.png"
                alt="Ventana de restauración desde una copia de seguridad en Databasus con la base de datos de Neon de destino"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <p>
                Databasus vacía la base de datos de destino antes de la
                restauración. El destino puede ser:
              </p>

              <ul>
                <li>una base de datos nueva en el mismo proyecto de Neon</li>
                <li>una base de datos en una rama de Neon</li>
                <li>una base de datos existente, incluida la propia base de origen</li>
              </ul>

              <p>
                La opción más segura es una rama nueva de Neon. Allí comprueba
                los datos restaurados, y la base de datos de producción queda
                intacta hasta que decida cambiar.
              </p>

              <p>Cuando termina la restauración, su estado pasa a Successful.</p>

              <img
                src="/images/faq/neon/databasus-restore-done.png"
                alt="Restauración terminada de una copia de seguridad de Neon en Databasus"
                width={1433}
                height={895}
                className={SCREENSHOT_CLASS_NAME}
                loading="lazy"
              />

              <h2 id="health-check">
                ¿La comprobación de disponibilidad mantiene activo el compute de
                Neon?
              </h2>

              <p>
                Por defecto, no. Neon suspende el compute cuando no hay nadie
                conectado y no lo cobra mientras está suspendido. La
                comprobación de disponibilidad se conecta cada minuto, así que
                el compute seguiría activo y facturándose las 24 horas. Por eso
                Databasus desactiva la comprobación de disponibilidad en las
                bases de datos de Neon. Si la activa, Databasus le pide antes
                una confirmación. Una copia de seguridad programada despierta
                el compute solo mientras dura, y después Neon lo vuelve a
                suspender.
              </p>

              <p>
                ¿También respalda Supabase? Consulte{" "}
                <a href={getLocalizedHref("es", "faq/supabase")}>
                  cómo hacer copias de seguridad de Supabase
                </a>
                .
              </p>

              {/* Navigation */}
              <div className="mt-12 border-t border-gray-200 pt-8">
                <a
                  href={getLocalizedHref("es", "faq")}
                  className="inline-flex items-center font-semibold text-blue-600 hover:text-blue-800"
                >
                  ← Volver a las preguntas frecuentes
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
