import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title:
    "Databasus vs WAL-G: comparación de herramientas de copia de seguridad de PostgreSQL",
  description:
    "Compare Databasus y WAL-G como herramientas de copia de seguridad de PostgreSQL. Vea las diferencias en enfoque de respaldo, soporte multibase, facilidad de uso, funciones de equipo y cuándo elegir cada herramienta.",
  keywords: [
    "Databasus vs WAL-G",
    "comparación de copias de seguridad de PostgreSQL",
    "alternativa a WAL-G",
    "herramientas de copia de seguridad de PostgreSQL",
    "comparación de respaldos de bases de datos",
    "pg_dump vs archivado WAL",
    "respaldo autoalojado",
    "PITR de PostgreSQL",
    "archivado WAL",
    "respaldo multibase",
  ],
  openGraph: {
    title:
      "Databasus vs WAL-G: comparación de herramientas de copia de seguridad de PostgreSQL",
    description:
      "Compare Databasus y WAL-G como herramientas de copia de seguridad de PostgreSQL. Vea las diferencias en enfoque de respaldo, soporte multibase, facilidad de uso, funciones de equipo y cuándo elegir cada herramienta.",
    type: "article",
    url: getLocalizedUrl("es", "databasus-vs-wal-g"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title:
      "Databasus vs WAL-G: comparación de herramientas de copia de seguridad de PostgreSQL",
    description:
      "Compare Databasus y WAL-G como herramientas de copia de seguridad de PostgreSQL. Vea las diferencias en enfoque de respaldo, soporte multibase, facilidad de uso, funciones de equipo y cuándo elegir cada herramienta.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "databasus-vs-wal-g"),
    languages: getLanguageAlternates("databasus-vs-wal-g"),
  },
  robots: "index, follow",
};

export default function DatabasusVsWalGPage() {
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
              "Databasus vs WAL-G: comparación de herramientas de copia de seguridad de PostgreSQL",
            description:
              "Una comparación completa de Databasus y WAL-G como herramientas de copia de seguridad de PostgreSQL: enfoque de respaldo, soporte multibase, facilidad de uso, funciones de equipo y cuándo elegir cada herramienta.",
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
              <h1 id="databasus-vs-wal-g">Databasus vs WAL-G</h1>

              <p className="text-lg text-gray-400">
                Databasus y WAL-G están diseñados para la recuperación ante
                desastres con RTO y RPO mínimos, y ambos admiten respaldos
                físicos de PostgreSQL, archivado WAL y recuperación a un punto
                en el tiempo (PITR). Databasus ejecuta estos respaldos de forma
                remota sobre la pila nativa de PostgreSQL 17, por lo que
                reutiliza las herramientas probadas del propio PostgreSQL en
                lugar de reinventarlas, todo detrás de una interfaz web
                intuitiva. Funciona con bases de datos de cualquier tamaño y
                complejidad. Los respaldos físicos requieren PostgreSQL 17 o
                superior; en versiones anteriores solo están disponibles los
                respaldos lógicos con <code>pg_dump</code>. WAL-G es una
                herramienta de línea de comandos con su propio motor, por lo que
                cubre respaldos físicos en versiones de PostgreSQL mucho más
                antiguas, usa un protocolo de streaming propio con un
                rendimiento ligeramente mejor, admite respaldos delta (solo
                páginas modificadas) y cubre más motores de bases de datos,
                incluidos MS SQL, FoundationDB y Greenplum.
              </p>

              <h2 id="quick-comparison">Comparación rápida</h2>

              <p>
                Este es un resumen rápido de las diferencias clave entre
                Databasus y WAL-G:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Función</th>
                    <th>Databasus</th>
                    <th>WAL-G</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Gestión de respaldos</td>
                    <td data-label="Databasus">✅ Sí (múltiples BD)</td>
                    <td data-label="WAL-G">❌ No (solo una BD)</td>
                  </tr>
                  <tr>
                    <td>Soporte de otras BD</td>
                    <td data-label="Databasus">
                      ✅ PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                    <td data-label="WAL-G">✅ PostgreSQL, MySQL, MS SQL</td>
                  </tr>
                  <tr>
                    <td>Interfaz</td>
                    <td data-label="Databasus">Interfaz web</td>
                    <td data-label="WAL-G">Solo línea de comandos</td>
                  </tr>
                  <tr>
                    <td>Tipo de respaldo</td>
                    <td data-label="Databasus">Lógico + físico</td>
                    <td data-label="WAL-G">Físico (archivado WAL)</td>
                  </tr>
                  <tr>
                    <td>Versión de PostgreSQL para respaldos físicos</td>
                    <td data-label="Databasus">17+ (nativo)</td>
                    <td data-label="WAL-G">9.x+ (motor propio)</td>
                  </tr>
                  <tr>
                    <td>Programación de respaldos</td>
                    <td data-label="Databasus">✅ Planificador integrado</td>
                    <td data-label="WAL-G">
                      Requiere herramienta externa (cron)
                    </td>
                  </tr>
                  <tr>
                    <td>Opciones de recuperación</td>
                    <td data-label="Databasus">✅ PITR</td>
                    <td data-label="WAL-G">✅ PITR</td>
                  </tr>
                  <tr>
                    <td>Respaldos incrementales</td>
                    <td data-label="Databasus">
                      ✅ A nivel de bloque (PG 17+)
                    </td>
                    <td data-label="WAL-G">
                      Respaldos delta (solo páginas modificadas)
                    </td>
                  </tr>
                  <tr>
                    <td>Respaldos remotos</td>
                    <td data-label="Databasus">✅ Sí</td>
                    <td data-label="WAL-G">❌ No (se ejecuta localmente)</td>
                  </tr>
                  <tr>
                    <td>Funciones de equipo</td>
                    <td data-label="Databasus">
                      ✅ Espacios de trabajo, RBAC, registros de auditoría
                    </td>
                    <td data-label="WAL-G">
                      ❌ Solo permisos a nivel del sistema operativo
                    </td>
                  </tr>
                  <tr>
                    <td>Notificaciones</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, correo electrónico
                    </td>
                    <td data-label="WAL-G">
                      ❌ Requiere scripts personalizados
                    </td>
                  </tr>
                  <tr>
                    <td>Cifrado</td>
                    <td data-label="Databasus">AES-256-GCM integrado</td>
                    <td data-label="WAL-G">GPG o libsodium</td>
                  </tr>
                  <tr>
                    <td>Curva de aprendizaje</td>
                    <td data-label="Databasus">Mínima</td>
                    <td data-label="WAL-G">Requiere dominio de la CLI</td>
                  </tr>
                  <tr>
                    <td>Instalación</td>
                    <td data-label="Databasus">Script de una línea o Docker</td>
                    <td data-label="WAL-G">
                      Descarga del binario + configuración
                    </td>
                  </tr>
                  <tr>
                    <td>Apto para BD autoalojadas</td>
                    <td data-label="Databasus">✅ Sí</td>
                    <td data-label="WAL-G">✅ Sí</td>
                  </tr>
                  <tr>
                    <td>Apto para BD en la nube</td>
                    <td data-label="Databasus">
                      ✅ Sí (RDS, Cloud SQL, Azure)
                    </td>
                    <td data-label="WAL-G">
                      ❌ Solo respaldo (sin restauración a la nube)
                    </td>
                  </tr>
                </tbody>
              </table>

              <h2 id="database-focus">Enfoque en bases de datos</h2>

              <p>
                Una de las diferencias más significativas entre estas
                herramientas es su alcance en cuanto a bases de datos:
              </p>

              <h3 id="focus-databasus">
                Databasus: gestión integral de respaldos
              </h3>

              <p>
                Databasus está construido para la gestión integral de respaldos
                en múltiples sistemas de bases de datos, con énfasis en la
                facilidad de uso:
              </p>

              <ul>
                <li>
                  <strong>Soporte multibase</strong>: Gestione los respaldos de
                  PostgreSQL, MySQL, MariaDB y MongoDB desde una única interfaz.
                </li>
                <li>
                  <strong>Experiencia unificada</strong>: La interfaz, los
                  flujos de trabajo y las funciones se comportan igual en todas
                  las bases de datos compatibles.
                </li>
                <li>
                  <strong>Soporte de versiones</strong>: Admite las versiones 12
                  a 18 de PostgreSQL, con optimizaciones específicas por
                  versión.
                </li>
                <li>
                  <strong>Gestión simplificada</strong>: Todo el esfuerzo de
                  desarrollo se dedica a mejorar la experiencia de gestión de
                  respaldos.
                </li>
              </ul>

              <h3 id="focus-wal-g">WAL-G: soporte multibase</h3>

              <p>
                WAL-G comenzó como una herramienta de respaldo para PostgreSQL,
                pero se ha ampliado para admitir varios sistemas de bases de
                datos:
              </p>

              <ul>
                <li>
                  <strong>PostgreSQL</strong>: La implementación original y más
                  madura.
                </li>
                <li>
                  <strong>MySQL/MariaDB</strong>: Admite respaldos basados en
                  binlog.
                </li>
                <li>
                  <strong>MS SQL Server</strong>: Respaldos de SQL Server en
                  Windows.
                </li>
                <li>
                  <strong>MongoDB</strong>: Soporte de respaldo para bases de
                  datos documentales.
                </li>
                <li>
                  <strong>FoundationDB</strong>: Soporte para bases de datos
                  distribuidas.
                </li>
                <li>
                  <strong>Greenplum</strong>: Soporte de respaldo para almacenes
                  de datos.
                </li>
              </ul>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">
                    Cuando la gestión integral importa:
                  </strong>{" "}
                  Si necesita gestionar los respaldos de varias bases de datos
                  con una interfaz unificada, Databasus ofrece una experiencia
                  simplificada. Obtiene una gestión centralizada de respaldos,
                  con funciones de equipo, sin la complejidad de combinar
                  herramientas distintas para cada base de datos.
                </p>
              </div>

              <h2 id="target-audience">Público objetivo</h2>

              <p>
                Las herramientas sirven a perfiles de usuario distintos según su
                filosofía de diseño:
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
                  <strong>Entornos multibase</strong>: Las organizaciones que
                  ejecutan PostgreSQL, MySQL, MariaDB o MongoDB se benefician de
                  una gestión centralizada de respaldos.
                </li>
                <li>
                  <strong>DBA y recuperación ante desastres</strong>: Respaldos
                  físicos, archivado WAL y PITR para sistemas críticos con
                  requisitos de pérdida de datos casi nula.
                </li>
                <li>
                  <strong>Ingenieros de DevOps</strong>: El modo agente se
                  integra en la infraestructura existente, mientras que la
                  interfaz web y la API ofrecen visibilidad y control sin
                  scripts personalizados.
                </li>
              </ul>

              <h3 id="audience-wal-g">Público de WAL-G</h3>

              <p>
                WAL-G está diseñado para usuarios cómodos con herramientas de
                línea de comandos:
              </p>

              <ul>
                <li>
                  <strong>Ingenieros de DevOps</strong>: Quienes prefieren la
                  infraestructura como código y los flujos de trabajo basados en
                  CLI.
                </li>
                <li>
                  <strong>Entornos multibase</strong>: Organizaciones que
                  ejecutan PostgreSQL junto con MySQL, MongoDB u otras bases de
                  datos compatibles.
                </li>
                <li>
                  <strong>Despliegues nativos de la nube</strong>: Equipos que
                  usan Kubernetes o entornos en contenedores donde las
                  herramientas de CLI se integran bien.
                </li>
                <li>
                  <strong>Soporte ampliado de bases de datos</strong>: Equipos
                  que necesitan respaldos de MS SQL, FoundationDB o Greenplum
                  junto con PostgreSQL.
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

              <h3 id="backup-wal-g">
                WAL-G: respaldos físicos con archivado WAL
              </h3>

              <p>
                WAL-G realiza respaldos a nivel de archivos (físicos) con
                archivado WAL continuo:
              </p>

              <ul>
                <li>
                  <strong>Respaldos base</strong>: Copias completas a nivel de
                  archivos del directorio de datos de PostgreSQL.
                </li>
                <li>
                  <strong>Respaldos delta</strong>: Solo se respaldan las
                  páginas modificadas, lo que ahorra almacenamiento y tiempo de
                  transferencia.
                </li>
                <li>
                  <strong>Archivado WAL</strong>: El archivado continuo de los
                  Write-Ahead Logs permite la recuperación a un punto en el
                  tiempo.
                </li>
                <li>
                  <strong>Optimización copy-on-write</strong>: Manejo eficiente
                  de los bloques de datos sin cambios.
                </li>
              </ul>

              <h2 id="recovery-options">Opciones de recuperación</h2>

              <p>
                Ambas herramientas ofrecen capacidades de recuperación, pero con
                distinta granularidad:
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

              <h3 id="recovery-wal-g">Recuperación con WAL-G</h3>

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
                  <strong>Restauración delta</strong>: Recuperación más rápida
                  al obtener solo las páginas modificadas.
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
                  herramientas admiten PITR. WAL-G ofrece además restauración
                  delta (obteniendo solo las páginas modificadas) y usa un
                  protocolo de streaming propio con un rendimiento ligeramente
                  mejor a gran escala.{" "}
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
                Las herramientas difieren significativamente en su enfoque de la
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

              <h3 id="ease-wal-g">Experiencia de usuario de WAL-G</h3>

              <ul>
                <li>
                  <strong>Interfaz de línea de comandos</strong>: Todas las
                  operaciones se realizan mediante comandos de terminal como{" "}
                  <code>wal-g backup-push</code>,{" "}
                  <code>wal-g backup-fetch</code>.
                </li>
                <li>
                  <strong>Variables de entorno</strong>: La configuración se
                  hace principalmente mediante variables de entorno en lugar de
                  archivos de configuración.
                </li>
                <li>
                  <strong>Programación externa</strong>: Requiere tareas cron u
                  orquestación externa para automatizar los respaldos.
                </li>
                <li>
                  <strong>Configuración del archivado WAL</strong>: Hay que
                  configurar el <code>archive_command</code> de PostgreSQL para
                  integrarlo con WAL-G.
                </li>
                <li>
                  <strong>Se espera dominio de la CLI</strong>: La documentación
                  asume familiaridad con herramientas de línea de comandos y
                  scripts de shell.
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

              <h3 id="team-wal-g">Capacidades de equipo de WAL-G</h3>

              <p>
                WAL-G es una herramienta de línea de comandos sin funciones de
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
                  y políticas IAM de la nube
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

              <h3 id="security-wal-g">Seguridad de WAL-G</h3>

              <ul>
                <li>
                  <strong>Cifrado GPG</strong>: Admite cifrado basado en GPG
                  para los archivos de respaldo.
                </li>
                <li>
                  <strong>Cifrado con libsodium</strong>: Cifrado alternativo
                  mediante la biblioteca libsodium.
                </li>
                <li>
                  <strong>Integración con IAM de la nube</strong>: Aprovecha el
                  IAM del proveedor de nube para el control de acceso al
                  almacenamiento.
                </li>
                <li>
                  <strong>Sin gestión de credenciales integrada</strong>:
                  Depende de variables de entorno o de una gestión de secretos
                  externa.
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
                Ambas herramientas admiten almacenamiento en la nube, con
                distintos focos:
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

              <h3 id="storage-wal-g">Almacenamiento de WAL-G</h3>

              <p>Opciones de almacenamiento nativas de la nube:</p>

              <ul>
                <li>Amazon S3</li>
                <li>Google Cloud Storage (GCS)</li>
                <li>Azure Blob Storage</li>
                <li>Swift (OpenStack)</li>
                <li>Sistema de archivos local</li>
                <li>SSH/SFTP</li>
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

              <h3 id="notifications-wal-g">Notificaciones de WAL-G</h3>

              <p>
                WAL-G no tiene soporte de notificaciones integrado. Las
                notificaciones requieren:
              </p>

              <ul>
                <li>
                  Scripts personalizados alrededor de los comandos de respaldo
                </li>
                <li>Integración con herramientas de monitoreo externas</li>
                <li>Análisis manual de registros y configuración de alertas</li>
                <li>
                  Integración con herramientas como Prometheus, Grafana o
                  soluciones personalizadas
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

              <h2 id="compression">Compresión</h2>

              <p>
                Ambas herramientas ofrecen compresión para reducir el tamaño de
                los respaldos:
              </p>

              <h3 id="compression-databasus">Compresión de Databasus</h3>

              <ul>
                <li>
                  <strong>Compresión zstd</strong>: Usa zstd en nivel 5 para un
                  equilibrio entre velocidad y ratio de compresión.
                </li>
                <li>
                  <strong>Reducción de tamaño de 4 a 8 veces</strong>: Ratios de
                  compresión típicos con solo ~20% de sobrecarga en tiempo de
                  ejecución.
                </li>
                <li>
                  <strong>Automática</strong>: La compresión está habilitada por
                  defecto sin necesidad de configuración.
                </li>
              </ul>

              <h3 id="compression-wal-g">Compresión de WAL-G</h3>

              <ul>
                <li>
                  <strong>Múltiples algoritmos</strong>: Admite LZ4, LZMA,
                  Brotli y zstd.
                </li>
                <li>
                  <strong>Niveles configurables</strong>: Ajuste fino del
                  equilibrio entre ratio de compresión y velocidad.
                </li>
                <li>
                  <strong>Compresión por archivo</strong>: Los archivos WAL y
                  los respaldos base pueden usar ajustes distintos.
                </li>
              </ul>

              <h2 id="conclusion">Conclusión</h2>

              <p>
                Databasus y WAL-G cubren necesidades distintas dentro del
                ecosistema de copias de seguridad de PostgreSQL. La elección
                correcta depende de su entorno de bases de datos, la estructura
                de su equipo y sus preferencias operativas.
              </p>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">Elija Databasus si:</strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Necesita una gestión integral de respaldos de PostgreSQL
                    desde una única interfaz
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
                    Quiere una programación integrada sin configurar cron
                    externo
                  </li>
                  <li>
                    Quiere gestionar los respaldos de varias bases de datos
                    desde un único panel con programación, notificaciones y
                    funciones de equipo
                  </li>
                  <li>
                    Quiere una puesta en marcha rápida con mínima experiencia en
                    bases de datos
                  </li>
                  <li>
                    El cifrado de respaldos integrado es importante para usted
                  </li>
                  <li>
                    Usa bases de datos gestionadas en la nube (AWS RDS, Google
                    Cloud SQL, Azure) o bases de datos autoalojadas
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] px-4 pt-4 my-6">
                <p className="text-white m-0">
                  <strong>Elija WAL-G si:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Necesita respaldos físicos o incrementales en PostgreSQL
                    anterior a 17 (WAL-G incluye su propio motor de respaldo)
                  </li>
                  <li>
                    Necesita respaldos delta (solo páginas modificadas) para
                    reducir el almacenamiento y el tiempo de transferencia
                  </li>
                  <li>
                    Necesita soporte para MS SQL, FoundationDB o Greenplum
                  </li>
                  <li>
                    Prefiere herramientas de línea de comandos y flujos de
                    trabajo de infraestructura como código
                  </li>
                  <li>
                    Quiere múltiples algoritmos de compresión (LZ4, LZMA,
                    Brotli, zstd) con control fino
                  </li>
                  <li>
                    Su equipo tiene experiencia en DevOps para gestionar
                    herramientas basadas en CLI
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
                WAL-G sigue siendo una excelente opción para equipos que
                prefieren flujos de trabajo basados en CLI y necesitan sus
                ventajas particulares: respaldos delta (solo páginas
                modificadas), un protocolo de streaming propio con un
                rendimiento ligeramente mejor y soporte para motores de bases de
                datos adicionales más allá de PostgreSQL.
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
