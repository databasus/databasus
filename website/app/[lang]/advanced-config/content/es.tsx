import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Configuración avanzada - Documentación de Databasus",
  description:
    "Variables de entorno opcionales para autoalojar Databasus: inicio de sesión con Google y GitHub, correo SMTP, captcha de Cloudflare Turnstile, telemetría, exportación de registros por OpenTelemetry y un script de analítica propio. No hacen falta en una instalación por defecto.",
  keywords: [
    "variables de entorno de Databasus",
    "configuración avanzada de Databasus",
    "configuración autoalojada",
    "GitHub OAuth",
    "Google OAuth",
    "configuración de correo SMTP",
    "Cloudflare Turnstile",
    "variables de entorno de Docker",
    "registros OpenTelemetry",
  ],
  openGraph: {
    title: "Configuración avanzada - Documentación de Databasus",
    description:
      "Variables de entorno opcionales para autoalojar Databasus: inicio de sesión con Google y GitHub, correo SMTP, captcha de Cloudflare Turnstile, telemetría, exportación de registros por OpenTelemetry y un script de analítica propio. No hacen falta en una instalación por defecto.",
    type: "article",
    url: getLocalizedUrl("es", "advanced-config"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Configuración avanzada - Documentación de Databasus",
    description:
      "Variables de entorno opcionales para autoalojar Databasus: inicio de sesión con Google y GitHub, correo SMTP, captcha de Cloudflare Turnstile, telemetría, exportación de registros por OpenTelemetry y un script de analítica propio. No hacen falta en una instalación por defecto.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "advanced-config"),
    languages: getLanguageAlternates("advanced-config"),
  },
  robots: "index, follow",
};

export default function AdvancedConfigPage() {
  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Configuración avanzada - Documentación de Databasus",
            description:
              "Variables de entorno opcionales para autoalojar Databasus: inicio de sesión con Google y GitHub, correo SMTP, captcha de Cloudflare Turnstile, telemetría, exportación de registros por OpenTelemetry y un script de analítica propio. No hacen falta en una instalación por defecto.",
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
              <h1 id="advanced-config">Configuración avanzada</h1>

              <p className="text-lg text-gray-400">
                Databasus funciona con valores por defecto razonables desde el
                primer momento: una instalación estándar de un solo contenedor
                no necesita configuración alguna. Todas las variables de esta
                página son <strong>opcionales</strong> y no hacen falta en el
                99% de los despliegues en producción.
              </p>

              <h2 id="oauth">OAuth</h2>

              <p>
                Por defecto Databasus usa inicio de sesión con correo y
                contraseña. Además, puede permitir que la gente inicie sesión
                con su cuenta de Google o GitHub. El botón de un proveedor
                aparece en cuanto se define su client ID, pero el inicio de
                sesión solo se completa cuando están presentes{" "}
                <strong>tanto</strong> el client ID como el client secret.
              </p>

              <p>
                Al registrar la aplicación OAuth, establezca su URL de
                redirección (callback) en{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>. Debido a
                esa redirección, el inicio de sesión OAuth necesita que su
                instancia se sirva por HTTPS en un dominio público; vea la nota
                más abajo.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    <strong>
                      HTTPS es obligatorio para el inicio de sesión y el correo.
                    </strong>{" "}
                    Tanto el inicio de sesión OAuth como el correo necesitan que
                    su instancia sea accesible por HTTPS en un dominio público:
                    los proveedores OAuth redirigen el navegador de vuelta a{" "}
                    <code>https://&lt;your-domain&gt;/auth/callback</code>, y
                    los enlaces dentro de los correos deben abrirse para quien
                    los recibe. Una instancia solo en localhost o en HTTP plano
                    no puede usar estas funciones. La forma más simple de
                    obtener HTTPS es la configuración del{" "}
                    <a
                      href="/es/installation/#caddy-reverse-proxy"
                      className="text-blue-400 hover:text-blue-300"
                    >
                      reverse proxy Caddy
                    </a>
                    .
                  </p>
                </div>
              </div>

              <h3 id="oauth-google">Google</h3>

              <p>
                Cree un cliente OAuth en la{" "}
                <a
                  href="https://console.cloud.google.com/apis/credentials"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Google Cloud Console
                </a>{" "}
                (APIs &amp; Services → Credentials → Create credentials → OAuth
                client ID, tipo de aplicación <em>Web application</em>) y añada{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code> como URI
                de redirección autorizada.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      Client ID de su cliente OAuth de Google. Al definirlo se
                      muestra el botón &quot;Sign in with Google&quot;.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GOOGLE_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      Client secret de su cliente OAuth de Google. Requerido
                      junto con el ID para que el inicio de sesión funcione.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h3 id="oauth-github">GitHub</h3>

              <p>
                Cree una aplicación OAuth en{" "}
                <a
                  href="https://github.com/settings/developers"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  GitHub Developer settings
                </a>{" "}
                (Settings → Developer settings → OAuth Apps → New OAuth App) y
                establezca la authorization callback URL en{" "}
                <code>https://&lt;your-domain&gt;/auth/callback</code>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_ID</code>
                    </td>
                    <td data-label="Description">
                      Client ID de su aplicación OAuth de GitHub. Al definirlo
                      se muestra el botón &quot;Sign in with GitHub&quot;.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>GITHUB_CLIENT_SECRET</code>
                    </td>
                    <td data-label="Description">
                      Client secret de su aplicación OAuth de GitHub. Requerido
                      junto con el ID para que el inicio de sesión funcione.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="email-smtp">Correo electrónico (SMTP)</h2>

              <p>
                Conecte un servidor de correo para que Databasus pueda enviar invitaciones a espacios de trabajo, códigos para restablecer la contraseña y códigos de acceso. El servidor de correo se considera configurado <strong>en cuanto se define <code>SMTP_HOST</code></strong>. Sin él, la pantalla de ajustes indica que el servidor de correo no está configurado y la pantalla de inicio de sesión oculta el enlace para restablecer la contraseña.
              </p>

              <p>
                Este es el servidor de correo de la instancia. Los canales de notificación por correo, que informan sobre las copias de seguridad, tienen su propia configuración SMTP en el formulario de cada canal. Por eso, que un canal de notificación entregue correo no dice nada de este servidor. Para comprobar el servidor de correo de la instancia, abra <strong>Databasus settings → Mail server</strong> y pulse <strong>Send test email</strong>. El mensaje llega a su propia dirección y, si la entrega falla, verá la respuesta del servidor de correo.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>SMTP_HOST</code>
                    </td>
                    <td data-label="Description">
                      Nombre de host del servidor de correo (por ejemplo, <code>smtp.gmail.com</code>). Al definirla se activa el correo de la instancia.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PORT</code>
                    </td>
                    <td data-label="Description">
                      Puerto del servidor de correo (por ejemplo, <code>587</code>). Debe ser un entero positivo cuando <code>SMTP_HOST</code> está definido.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_SECURITY</code>
                    </td>
                    <td data-label="Description">
                      Seguridad de la conexión: <code>tls</code>, <code>starttls</code> o <code>none</code>, descritos más abajo. Por defecto es <code>tls</code> en el puerto 465 y <code>starttls</code> en cualquier otro. Con otro valor, la instancia no arranca.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_USER</code>
                    </td>
                    <td data-label="Description">
                      Nombre de usuario para la autenticación SMTP.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_PASSWORD</code>
                    </td>
                    <td data-label="Description">
                      Contraseña para la autenticación SMTP. Para Gmail, use una App Password, no la contraseña de su cuenta.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_FROM</code>
                    </td>
                    <td data-label="Description">
                      Dirección del remitente, sola (<code>noreply@example.com</code>) o con un nombre (<code>Acme Backups &lt;noreply@example.com&gt;</code>). Sin nombre, los destinatarios ven <code>Databasus</code>. Si no está definida, se usa <code>SMTP_USER</code> cuando es una dirección de correo y, si no, <code>noreply@</code> seguido de <code>SMTP_HOST</code>, de modo que un usuario como <code>apikey</code> nunca pasa a ser el remitente. Si el valor no se puede leer como dirección, la instancia no arranca.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_HELO_NAME</code>
                    </td>
                    <td data-label="Description">
                      El nombre con el que la instancia saluda al servidor de correo. Por defecto es el host de <code>DATABASUS_URL</code> y, si no existe, el nombre del equipo. Debe ser un nombre de host o un literal de dirección como <code>[192.0.2.1]</code> o <code>[IPv6:2001:db8::1]</code>; de lo contrario, la instancia no arranca.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>SMTP_INSECURE_SKIP_VERIFY</code>
                    </td>
                    <td data-label="Description">
                      Póngalo en <code>true</code> para omitir la verificación del certificado TLS en los modos <code>tls</code> y <code>starttls</code>. Por defecto es <code>false</code>. Úselo solo con servidores con certificado autofirmado en una red de confianza: desactiva la protección contra ataques de intermediario (man-in-the-middle).
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>DATABASUS_URL</code>
                    </td>
                    <td data-label="Description">
                      URL base pública de su instancia (por ejemplo, <code>https://backup.example.com</code>). Opcional: añade un enlace a los correos de invitación y aporta el nombre de saludo predeterminado. El correo funciona sin ella.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                <code>SMTP_SECURITY</code> elige uno de tres modos. Los canales de notificación por correo ofrecen la misma elección en <strong>Advanced settings</strong>.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Modo</th>
                    <th>Comportamiento</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>tls</code>
                    </td>
                    <td data-label="Comportamiento">
                      La conexión va cifrada desde el primer byte. Lo habitual en el puerto 465.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>starttls</code>
                    </td>
                    <td data-label="Comportamiento">
                      La conexión empieza sin cifrar y se cifra antes de enviar la contraseña o cualquier mensaje. Si el servidor no lo admite, la entrega falla en lugar de continuar sin cifrado. Lo habitual en el puerto 587.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>none</code>
                    </td>
                    <td data-label="Comportamiento">
                      La conexión nunca se cifra. Úselo para un relé que no admite STARTTLS, como un Postfix local en el puerto 25.
                    </td>
                  </tr>
                </tbody>
              </table>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-red-500 rounded-lg px-4 py-4 flex items-start gap-3">
                <svg
                  width="20"
                  height="20"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="2"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  className="text-red-500 mt-0.5 shrink-0"
                >
                  <path d="M10.29 3.86 1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z" />
                  <path d="M12 9v4M12 17h.01" />
                </svg>
                <div>
                  <p className="text-gray-300 my-0!">
                    <code>none</code> envía la contraseña SMTP y todos los mensajes sin cifrar, y cualquiera que esté entre Databasus y el relé puede leerlos. Úselo solo con un relé en el mismo host o en una red privada de confianza.
                  </p>
                </div>
              </div>

              <h2 id="signup-captcha">
                Captcha de registro (Cloudflare Turnstile)
              </h2>

              <p>
                Si su instancia es accesible desde internet, puede poner un
                desafío de{" "}
                <a
                  href="https://www.cloudflare.com/products/turnstile/"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-blue-400 hover:text-blue-300"
                >
                  Cloudflare Turnstile
                </a>{" "}
                en los formularios de registro e inicio de sesión para dejar
                fuera a los bots. Ambas claves salen del panel de Turnstile, y
                el desafío se activa solo cuando las dos están definidas.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    Para bloquear por completo los registros externos, en lugar
                    de solo ponerles un desafío, no necesita captcha alguno:
                    abra <strong>Databasus settings → Allow sign up</strong> en
                    la interfaz y desactívelo. Eso cierra el formulario de
                    registro por completo.
                  </p>
                </div>
              </div>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SITE_KEY</code>
                    </td>
                    <td data-label="Description">
                      Clave pública de sitio de Turnstile, usada para renderizar
                      el widget en el navegador.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>CLOUDFLARE_TURNSTILE_SECRET_KEY</code>
                    </td>
                    <td data-label="Description">
                      Clave secreta de Turnstile, usada por el backend para
                      validar las respuestas al desafío.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="docker-storage-permissions">Permisos del almacenamiento Docker</h2>

              <p>
                Databasus suele tomar los ID numéricos del usuario y del grupo
                del directorio pgdata existente, después del mount de copias de
                seguridad o de la raíz de datos y, como último recurso, usa{" "}
                <code>999</code>. Configure <code>PUID</code> o <code>PGID</code>{" "}
                solo si esa selección automática no sirve para su bind mount,
                recurso CIFS o exportación NFS. Deben ser enteros decimales de{" "}
                <code>1</code> a <code>4294967294</code>.
              </p>

              <table>
                <thead><tr><th>Variable</th><th>Valor automático</th><th>Cuenta</th></tr></thead>
                <tbody>
                  <tr><td><code>PUID</code></td><td>Propietario del mount o <code>999</code></td><td>Usuario <code>databasus</code></td></tr>
                  <tr><td><code>PGID</code></td><td>Grupo del mount o <code>999</code></td><td>Grupo principal <code>databasus</code></td></tr>
                </tbody>
              </table>

              <p>
                Databasus y PostgreSQL usan la misma cuenta del sistema
                operativo sin privilegios root, llamada <code>databasus</code>.
              </p>

              <p>
                El entrypoint comienza como root dentro del contenedor para
                elegir los ID e intentar ejecutar <code>chown</code> y{" "}
                <code>chmod</code>. Después comprueba todo el ciclo de archivos
                como <code>databasus</code>. Un mount puede funcionar aunque
                rechace cambios de propietario o modo. No se admite un valor
                arbitrario de Docker <code>user:</code>.
              </p>

              <pre><code>{`ERROR: Databasus cannot write to local storage paths /databasus-data/temp and /databasus-data/backups as UID 999 and GID 999.
Required operation: save a file through local storage.
Set PUID and PGID or fix the mounted directory permissions: https://databasus.com/advanced-config/#docker-storage-permissions
Details: permission denied`}</code></pre>

              <p>
                Las cuatro variables anteriores para servicios separados se
                eliminaron de forma deliberada. Quítelas de la configuración al
                actualizar. Si los ID elegidos automáticamente no pueden acceder
                a un mount, el inicio se detiene e indica la operación fallida y
                el enlace a esta documentación.
              </p>

              <h2 id="telemetry">Telemetría</h2>

              <p>
                Databasus envía por defecto telemetría de uso anónima y no
                identificable. No contiene datos personales y nos ayuda a
                entender cómo se usa el proyecto. Puede leer exactamente qué se
                recopila en la{" "}
                <a
                  href="/privacy"
                  className="text-blue-400 hover:text-blue-300"
                >
                  política de privacidad
                </a>
                , y puede desactivarla por completo.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Por defecto</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>IS_DISABLE_ANONYMOUS_TELEMETRY</code>
                    </td>
                    <td data-label="Default">
                      <code>false</code>
                    </td>
                    <td data-label="Description">
                      Póngalo en <code>true</code> para desactivar la telemetría
                      de uso anónima.
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="logging">Registros</h2>

              <p>
                Databasus escribe sus registros en stdout y los duplica como
                JSON en <code>databasus.log</code> en el volumen de datos.
                Defina <code>OPEN_TELEMETRY_URL</code> y también los exportará
                por OpenTelemetry a un backend como VictoriaLogs, Graylog,
                SigNoz, Grafana Loki, Datadog o Honeycomb, o a un OpenTelemetry
                Collector, que es a su vez un receptor OTLP.
              </p>

              <ul>
                <li>
                  <strong>El transporte</strong> sigue el esquema.{" "}
                  <code>http://</code> y <code>https://</code> envían OTLP/HTTP
                  y usan la URL literal, ruta incluida; <code>grpc://</code> y{" "}
                  <code>grpcs://</code> envían OTLP/gRPC y usan solo el host y
                  el puerto.
                </li>
                <li>
                  <strong>La autenticación</strong> va en{" "}
                  <code>OPEN_TELEMETRY_HEADERS</code> o en la URL como{" "}
                  <code>user:password@host</code>.
                </li>
                <li>
                  <strong>Los secretos</strong> (contraseñas, tokens,
                  credenciales) dentro de las URL se ocultan antes de que un
                  registro salga del proceso.
                </li>
                <li>
                  <strong>Las entradas de auditoría</strong> se envían con los
                  registros de la aplicación etiquetadas como{" "}
                  <code>log_type=audit</code> e ignoran <code>LOG_LEVEL</code>,
                  así que subir el nivel nunca elimina el rastro de auditoría.
                </li>
              </ul>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Por defecto</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_URL</code>
                    </td>
                    <td data-label="Default">—</td>
                    <td data-label="Description">
                      URL completa del endpoint OTLP, ruta incluida. Déjela sin
                      definir para mantener los registros en el contenedor. Una
                      query string, un host ausente o un esquema desconocido
                      detiene el contenedor en el arranque, en lugar de quedarse
                      sin exportar en silencio.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </td>
                    <td data-label="Default">—</td>
                    <td data-label="Description">
                      Pares <code>key=value</code> separados por comas que se
                      envían con cada exportación, normalmente una clave de API.
                      Los valores usan codificación porcentual, como en el
                      formato estándar de{" "}
                      <code>OTEL_EXPORTER_OTLP_HEADERS</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_LEVEL</code>
                    </td>
                    <td data-label="Default">
                      <code>info</code>
                    </td>
                    <td data-label="Description">
                      Uno de <code>debug</code>, <code>info</code>,{" "}
                      <code>warn</code> o <code>error</code>. Un valor no
                      reconocido vuelve a <code>info</code>.
                    </td>
                  </tr>
                  <tr>
                    <td>
                      <code>LOG_FILE_IS_ENABLED</code>
                    </td>
                    <td data-label="Default">
                      <code>true</code>
                    </td>
                    <td data-label="Description">
                      Escribe <code>databasus.log</code> junto al resto de los
                      datos, con rotación a los 5 MB y conservando 3 archivos
                      antiguos. Póngalo en <code>false</code> si su plataforma
                      ya recolecta stdout.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Valores para los backends habituales, cada uno con la cabecera
                que lo autentica. Reemplace hosts, regiones y claves por los
                suyos:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Backend</th>
                    <th>
                      <code>OPEN_TELEMETRY_URL</code>
                    </th>
                    <th>
                      <code>OPEN_TELEMETRY_HEADERS</code>
                    </th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>VictoriaLogs</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>
                        http://victoria-logs:9428/insert/opentelemetry/v1/logs
                      </code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Basic%20dXNlcjpwYXNzd29yZA==</code> —
                      las credenciales que espera su <code>vmauth</code> o
                      reverse proxy, ya que VictoriaLogs no tiene autenticación
                      propia en la ruta de ingesta.
                    </td>
                  </tr>
                  <tr>
                    <td>OpenTelemetry Collector</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://otel-collector:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> —
                      corresponde a una extensión <code>bearertokenauth</code> o{" "}
                      <code>basicauth</code> en el receptor. Un Collector
                      accesible solo dentro de su red normalmente no necesita
                      ninguna.
                    </td>
                  </tr>
                  <tr>
                    <td>Graylog 6.2+</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://graylog:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Bearer%20your-token</code> — el token
                      definido en la entrada OpenTelemetry (gRPC). La entrada
                      también acepta mTLS en su lugar.
                    </td>
                  </tr>
                  <tr>
                    <td>SigNoz Cloud</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpcs://ingest.eu.signoz.cloud:443</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>signoz-ingestion-key=your-ingestion-key</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Grafana Cloud</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>
                        https://otlp-gateway-prod-eu-west-0.grafana.net/otlp/v1/logs
                      </code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>Authorization=Basic%20&lt;base64&gt;</code> — base64
                      de <code>instance-id:api-token</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Honeycomb</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>https://api.honeycomb.io/v1/logs</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      <code>x-honeycomb-team=your-api-key</code>
                    </td>
                  </tr>
                  <tr>
                    <td>Datadog Agent</td>
                    <td data-label="OPEN_TELEMETRY_URL">
                      <code>grpc://datadog-agent:4317</code>
                    </td>
                    <td data-label="OPEN_TELEMETRY_HEADERS">
                      Ninguna: el Agent guarda la clave de API y reenvía los
                      registros en su nombre.
                    </td>
                  </tr>
                </tbody>
              </table>

              <p>
                Los valores de las cabeceras usan codificación porcentual, así
                que el espacio después de <code>Basic</code> o{" "}
                <code>Bearer</code> se escribe como <code>%20</code> y una coma
                dentro de un valor como <code>%2C</code>. La autenticación
                básica también puede ir directamente en la URL como{" "}
                <code>https://user:password@host/path</code>: Databasus la
                convierte en la misma cabecera y la mantiene fuera de los
                registros. Por <code>http://</code> y <code>grpc://</code> las
                claves y contraseñas viajan en claro, así que use{" "}
                <code>https://</code> o <code>grpcs://</code> fuera de una red
                de confianza.
              </p>

              <h2 id="analytics-script">Script de analítica</h2>

              <p>
                Databasus puede inyectar en la aplicación su propio fragmento de
                analítica o seguimiento: Google Analytics, Plausible, Umami y
                similares. Cuando <code>ANALYTICS_SCRIPT</code> está definida,
                su valor se inserta en el <code>&lt;head&gt;</code> de la página
                al arrancar.
              </p>

              <p>
                <strong>Advertencia de seguridad:</strong> el valor se inyecta
                tal cual como HTML y JavaScript sin procesar, y se ejecuta con
                acceso completo a la interfaz de Databasus en el navegador de
                cada visitante. Establézcalo únicamente con un fragmento que
                controle y en el que confíe plenamente.
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Variable</th>
                    <th>Descripción</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>
                      <code>ANALYTICS_SCRIPT</code>
                    </td>
                    <td data-label="Description">
                      Marcado <code>&lt;script&gt;</code> personalizado que se
                      inyecta antes de la etiqueta de cierre{" "}
                      <code>&lt;/head&gt;</code>. Déjela sin definir para no
                      añadir analítica.
                    </td>
                  </tr>
                </tbody>
              </table>
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
