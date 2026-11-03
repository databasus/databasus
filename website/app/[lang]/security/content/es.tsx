import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "Seguridad - Cómo protege Databasus sus datos | Databasus",
  description:
    "Descubra cómo Databasus garantiza seguridad de nivel empresarial con cifrado AES-256-GCM para los datos sensibles y las copias de seguridad, acceso de solo lectura a la base de datos y registro completo de auditoría.",
  keywords: [
    "seguridad de Databasus",
    "seguridad de copia de seguridad de PostgreSQL",
    "cifrado AES-256-GCM",
    "cifrado de bases de datos",
    "cifrado de copias de seguridad",
    "acceso de solo lectura a la base de datos",
    "seguridad empresarial",
    "protección de datos",
    "respaldos seguros",
  ],
  openGraph: {
    title: "Seguridad - Cómo protege Databasus sus datos | Databasus",
    description:
      "Descubra cómo Databasus garantiza seguridad de nivel empresarial con cifrado AES-256-GCM para los datos sensibles y las copias de seguridad, acceso de solo lectura a la base de datos y registro completo de auditoría.",
    type: "article",
    url: getLocalizedUrl("es", "security"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Seguridad - Cómo protege Databasus sus datos | Databasus",
    description:
      "Descubra cómo Databasus garantiza seguridad de nivel empresarial con cifrado AES-256-GCM para los datos sensibles y las copias de seguridad, acceso de solo lectura a la base de datos y registro completo de auditoría.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "security"),
    languages: getLanguageAlternates("security"),
  },
  robots: "index, follow",
};

export default function SecurityPage() {
  const encryptionPipeline = `PostgreSQL pg_dump → Compression → Encryption → Cloud Storage`;
  const disableTwoFactorCommand = `docker exec -it databasus ./main --disable-2fa`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Seguridad - Cómo protege Databasus sus datos",
            description:
              "Descubra cómo Databasus garantiza seguridad de nivel empresarial con cifrado AES-256-GCM para los datos sensibles y las copias de seguridad, acceso de solo lectura a la base de datos y registro completo de auditoría.",
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
              <h1 id="security">¿Cómo garantiza Databasus la seguridad?</h1>

              <p className="text-lg text-gray-400">
                Databasus es responsable de datos sensibles:
              </p>

              <ul>
                <li>accede a su base de datos;</li>
                <li>la respalda (es decir, hace una copia de los datos);</li>
                <li>
                  guarda credenciales para poder acceder a su base de datos de
                  forma regular;
                </li>
                <li>
                  guarda las copias de seguridad en su S3 u otros
                  almacenamientos en la nube (si lo activa);
                </li>
              </ul>

              <p>
                Por lo tanto,{" "}
                <strong>
                  la prioridad principal de Databasus es ser seguro y fiable a
                  nivel empresarial
                </strong>
                .
              </p>

              <p>Databasus garantiza que:</p>

              <ul>
                <li>
                  los datos sensibles nunca se exponen y siempre están cifrados;
                </li>
                <li>
                  las copias de seguridad están cifradas y son inútiles aunque
                  alguien las vea en el almacenamiento en la nube;
                </li>
                <li>
                  Databasus ni siquiera recibe acceso de escritura o
                  actualización a la base de datos;
                </li>
                <li>todas las acciones se registran y pueden auditarse;</li>
              </ul>

              <p>
                Todos estos pasos protegen sus datos. Ningún sistema es 100%
                seguro, pero hacemos todo lo posible por acercarnos: incluso en
                caso de un ataque, nadie podrá corromper sus datos.
              </p>

              <p>Databasus aplica la seguridad en tres niveles:</p>

              <ol>
                <li>Cifrado de los datos sensibles;</li>
                <li>Cifrado de las copias de seguridad;</li>
                <li>Acceso de solo lectura a la base de datos.</li>
              </ol>

              <h2 id="level-1-sensitive-data-encryption">
                Nivel 1: cifrado de los datos sensibles
              </h2>

              <p>
                Internamente, Databasus usa una base de datos PostgreSQL para
                guardar los datos de conexión, las configuraciones y los ajustes
                de notificadores y almacenamientos (S3, Google Drive, Dropbox,
                etc.).
              </p>

              <p>Todo dato sensible se cifra. Por ejemplo:</p>

              <ul>
                <li>contraseñas</li>
                <li>tokens</li>
                <li>webhooks con secretos</li>
              </ul>

              <p>
                Así, en la base de datos Databasus solo guarda hashes o valores
                cifrados. Para el cifrado se usa el algoritmo{" "}
                <strong>AES-256-GCM</strong>. Además, a pesar del cifrado, esos
                valores nunca se exponen a través de la API ni de la interfaz.
              </p>

              <p>
                La clave secreta usada para el cifrado se guarda en el
                almacenamiento local (<code>./databasus-data/secret.key</code>{" "}
                por defecto) y no está presente en la propia base de datos. Así,
                comprometer la base de datos no da acceso a los datos sensibles.
              </p>

              <h2 id="level-2-backups-encryption">
                Nivel 2: cifrado de las copias de seguridad
              </h2>

              <p>
                Cada archivo de copia de seguridad se cifra al vuelo durante su
                creación. Databasus usa el algoritmo de cifrado{" "}
                <strong>AES-256-GCM</strong>, que garantiza que los datos de la
                copia no puedan leerse sin la clave de cifrado y que cualquier
                manipulación se detecte durante el descifrado.
              </p>

              <p>Las copias pasan por esta canalización:</p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{encryptionPipeline}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={encryptionPipeline} />
                </div>
              </div>

              <p>
                Cada copia recibe su propia clave de cifrado única derivada de:
              </p>

              <ul>
                <li>
                  La clave maestra (guardada en{" "}
                  <code>./databasus-data/secret.key</code>)
                </li>
                <li>El ID de la copia</li>
                <li>Una sal aleatoria (única por copia)</li>
              </ul>

              <p>
                <strong>Resultado</strong>: aunque alguien obtenga acceso a su
                almacenamiento en la nube (S3, Google Drive, etc.), no podrá
                leer las copias sin su clave maestra.
              </p>

              <h2 id="level-3-read-only-access">
                Nivel 3: acceso de solo lectura a la base de datos
              </h2>

              <p>
                Databasus aplica el principio de mínimo privilegio: solo
                necesita acceso de lectura para crear copias de seguridad, nunca
                de escritura. Esto protege su base de datos de la corrupción de
                datos, accidental o malintencionada, a través de la herramienta
                de respaldo.
              </p>

              <p>
                Antes de aceptar las credenciales de la base de datos, Databasus
                realiza comprobaciones en tres niveles:
              </p>

              <ol>
                <li>
                  <strong>Nivel de rol</strong>: verifica que el usuario NO es
                  superusuario y no puede crear roles ni bases de datos
                </li>
                <li>
                  <strong>Nivel de base de datos</strong>: se asegura de que no
                  hay privilegios CREATE ni TEMP
                </li>
                <li>
                  <strong>Nivel de tabla</strong>: confirma que no hay ningún
                  permiso de escritura (INSERT, UPDATE, DELETE, TRUNCATE, etc.)
                </li>
              </ol>

              <p>
                El usuario de la base de datos debe pasar las tres
                comprobaciones para considerarse de solo lectura. Si se detecta
                cualquier privilegio de escritura, Databasus le avisará.
              </p>

              <p>
                Databasus le sugiere crear usuarios de solo lectura con los
                permisos adecuados:
              </p>

              <ul>
                <li>
                  Concede SELECT sobre todas las tablas actuales y futuras
                </li>
                <li>Concede USAGE sobre los esquemas (pero no CREATE)</li>
                <li>
                  Revoca explícitamente todos los privilegios de escritura
                </li>
              </ul>

              <p>
                <strong>Resultado</strong>: aunque Databasus se vea
                comprometido, el servidor sea atacado, la clave secreta sea
                robada y las credenciales sean descifradas, los atacantes no
                podrán corromper su base de datos.
              </p>

              <h2 id="two-factor-authentication">
                Autenticación de dos factores al iniciar sesión
              </h2>

              <p>
                Una contraseña por sí sola abre todas las credenciales de bases
                de datos, las claves de almacenamiento y las copias de seguridad
                que guarda la instancia, así que un administrador puede exigir
                un segundo factor al iniciar sesión con contraseña. Con la
                opción activada, una contraseña correcta envía un código de seis
                dígitos al correo de la cuenta y el inicio de sesión termina
                solo cuando se introduce ese código. El código deja de funcionar
                diez minutos después de enviarse o tras cinco intentos fallidos,
                y pedir otro anula el anterior. Activar el ajuste protege el
                siguiente inicio de sesión y deja funcionando las sesiones ya
                abiertas.
              </p>

              <p>
                El ajuste cubre el inicio de sesión con contraseña. Entrar con
                Google o GitHub sigue entregando un token directamente, porque
                esos proveedores hacen sus propias comprobaciones de varios
                factores: una instancia que quiera el segundo factor para todo
                el mundo simplemente no los configura.
              </p>

              <p>
                Solo se puede activar cuando la instancia tiene servidor de
                correo y cada administrador activo lleva una dirección real, de
                modo que nadie lo active y deje la instancia cerrada. Si más
                tarde el correo deja de entregarse, nadie entra con contraseña,
                y eso es deliberado: un servidor de correo averiado no debe
                rebajar en silencio la autenticación a un solo factor. Un
                operador con acceso al servidor vuelve a desactivarlo:
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
                El comando informa de lo que ha cambiado y lo registra en el
                historial de auditoría, así que una instancia que dejó de pedir
                códigos explica por qué. Está documentado junto a la
                recuperación de contraseña en{" "}
                <a
                  href="/es/password#disable-two-factor"
                  className="text-blue-400 hover:text-blue-600"
                >
                  la página de contraseña
                </a>
                .
              </p>

              <h2 id="security-and-reliability-engineering">
                🛡️ Ingeniería de seguridad y fiabilidad
              </h2>

              <p>
                Databasus trabaja con datos sensibles, así que prevenir
                vulnerabilidades, accesos no autorizados y fugas de datos es una
                preocupación primordial. Invertimos en ello en ambos lados del
                sistema: en el propio código (comprobaciones de permisos,
                cifrado, manejo cuidadoso de los secretos) y en la
                infraestructura que lo rodea (análisis de dependencias,
                respuesta a CVE, prácticas DevSecOps). La canalización que se
                describe a continuación se ejecuta automáticamente en cada
                commit y PR: ninguna capa basta por sí sola, pero juntas reducen
                la probabilidad de que código vulnerable, dependencias
                inseguras, imágenes rotas o copias no restaurables lleguen a una
                versión publicada.
              </p>

              <h3 id="static-analysis">Análisis estático</h3>

              <p>
                El análisis estático se ejecuta en varias pasadas
                independientes. CodeQL escanea todo el código en busca de
                problemas de seguridad. En cada PR, <strong>gitleaks</strong> busca
                secretos filtrados y <strong>semgrep</strong> revisa el código con
                reglas de seguridad. Los
                Dockerfiles y los flujos de CI tienen reglas adicionales propias
                (referencias de acciones fijadas, permisos de mínimo privilegio,
                imágenes base sospechosas), de modo que los patrones inseguros
                se señalan antes de fusionarse.
              </p>

              <p>
                Además de estas comprobaciones por PR,{" "}
                <strong>Codex Security</strong> de OpenAI realiza auditorías
                periódicas y más profundas de todo el código. Es un programa
                aparte que detecta problemas arquitectónicos y transversales que
                los escaneos limitados al momento del PR pasan por alto.
              </p>

              <h3 id="dependency-management">Gestión de dependencias</h3>

              <p>
                Dependabot vigila todas nuestras dependencias contra la GitHub
                Advisory Database y detecta los CVE a los pocos minutos de su
                publicación. Las actualizaciones pasan por un periodo de espera
                para que las versiones recién publicadas maduren antes de
                adoptarlas, una defensa deliberada contra incidentes de paquetes
                comprometidos como los ataques a la cadena de suministro.
              </p>

              <p>
                La <strong>Dependency Review Action</strong> hace fallar la
                comprobación de cualquier PR que introduzca un CVE nuevo de
                nivel{" "}
                <strong>HIGH</strong> o <strong>CRITICAL</strong>.
              </p>

              <h3 id="container-and-ci-hardening">
                Endurecimiento de contenedores y CI
              </h3>

              <ul>
                <li>
                  La imagen de la aplicación se escanea con <strong>Trivy</strong> en
                  ambas arquitecturas antes de cada versión.
                </li>
                <li>
                  Una pasada aparte de Trivy sobre el Dockerfile detecta
                  configuraciones incorrectas antes de que lleguen a una imagen.
                </li>
                <li>
                  Todas las GitHub Actions están fijadas a SHA de commit
                  completos en lugar de etiquetas flotantes como{" "}
                  <code>@v4</code> o <code>@main</code>, que han sido un vector
                  de ataque activo en 2025.
                </li>
                <li>
                  Los flujos de trabajo usan por defecto permisos de mínimo
                  privilegio y solo los elevan por trabajo cuando es realmente
                  necesario.
                </li>
              </ul>

              <h3 id="testing-and-verification">Pruebas y verificación</h3>

              <p>
                Las rutas críticas están cubiertas por pruebas unitarias y de
                integración, ejecutadas contra contenedores de bases de datos
                reales para cada motor y versión mayor compatibles.
              </p>

              <p>
                La restauración es la ruta que más importa en una herramienta de
                respaldo, así que la probamos explícitamente: cada PR ejecuta
                ciclos completos de copia y restauración contra esos mismos
                contenedores reales, verificando que las copias realmente pueden
                restaurarse de extremo a extremo, no solo escribirse con éxito.
              </p>

              <p>
                El resto de la canalización de CI/CD ejecuta lint, comprobación
                de tipos, la suite de pruebas completa, pruebas de humo de las
                imágenes y builds multiarquitectura en cada PR. Una versión se
                publica solo cuando todo esto y los escaneos de seguridad pasan
                y no queda ninguna alerta crítica de CodeQL abierta.
              </p>

              <h3 id="reporting-a-vulnerability">
                Informar de una vulnerabilidad
              </h3>

              <p>
                ¿Ha encontrado una vulnerabilidad? Infórmela a través de la
                pestaña Security de GitHub; consulte{" "}
                <a
                  href="https://github.com/databasus/databasus?tab=security-ov-file#readme"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  SECURITY.md
                </a>
                . Los informes de seguridad se atienden con la máxima prioridad.
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
