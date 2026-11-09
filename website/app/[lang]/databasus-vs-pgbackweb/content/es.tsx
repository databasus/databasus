import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Databasus vs PgBackWeb: comparación de herramientas de copia de seguridad de PostgreSQL",
  description:
    "Compare Databasus y PgBackWeb como herramientas de copia de seguridad de PostgreSQL. Vea las diferencias en funciones, seguridad, soporte de equipos, opciones de almacenamiento, notificaciones y facilidad de uso.",
  keywords: [
    "Databasus vs PgBackWeb",
    "comparación de copias de seguridad de PostgreSQL",
    "alternativa a PgBackWeb",
    "herramientas de copia de seguridad de PostgreSQL",
    "comparación de respaldos de bases de datos",
    "interfaz gráfica para pg_dump",
    "respaldo autoalojado",
    "seguridad de copias de seguridad de PostgreSQL",
  ],
  openGraph: {
    title:
      "Databasus vs PgBackWeb: comparación de herramientas de copia de seguridad de PostgreSQL",
    description:
      "Compare Databasus y PgBackWeb como herramientas de copia de seguridad de PostgreSQL. Vea las diferencias en funciones, seguridad, soporte de equipos, opciones de almacenamiento, notificaciones y facilidad de uso.",
    type: "article",
    url: getLocalizedUrl("es", "databasus-vs-pgbackweb"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs PgBackWeb: comparación de herramientas de copia de seguridad de PostgreSQL",
    description:
      "Compare Databasus y PgBackWeb como herramientas de copia de seguridad de PostgreSQL. Vea las diferencias en funciones, seguridad, soporte de equipos, opciones de almacenamiento, notificaciones y facilidad de uso.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "databasus-vs-pgbackweb"),
    languages: getLanguageAlternates("databasus-vs-pgbackweb"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackWebPage() {
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
              "Databasus vs PgBackWeb: comparación de herramientas de copia de seguridad de PostgreSQL",
            description:
              "Una comparación completa de Databasus y PgBackWeb como herramientas de copia de seguridad de PostgreSQL: funciones, seguridad, soporte de equipos, opciones de almacenamiento y facilidad de uso.",
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
              <h1 id="databasus-vs-pgbackweb">Databasus vs PgBackWeb</h1>

              <p className="text-lg text-gray-400">
                Databasus y PgBackWeb son herramientas de código abierto
                diseñadas para simplificar la gestión de copias de seguridad de
                PostgreSQL mediante interfaces web. Aunque comparten el objetivo
                común de hacer los respaldos más accesibles, difieren
                significativamente en funciones, seguridad, soporte de equipos y
                facilidad de uso.
              </p>

              <h2 id="quick-comparison">Comparación rápida</h2>

              <p>
                Este es un resumen rápido de las diferencias clave entre
                Databasus y PgBackWeb:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Función</th>
                    <th>Databasus</th>
                    <th>PgBackWeb</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Licencia</td>
                    <td data-label="Databasus">Apache 2.0</td>
                    <td data-label="PgBackWeb">AGPL-3.0</td>
                  </tr>
                  <tr>
                    <td>Gestión de respaldos</td>
                    <td data-label="Databasus">✅ Múltiples BD</td>
                    <td data-label="PgBackWeb">✅ Múltiples BD</td>
                  </tr>
                  <tr>
                    <td>Soporte de otras BD</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="PgBackWeb">❌ Solo PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>Opciones de almacenamiento</td>
                    <td data-label="Databasus">
                      Local, S3, Cloudflare R2, Azure, NAS,
                      FTP
                    </td>
                    <td data-label="PgBackWeb">
                      Solo local y compatible con S3
                    </td>
                  </tr>
                  <tr>
                    <td>Notificaciones</td>
                    <td data-label="Databasus">
                      Slack, Discord, Telegram, Teams, correo electrónico,
                      webhooks
                    </td>
                    <td data-label="PgBackWeb">Solo webhooks</td>
                  </tr>
                  <tr>
                    <td>Seguridad</td>
                    <td data-label="Databasus">
                      ✅ AES-256-GCM, claves únicas por respaldo, exigencia de
                      solo lectura
                    </td>
                    <td data-label="PgBackWeb">✅ Cifrado PGP</td>
                  </tr>
                  <tr>
                    <td>Funciones de equipo</td>
                    <td data-label="Databasus">
                      ✅ Espacios de trabajo, acceso basado en roles, registros
                      de auditoría
                    </td>
                    <td data-label="PgBackWeb">❌ No disponible</td>
                  </tr>
                  <tr>
                    <td>Monitoreo de salud</td>
                    <td data-label="Databasus">✅ Integrado</td>
                    <td data-label="PgBackWeb">❌ No disponible</td>
                  </tr>
                  <tr>
                    <td>Instalación</td>
                    <td data-label="Databasus">
                      Script de una línea, Docker o Helm
                    </td>
                    <td data-label="PgBackWeb">
                      Configuración manual de Docker
                    </td>
                  </tr>
                  <tr>
                    <td>Respaldos físicos</td>
                    <td data-label="Databasus">✅ Sí</td>
                    <td data-label="PgBackWeb">❌ No disponible</td>
                  </tr>
                  <tr>
                    <td>Respaldos incrementales</td>
                    <td data-label="Databasus">
                      ✅ A nivel de bloque (PG 17+)
                    </td>
                    <td data-label="PgBackWeb">❌ No disponible</td>
                  </tr>
                  <tr>
                    <td>Archivado WAL</td>
                    <td data-label="Databasus">✅ Streaming continuo</td>
                    <td data-label="PgBackWeb">❌ No disponible</td>
                  </tr>
                  <tr>
                    <td>Recuperación a un punto en el tiempo</td>
                    <td data-label="Databasus">✅ Sí</td>
                    <td data-label="PgBackWeb">❌ No disponible</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="backup-features">Funciones de respaldo</h2>

              <p>
                Ambas herramientas admiten respaldos programados con horarios
                flexibles:
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong>: Admite programaciones horarias,
                  diarias, semanales, mensuales o con cron, con horarios
                  precisos (por ejemplo, a las 4 AM). Implementa{" "}
                  <strong>compresión equilibrada con zstd (nivel 5)</strong>:
                  los respaldos ocupan entre 4 y 8 veces menos con solo ~20% de
                  sobrecarga en tiempo de ejecución. Es significativamente más
                  eficiente que gzip.
                </li>
                <li>
                  <strong>PgBackWeb</strong>: Admite la programación basada en
                  cron para la ejecución de respaldos. Usa compresión gzip para
                  los respaldos, que es más lenta y menos eficiente que zstd.
                </li>
              </ul>

              <p>
                Más allá de los respaldos lógicos, Databasus también admite
                respaldos físicos, incrementales y de WAL. Están construidos
                sobre la pila de respaldo nativa de PostgreSQL 17 y se ejecutan
                de forma remota, así que no se instala nada en el servidor de
                base de datos y a las redes cerradas se llega mediante un túnel
                SSH. Esto le da respaldos incrementales a nivel de bloque,
                streaming continuo de WAL y recuperación a un punto en el tiempo
                para la recuperación ante desastres con pérdida de datos casi
                nula: puede restaurar a cualquier segundo entre respaldos.
                PgBackWeb no ofrece nada de esto.
              </p>

              <h2 id="storage-options">Opciones de almacenamiento</h2>

              <p>
                La flexibilidad de almacenamiento es crucial en las estrategias
                de respaldo. Así se comparan las dos herramientas:
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong>: Admite una amplia gama de destinos
                  de almacenamiento:
                  <ul>
                    <li>Almacenamiento local</li>
                    <li>Amazon S3 y servicios compatibles con S3</li>
                    <li>Cloudflare R2</li>
                    <li>Azure Blob Storage</li>
                    <li>NAS (almacenamiento conectado en red)</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>: Limitado únicamente a
                  almacenamiento local y compatible con S3.
                </li>
              </ul>

              <p>
                <a
                  href="/es/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Ver todas las opciones de almacenamiento de Databasus →
                </a>
              </p>

              <h2 id="security">Seguridad</h2>

              <p>
                La seguridad es un aspecto crítico de la gestión de respaldos.
                Databasus implementa seguridad de nivel empresarial en tres
                niveles:
              </p>

              <h3 id="security-databasus">Modelo de seguridad de Databasus</h3>

              <ol>
                <li>
                  <strong>Cifrado de datos sensibles</strong>: Todas las
                  contraseñas, tokens y credenciales se cifran con AES-256-GCM.
                  La clave de cifrado se almacena separada de la base de datos,
                  de modo que aunque la base de datos se vea comprometida, los
                  datos sensibles permanecen protegidos.
                </li>
                <li>
                  <strong>Cifrado de respaldos</strong>: Cada archivo de
                  respaldo se cifra con una clave única derivada de la clave
                  maestra, el ID del respaldo y una sal aleatoria. Aunque
                  alguien acceda a su almacenamiento en la nube, no puede leer
                  los respaldos sin su clave de cifrado.
                </li>
                <li>
                  <strong>Acceso de solo lectura a la base de datos</strong>:
                  Databasus exige acceso de solo lectura comprobando los
                  permisos a nivel de rol, de base de datos y de tabla. Solo
                  requiere permisos SELECT y le advertirá si detecta privilegios
                  de escritura. Esto evita la corrupción de datos incluso si
                  Databasus se ve comprometido.
                </li>
              </ol>

              <h3 id="security-pgbackweb">Modelo de seguridad de PgBackWeb</h3>

              <ul>
                <li>
                  <strong>Cifrado PGP</strong>: PgBackWeb ofrece cifrado PGP
                  para los archivos de respaldo.
                </li>
                <li>
                  <strong>Sin exigencia de solo lectura</strong>: PgBackWeb no
                  exige ni verifica el acceso de solo lectura a la base de
                  datos, lo que significa que los respaldos pueden crearse con
                  usuarios que tienen permisos de escritura.
                </li>
              </ul>

              <p>
                <a
                  href="/es/security"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Más información sobre la seguridad de Databasus →
                </a>
              </p>

              <h2 id="notifications">Notificaciones</h2>

              <p>
                Mantenerse informado del estado de los respaldos es esencial
                para una operación fiable:
              </p>

              <ul>
                <li>
                  <strong>Databasus</strong>: Proporciona notificaciones en
                  tiempo real a través de múltiples canales:
                  <ul>
                    <li>Slack</li>
                    <li>Discord</li>
                    <li>Telegram</li>
                    <li>Microsoft Teams</li>
                    <li>Correo electrónico</li>
                    <li>Webhooks</li>
                  </ul>
                </li>
                <li>
                  <strong>PgBackWeb</strong>: Solo admite webhooks para las
                  notificaciones. Para recibir alertas por Slack, Telegram u
                  otras plataformas necesita configurar middleware o servicios
                  adicionales.
                </li>
              </ul>

              <p>
                <a
                  href="/es/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Ver todos los canales de notificación de Databasus →
                </a>
              </p>

              <h2 id="team-features">Funciones de equipo</h2>

              <p>
                Para organizaciones y equipos de DevOps, las funciones de
                colaboración son esenciales. Aquí es donde Databasus supera
                claramente a PgBackWeb:
              </p>

              <h3 id="team-databasus">Capacidades de equipo de Databasus</h3>

              <ul>
                <li>
                  <strong>Espacios de trabajo</strong>: Agrupe bases de datos,
                  notificadores y almacenamientos para distintos proyectos o
                  equipos. Los usuarios solo ven los espacios de trabajo a los
                  que han sido invitados.
                </li>
                <li>
                  <strong>Control de acceso basado en roles</strong>: Niveles de
                  permiso para controlar lo que puede hacer cada miembro del
                  equipo dentro de los espacios de trabajo.
                </li>
                <li>
                  <strong>Registros de auditoría</strong>: Rastree todas las
                  actividades del sistema y los cambios realizados por los
                  usuarios. Esencial para el cumplimiento de seguridad y la
                  rendición de cuentas del equipo.
                </li>
              </ul>

              <h3 id="team-pgbackweb">Capacidades de equipo de PgBackWeb</h3>

              <p>
                PgBackWeb no tiene gestión de usuarios, espacios de trabajo ni
                registros de auditoría integrados. Está diseñado principalmente
                para escenarios de usuario único.
              </p>

              <p>
                <a
                  href="/es/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Más información sobre la gestión de accesos de Databasus →
                </a>
              </p>

              <h2 id="ease-of-use">Facilidad de uso</h2>

              <p>
                <strong>
                  Databasus está diseñado para ser significativamente más fácil
                  de usar
                </strong>{" "}
                que PgBackWeb, con un enfoque en una experiencia intuitiva y un
                tiempo de configuración mínimo:
              </p>

              <h3 id="ease-databasus">Experiencia de usuario de Databasus</h3>

              <ul>
                <li>
                  <strong>Instalación sencilla</strong>: Use Docker directamente
                  o ejecute un script de una línea que instala Docker (si hace
                  falta), configura Databasus y activa el arranque automático.
                  Tiempo total: ~2 minutos.
                </li>
                <li>
                  <strong>Interfaz web intuitiva</strong>: Una interfaz pulida
                  por diseñadores que le guía paso a paso en la configuración de
                  respaldos. No requiere experiencia en PostgreSQL.
                </li>
                <li>
                  <strong>Temas oscuro y claro</strong>: Elija el aspecto que se
                  ajuste a su forma de trabajar.
                </li>
                <li>
                  <strong>Adaptada a móviles</strong>: Revise sus respaldos
                  desde cualquier lugar y en cualquier dispositivo.
                </li>
                <li>
                  <strong>Monitoreo de salud integrado</strong>: Comprobaciones
                  de salud configurables con gráficos visuales de
                  disponibilidad.
                </li>
                <li>
                  <strong>Restauración con un clic</strong>: Descargue y
                  restaure desde cualquier respaldo con un solo clic.
                </li>
              </ul>

              <h3 id="ease-pgbackweb">Experiencia de usuario de PgBackWeb</h3>

              <ul>
                <li>
                  <strong>Configuración manual de Docker</strong>: Requiere
                  configurar variables de entorno y montar una base de datos
                  PostgreSQL externa para almacenar la configuración.
                </li>
                <li>
                  <strong>Interfaz web básica</strong>: Interfaz funcional pero
                  menos pulida en comparación con Databasus. Tema oscuro
                  disponible.
                </li>
                <li>
                  <strong>Sin monitoreo de salud</strong>: El monitoreo de la
                  disponibilidad de la base de datos debe configurarse por
                  separado.
                </li>
              </ul>

              <h2 id="installation">Instalación y despliegue</h2>

              <h3 id="install-databasus">Instalación de Databasus</h3>

              <p>
                Databasus ofrece tres métodos de instalación; el script
                automatizado es el más rápido:
              </p>

              <ul>
                <li>
                  <strong>Script automatizado (recomendado)</strong>: Un comando
                  cURL de una línea que instala Docker, configura Databasus y
                  activa el arranque automático.
                </li>
                <li>
                  <strong>Docker run</strong>: Un único comando para iniciar
                  Databasus con PostgreSQL integrado.
                </li>
                <li>
                  <strong>Docker Compose</strong>: Para más control sobre el
                  despliegue.
                </li>
              </ul>

              <p>
                <a
                  href="/es/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Ver la guía de instalación de Databasus →
                </a>
              </p>

              <h3 id="install-pgbackweb">Instalación de PgBackWeb</h3>

              <p>
                PgBackWeb requiere Docker y la configuración manual de variables
                de entorno. También necesita montar una base de datos PostgreSQL
                externa para almacenar la configuración de PgBackWeb.
              </p>

              <h2 id="licensing">Licencias</h2>

              <p>
                El modelo de licencia puede afectar significativamente a cómo
                puede usar y modificar el software:
              </p>

              <ul>
                <li>
                  <strong>Databasus (Apache 2.0)</strong>: Licencia permisiva
                  que permite el uso comercial, la modificación y la
                  distribución sin restricciones. Puede usar Databasus en
                  proyectos propietarios sin ninguna preocupación de licencia.
                </li>
                <li>
                  <strong>PgBackWeb (AGPL-3.0)</strong>: Licencia copyleft que
                  exige que cualquier obra derivada o modificación también sea
                  de código abierto bajo AGPL-3.0. Si modifica PgBackWeb y lo
                  ofrece como servicio, debe publicar sus modificaciones.
                </li>
              </ul>

              <h2 id="conclusion">Conclusión</h2>

              <p>
                Tanto Databasus como PgBackWeb son herramientas capaces para las
                copias de seguridad de PostgreSQL, pero cubren necesidades
                distintas:
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">
                    Elija Databasus si necesita:
                  </strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Seguridad de nivel empresarial con protección en 3 niveles
                  </li>
                  <li>
                    Colaboración en equipo con espacios de trabajo y registros
                    de auditoría
                  </li>
                  <li>
                    Múltiples destinos de almacenamiento (FTP, SFTP, Azure,
                    etc.)
                  </li>
                  <li>
                    Notificaciones integradas a Slack, Teams, Telegram, etc.
                  </li>
                  <li>
                    Instalación rápida con un script de una línea o Docker
                  </li>
                  <li>
                    Una interfaz moderna e intuitiva con una curva de
                    aprendizaje mínima
                  </li>
                  <li>Una licencia permisiva Apache 2.0 para uso comercial</li>
                  <li>
                    Respaldos físicos, respaldos incrementales, archivado WAL y
                    PITR para la recuperación ante desastres
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>Elija PgBackWeb si:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Necesita una solución de respaldo sencilla para un solo
                    usuario
                  </li>
                  <li>Le basta el almacenamiento local o S3</li>
                  <li>Le bastan las notificaciones por webhooks</li>
                  <li>La licencia AGPL-3.0 es aceptable para su caso de uso</li>
                </ul>
              </div>

              <p>
                Para la mayoría de los usuarios, especialmente los equipos y
                organizaciones que requieren una seguridad sólida, múltiples
                opciones de almacenamiento y canales de notificación completos,{" "}
                <strong>Databasus es la opción recomendada</strong>.
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
