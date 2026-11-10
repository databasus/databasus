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
                de notificadores y almacenamientos (S3, Azure Blob Storage, FTP,
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
                almacenamiento en la nube (S3, Azure Blob Storage, etc.), no podrá
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
                Databasus maneja datos sensibles. Antes de cada versión, las comprobaciones
                automáticas buscan problemas de seguridad y verifican que las copias se
                puedan restaurar. Estas comprobaciones reducen el riesgo, pero no garantizan
                la ausencia de vulnerabilidades.
              </p>

              <h3 id="static-analysis">Análisis estático</h3>

              <p>
                CodeQL analiza Go, JavaScript/TypeScript y los flujos de GitHub Actions.
                gitleaks busca secretos expuestos en el historial de Git de la versión.
                semgrep comprueba el código fuente, los Dockerfiles y los flujos de CI con
                reglas de seguridad. Estos análisis se ejecutan antes de cada versión y
                semanalmente.
              </p>

              <p>
                Los hallazgos de CodeQL de nivel <strong>CRITICAL</strong> abiertos en el
                análisis de la versión bloquean su publicación. Los de nivel{" "}
                <strong>HIGH</strong> generan advertencias.
              </p>

              <p>
                Codex Security de OpenAI también realiza auditorías periódicas más
                profundas. Pueden detectar problemas de arquitectura que los análisis más
                específicos pasen por alto. Se realizan por separado de las comprobaciones
                automáticas de publicación.
              </p>

              <h3 id="dependency-management">Gestión de dependencias</h3>

              <p>
                Las actualizaciones ordinarias esperan siete días completos para paquetes,
                herramientas de desarrollo y GitHub Actions. El plazo es de catorce días
                para imágenes, clientes de bases de datos y extensiones de verificación
                independientes. Los responsables revisan las fechas de publicación y
                la compatibilidad, incluidas las versiones exactas y las actualizaciones de
                seguridad que pueden saltarse los filtros de antigüedad. Toda excepción se
                comunica con su motivo.
              </p>

              <p>
                Dependency Review Action compara la versión candidata con la anterior y
                bloquea las nuevas vulnerabilidades de nivel <strong>HIGH</strong> o{" "}
                <strong>CRITICAL</strong>. govulncheck bloquea la comprobación de Go cuando
                nuestro código puede llamar a una función con una vulnerabilidad conocida.
                El comando <code>go mod verify</code> contrasta los módulos Go descargados
                con sus sumas de verificación registradas.
              </p>

              <p>
                Antes de publicar una versión o imágenes de desarrollo,{" "}
                <code>mise run verify-deps</code> comprueba las versiones fijas de
                herramientas, imágenes y GitHub Actions. También verifica que las versiones
                relacionadas de los entornos de ejecución coincidan y comprueba los hashes y
                permisos de ejecución de los clientes de bases de datos incluidos en el
                repositorio. Estas comprobaciones cubren una lista definida de declaraciones
                y archivos locales. Los gestores de paquetes verifican por separado las
                versiones de los archivos de bloqueo y las sumas de verificación de los
                paquetes descargados.
              </p>

              <p>
                La comprobación local de dependencias no certifica la seguridad del código
                de terceros ni impone una antigüedad mínima a cada dependencia. La selección
                y revisión de versiones siguen siendo responsabilidad de los responsables
                del proyecto.
              </p>

              <h3 id="container-and-ci-hardening">Seguridad de contenedores y CI</h3>

              <ul>
                <li>
                  Trivy analiza la imagen de la aplicación en amd64 y arm64. Las
                  vulnerabilidades <strong>CRITICAL</strong> con una corrección disponible
                  hacen fallar la comprobación. Las de nivel <strong>HIGH</strong> con
                  corrección se notifican sin bloquearla. Las vulnerabilidades sin corrección
                  se excluyen de esta comprobación.
                </li>
                <li>
                  Un análisis de configuración independiente de Trivy bloquea hallazgos{" "}
                  <strong>HIGH</strong> y <strong>CRITICAL</strong> en los Dockerfiles de la
                  aplicación, del agente de verificación y de determinadas imágenes de prueba.
                </li>
                <li>
                  Las GitHub Actions de terceros se fijan a SHA completos de commit. Así,
                  cambiar un tag no puede sustituir silenciosamente el código de la acción
                  ejecutada.
                </li>
                <li>
                  Los flujos tienen acceso de solo lectura al repositorio por defecto. Cada
                  tarea recibe permisos adicionales únicamente cuando los necesita.
                </li>
              </ul>

              <h3 id="testing-and-verification">Pruebas y verificación</h3>

              <p>
                Las pruebas unitarias y de integración cubren el comportamiento crítico.
                Antes de cada versión, se ejecutan ciclos completos de copia de seguridad y
                restauración en contenedores reales para cada motor y versión principal
                compatibles. Verifican que los datos se puedan restaurar, no solo que se
                haya creado un archivo.
              </p>

              <p>
                Las comprobaciones también incluyen análisis de estilo, comprobación de
                tipos, pruebas en navegador, pruebas básicas de imágenes y compilaciones y
                pruebas nativas en amd64 y arm64. La publicación espera a que pasen todas
                las tareas obligatorias, incluidas las comprobaciones de seguridad descritas
                arriba.
              </p>

              <h3 id="reporting-a-vulnerability">Cómo comunicar una vulnerabilidad</h3>

              <p>
                Comunique las vulnerabilidades de forma privada mediante las opciones de
                contacto de{" "}
                <a
                  href="https://github.com/databasus/databasus?tab=security-ov-file#readme"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  SECURITY.md
                </a>
                . No abra un issue público. Los informes de seguridad tienen prioridad.
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
