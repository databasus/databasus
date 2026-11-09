import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Databasus vs Barman: comparación de herramientas de copia de seguridad de PostgreSQL",
  description:
    "Compare Databasus y Barman como herramientas de copia de seguridad de PostgreSQL. Vea las diferencias en enfoque de respaldo, capacidades de PITR, facilidad de uso, funciones de equipo y cuándo elegir cada herramienta.",
  keywords: [
    "Databasus vs Barman",
    "comparación de copias de seguridad de PostgreSQL",
    "alternativa a Barman",
    "herramientas de copia de seguridad de PostgreSQL",
    "comparación de respaldos de bases de datos",
    "pg_dump vs respaldo físico",
    "respaldo autoalojado",
    "PITR de PostgreSQL",
    "archivado WAL",
    "recuperación ante desastres PostgreSQL",
  ],
  openGraph: {
    title:
      "Databasus vs Barman: comparación de herramientas de copia de seguridad de PostgreSQL",
    description:
      "Compare Databasus y Barman como herramientas de copia de seguridad de PostgreSQL. Vea las diferencias en enfoque de respaldo, capacidades de PITR, facilidad de uso, funciones de equipo y cuándo elegir cada herramienta.",
    type: "article",
    url: getLocalizedUrl("es", "databasus-vs-barman"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs Barman: comparación de herramientas de copia de seguridad de PostgreSQL",
    description:
      "Compare Databasus y Barman como herramientas de copia de seguridad de PostgreSQL. Vea las diferencias en enfoque de respaldo, capacidades de PITR, facilidad de uso, funciones de equipo y cuándo elegir cada herramienta.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "databasus-vs-barman"),
    languages: getLanguageAlternates("databasus-vs-barman"),
  },
  robots: "index, follow",
};

export default function DatabasusVsBarmanPage() {
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
              "Databasus vs Barman: comparación de herramientas de copia de seguridad de PostgreSQL",
            description:
              "Una comparación completa de Databasus y Barman como herramientas de copia de seguridad de PostgreSQL: enfoque de respaldo, capacidades de PITR, facilidad de uso, funciones de equipo y cuándo elegir cada herramienta.",
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
              <h1 id="databasus-vs-barman">Databasus vs Barman</h1>

              <p className="text-lg text-gray-400">
                Databasus y Barman están diseñados para la recuperación ante
                desastres con RTO y RPO mínimos, y ambos admiten respaldos
                físicos, archivado WAL y recuperación a un punto en el tiempo
                (PITR). Databasus ejecuta estos respaldos de forma remota sobre
                la pila nativa de PostgreSQL 17, por lo que reutiliza las
                herramientas probadas del propio PostgreSQL en lugar de
                reinventarlas, todo detrás de una interfaz web intuitiva con
                funciones de equipo y soporte para múltiples motores de bases de
                datos. Funciona con bases de datos de cualquier tamaño y
                complejidad. Los respaldos físicos requieren PostgreSQL 17 o
                superior; en versiones anteriores solo están disponibles los
                respaldos lógicos con <code>pg_dump</code>. Barman (Backup and
                Recovery Manager) incluye su propio motor de respaldo, por lo
                que cubre respaldos físicos en versiones de PostgreSQL mucho más
                antiguas y añade funciones avanzadas como respaldos
                incrementales basados en rsync, integración con la replicación
                por streaming y georredundancia de Barman a Barman.
              </p>

              <h2 id="quick-comparison">Comparación rápida</h2>

              <p>
                Este es un resumen rápido de las diferencias clave entre
                Databasus y Barman:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Función</th>
                    <th>Databasus</th>
                    <th>Barman</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Público objetivo</td>
                    <td data-label="Databasus">
                      Particulares, equipos, DBA, empresas
                    </td>
                    <td data-label="Barman">DBA, empresas</td>
                  </tr>
                  <tr>
                    <td>Soporte de otras BD</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="Barman">❌ Solo PostgreSQL</td>
                  </tr>
                  <tr>
                    <td>Interfaz</td>
                    <td data-label="Databasus">Interfaz web</td>
                    <td data-label="Barman">Solo línea de comandos</td>
                  </tr>
                  <tr>
                    <td>Tipo de respaldo</td>
                    <td data-label="Databasus">Lógico + físico</td>
                    <td data-label="Barman">Físico (a nivel de archivos)</td>
                  </tr>
                  <tr>
                    <td>Versión de PostgreSQL para respaldos físicos</td>
                    <td data-label="Databasus">17+ (nativo)</td>
                    <td data-label="Barman">9.x+ (motor propio)</td>
                  </tr>
                  <tr>
                    <td>Opciones de recuperación</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="Barman">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Respaldos incrementales</td>
                    <td data-label="Databasus">
                      ✅ A nivel de bloque (PG 17+)
                    </td>
                    <td data-label="Barman">Incremental basado en rsync</td>
                  </tr>
                  <tr>
                    <td>Respaldos remotos</td>
                    <td data-label="Databasus">✅ Sí</td>
                    <td data-label="Barman">
                      ❌ No (requiere acceso al sistema de archivos)
                    </td>
                  </tr>
                  <tr>
                    <td>Gestión multiservidor</td>
                    <td data-label="Databasus">
                      Programación por base de datos
                    </td>
                    <td data-label="Barman">
                      Servidor de respaldo centralizado
                    </td>
                  </tr>
                  <tr>
                    <td>Funciones de equipo</td>
                    <td data-label="Databasus">
                      ✅ Espacios de trabajo, RBAC, registros de auditoría
                    </td>
                    <td data-label="Barman">
                      ❌ Solo permisos a nivel del sistema operativo
                    </td>
                  </tr>
                  <tr>
                    <td>Notificaciones</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, correo electrónico
                    </td>
                    <td data-label="Barman">
                      ❌ Requiere scripts personalizados
                    </td>
                  </tr>
                  <tr>
                    <td>Curva de aprendizaje</td>
                    <td data-label="Databasus">Mínima</td>
                    <td data-label="Barman">Requiere experiencia de DBA</td>
                  </tr>
                  <tr>
                    <td>Instalación</td>
                    <td data-label="Databasus">Script de una línea o Docker</td>
                    <td data-label="Barman">Requiere configuración manual</td>
                  </tr>
                  <tr>
                    <td>Gestión de respaldos</td>
                    <td data-label="Databasus">✅ Sí</td>
                    <td data-label="Barman">❌ No</td>
                  </tr>
                  <tr>
                    <td>Apto para BD autoalojadas</td>
                    <td data-label="Databasus">✅ Sí</td>
                    <td data-label="Barman">✅ Sí</td>
                  </tr>
                  <tr>
                    <td>Apto para BD en la nube</td>
                    <td data-label="Databasus">
                      ✅ Sí (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="Barman">
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

              <h3 id="audience-barman">Público de Barman</h3>

              <p>
                Barman está diseñado específicamente para administradores de
                bases de datos (DBA) que gestionan infraestructura PostgreSQL
                empresarial:
              </p>

              <ul>
                <li>
                  <strong>DBA empresariales</strong>: Profesionales que
                  necesitan una gestión centralizada de respaldos de varios
                  servidores PostgreSQL desde un servidor de respaldo dedicado.
                </li>
                <li>
                  <strong>Equipos que necesitan incrementales con rsync</strong>
                  : La comparación a nivel de archivos reduce el tiempo de
                  respaldo y el uso de red en clústeres grandes.
                </li>
                <li>
                  <strong>Requisitos de georredundancia</strong>: Replicación de
                  Barman a Barman para redundancia geográfica entre centros de
                  datos.
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
                  versiones de PostgreSQL. Es también el único tipo de respaldo
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
                  <strong>Compresión eficiente</strong>: Usa compresión zstd
                  (nivel 5) tanto para respaldos lógicos como físicos.
                </li>
                <li>
                  <strong>Acceso de solo lectura</strong>: Los respaldos lógicos
                  solo necesitan permisos SELECT, lo que reduce el riesgo de
                  seguridad.
                </li>
              </ul>

              <h3 id="backup-barman">Barman: respaldos físicos</h3>

              <p>
                Barman realiza respaldos a nivel de archivos (físicos) del
                directorio de datos de PostgreSQL:
              </p>

              <ul>
                <li>
                  <strong>Respaldo completo del clúster</strong>: Captura el
                  clúster de base de datos completo a nivel del sistema de
                  archivos usando rsync o pg_basebackup.
                </li>
                <li>
                  <strong>Archivado WAL</strong>: Archiva continuamente los
                  Write-Ahead Logs para la recuperación a un punto en el tiempo.
                </li>
                <li>
                  <strong>Incremental con rsync</strong>: Usa rsync para
                  transferir solo los archivos modificados, lo que acorta el
                  respaldo y reduce el uso de red.
                </li>
                <li>
                  <strong>Integración con la replicación por streaming</strong>:
                  Puede recibir archivos WAL mediante el protocolo de
                  replicación por streaming para un archivado en tiempo real.
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

              <h3 id="recovery-barman">Recuperación con Barman</h3>

              <ul>
                <li>
                  <strong>Recuperación a un punto en el tiempo (PITR)</strong>:
                  Restaure a cualquier segundo específico mediante la
                  reproducción de WAL, con una pérdida de datos mínima.
                </li>
                <li>
                  <strong>Restauración completa del clúster</strong>: Restaure
                  el clúster de base de datos completo a un punto específico en
                  el tiempo.
                </li>
                <li>
                  <strong>Recuperación remota</strong>: Recupere bases de datos
                  en servidores remotos mediante SSH.
                </li>
                <li>
                  <strong>Creación de réplicas standby</strong>: Cree réplicas
                  de PostgreSQL a partir de respaldos para configuraciones de
                  alta disponibilidad.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Nota:</strong> Ambas
                  herramientas admiten PITR. Barman ofrece además la creación de
                  réplicas standby a partir de respaldos y la recuperación
                  remota por SSH en otros servidores, lo cual puede resultar
                  valioso en configuraciones de alta disponibilidad.{" "}
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

              <h3 id="ease-barman">Experiencia de usuario de Barman</h3>

              <ul>
                <li>
                  <strong>Interfaz de línea de comandos</strong>: Todas las
                  operaciones se realizan mediante comandos de terminal como{" "}
                  <code>barman backup</code>, <code>barman recover</code>.
                </li>
                <li>
                  <strong>Archivos de configuración</strong>: Requiere editar
                  manualmente archivos de configuración de tipo INI para cada
                  servidor.
                </li>
                <li>
                  <strong>Configuración del archivado WAL</strong>: Hay que
                  configurar el <code>archive_command</code> de PostgreSQL o los
                  ajustes de replicación por streaming.
                </li>
                <li>
                  <strong>Gestión de claves SSH</strong>: Requiere configurar
                  claves SSH entre el servidor de Barman y los servidores
                  PostgreSQL.
                </li>
                <li>
                  <strong>Se espera experiencia de DBA</strong>: La
                  documentación asume familiaridad con los internos de
                  PostgreSQL y la mecánica de WAL.
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

              <h3 id="team-barman">Capacidades de equipo de Barman</h3>

              <p>
                Barman es una herramienta de línea de comandos sin funciones de
                equipo integradas:
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
                  y claves SSH
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
                Ambas herramientas proporcionan funciones de seguridad, pero con
                enfoques distintos:
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

              <h3 id="security-barman">Seguridad de Barman</h3>

              <ul>
                <li>
                  <strong>Comunicación basada en SSH</strong>: Usa SSH para una
                  comunicación segura entre el servidor de Barman y los
                  servidores PostgreSQL.
                </li>
                <li>
                  <strong>Sin cifrado integrado</strong>: Barman no ofrece
                  cifrado de respaldos integrado. Hay que usar herramientas
                  externas o almacenamiento cifrado.
                </li>
                <li>
                  <strong>Seguridad a nivel del sistema operativo</strong>:
                  Depende de los permisos del sistema de archivos y de la
                  gestión de claves SSH para el control de acceso.
                </li>
                <li>
                  <strong>Verificación de sumas de comprobación</strong>: Valida
                  la integridad de los respaldos mediante sumas de comprobación.
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
                Las herramientas admiten destinos de almacenamiento distintos:
              </p>

              <h3 id="storage-databasus">Almacenamiento de Databasus</h3>

              <p>Opciones accesibles para diversos casos de uso:</p>

              <ul>
                <li>Almacenamiento local</li>
                <li>Amazon S3 y servicios compatibles con S3</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (almacenamiento conectado en red)</li>
                <li>FTP</li>
              </ul>

              <h3 id="storage-barman">Almacenamiento de Barman</h3>

              <p>Opciones de almacenamiento orientadas a empresas:</p>

              <ul>
                <li>Almacenamiento local (sistemas de archivos POSIX)</li>
                <li>Amazon S3 y almacenamiento de objetos compatible con S3</li>
                <li>
                  Redundancia geográfica mediante replicación de Barman a Barman
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

              <h3 id="notifications-barman">Notificaciones de Barman</h3>

              <p>
                Barman no tiene soporte de notificaciones integrado. Las
                notificaciones requieren:
              </p>

              <ul>
                <li>
                  Scripts personalizados alrededor de los comandos de respaldo
                </li>
                <li>Integración con herramientas de monitoreo externas</li>
                <li>Análisis manual de registros y configuración de alertas</li>
                <li>
                  Integración con herramientas como Nagios, Zabbix o soluciones
                  personalizadas
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

              <h2 id="multi-server-management">Gestión multiservidor</h2>

              <p>
                Ambas herramientas pueden gestionar respaldos de varios
                servidores PostgreSQL, pero con enfoques distintos:
              </p>

              <h3 id="multi-databasus">Enfoque de Databasus</h3>

              <ul>
                <li>
                  <strong>Programación por base de datos</strong>: Cada base de
                  datos puede tener su propio calendario de respaldo y destino
                  de almacenamiento.
                </li>
                <li>
                  <strong>Organización en espacios de trabajo</strong>: Agrupe
                  las bases de datos relacionadas en espacios de trabajo para
                  una gestión más sencilla.
                </li>
                <li>
                  <strong>Panel unificado</strong>: Vea todos los respaldos de
                  bases de datos y su estado en una única interfaz web.
                </li>
              </ul>

              <h3 id="multi-barman">Enfoque de Barman</h3>

              <ul>
                <li>
                  <strong>Servidor de respaldo centralizado</strong>: Un
                  servidor de Barman dedicado gestiona los respaldos de varias
                  instancias PostgreSQL.
                </li>
                <li>
                  <strong>Configuración por servidor</strong>: Cada servidor
                  PostgreSQL requiere su propio archivo de configuración en el
                  servidor de Barman.
                </li>
                <li>
                  <strong>Georredundancia</strong>: Los servidores de Barman
                  pueden replicarse a otros servidores de Barman para
                  redundancia geográfica.
                </li>
              </ul>

              <h2 id="conclusion">Conclusión</h2>

              <p>
                Databasus y Barman cubren necesidades distintas dentro del
                ecosistema de copias de seguridad de PostgreSQL. La elección
                correcta depende de sus requisitos de recuperación, la
                estructura de su equipo y su experiencia técnica.
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
                    El cifrado de respaldos integrado es importante para usted
                  </li>
                  <li>
                    Usa bases de datos gestionadas en la nube (AWS RDS, Google
                    Cloud SQL, Azure) o PostgreSQL autoalojado
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Elija Barman si:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Necesita respaldos físicos o incrementales en PostgreSQL
                    anterior a 17 (Barman incluye su propio motor de respaldo)
                  </li>
                  <li>
                    Necesita respaldos incrementales basados en rsync
                    (comparación a nivel de archivos) para reducir el tiempo de
                    transferencia
                  </li>
                  <li>
                    Necesita la integración con la replicación por streaming
                    para un archivado WAL en tiempo real
                  </li>
                  <li>Necesita la redundancia geográfica de Barman a Barman</li>
                  <li>
                    Necesita crear réplicas standby a partir de respaldos para
                    configuraciones de alta disponibilidad
                  </li>
                  <li>
                    Se siente cómodo con las herramientas de línea de comandos y
                    los internos de PostgreSQL
                  </li>
                  <li>
                    Su organización dispone de experiencia dedicada de DBA
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
                nube. Barman encaja mejor cuando necesita respaldos
                incrementales basados en rsync, integración con la replicación
                por streaming, georredundancia de Barman a Barman o la creación
                de réplicas standby a partir de respaldos.
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
