import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Alternativa a pg_dump - Databasus, copia de seguridad de PostgreSQL",
  description:
    "Databasus se construye sobre pg_dump y amplía sus funciones con gestión de copias de seguridad, interfaz web, programación automática, almacenamiento en la nube, notificaciones, trabajo en equipo y cifrado.",
  keywords: [
    "alternativa a pg_dump",
    "pg_dump GUI",
    "automatización de pg_dump",
    "interfaz web para pg_dump",
    "herramienta de copia de seguridad de PostgreSQL",
    "programador de pg_dump",
    "pg_dump almacenamiento en la nube",
    "cifrado de pg_dump",
    "automatización de copias de seguridad de PostgreSQL",
    "wrapper de pg_dump",
  ],
  openGraph: {
    title:
      "Alternativa a pg_dump - Databasus, copia de seguridad de PostgreSQL",
    description:
      "Databasus se construye sobre pg_dump y amplía sus funciones con gestión de copias de seguridad, interfaz web, programación automática, almacenamiento en la nube, notificaciones, trabajo en equipo y cifrado.",
    type: "article",
    url: getLocalizedUrl("es", "pgdump-alternative"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title:
      "Alternativa a pg_dump - Databasus, copia de seguridad de PostgreSQL",
    description:
      "Databasus se construye sobre pg_dump y amplía sus funciones con gestión de copias de seguridad, interfaz web, programación automática, almacenamiento en la nube, notificaciones, trabajo en equipo y cifrado.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "pgdump-alternative"),
    languages: getLanguageAlternates("pgdump-alternative"),
  },
  robots: "index, follow",
};

export default function PgDumpAlternativePage() {
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
              "Alternativa a pg_dump - Databasus, copia de seguridad de PostgreSQL",
            description:
              "Una guía completa de Databasus como alternativa a pg_dump: cómo se construye sobre pg_dump y amplía sus capacidades con automatización, almacenamiento en la nube, notificaciones y funciones para equipos.",
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
              <h1 id="pgdump-alternative">Alternativa a pg_dump</h1>

              <p className="text-lg text-gray-400">
                Para las copias de seguridad lógicas, Databasus se construye
                sobre <code>pg_dump</code>. En lugar de reemplazar{" "}
                <code>pg_dump</code>, Databasus amplía sus capacidades con
                gestión de copias de seguridad, interfaz web, programación
                automática, integración con almacenamiento en la nube,
                notificaciones, funciones de trabajo en equipo y cifrado
                integrado. Más allá de los respaldos lógicos, Databasus también
                admite copias físicas, copias incrementales con archivado de WAL
                y recuperación a un punto en el tiempo (Point-in-Time Recovery).
              </p>

              <h2 id="quick-comparison">Comparación rápida</h2>

              <p>
                Este es un resumen de cómo Databasus amplía la funcionalidad
                básica de <code>pg_dump</code>:
              </p>

              <table>
                <thead>
                  <tr>
                    <th>Función</th>
                    <th>pg_dump</th>
                    <th>Databasus</th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td>Motor de copias</td>
                    <td data-label="pg_dump">pg_dump</td>
                    <td data-label="Databasus">Construido sobre pg_dump</td>
                  </tr>
                  <tr>
                    <td>Gestión de copias de seguridad</td>
                    <td data-label="pg_dump">❌ No</td>
                    <td data-label="Databasus">✅ Sí</td>
                  </tr>
                  <tr>
                    <td>Soporte de otras bases de datos</td>
                    <td data-label="pg_dump">Solo PostgreSQL</td>
                    <td data-label="Databasus">
                      PostgreSQL, MySQL, MariaDB, MongoDB
                    </td>
                  </tr>
                  <tr>
                    <td>Interfaz</td>
                    <td data-label="pg_dump">Línea de comandos</td>
                    <td data-label="Databasus">Interfaz web + API</td>
                  </tr>
                  <tr>
                    <td>Programación</td>
                    <td data-label="pg_dump">Manual o scripts de cron</td>
                    <td data-label="Databasus">✅ Programador integrado</td>
                  </tr>
                  <tr>
                    <td>Destinos de almacenamiento</td>
                    <td data-label="pg_dump">Sistema de archivos local</td>
                    <td data-label="Databasus">
                      Local, S3, R2, Azure, NAS, FTP
                    </td>
                  </tr>
                  <tr>
                    <td>Compresión</td>
                    <td data-label="pg_dump">gzip, LZ4, zstd (manual)</td>
                    <td data-label="Databasus">
                      zstd (automática, optimizada)
                    </td>
                  </tr>
                  <tr>
                    <td>Cifrado</td>
                    <td data-label="pg_dump">Requiere herramientas externas</td>
                    <td data-label="Databasus">✅ AES-256-GCM integrado</td>
                  </tr>
                  <tr>
                    <td>Notificaciones</td>
                    <td data-label="pg_dump">❌ Ninguna</td>
                    <td data-label="Databasus">
                      ✅ Slack, Teams, Telegram, correo, webhooks
                    </td>
                  </tr>
                  <tr>
                    <td>Funciones para equipos</td>
                    <td data-label="pg_dump">❌ Ninguna</td>
                    <td data-label="Databasus">
                      ✅ Espacios de trabajo, RBAC, registros de auditoría
                    </td>
                  </tr>
                  <tr>
                    <td>Políticas de retención</td>
                    <td data-label="pg_dump">Scripts de limpieza manuales</td>
                    <td data-label="Databasus">✅ Retención automática</td>
                  </tr>
                  <tr>
                    <td>Monitorización de salud</td>
                    <td data-label="pg_dump">❌ Ninguna</td>
                    <td data-label="Databasus">
                      ✅ Comprobaciones de salud integradas
                    </td>
                  </tr>
                  <tr>
                    <td>Copias físicas</td>
                    <td data-label="pg_dump">❌ No</td>
                    <td data-label="Databasus">✅ Sí</td>
                  </tr>
                  <tr>
                    <td>Copias incrementales</td>
                    <td data-label="pg_dump">❌ No</td>
                    <td data-label="Databasus">
                      ✅ A nivel de bloque (PG 17+)
                    </td>
                  </tr>
                  <tr>
                    <td>Point-in-Time Recovery</td>
                    <td data-label="pg_dump">❌ No</td>
                    <td data-label="Databasus">✅ Sí</td>
                  </tr>
                  <tr>
                    <td>Copias remotas</td>
                    <td data-label="pg_dump">✅ Sí (CLI)</td>
                    <td data-label="Databasus">✅ Sí</td>
                  </tr>
                </tbody>
              </table>

              <h2 id="what-is-pgdump">¿Qué es pg_dump?</h2>

              <p>
                <code>pg_dump</code> es la utilidad nativa de PostgreSQL para
                crear copias de seguridad lógicas. Forma parte de PostgreSQL
                desde el principio y es la herramienta estándar para exportar
                bases de datos.
              </p>

              <h3 id="pgdump-strengths">Puntos fuertes de pg_dump</h3>

              <ul>
                <li>
                  <strong>Copias portables</strong>: crea volcados en formato
                  SQL o personalizado que pueden restaurarse en distintas
                  versiones de PostgreSQL.
                </li>
                <li>
                  <strong>Copias selectivas</strong>: puede exportar tablas o
                  esquemas concretos, o bases de datos completas.
                </li>
                <li>
                  <strong>Instantáneas consistentes</strong>: usa el MVCC de
                  PostgreSQL para crear copias consistentes sin bloquear las
                  escrituras.
                </li>
                <li>
                  <strong>Soporte universal</strong>: disponible en cualquier
                  instalación de PostgreSQL, bien documentado y probado durante
                  años.
                </li>
                <li>
                  <strong>Formatos de salida flexibles</strong>: SQL plano,
                  personalizado, directorio o tar.
                </li>
              </ul>

              <h3 id="pgdump-limitations">Limitaciones de pg_dump</h3>

              <p>
                Aunque <code>pg_dump</code> es potente, usarlo en producción
                suele requerir scripts adicionales:
              </p>

              <ul>
                <li>
                  <strong>Sin programación integrada</strong>: requiere tareas
                  de cron o programadores externos.
                </li>
                <li>
                  <strong>Solo almacenamiento local</strong>: escribe en el
                  sistema de archivos local; subir a la nube requiere scripts
                  adicionales.
                </li>
                <li>
                  <strong>Sin cifrado</strong>: los archivos de respaldo no
                  están cifrados por defecto; hay que encadenar la salida con
                  gpg o herramientas similares.
                </li>
                <li>
                  <strong>Sin notificaciones</strong>: no hay forma de recibir
                  avisos de éxito o fallo sin scripts propios.
                </li>
                <li>
                  <strong>Sin gestión de retención</strong>: las copias antiguas
                  deben limpiarse a mano o con scripts.
                </li>
                <li>
                  <strong>Solo línea de comandos</strong>: sin interfaz visual
                  para monitorizar o administrar.
                </li>
              </ul>

              <h2 id="how-databasus-extends">Cómo Databasus amplía pg_dump</h2>

              <p>
                Databasus usa <code>pg_dump</code> como motor de copias:
                conserva todas las ventajas de los respaldos lógicos y añade
                funciones empresariales.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-gray-300 m-0">
                  <strong className="text-amber-400">Bajo el capó:</strong>{" "}
                  cuando lanza una copia de seguridad en Databasus, este ejecuta{" "}
                  <code className="bg-[#374151] text-gray-200">pg_dump</code>{" "}
                  con parámetros optimizados y luego se encarga de la
                  compresión, el cifrado y la subida al destino de
                  almacenamiento configurado.
                </p>
              </div>

              <h3 id="web-interface">Interfaz web</h3>

              <p>
                En lugar de memorizar las opciones de línea de comandos de{" "}
                <code>pg_dump</code>, Databasus ofrece una interfaz web donde
                puede:
              </p>

              <ul>
                <li>
                  Añadir bases de datos con un asistente de conexión guiado
                </li>
                <li>Configurar horarios de copia con controles visuales</li>
                <li>
                  Ver el historial y el estado de las copias de un vistazo
                </li>
                <li>Descargar o restaurar copias con un clic</li>
                <li>
                  Consultar gráficos de salud y disponibilidad de las bases de
                  datos
                </li>
              </ul>

              <h3 id="optimized-compression">Compresión optimizada</h3>

              <p>
                Databasus usa por defecto compresión zstd (nivel 5), que ofrece:
              </p>

              <ul>
                <li>
                  <strong>Archivos de 4 a 8 veces más pequeños</strong> que los
                  volcados sin comprimir
                </li>
                <li>
                  <strong>~20% de sobrecoste en tiempo de ejecución</strong>,
                  mucho más rápido que gzip
                </li>
                <li>
                  <strong>Gestión automática</strong>, sin necesidad de
                  encadenar herramientas de compresión
                </li>
              </ul>

              <h2 id="beyond-pgdump">
                Más allá de pg_dump: copias físicas y PITR
              </h2>

              <p>
                Aunque Databasus se apoya en <code>pg_dump</code> para los
                respaldos lógicos, también va más allá de lo que{" "}
                <code>pg_dump</code> puede ofrecer:
              </p>

              <ul>
                <li>
                  <strong>Copias físicas</strong>: copias a nivel de archivos de
                  todo el clúster de la base de datos mediante{" "}
                  <code>pg_basebackup</code>. Copia y restauración más rápidas
                  para bases de datos grandes.
                </li>
                <li>
                  <strong>Copias incrementales y de WAL</strong>: copias
                  incrementales a nivel de bloque mediante{" "}
                  <code>pg_basebackup --incremental</code> (basadas en los
                  resúmenes de WAL del servidor) más streaming continuo de WAL
                  mediante <code>pg_receivewal</code>, lo que habilita la
                  recuperación a un punto en el tiempo: restaure a cualquier
                  segundo entre copias.
                </li>
                <li>
                  <strong>Recuperación ante desastres</strong>: diseñado para
                  requisitos de pérdida de datos casi nula con copias base
                  físicas y streaming continuo de WAL.
                </li>
              </ul>

              <p>
                Estas copias se apoyan en el mecanismo de respaldo nativo de
                PostgreSQL 17, de modo que Databasus reutiliza las herramientas
                probadas de PostgreSQL en lugar de reinventarlas. Requieren
                PostgreSQL 17 o superior; en versiones anteriores solo están
                disponibles las copias lógicas con <code>pg_dump</code>. Todo se
                ejecuta de forma remota desde el host de Databasus a través del
                protocolo de replicación, así que no se instala nada en el
                servidor de la base de datos. A las redes cerradas se llega
                mediante un túnel SSH hacia un host interno o un bastión, por lo
                que la base de datos nunca tiene que exponerse públicamente.{" "}
                <a
                  href="/es/faq#pitr"
                  className="text-blue-400 hover:text-blue-600"
                >
                  Lea cómo funcionan las copias físicas y PITR
                </a>
                .
              </p>

              <h2 id="backup-automation">Automatización de copias</h2>

              <p>
                Uno de los retos más habituales con <code>pg_dump</code> es
                montar copias automáticas fiables.
              </p>

              <h3 id="automation-pgdump">
                Automatización tradicional con pg_dump
              </h3>

              <p>
                Un script típico de automatización de <code>pg_dump</code> puede
                parecerse a esto:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{`#!/bin/bash
# Backup script for pg_dump
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
DB_NAME="mydb"

# Create backup
pg_dump -Fc -h localhost -U postgres $DB_NAME > $BACKUP_DIR/$DB_NAME_$DATE.dump

# Compress (if not using custom format)
# gzip $BACKUP_DIR/$DB_NAME_$DATE.sql

# Encrypt
gpg --encrypt --recipient backup@company.com $BACKUP_DIR/$DB_NAME_$DATE.dump

# Upload to S3
aws s3 cp $BACKUP_DIR/$DB_NAME_$DATE.dump.gpg s3://my-bucket/backups/

# Cleanup old backups (keep last 7 days)
find $BACKUP_DIR -name "*.dump*" -mtime +7 -delete

# Send notification on failure
if [ $? -ne 0 ]; then
  curl -X POST https://hooks.slack.com/... -d '{"text":"Backup failed!"}'
fi`}</code>
                </pre>
              </div>

              <p>
                Este script hay que mantenerlo, probarlo y monitorizarlo. Cada
                base de datos requiere su propia entrada de cron.
              </p>

              <h3 id="automation-databasus">Automatización con Databasus</h3>

              <p>Con Databasus, la misma funcionalidad viene integrada:</p>

              <ul>
                <li>
                  <strong>Programador visual</strong>: configure copias cada
                  hora, diarias, semanales, mensuales o con cron a horas
                  concretas.
                </li>
                <li>
                  <strong>Compresión automática</strong>: la compresión zstd se
                  aplica automáticamente.
                </li>
                <li>
                  <strong>Cifrado integrado</strong>: cifrado AES-256-GCM con
                  claves únicas por copia.
                </li>
                <li>
                  <strong>Subida a la nube</strong>: subida directa a S3, Cloudflare R2, Azure y otros destinos.
                </li>
                <li>
                  <strong>Políticas de retención</strong>: limpieza automática
                  de copias antiguas según su configuración de retención.
                </li>
                <li>
                  <strong>Notificaciones</strong>: avisos a Slack, Teams,
                  Telegram o correo electrónico en caso de éxito o fallo.
                </li>
              </ul>

              <h2 id="storage-options">Opciones de almacenamiento</h2>

              <p>
                <code>pg_dump</code> escribe en el sistema de archivos local.
                Llevar las copias a un almacenamiento en la nube requiere
                herramientas y scripts adicionales.
              </p>

              <h3 id="storage-databasus">
                Destinos de almacenamiento de Databasus
              </h3>

              <p>
                Databasus admite varios destinos de almacenamiento de serie:
              </p>

              <ul>
                <li>Almacenamiento local</li>
                <li>Amazon S3 y servicios compatibles con S3</li>
                <li>Cloudflare R2</li>
                <li>Azure Blob Storage</li>
                <li>NAS (almacenamiento conectado en red)</li>
                <li>FTP</li>
              </ul>

              <p>
                Cada base de datos puede tener su propio destino de
                almacenamiento y puede configurar varios destinos para
                redundancia.
              </p>

              <p>
                <a
                  href="/es/storages"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Ver todas las opciones de almacenamiento →
                </a>
              </p>

              <h2 id="notifications">Notificaciones</h2>

              <p>
                Saber cuándo las copias tienen éxito o fallan es fundamental
                para proteger los datos.
              </p>

              <h3 id="notifications-pgdump">Notificaciones con pg_dump</h3>

              <p>
                <code>pg_dump</code> no tiene sistema de notificaciones.
                Necesita:
              </p>

              <ul>
                <li>
                  Escribir scripts envoltorio que comprueben los códigos de
                  salida
                </li>
                <li>Integrarse con herramientas de monitorización externas</li>
                <li>Montar sus propios canales de alertas</li>
              </ul>

              <h3 id="notifications-databasus">Notificaciones de Databasus</h3>

              <p>Databasus incluye notificaciones integradas para:</p>

              <ul>
                <li>Slack</li>
                <li>Discord</li>
                <li>Telegram</li>
                <li>Microsoft Teams</li>
                <li>Correo electrónico</li>
                <li>Webhooks (para integraciones personalizadas)</li>
              </ul>

              <p>
                Configure qué eventos disparan notificaciones: copia con éxito,
                copia fallida o ambos.
              </p>

              <p>
                <a
                  href="/es/notifiers"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Ver todos los canales de notificación →
                </a>
              </p>

              <h2 id="team-features">Funciones para equipos</h2>

              <p>
                <code>pg_dump</code> es una herramienta de línea de comandos
                para un solo usuario. Databasus añade funciones de colaboración
                para equipos:
              </p>

              <h3 id="team-databasus">Capacidades de Databasus para equipos</h3>

              <ul>
                <li>
                  <strong>Espacios de trabajo</strong>: organice bases de datos,
                  notificadores y almacenamientos por proyecto o equipo. Los
                  usuarios solo ven los espacios de trabajo a los que fueron
                  invitados.
                </li>
                <li>
                  <strong>Control de acceso basado en roles</strong>: asigne
                  permisos de lector, editor o administrador para controlar lo
                  que puede hacer cada miembro del equipo.
                </li>
                <li>
                  <strong>Registros de auditoría</strong>: siga todas las
                  actividades y cambios del sistema. Esencial para el
                  cumplimiento de seguridad y la trazabilidad.
                </li>
                <li>
                  <strong>Notificaciones compartidas</strong>: los canales del
                  equipo reciben automáticamente el estado de las copias.
                </li>
              </ul>

              <p>
                <a
                  href="/es/access-management"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Más sobre la gestión de accesos →
                </a>
              </p>

              <h2 id="security">Seguridad</h2>

              <p>
                La seguridad es donde Databasus aporta más valor frente al uso
                directo de <code>pg_dump</code>.
              </p>

              <h3 id="security-pgdump">Seguridad con pg_dump</h3>

              <p>
                <code>pg_dump</code> crea archivos de respaldo sin cifrar.
                Protegerlos requiere:
              </p>

              <ul>
                <li>
                  Encadenar la salida con herramientas de cifrado (gpg, openssl)
                </li>
                <li>Gestionar las claves de cifrado por separado</li>
                <li>
                  Garantizar el almacenamiento seguro y la rotación de las
                  claves
                </li>
                <li>Configurar permisos de archivo adecuados</li>
              </ul>

              <h3 id="security-databasus">Seguridad de Databasus</h3>

              <p>Databasus implementa la seguridad en varios niveles:</p>

              <ul>
                <li>
                  <strong>Cifrado AES-256-GCM</strong>: todas las contraseñas,
                  tokens y credenciales están cifrados. La clave de cifrado se
                  guarda separada de la base de datos.
                </li>
                <li>
                  <strong>Cifrado único por copia</strong>: cada archivo de
                  respaldo se cifra con una clave única derivada de la clave
                  maestra, el ID de la copia y una sal aleatoria.
                </li>
                <li>
                  <strong>Acceso de solo lectura a la base de datos</strong>:
                  exige permisos solo de SELECT; así, incluso si la cuenta se ve
                  comprometida, los datos no pueden corromperse.
                </li>
              </ul>

              <p>
                <a
                  href="/es/security"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Más sobre la seguridad de Databasus →
                </a>
              </p>

              <h2 id="restore-process">Proceso de restauración</h2>

              <p>
                Ambas herramientas permiten restaurar copias, pero con flujos
                distintos.
              </p>

              <h3 id="restore-pgdump">Restaurar copias de pg_dump</h3>

              <p>
                Restaurar una copia de <code>pg_dump</code> requiere:
              </p>

              <ol>
                <li>Localizar el archivo de respaldo</li>
                <li>Descifrarlo si está cifrado</li>
                <li>Descomprimirlo si está comprimido</li>
                <li>
                  Ejecutar <code>pg_restore</code> o <code>psql</code> con los
                  parámetros correctos
                </li>
              </ol>

              <h3 id="restore-databasus">Restaurar copias de Databasus</h3>

              <p>Databasus simplifica la restauración:</p>

              <ul>
                <li>
                  <strong>Descarga con un clic</strong>: descargue cualquier
                  copia directamente desde la interfaz web.
                </li>
                <li>
                  <strong>Descifrado automático</strong>: las copias se
                  descifran automáticamente al descargarlas.
                </li>
                <li>
                  <strong>Comandos de restauración listos</strong>: Databasus
                  muestra el comando <code>pg_restore</code> exacto para cada
                  copia.
                </li>
                <li>
                  <strong>Restauración en paralelo</strong>: aproveche varios
                  núcleos de CPU para restaurar más rápido bases de datos
                  grandes.
                </li>
              </ul>

              <h2 id="installation">Instalación</h2>

              <h3 id="install-pgdump">Instalación de pg_dump</h3>

              <p>
                <code>pg_dump</code> viene con PostgreSQL. Si tiene PostgreSQL
                instalado, ya tiene <code>pg_dump</code>.
              </p>

              <h3 id="install-databasus">Instalación de Databasus</h3>

              <p>Databasus ofrece varios métodos de instalación:</p>

              <ul>
                <li>
                  <strong>Script de una línea</strong>: instala Docker (si hace
                  falta), configura Databasus y activa el arranque automático.
                </li>
                <li>
                  <strong>Docker run</strong>: un solo comando para arrancar con
                  PostgreSQL incorporado.
                </li>
                <li>
                  <strong>Docker Compose</strong>: para tener más control sobre
                  el despliegue.
                </li>
              </ul>

              <p>
                <a
                  href="/es/installation"
                  className="font-semibold text-blue-600 hover:text-blue-800"
                >
                  Ver la guía de instalación →
                </a>
              </p>

              <h2 id="conclusion">Conclusión</h2>

              <p>
                <code>pg_dump</code> es la utilidad de respaldo probada de
                PostgreSQL, y Databasus se construye directamente sobre ella.
                Elegir entre usar <code>pg_dump</code> directamente o a través
                de Databasus depende de sus necesidades.
              </p>

              <div className="rounded-lg border border-[#ffffff20] bg-[#1f2937] p-4 my-6">
                <p className="text-white m-0">
                  <strong>Use pg_dump directamente si:</strong>
                </p>
                <ul className="text-white mb-0">
                  <li>
                    Necesita exportaciones puntuales o bajo demanda de la base
                    de datos
                  </li>
                  <li>
                    Se siente cómodo escribiendo y manteniendo scripts de shell
                  </li>
                  <li>
                    Ya tiene infraestructura de automatización (Ansible,
                    Terraform, etc.)
                  </li>
                  <li>
                    Solo necesita copias locales sin almacenamiento en la nube
                  </li>
                  <li>
                    Es un desarrollador que trabaja solo y con necesidades
                    simples
                  </li>
                </ul>
              </div>

              <div className="rounded-lg border border-blue-500/30 bg-blue-500/10 p-4 my-6">
                <p className="text-blue-300 m-0">
                  <strong className="text-blue-400">Use Databasus si:</strong>
                </p>
                <ul className="text-blue-200 mb-0">
                  <li>
                    Quiere copias automáticas y programadas sin escribir scripts
                  </li>
                  <li>
                    Necesita guardar las copias en la nube (S3, Azure Blob Storage,
                    etc.)
                  </li>
                  <li>Quiere cifrado integrado sin gestionar claves a mano</li>
                  <li>
                    Necesita notificaciones cuando las copias tienen éxito o
                    fallan
                  </li>
                  <li>
                    Trabaja en equipo y necesita funciones de colaboración
                  </li>
                  <li>
                    Prefiere una interfaz visual antes que herramientas de línea
                    de comandos
                  </li>
                  <li>Quiere políticas de retención y limpieza automáticas</li>
                  <li>
                    Necesita copias físicas, copias incrementales o
                    Point-in-Time Recovery para la recuperación ante desastres
                  </li>
                </ul>
              </div>

              <p>
                Databasus se construye sobre <code>pg_dump</code> para los
                respaldos lógicos y lo amplía con automatización, seguridad y
                funciones para equipos. Además admite copias físicas, copias
                incrementales con archivado de WAL y Point-in-Time Recovery,
                capacidades que <code>pg_dump</code> sencillamente no puede
                ofrecer.
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
