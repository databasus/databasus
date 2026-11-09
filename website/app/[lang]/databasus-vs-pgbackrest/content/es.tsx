import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Databasus vs pgBackRest: comparación de herramientas de copia de seguridad de PostgreSQL",
  description:
    "Compare Databasus y pgBackRest como herramientas de copia de seguridad de PostgreSQL. Vea las diferencias en enfoque de respaldo, público objetivo, facilidad de uso, opciones de recuperación y cuándo elegir cada herramienta.",
  keywords: [
    "Databasus vs pgBackRest",
    "comparación de copias de seguridad de PostgreSQL",
    "alternativa a pgBackRest",
    "herramientas de copia de seguridad de PostgreSQL",
    "comparación de respaldos de bases de datos",
    "pg_dump vs respaldo físico",
    "respaldo autoalojado",
    "PITR de PostgreSQL",
    "respaldo de bases de datos grandes",
    "herramientas de respaldo para DBA",
  ],
  openGraph: {
    title:
      "Databasus vs pgBackRest: comparación de herramientas de copia de seguridad de PostgreSQL",
    description:
      "Compare Databasus y pgBackRest como herramientas de copia de seguridad de PostgreSQL. Vea las diferencias en enfoque de respaldo, público objetivo, facilidad de uso, opciones de recuperación y cuándo elegir cada herramienta.",
    type: "article",
    url: getLocalizedUrl("es", "databasus-vs-pgbackrest"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs pgBackRest: comparación de herramientas de copia de seguridad de PostgreSQL",
    description:
      "Compare Databasus y pgBackRest como herramientas de copia de seguridad de PostgreSQL. Vea las diferencias en enfoque de respaldo, público objetivo, facilidad de uso, opciones de recuperación y cuándo elegir cada herramienta.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "databasus-vs-pgbackrest"),
    languages: getLanguageAlternates("databasus-vs-pgbackrest"),
  },
  robots: "index, follow",
};

export default function DatabasusVsPgBackRestPage() {
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
              "Databasus vs pgBackRest: comparación de herramientas de copia de seguridad de PostgreSQL",
            description:
              "Una comparación completa de Databasus y pgBackRest como herramientas de copia de seguridad de PostgreSQL: enfoque de respaldo, público objetivo, facilidad de uso, opciones de recuperación y cuándo elegir cada herramienta.",
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
              <h1 id="databasus-vs-pgbackrest">Databasus vs pgBackRest</h1>

              <p className="text-lg text-gray-400">
                Databasus y pgBackRest están diseñados para la recuperación ante
                desastres con RTO y RPO mínimos, y ambos admiten respaldos
                físicos, archivado WAL y recuperación a un punto en el tiempo
                (PITR). Databasus ejecuta estos respaldos de forma remota sobre
                la pila nativa de PostgreSQL 17, por lo que reutiliza las
                herramientas probadas del propio PostgreSQL en lugar de
                reinventarlas, todo detrás de una interfaz web intuitiva.
                Funciona con bases de datos de cualquier tamaño y complejidad.
                Los respaldos físicos requieren PostgreSQL 17 o superior; en
                versiones anteriores solo están disponibles los respaldos
                lógicos con <code>pg_dump</code>. pgBackRest incluye su propio
                motor de respaldo, por lo que también cubre respaldos físicos e
                incrementales en versiones de PostgreSQL mucho más antiguas, y
                añade funciones avanzadas como respaldos diferenciales y
                restauración delta.
              </p>

              <h2 id="quick-comparison">Comparación rápida</h2>

              <p>
                Este es un resumen rápido de las diferencias clave entre
                Databasus y pgBackRest:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Función</th>
                    <th>Databasus</th>
                    <th>pgBackRest</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Público objetivo</td>
                    <td data-label="Databasus">
                      Particulares, equipos, DBA, empresas
                    </td>
                    <td data-label="pgBackRest">
                      DBA, flujos de trabajo de CLI e IaC
                    </td>
                  </tr>
                  <tr>
                    <td>Gestión de respaldos</td>
                    <td data-label="Databasus">✅ Múltiples BD</td>
                    <td data-label="pgBackRest">❌ Solo una BD</td>
                  </tr>
                  <tr>
                    <td>Soporte de otras BD</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="pgBackRest">❌ Solo PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>Interfaz</td>
                    <td data-label="Databasus">Interfaz web</td>
                    <td data-label="pgBackRest">
                      Línea de comandos, archivos de configuración
                    </td>
                  </tr>
                  <tr>
                    <td>Tipo de respaldo</td>
                    <td data-label="Databasus">Lógico + físico</td>
                    <td data-label="pgBackRest">
                      Físico (a nivel de archivos)
                    </td>
                  </tr>
                  <tr>
                    <td>Versión de PostgreSQL para respaldos físicos</td>
                    <td data-label="Databasus">17+ (nativo)</td>
                    <td data-label="pgBackRest">9.4+ (motor propio)</td>
                  </tr>
                  <tr>
                    <td>Opciones de recuperación</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="pgBackRest">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Operaciones en paralelo</td>
                    <td data-label="Databasus">✅ Sí</td>
                    <td data-label="pgBackRest">✅ Sí</td>
                  </tr>
                  <tr>
                    <td>Respaldos incrementales</td>
                    <td data-label="Databasus">
                      ✅ A nivel de bloque (PG 17+)
                    </td>
                    <td data-label="pgBackRest">
                      Incremental a nivel de bloque
                    </td>
                  </tr>
                  <tr>
                    <td>Respaldos diferenciales</td>
                    <td data-label="Databasus">❌ No</td>
                    <td data-label="pgBackRest">✅ Sí</td>
                  </tr>
                  <tr>
                    <td>Restauración delta</td>
                    <td data-label="Databasus">❌ No</td>
                    <td data-label="pgBackRest">
                      ✅ Sí (solo archivos modificados)
                    </td>
                  </tr>
                  <tr>
                    <td>Respaldos remotos</td>
                    <td data-label="Databasus">✅ Sí</td>
                    <td data-label="pgBackRest">
                      ❌ No (requiere acceso al sistema de archivos)
                    </td>
                  </tr>
                  <tr>
                    <td>Funciones de equipo</td>
                    <td data-label="Databasus">
                      ✅ Espacios de trabajo, RBAC, registros de auditoría
                    </td>
                    <td data-label="pgBackRest">❌ Usuario único</td>
                  </tr>
                  <tr>
                    <td>Curva de aprendizaje</td>
                    <td data-label="Databasus">Mínima</td>
                    <td data-label="pgBackRest">Requiere experiencia de DBA</td>
                  </tr>
                  <tr>
                    <td>Instalación</td>
                    <td data-label="Databasus">Script de una línea o Docker</td>
                    <td data-label="pgBackRest">
                      Requiere configuración manual
                    </td>
                  </tr>
                  <tr>
                    <td>Apto para BD autoalojadas</td>
                    <td data-label="Databasus">✅ Sí</td>
                    <td data-label="pgBackRest">✅ Sí</td>
                  </tr>
                  <tr>
                    <td>Apto para BD en la nube</td>
                    <td data-label="Databasus">
                      ✅ Sí (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="pgBackRest">
                      ❌ No (requiere acceso al sistema de archivos)
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="target-audience">Público objetivo</h2>

              <p>
                La diferencia más significativa entre estas herramientas es para
                quién están diseñadas:
              </p>

              <h3 id="audience-databasus">Público de Databasus</h3>

              <p>
                Databasus está construido para un público amplio, desde
                desarrolladores individuales hasta grandes empresas:
              </p>

              <ul>
                <li>
                  <strong>Desarrolladores individuales</strong>: La
                  configuración sencilla y la interfaz intuitiva facilitan
                  proteger proyectos personales sin conocimientos profundos de
                  PostgreSQL.
                </li>
                <li>
                  <strong>Equipos de desarrollo</strong>: Los espacios de
                  trabajo, el control de acceso basado en roles y los registros
                  de auditoría permiten una colaboración segura entre los
                  miembros del equipo.
                </li>
                <li>
                  <strong>Empresas</strong>: Escala para cubrir necesidades
                  empresariales con seguridad completa, múltiples destinos de
                  almacenamiento y canales de notificación.
                </li>
                <li>
                  <strong>DBA y recuperación ante desastres</strong>: Respaldos
                  físicos, archivado WAL y PITR para sistemas críticos con
                  requisitos de pérdida de datos casi nula.
                </li>
              </ul>

              <h3 id="audience-pgbackrest">Público de pgBackRest</h3>

              <p>
                pgBackRest es una herramienta de línea de comandos dirigida a
                equipos que quieren ejecutar el motor de respaldo por su cuenta:
              </p>

              <ul>
                <li>
                  <strong>Flujos de trabajo de CLI e IaC</strong>: Equipos que
                  prefieren configurar los respaldos mediante archivos y scripts
                  en lugar de una interfaz web.
                </li>
                <li>
                  <strong>Versiones antiguas de PostgreSQL</strong>: pgBackRest
                  incluye su propio motor, por lo que puede ejecutar respaldos
                  físicos e incrementales en versiones de PostgreSQL anteriores
                  a 17.
                </li>
                <li>
                  <strong>Funciones avanzadas a gran escala</strong>: Donde los
                  respaldos diferenciales, la restauración delta y la creación
                  de réplicas standby justifican la configuración adicional.
                </li>
              </ul>

              <h2 id="backup-approach">Enfoque de respaldo</h2>

              <p>
                Las herramientas usan estrategias de respaldo fundamentalmente
                distintas, cada una con ventajas propias:
              </p>

              <h3 id="backup-databasus">
                Databasus: respaldos lógicos + físicos
              </h3>

              <p>
                Databasus admite tanto estrategias de respaldo lógico como
                físico:
              </p>

              <ul>
                <li>
                  <strong>Respaldos físicos, incrementales y de WAL</strong>: Se
                  ejecutan de forma remota mediante el protocolo de replicación
                  de PostgreSQL sobre la pila nativa de PostgreSQL 17:{" "}
                  <code>pg_basebackup</code>,{" "}
                  <code>pg_basebackup --incremental</code> a nivel de bloque
                  impulsado por resúmenes WAL del lado del servidor,{" "}
                  <code>pg_receivewal</code> y <code>pg_combinebackup</code>.
                  Databasus reutiliza las herramientas probadas del propio
                  PostgreSQL en lugar de reinventarlas. Requiere PostgreSQL 17 o
                  superior.
                </li>
                <li>
                  <strong>Respaldos lógicos</strong>: Usa <code>pg_dump</code>{" "}
                  para respaldos portables que pueden restaurarse en distintas
                  versiones de PostgreSQL, con restauración selectiva de tablas
                  o esquemas específicos. Es también el único tipo de respaldo
                  en PostgreSQL anterior a 17 y la vía para MySQL, MariaDB y
                  MongoDB.
                </li>
                <li>
                  <strong>Nada instalado en la base de datos</strong>: Los
                  respaldos se conectan de forma remota; a las redes cerradas se
                  llega mediante un túnel SSH a un host interno o un bastión, de
                  modo que la base de datos nunca tiene que exponerse
                  públicamente.
                </li>
                <li>
                  <strong>Compresión eficiente</strong>: Usa zstd (nivel 5) para
                  ambos tipos de respaldo; el archivo resultante ocupa entre 4 y
                  8 veces menos.
                </li>
                <li>
                  <strong>Acceso de solo lectura</strong>: Los respaldos lógicos
                  solo necesitan permisos SELECT, lo que reduce el riesgo de
                  seguridad.
                </li>
              </ul>

              <h3 id="backup-pgbackrest">pgBackRest: respaldos físicos</h3>

              <p>
                pgBackRest realiza respaldos a nivel de archivos (físicos) del
                directorio de datos de PostgreSQL:
              </p>

              <ul>
                <li>
                  <strong>Incremental a nivel de bloque</strong>: Solo se
                  respaldan los bloques modificados, lo que acorta el respaldo y
                  ahorra almacenamiento en bases de datos muy grandes.
                </li>
                <li>
                  <strong>Archivado WAL</strong>: El archivado continuo de los
                  Write-Ahead Logs permite una recuperación precisa a un punto
                  en el tiempo.
                </li>
                <li>
                  <strong>Completo, diferencial, incremental</strong>: Múltiples
                  estrategias de respaldo para distintos escenarios de
                  recuperación.
                </li>
                <li>
                  <strong>Optimizado para la escala</strong>: Diseñado para
                  bases de datos donde los respaldos lógicos tardarían
                  demasiado.
                </li>
              </ul>

              <h2 id="recovery-options">Opciones de recuperación</h2>

              <p>
                Ambas herramientas ofrecen opciones de recuperación flexibles,
                pero con distinta granularidad:
              </p>

              <h3 id="recovery-databasus">Recuperación con Databasus</h3>

              <ul>
                <li>
                  <strong>Recuperación a un punto en el tiempo</strong>:
                  Restaure a cualquier segundo específico mediante la
                  reproducción de WAL.
                </li>
                <li>
                  <strong>Restauración completa del clúster</strong>: Restaure
                  el clúster de base de datos completo a un punto específico en
                  el tiempo desde respaldos físicos.
                </li>
                <li>
                  <strong>Restauración lógica</strong>: Restaure desde respaldos
                  lógicos programados a cualquier punto de respaldo.
                </li>
                <li>
                  <strong>Restauración con un clic</strong>: Descargue y
                  restaure respaldos lógicos directamente desde la interfaz web.
                </li>
                <li>
                  <strong>Compatibilidad entre versiones</strong>: Los respaldos
                  lógicos pueden restaurarse en distintas versiones de
                  PostgreSQL.
                </li>
              </ul>

              <h3 id="recovery-pgbackrest">Recuperación con pgBackRest</h3>

              <ul>
                <li>
                  <strong>Recuperación a un punto en el tiempo (PITR)</strong>:
                  Restaure a cualquier segundo específico mediante la
                  reproducción de WAL.
                </li>
                <li>
                  <strong>Restauración completa del clúster</strong>: Restaure
                  el clúster de base de datos completo a un punto específico en
                  el tiempo desde respaldos físicos.
                </li>
                <li>
                  <strong>Restauración en paralelo</strong>: Restauración
                  multihilo para una recuperación más rápida de bases de datos
                  grandes.
                </li>
                <li>
                  <strong>Restauración delta</strong>: Restaure solo los
                  archivos modificados, lo que acelera la recuperación.
                </li>
                <li>
                  <strong>Creación de réplicas standby</strong>: Cree réplicas
                  de PostgreSQL a partir de respaldos.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Nota:</strong> Ambas
                  herramientas admiten PITR. pgBackRest ofrece además
                  restauración delta (obteniendo solo los archivos modificados),
                  respaldos diferenciales y la creación de réplicas standby a
                  partir de respaldos.{" "}
                  <a
                    href="/es/faq#pitr"
                    className="text-blue-400 hover:text-blue-300"
                  >
                    Descubra cómo Databasus admite PITR →
                  </a>
                </p>
              </div>

              <h2 id="ease-of-use">Facilidad de uso</h2>

              <p>
                Las herramientas difieren drásticamente en su enfoque de la
                experiencia de usuario:
              </p>

              <h3 id="ease-databasus">Experiencia de usuario de Databasus</h3>

              <ul>
                <li>
                  <strong>Interfaz web</strong>: Configuración de todos los
                  ajustes de respaldo con apuntar y hacer clic. No requiere
                  línea de comandos.
                </li>
                <li>
                  <strong>Instalación en 2 minutos</strong>: Un script cURL de
                  una línea o un simple comando Docker lo pone en marcha de
                  inmediato.
                </li>
                <li>
                  <strong>Monitoreo visual</strong>: El panel muestra de un
                  vistazo el estado de los respaldos, las comprobaciones de
                  salud y el historial.
                </li>
                <li>
                  <strong>Notificaciones integradas</strong>: Configure alertas
                  de Slack, Teams, Telegram, correo electrónico o webhooks
                  directamente en la interfaz.
                </li>
                <li>
                  <strong>Sin necesidad de experiencia en PostgreSQL</strong>:
                  Diseñado para desarrolladores que quieren respaldos fiables
                  sin convertirse en expertos en bases de datos.
                </li>
              </ul>

              <h3 id="ease-pgbackrest">Experiencia de usuario de pgBackRest</h3>

              <ul>
                <li>
                  <strong>Interfaz de línea de comandos</strong>: Todas las
                  operaciones se realizan mediante comandos de terminal.
                </li>
                <li>
                  <strong>Archivos de configuración</strong>: Requiere editar
                  manualmente archivos de configuración de tipo INI.
                </li>
                <li>
                  <strong>Configuración del archivado WAL</strong>: Hay que
                  configurar el <code>archive_command</code> de PostgreSQL y los
                  ajustes relacionados.
                </li>
                <li>
                  <strong>Curva de aprendizaje pronunciada</strong>: Requiere
                  entender los internos de PostgreSQL, la mecánica de WAL y las
                  estrategias de respaldo.
                </li>
                <li>
                  <strong>Se espera experiencia de DBA</strong>: La
                  documentación asume familiaridad con los conceptos de
                  administración de bases de datos.
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

              <h2 id="team-features">Funciones de equipo</h2>

              <p>
                Para organizaciones con varios miembros del equipo gestionando
                respaldos:
              </p>

              <h3 id="team-databasus">Capacidades de equipo de Databasus</h3>

              <ul>
                <li>
                  <strong>Espacios de trabajo</strong>: Organice bases de datos,
                  notificadores y almacenamientos por proyecto o equipo. Los
                  usuarios solo ven los espacios de trabajo a los que han sido
                  invitados.
                </li>
                <li>
                  <strong>Control de acceso basado en roles</strong>: Asigne
                  permisos de lector, editor o administrador para controlar lo
                  que puede hacer cada miembro del equipo.
                </li>
                <li>
                  <strong>Registros de auditoría</strong>: Rastree todas las
                  actividades y cambios del sistema. Esencial para el
                  cumplimiento de seguridad y la rendición de cuentas.
                </li>
                <li>
                  <strong>Notificaciones compartidas</strong>: Los canales del
                  equipo reciben automáticamente las actualizaciones del estado
                  de los respaldos.
                </li>
              </ul>

              <h3 id="team-pgbackrest">Capacidades de equipo de pgBackRest</h3>

              <p>
                pgBackRest es una herramienta de línea de comandos sin funciones
                de equipo integradas:
              </p>

              <ul>
                <li>Sin gestión de usuarios ni control de acceso</li>
                <li>Sin registro de auditoría de las operaciones</li>
                <li>
                  La coordinación del equipo requiere herramientas y procesos
                  externos
                </li>
                <li>
                  El acceso se controla mediante permisos del sistema operativo
                  sobre los archivos de configuración
                </li>
              </ul>

              <p>
                <a
                  href="/es/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Más información sobre la gestión de accesos de Databasus →
                </a>
              </p>

              <h2 id="security">Seguridad</h2>

              <p>
                Ambas herramientas proporcionan funciones de seguridad sólidas:
              </p>

              <h3 id="security-databasus">Seguridad de Databasus</h3>

              <ul>
                <li>
                  <strong>Cifrado AES-256-GCM</strong>: Todas las contraseñas,
                  tokens y credenciales se cifran. La clave de cifrado se
                  almacena separada de la base de datos.
                </li>
                <li>
                  <strong>Cifrado único por respaldo</strong>: Cada archivo de
                  respaldo se cifra con una clave única derivada de la clave
                  maestra, el ID del respaldo y una sal aleatoria.
                </li>
                <li>
                  <strong>Acceso de solo lectura a la base de datos</strong>:
                  Solo exige permisos SELECT; así, incluso si la cuenta se ve
                  comprometida, los datos no pueden corromperse.
                </li>
              </ul>

              <h3 id="security-pgbackrest">Seguridad de pgBackRest</h3>

              <ul>
                <li>
                  <strong>Cifrado del repositorio</strong>: Los repositorios de
                  respaldo pueden cifrarse con AES-256.
                </li>
                <li>
                  <strong>Transporte TLS/SSH</strong>: Comunicación segura para
                  operaciones remotas.
                </li>
                <li>
                  <strong>Verificación de sumas de comprobación</strong>: Valida
                  la integridad de los respaldos durante la creación y la
                  restauración.
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

              <h2 id="storage-options">Opciones de almacenamiento</h2>

              <p>
                Ambas herramientas admiten múltiples destinos de almacenamiento;
                Databasus ofrece opciones más accesibles:
              </p>

              <h3 id="storage-databasus">Almacenamiento de Databasus</h3>

              <ul>
                <li>Almacenamiento local</li>
                <li>Amazon S3 y servicios compatibles con S3</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (almacenamiento conectado en red)</li>
                <li>FTP</li>
              </ul>

              <h3 id="storage-pgbackrest">Almacenamiento de pgBackRest</h3>

              <ul>
                <li>Almacenamiento local (POSIX, CIFS)</li>
                <li>Amazon S3 y servicios compatibles con S3</li>
                <li>Cloudflare R2 (compatible con S3)</li>
                <li>Azure Blob Storage</li>
                <li>NAS (almacenamiento conectado en red)</li>
                <li>Google Cloud Storage</li>
                <li>SFTP</li>
              </ul>

              <p>
                <a
                  href="/es/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Ver todas las opciones de almacenamiento de Databasus →
                </a>
              </p>

              <h2 id="notifications">Notificaciones</h2>

              <p>Mantenerse informado del estado de los respaldos:</p>

              <h3 id="notifications-databasus">Notificaciones de Databasus</h3>

              <p>Soporte integrado para múltiples canales de notificación:</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>Correo electrónico</li>
                <li>Webhooks</li>
              </ul>

              <h3 id="notifications-pgbackrest">
                Notificaciones de pgBackRest
              </h3>

              <p>
                pgBackRest no tiene soporte de notificaciones integrado. Las
                notificaciones requieren:
              </p>

              <ul>
                <li>
                  Scripts personalizados alrededor de los comandos de respaldo
                </li>
                <li>Integración con herramientas de monitoreo externas</li>
                <li>Análisis manual de registros y configuración de alertas</li>
              </ul>

              <p>
                <a
                  href="/es/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Ver todos los canales de notificación de Databasus →
                </a>
              </p>

              <h2 id="conclusion">Conclusión</h2>

              <p>
                Databasus y pgBackRest cubren necesidades distintas dentro del
                ecosistema de copias de seguridad de PostgreSQL. La elección
                correcta depende del tamaño de su base de datos, la estructura
                de su equipo y sus requisitos técnicos.
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">Elija Databasus si:</strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Es un desarrollador individual, un equipo o una empresa en
                    busca de una solución de respaldo intuitiva
                  </li>
                  <li>
                    Prefiere una interfaz web a las herramientas de línea de
                    comandos
                  </li>
                  <li>
                    Necesita funciones de colaboración en equipo (espacios de
                    trabajo, RBAC, registros de auditoría)
                  </li>
                  <li>
                    Quiere notificaciones integradas a Slack, Teams, Telegram,
                    etc.
                  </li>
                  <li>
                    Quiere gestionar los respaldos de varias bases de datos
                    desde un único panel con programación, notificaciones y
                    funciones de equipo
                  </li>
                  <li>
                    Quiere una puesta en marcha rápida con mínima experiencia en
                    PostgreSQL
                  </li>
                  <li>
                    Usa bases de datos gestionadas en la nube (AWS RDS, Google
                    Cloud SQL, Azure) o PostgreSQL autoalojado
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Elija pgBackRest si:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Necesita respaldos físicos o incrementales en PostgreSQL
                    anterior a 17 (pgBackRest incluye su propio motor de
                    respaldo)
                  </li>
                  <li>
                    Necesita respaldos diferenciales o restauración delta (solo
                    archivos modificados)
                  </li>
                  <li>
                    Necesita crear réplicas standby a partir de respaldos para
                    alta disponibilidad
                  </li>
                  <li>
                    Prefiere herramientas de línea de comandos y flujos de
                    trabajo de infraestructura como código
                  </li>
                  <li>
                    Su equipo tiene la experiencia en PostgreSQL para ejecutarlo
                    y ajustarlo
                  </li>
                </ul>
              </div>

              <p>
                Ambas herramientas admiten respaldos físicos, archivado WAL y
                PITR, y ambas están diseñadas para la recuperación ante
                desastres con RTO y RPO mínimos. Databasus funciona con bases de
                datos de cualquier tamaño y complejidad, y le ofrece una
                interfaz web, funciones de equipo y respaldos tanto lógicos como
                físicos para bases de datos autoalojadas y gestionadas en la
                nube.
                <br />
                <br />
                pgBackRest encaja mejor cuando quiere ejecutar el motor de
                respaldo por su cuenta, necesita respaldos físicos en PostgreSQL
                anterior a 17, o depende de sus respaldos diferenciales y su
                restauración delta.
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
