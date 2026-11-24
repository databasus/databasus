import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Verificación de restauración - Documentación de Databasus",
  description:
    "Demuestre que sus copias de seguridad realmente se pueden restaurar. Databasus toma la última copia, la restaura en un contenedor de base de datos desechable, compara la base restaurada con la de origen e informa del recuento de filas por tabla en cada ejecución.",
  keywords: [
    "verificación de restauración",
    "restauración de bases de datos",
    "verificación de copias de seguridad",
    "recuperación ante desastres",
    "pruebas de copias de seguridad",
    "agente de verificación de Databasus",
    "integridad de las copias de seguridad",
    "prueba de restauración automatizada",
  ],
  openGraph: {
    title: "Verificación de restauración - Documentación de Databasus",
    description:
      "Demuestre que sus copias de seguridad realmente se pueden restaurar. Databasus toma la última copia, la restaura en un contenedor de base de datos desechable, compara la base restaurada con la de origen e informa del recuento de filas por tabla en cada ejecución.",
    type: "article",
    url: getLocalizedUrl("es", "restore-verification"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Verificación de restauración - Documentación de Databasus",
    description:
      "Demuestre que sus copias de seguridad realmente se pueden restaurar. Databasus toma la última copia, la restaura en un contenedor de base de datos desechable, compara la base restaurada con la de origen e informa del recuento de filas por tabla en cada ejecución.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "restore-verification"),
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
            headline:
              "Verificación de restauración - Documentación de Databasus",
            description:
              "Demuestre que sus copias de seguridad realmente se pueden restaurar. Databasus toma la última copia, la restaura en un contenedor de base de datos desechable, compara la base restaurada con la de origen e informa del recuento de filas por tabla en cada ejecución.",
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
            name: "Cómo configurar la verificación de restauración en Databasus",
            description:
              "Guía paso a paso para registrar un agente de verificación, lanzarlo en su servidor y configurar la verificación de restauración programada.",
            step: [
              {
                "@type": "HowToStep",
                name: "Crear un agente de verificación en la interfaz",
                text: "Vaya a Settings → Verification agents, haga clic en Create verification agent, asígnele un nombre y copie el token y el ID del agente desde el diálogo.",
              },
              {
                "@type": "HowToStep",
                name: "Descargar el binario del agente",
                text: "Ejecute el comando curl en el host donde se ejecutará la verificación, eligiendo amd64 o arm64 según su arquitectura.",
              },
              {
                "@type": "HowToStep",
                name: "Lanzar el agente",
                text: "Arranque el agente con --agent-id, --token y los presupuestos de recursos (--max-cpu, --max-ram-mb, --max-disk-gb, --max-concurrent-jobs).",
              },
              {
                "@type": "HowToStep",
                name: "Programar las verificaciones",
                text: "Abra la configuración de verificación de la base de datos, active Scheduled verification y elija un intervalo (After backup, Hourly, Daily, Weekly, Monthly o Cron).",
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
              <h1 id="restore-verification">Verificación de restauración</h1>

              <p className="text-lg text-gray-400">
                Una copia de seguridad que termina sin errores no es lo mismo
                que una copia que realmente puede restaurar. La única prueba
                real es restaurarla. Databasus lo hace por usted de forma
                programada:
              </p>

              <ul>
                <li>toma la última copia de seguridad</li>
                <li>
                  ejecuta la restauración en un contenedor de base de datos
                  desechable
                </li>
                <li>compara la base de datos restaurada con la de origen</li>
                <li>elimina el contenedor</li>
                <li>informa del resultado</li>
              </ul>

              <img
                src="/images/restore-verification/verified-backups.png"
                alt="Verified backups tab"
                loading="lazy"
              />

              <img
                className="mt-5"
                src="/images/restore-verification/verifications.png"
                alt="Verifications tab"
                loading="lazy"
              />

              <h2 id="what-is-verification-agent">
                ¿Qué es un agente de verificación?
              </h2>

              <p>
                El agente de verificación es un pequeño binario en Go que se
                ejecuta en una máquina que usted controla: sirve cualquier
                equipo con CPU, RAM y disco libres. El agente se registra en
                Databasus, recoge trabajos de verificación de una cola, los
                ejecuta localmente y devuelve los resultados.
              </p>

              <h3 id="what-you-need">Qué necesita</h3>

              <ul>
                <li>
                  Un host con acceso HTTPS de salida a la URL de su Databasus.
                </li>
                <li>
                  Docker disponible en ese host: el agente crea contenedores de
                  base de datos efímeros con la versión mayor correspondiente
                  para cada trabajo.
                </li>
                <li>
                  Capacidad de disco para cada trabajo de verificación que cubra
                  el <strong>tamaño del archivo de la copia</strong>, el{" "}
                  <strong>tamaño bruto de la base de datos</strong>, una reserva y los{" "}
                  <strong>índices más grandes</strong> que se construyen a la vez.
                </li>
                <li>
                  Al menos 1 núcleo de CPU y 512 MB de RAM disponibles por
                  trabajo concurrente.
                </li>
              </ul>

              <h3 id="why-not-just-checksums">
                ¿Por qué no basta con checksums?
              </h3>

              <p>
                Los checksums y los códigos de salida detectan algunos modos de
                fallo, pero pasan otros completamente por alto:
              </p>

              <ul>
                <li>
                  <strong>Los checksums</strong> detectan corrupción de bits en
                  el archivo, pero no dicen nada sobre si el volcado en sí está
                  completo o es semánticamente válido.
                </li>
                <li>
                  <strong>El código de salida del volcado</strong> indica que el
                  comando se ejecutó. No detecta un rol sin permisos de lectura
                  sobre ciertos objetos, una extensión ausente en el origen o
                  una discrepancia de tablespaces, situaciones que pueden hacer
                  que se omitan o recorten objetos en silencio.
                </li>
                <li>
                  <strong>La verificación de restauración</strong> pasa el
                  archivo por la herramienta de restauración nativa de la base
                  de datos y cuenta las filas de cada tabla. Es la única
                  comprobación que detecta todo lo anterior: si una copia no se
                  puede restaurar, lo descubre antes de necesitarla, no en medio
                  de un desastre.
                </li>
              </ul>

              <h2 id="configuration">Configuración</h2>

              <h3 id="create-on-ui">Crear un agente en la interfaz</h3>

              <p>
                Abra <strong>Settings → Verification agents</strong> y haga clic
                en <strong>Create verification agent</strong>. Elija un nombre
                descriptivo como <code>staging-verifier</code> o{" "}
                <code>eu-west-host-1</code>. El siguiente diálogo muestra el{" "}
                <strong>token</strong> y el <strong>ID</strong> del agente.
              </p>

              <p>
                El token se muestra <strong>una sola vez</strong>: cópielo antes
                de cerrar el diálogo. Si lo pierde más adelante, use la acción{" "}
                <strong>Rotate token</strong> en la fila del agente para emitir
                uno nuevo; el token antiguo deja de funcionar en el siguiente
                latido del agente. El diálogo posterior muestra los comandos de
                instalación para la arquitectura de su servidor, los mismos que
                se describen a continuación.
              </p>

              <h3 id="launch">Lanzar el agente en su servidor</h3>

              <p>
                Conéctese por SSH a la máquina que ejecutará las verificaciones.
                Primero descargue el binario del agente. Sustituya{" "}
                <code>https://your-databasus-host</code> por la URL de su
                Databasus y cambie <code>amd64</code> por <code>arm64</code> si
                su servidor es ARM:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{downloadAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={downloadAgent} />
                </div>
              </div>

              <p>
                Después lance el agente. El ID del agente y el token provienen
                del diálogo del paso anterior:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startAgent}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={startAgent} />
                </div>
              </div>

              <p>
                <code>start</code> convierte el agente en un daemon y escribe
                sus flags en <code>databasus-verification.json</code> en el
                directorio de trabajo, de modo que los siguientes reinicios
                pueden usar <code>./verification-agent start</code> sin ningún
                flag. Los logs se escriben en{" "}
                <code>databasus-verification.log</code> junto al binario.
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
                    El host de Databasus debe ser <code>https://</code>. El HTTP
                    plano solo se permite si añade{" "}
                    <code>--allow-insecure-http</code>, y está pensado para
                    pruebas locales; nunca exponga un agente de producción sobre
                    HTTP sin cifrar.
                  </p>
                </div>
              </div>

              <p>
                Los cuatro flags <code>--max-*</code> son{" "}
                <strong>presupuestos</strong>, no asignaciones por trabajo. El
                agente los comunica a Databasus en cada latido, y Databasus los
                reparte entre los trabajos concurrentes que usted permita. Con{" "}
                <code>
                  --max-cpu=2 --max-ram-mb=2048 --max-concurrent-jobs=1
                </code>{" "}
                el único trabajo recibe las 2 CPU y los 2 GB de RAM. Con{" "}
                <code>--max-concurrent-jobs=2</code>, cada trabajo recibe 1 CPU
                y 1 GB. El mínimo es 1 CPU y 512 MB por trabajo: si su
                presupuesto no alcanza ese mínimo, el agente anuncia una
                concurrencia menor. El presupuesto de disco es el más fácil de
                calcular mal: cada trabajo necesita espacio para el{" "}
                <strong>tamaño del archivo de la copia</strong>, el{" "}
                <strong>tamaño bruto de la base de datos</strong>, una{" "}
                <strong>reserva de unos 1,5 GB</strong> y sus{" "}
                <strong>índices más grandes</strong>, uno por cada trabajo de
                restauración en paralelo, porque cada índice en construcción ocupa
                en disco una ordenación de aproximadamente su mismo tamaño.
                Databasus reduce el número de trabajos de restauración en paralelo
                hasta que todo eso cabe en el disco libre del agente, así que un{" "}
                <code>--max-cpu</code> mayor necesita más disco y un{" "}
                <code>--max-disk-gb</code> mayor permite restaurar más rápido una
                base de datos grande.
              </p>

              <h3 id="manage">Gestionar el agente</h3>

              <p>El mismo binario ofrece cuatro subcomandos:</p>

              <ul>
                <li>
                  <code>./verification-agent status</code> — muestra si el
                  daemon está en ejecución y qué trabajos tiene en curso.
                </li>
                <li>
                  <code>./verification-agent stop</code> — detiene el daemon.
                  Las verificaciones en curso se reportan a Databasus como
                  fallidas y vuelven a la cola.
                </li>
                <li>
                  <code>./verification-agent start</code> — relanza el daemon.
                  Los flags se recuerdan desde el primer arranque; pase{" "}
                  <code>--token=&lt;NEW&gt;</code> tras una rotación para
                  actualizar el token guardado.
                </li>
                <li>
                  <code>./verification-agent run</code> — se ejecuta en primer
                  plano en lugar de como daemon. Úselo al envolver el agente en
                  una unidad de systemd o un contenedor Docker: esos
                  supervisores esperan que el proceso no haga fork.
                </li>
              </ul>

              <p>
                La página Settings muestra tres acciones con icono en la fila de
                cada agente: ver de nuevo los comandos de instalación (sin
                revelar el token), rotar el token y eliminar el agente. Eliminar
                es seguro: las verificaciones asignadas a ese agente vuelven a
                la cola y las recoge otro agente si hay alguno disponible.
              </p>

              <h2 id="schedules-and-notifications">
                Programación y notificaciones
              </h2>

              <p>
                La verificación de restauración se configura por base de datos.
                Abra la configuración de verificación de la base de datos,
                active <strong>Scheduled verification</strong> y elija un
                intervalo.
              </p>

              <h3 id="interval-options">Opciones de intervalo</h3>

              <ul>
                <li>
                  <strong>After backup</strong> — la garantía más fuerte: cada
                  copia de seguridad correcta se verifica en cuanto termina.
                </li>
                <li>
                  <strong>Hourly, daily, weekly, monthly</strong> — elija una
                  cadencia y una hora del día.
                </li>
                <li>
                  <strong>Cron</strong> — una expresión cron en UTC para lo que
                  los preajustes no cubran. Ejemplos: <code>0 4 * * 0</code>{" "}
                  (cada domingo a las 4:00 UTC) y <code>0 */6 * * *</code> (cada
                  seis horas).
                </li>
              </ul>

              <h3 id="how-the-queue-works">
                Cómo maneja la cola &quot;After backup&quot;
              </h3>

              <p>
                Una verificación suele ser más lenta que la copia que la
                originó, así que si las copias llegan más rápido de lo que
                terminan las verificaciones, la cola crecería sin límite.
                Databasus lo evita{" "}
                <strong>
                  cancelando cualquier verificación pendiente de la misma base
                  de datos cuando llega una copia nueva
                </strong>
                : solo la copia más reciente espera en la cola. El compromiso es
                deliberado: es mejor saltarse la verificación de una copia
                obsoleta que pasar horas verificando algo desde lo que nunca
                restauraría.
              </p>

              <h3 id="manual-runs">Ejecuciones manuales</h3>

              <p>
                También puede lanzar una verificación puntual desde la pestaña{" "}
                <strong>Restore verifications</strong> de la base de datos sin
                cambiar la programación. Es útil para comprobar una copia
                concreta o probar de extremo a extremo un agente nuevo antes de
                confiarle la carga programada.
              </p>

              <h3 id="notifications">Notificaciones</h3>

              <p>
                El éxito y el fallo pueden enviarse por cualquier notificador ya
                configurado para la base de datos. Las dos casillas,{" "}
                <strong>Verification success</strong> y{" "}
                <strong>Verification failed</strong>, son independientes. La
                mayoría de los equipos activa solo la de fallo para evitar la
                fatiga de notificaciones. Consulte la{" "}
                <a
                  href="/es/notifiers"
                  className="text-blue-400 hover:text-blue-300"
                >
                  documentación de notificadores
                </a>{" "}
                para configurar Slack, Microsoft Teams, Discord, correo y otros.
              </p>

              <h3 id="results">Interpretar los resultados</h3>

              <p>
                Cada intento de verificación aparece como una fila en la pestaña{" "}
                <strong>Restore verifications</strong> de la base de datos. El
                estado es uno de <strong>Pending</strong>,{" "}
                <strong>Running</strong>, <strong>Successful</strong>,{" "}
                <strong>Failed</strong> o <strong>Canceled</strong>. Al hacer
                clic en una fila se abre un panel con la cronología completa, el
                código de salida de la restauración, el tamaño de la base de
                datos restaurada, el número de esquemas y tablas, y el desglose
                del recuento de filas por tabla. Las ejecuciones fallidas
                muestran el mensaje de error en la parte superior del panel.
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
