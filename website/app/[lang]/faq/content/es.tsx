import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedHref, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";
import { CopyButton } from "@/app/components/CopyButton";

export const metadata: Metadata = {
  title: "FAQ - Preguntas frecuentes | Databasus",
  description:
    "Preguntas frecuentes sobre Databasus, la herramienta de copia de seguridad de PostgreSQL con soporte de MySQL, MariaDB y MongoDB. Aprenda a respaldar bases de datos en localhost y conozca los formatos de copia, los métodos de compresión y más.",
  keywords: [
    "Databasus FAQ",
    "preguntas sobre copias de seguridad de PostgreSQL",
    "copia de seguridad de base de datos en localhost",
    "formatos de copia de seguridad",
    "compresión de pg_dump",
    "compresión zstd",
    "ayuda con copias de seguridad de PostgreSQL",
    "guía de respaldo de bases de datos",
  ],
  openGraph: {
    title: "FAQ - Preguntas frecuentes | Databasus",
    description:
      "Preguntas frecuentes sobre Databasus, la herramienta de copia de seguridad de PostgreSQL con soporte de MySQL, MariaDB y MongoDB. Aprenda a respaldar bases de datos en localhost y conozca los formatos de copia, los métodos de compresión y más.",
    type: "article",
    url: getLocalizedUrl("es", "faq"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "FAQ - Preguntas frecuentes | Databasus",
    description:
      "Preguntas frecuentes sobre Databasus, la herramienta de copia de seguridad de PostgreSQL con soporte de MySQL, MariaDB y MongoDB. Aprenda a respaldar bases de datos en localhost y conozca los formatos de copia, los métodos de compresión y más.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "faq"),
    languages: getLanguageAlternates("faq"),
  },
  robots: "index, follow",
};

export default function FAQPage() {
  const startupRecoveryCommand = `docker run -d \\
  --name databasus \\
  -p 4005:4005 \\
  -v ./databasus-data:/databasus-data \\
  -e RECOVERY_DUMP_PATH=/databasus-data/recovery/backup.dbbackup \\
  -e RECOVERY_MASTER_KEY_PATH=/databasus-data/recovery/master.key \\
  --restart unless-stopped \\
  databasus/databasus:<version>`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "FAQPage",
            mainEntity: [
              {
                "@type": "Question",
                name: "¿Por qué Databasus no usa el formato de volcado SQL plano para las copias de seguridad lógicas de PostgreSQL?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Para las copias lógicas, Databasus usa el formato personalizado de pg_dump con compresión zstd de nivel 5 porque, tras pruebas exhaustivas, resultó ser el equilibrio óptimo entre velocidad de creación de la copia, velocidad de restauración y tamaño del archivo.",
                },
              },
              {
                "@type": "Question",
                name: "¿Dónde se instala Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus se instala en /opt/databasus/",
                },
              },
              {
                "@type": "Question",
                name: "¿Cómo funcionan las copias de seguridad físicas y PITR (Point-in-Time Recovery)?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus ejecuta las copias físicas de forma remota desde su propio host, conectándose a su PostgreSQL por el protocolo de replicación estándar, así que no hace falta instalar nada en el servidor de la base de datos. A las bases de datos en redes cerradas se llega mediante un túnel SSH. Las copias físicas usan la pila nativa de PostgreSQL 17: copias completas con pg_basebackup, incrementales a nivel de bloque con pg_basebackup --incremental basadas en los resúmenes de WAL del servidor (summarize_wal = on) y streaming continuo de WAL con pg_receivewal. Las copias físicas requieren PostgreSQL 17 o superior; en versiones anteriores se usan las copias lógicas con pg_dump. Para restaurar a un punto en el tiempo, pg_combinebackup reconstruye un directorio de datos ejecutable a partir de la copia completa y su cadena incremental, y PostgreSQL luego reproduce el WAL hasta el momento objetivo que elija, de modo que puede restaurar a cualquier segundo entre copias. La interfaz de Databasus da instrucciones paso a paso para restaurar a un host o a una base de datos en Docker, ya sea mediante un script listo que convierte la restauración en un solo comando o descargando las copias y reconstruyendo la cadena de partes completas, incrementales y WAL. Las incrementales y el WAL son opcionales: puede tomar solo una copia completa, y el WAL no es obligatorio. Usamos las copias nativas de PostgreSQL 17 porque reutilizan la maquinaria de respaldo probada de PostgreSQL en lugar de reinventarla y dan una pérdida de datos casi nula.",
                },
              },
              {
                "@type": "Question",
                name: "¿Por qué Databasus abandonó las copias de seguridad basadas en agente?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Una versión anterior de Databasus incluía un agente de respaldo: un binario que se ejecutaba en el host de la base de datos para transmitir WAL y crear copias físicas localmente. Esa primera implementación resultó ser un error y fue eliminada. Era una implementación ingenua que solo copiaba WAL sobre copias completas, lo que llevaba a un RTO largo. Los usuarios tenían que configurar tanto Databasus como un agente separado, cuando hacerlo todo de forma remota desde un solo lugar es mucho más simple. Como el agente vivía fuera del sistema principal, era difícil cubrir todos los casos de prueba. En realidad solo hay un problema que un agente resuelve: alcanzar una base de datos no accesible desde fuera, y para el 99% de los usuarios eso ya se resuelve ejecutando Databasus dentro de la red privada o conectando por SSH, así que el agente reinventaba la rueda y complicaba mucho un problema simple. Tampoco podía ejecutarse en bases de datos gestionadas como RDS y Cloud SQL, que prohíben instalar software en el host pero ya exponen el protocolo de replicación, así que de todos modos hacía falta una vía remota. Además traía muchos casos límite en torno a conexiones rotas, gestión de actualizaciones del agente y recopilación de registros de un proceso separado, y cuantas menos piezas móviles tiene un sistema, más fiable es en el uso diario. Las copias físicas ahora se ejecutan de forma remota desde el host de Databasus. Las copias existentes están a salvo: si actualiza desde una versión que aún tiene copias de agente, Databasus no lo hará en silencio, sino que le avisará del cambio y le dejará quedarse en la versión soportada 3.42.0 o eliminar las copias de agente antiguas antes de actualizar. La implementación basada en agente sigue disponible hasta la versión 3.42.0 y seguirá funcionando durante mucho tiempo.",
                },
              },
              {
                "@type": "Question",
                name: "¿Cómo respaldar el propio Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "Databasus está pensado para poder recuperarse aunque se pierda o se corrompa su propio servidor. Puede hacer copias de seguridad de su propia base de datos interna: cuentas, ajustes, las credenciales cifradas de bases de datos, almacenamientos y canales de notificación, programaciones y el catálogo de copias. Actívelo en Databasus settings → Databasus backups eligiendo uno o varios almacenamientos existentes; por defecto, la copia de seguridad de Databasus se ejecuta cada hora y se conserva 30 días. El archivo se cifra con la clave maestra y no contiene ni la clave ni los archivos de copia de sus bases de datos, así que guarde la clave maestra por separado: Databasus le pide que la descargue, y también está en . /databasus-data/secret. key en el host. La recuperación necesita el archivo de copia, su clave maestra y exactamente la misma versión de Databasus, incluido el parche. Puede poner ambos archivos en el directorio de datos montado y arrancar Databasus con RECOVERY_DUMP_PATH y RECOVERY_MASTER_KEY_PATH en un servidor sin cuentas ni ajustes cambiados, o bien subir el archivo y pegar la clave en Databasus settings → Databasus backups, lo que reemplaza la instalación actual.",
                },
              },
              {
                "@type": "Question",
                name: "¿Cómo apoyan a Databasus los programas de código abierto de Anthropic y OpenAI?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "En marzo de 2026, Databasus fue aceptado tanto en Claude for Open Source de Anthropic como en Codex for Open Source de OpenAI. Ambos programas evalúan los proyectos de forma independiente antes de aceptarlos.",
                },
              },
              {
                "@type": "Question",
                name: "¿Cómo se utiliza la IA en el desarrollo de Databasus?",
                acceptedAnswer: {
                  "@type": "Answer",
                  text: "La IA nos ayuda a desarrollar Databasus, desde la planificación y la implementación hasta las pruebas y la documentación. Las personas definen el rumbo y toman las decisiones finales. Una persona lee cada línea de código antes de integrar un cambio. Utilizamos desarrollo basado en especificaciones (SDD) con OpenSpec para acordar el comportamiento antes de implementar. Revisión independiente en cada etapa Los hooks automáticos inician sesiones de revisión separadas con acceso de solo lectura al repositorio. Cada agente inspecciona el código pertinente y devuelve sus propios hallazgos. El agente que implementa no puede aprobar su propio trabajo. La revisión empieza antes de implementar. Agentes de planificación separados buscan soluciones más sencillas, comprueban los nombres propuestos y evalúan el diseño. También contrastan los requisitos con las tareas de verificación. Revisor de código: Busca errores, casos límite omitidos, fallos en el manejo de errores, condiciones de carrera, fugas de recursos y problemas de seguridad. Revisor de requisitos: Localiza la implementación y las pruebas de cada requisito. Comprueba que las pruebas demuestren el comportamiento prometido. Revisor de arquitectura: Comprueba los límites entre módulos, la dirección de las dependencias, la propiedad de los datos, las migraciones y las reglas de API. Revisor de estructura del código: Comprueba nombres, responsabilidades de archivos y funciones, comentarios y legibilidad. Revisor de pruebas: Evalúa si las pruebas detectarían un comportamiento ausente o una regresión. También comprueba las esperas por condiciones, la limpieza y las evidencias de verificación. Revisor de documentación y tareas: Contrasta la documentación con el código, comprueba que las tareas estén completas y verifica las evidencias de las comprobaciones declaradas. Las revisiones completas de implementación se ejecutan en paralelo. Cada especificación modificada tiene su propio revisor de requisitos. Arquitectura, pruebas y documentación tienen revisores separados. Un cambio que abarca dos especificaciones recibe, por tanto, cinco agentes de revisión completa. Los hallazgos bloqueantes impiden aceptar el trabajo. El agente que implementa los corrige o explica por qué un hallazgo es incorrecto. Un revisor posterior comprueba esa resolución y los cambios posteriores. Una revisión que no pudo ejecutarse no se considera aprobada. Comprobaciones automáticas y restauraciones reales La revisión independiente se complementa con comprobaciones automáticas estrictas. golangci-lint y nuestro analizador de arquitectura comprueban las dependencias del backend, los límites entre módulos y las reglas de migración. Las comprobaciones de API validan los contratos OpenAPI y el código generado. El frontend comprueba TypeScript, React, importaciones entre capas, estilos y localización. Ruff y ty comprueban el formato, las reglas de código y los tipos de Python. ShellCheck comprueba los scripts de shell y actionlint, los flujos de GitHub Actions. Los hooks previos al commit comprueban los archivos afectados del backend, frontend y Python sin reescribirlos. CI también ejecuta el análisis estático. Las pruebas comprueban comportamientos reales. El backend se prueba mediante la API de la aplicación y los motores, con bases de datos reales en contenedores. Los escenarios de extremo a extremo crean copias de seguridad, las restauran y comprueban los datos restaurados. Otras pruebas cubren actualizaciones, pantallas del frontend y flujos del navegador con comparaciones visuales. Las comprobaciones se eligen según el cambio. Las suites obligatorias de publicación se ejecutan en amd64 y arm64. El agente que implementa ejecuta los analizadores y las pruebas pertinentes antes de la revisión. Los revisores inspeccionan las evidencias sin volver a ejecutar esos comandos. Antes de publicar, CodeQL y Semgrep analizan el código, Gitleaks busca secretos y las dependencias se comprueban para detectar vulnerabilidades. Trivy analiza las imágenes de contenedor y los Dockerfile. Las comprobaciones obligatorias de CI deben superarse antes de publicar una versión.",
                },
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
              <h1 id="faq">Preguntas frecuentes</h1>

              <p className="text-lg text-gray-400">
                Encuentre respuestas a las preguntas más habituales sobre
                Databasus, incluidas la instalación, la configuración y las
                estrategias de copia de seguridad.
              </p>

              <h2 id="why-no-raw-sql-dump">
                ¿Por qué Databasus no usa el formato de volcado SQL plano para
                las copias de seguridad lógicas de PostgreSQL?
              </h2>

              <p>
                Para las copias lógicas, Databasus usa el{" "}
                <strong>formato personalizado</strong> de <code>pg_dump</code>{" "}
                con <strong>compresión zstd de nivel 5</strong> en lugar del
                formato SQL plano porque ofrece el equilibrio más eficiente
                entre:
              </p>

              <ul>
                <li>Velocidad de creación de la copia</li>
                <li>Velocidad de restauración</li>
                <li>
                  Compresión del tamaño del archivo (hasta 20 veces más pequeño
                  que el formato SQL plano)
                </li>
              </ul>

              <p>
                Esta decisión se tomó tras pruebas y comparativas exhaustivas de
                distintos formatos de copia y métodos de compresión de
                PostgreSQL. Puede leer más sobre las pruebas aquí:{" "}
                <a
                  href="https://dev.to/rostislav_dugin/postgresql-backups-comparing-pgdump-speed-in-different-formats-and-with-different-compression-4pmd"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  PostgreSQL backups: comparing pg_dump speed in different
                  formats and with different compression
                </a>
                .
              </p>

              <p>
                Databasus no incluirá el formato de volcado SQL plano, porque:
              </p>

              <ul>
                <li>la variedad extra perjudica la experiencia de uso;</li>
                <li>hace más difícil mantener el código;</li>
                <li>
                  el formato de volcado actual sirve para el 99% de los casos
                </li>
              </ul>

              <h2 id="installation-directory">
                ¿Dónde queda instalado Databasus si se instala con el script
                .sh?
              </h2>

              <p>
                Databasus se instala en el directorio{" "}
                <code>/opt/databasus/</code>.
              </p>

              <h2 id="pitr">
                ¿Cómo funcionan las copias físicas y PITR (Point-in-Time
                Recovery)?
              </h2>

              <p>
                Databasus ejecuta las copias físicas{" "}
                <strong>de forma remota desde su propio host</strong>,
                conectándose a su PostgreSQL por el{" "}
                <strong>protocolo de replicación</strong> estándar, así que no
                hace falta instalar nada en el servidor de la base de datos. Si
                la base de datos está en una red cerrada, Databasus puede
                alcanzarla mediante un túnel SSH hacia un host interno o un
                bastión, por lo que la base de datos nunca tiene que exponerse
                públicamente.
              </p>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    <strong>Por qué esto es posible ahora:</strong> durante años
                    herramientas como pgBackRest y WAL-G tuvieron que construir
                    sus propios motores de copias incrementales a nivel de
                    bloque, porque PostgreSQL no tenía uno nativo. Eso cambió
                    con PostgreSQL 17, donde la función fue desarrollada por{" "}
                    <strong>Robert Haas</strong> con la ayuda de{" "}
                    <strong>David Steele</strong>, el autor de pgBackRest.
                    PostgreSQL ahora incluye de serie copias incrementales
                    nativas a nivel de bloque en el lado del servidor (
                    <code>pg_basebackup --incremental</code> y{" "}
                    <code>summarize_wal</code>), así que Databasus se apoya en
                    eso en lugar de reinventarlo.
                  </p>
                </div>
              </div>

              <p>
                <strong>Cómo funcionan las copias:</strong>
              </p>

              <ul>
                <li>
                  Las copias completas se crean con <code>pg_basebackup</code>,
                  transmitidas directamente a Databasus
                </li>
                <li>
                  Las incrementales a nivel de bloque usan{" "}
                  <code>pg_basebackup --incremental</code>, donde los resúmenes
                  de WAL del servidor de PostgreSQL 17 (
                  <code>summarize_wal = on</code>) rastrean los cambios para
                  transferir solo los bloques modificados
                </li>
                <li>
                  El WAL se transmite de forma continua con{" "}
                  <code>pg_receivewal</code> para mantener completa la cadena de
                  recuperación entre copias
                </li>
                <li>
                  Las copias físicas requieren{" "}
                  <strong>PostgreSQL 17 o superior</strong>; en versiones
                  anteriores se usan las copias lógicas con <code>pg_dump</code>
                </li>
              </ul>

              <p>
                <strong>Cómo funciona la restauración:</strong>
              </p>

              <ul>
                <li>
                  <code>pg_combinebackup</code> reconstruye un directorio de
                  datos ejecutable a partir de la copia completa y su cadena
                  incremental
                </li>
                <li>
                  PostgreSQL luego reproduce el WAL hasta el momento objetivo
                  que elija, de modo que puede restaurar a cualquier segundo
                  entre copias
                </li>
                <li>
                  Al arrancar PostgreSQL, este termina la recuperación, se
                  promociona a primario y reanuda el funcionamiento normal
                </li>
              </ul>

              <div className="bg-[#1f2937]/50 border border-[#ffffff20] border-l-[3px] my-4 border-l-blue-500 rounded-lg px-4 py-4 flex items-start gap-3">
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
                    <strong>No tiene que hacerlo todo a mano.</strong> La
                    interfaz de Databasus le da instrucciones paso a paso para
                    restaurar a un host o a una base de datos en Docker, ya sea
                    mediante un script listo o descargando las copias
                    manualmente. Preparamos el script para que una restauración
                    sea un solo comando, pero si lo prefiere también puede
                    reconstruir la cadena de partes completas, incrementales y
                    WAL. Las incrementales y el WAL también son opcionales:
                    puede tomar solo una copia completa, sin incrementales, y el
                    WAL no es obligatorio.
                  </p>
                </div>
              </div>

              <p>
                <strong>Por qué usamos las copias nativas de PG 17:</strong>
              </p>

              <ul>
                <li>
                  Reutilizan la propia maquinaria de respaldo de PostgreSQL en
                  lugar de reinventarla, así que obtiene internos probados con
                  miles de tests y casos límite a sus espaldas
                </li>
                <li>
                  Dan una pérdida de datos casi nula: puede restaurar a
                  cualquier segundo entre copias
                </li>
              </ul>

              <h2 id="why-no-agent">
                ¿Por qué Databasus abandonó las copias basadas en agente?
              </h2>

              <p>
                Una versión anterior de Databasus incluía un{" "}
                <strong>agente</strong> de respaldo: un binario que se ejecutaba
                en el host de la base de datos para transmitir WAL y crear
                copias físicas localmente. Esa primera implementación resultó
                ser un error, y la eliminamos. Las copias físicas ahora se
                ejecutan de forma remota desde el host de Databasus, como se
                describe arriba.
              </p>

              <p>
                <strong>Por qué el agente era el enfoque equivocado:</strong>
              </p>

              <ul>
                <li>
                  Era una implementación ingenua que solo copiaba WAL sobre
                  copias completas, lo que llevaba a un RTO largo
                </li>
                <li>
                  Los usuarios tenían que configurar tanto Databasus como un
                  agente separado, cuando hacerlo todo de forma remota desde un
                  solo lugar es mucho más simple
                </li>
                <li>
                  Como el agente vivía fuera del sistema principal, era difícil
                  cubrir todos los casos de prueba
                </li>
                <li>
                  En realidad solo hay un problema que un agente resuelve:
                  alcanzar una base de datos que no es accesible desde fuera.
                  Para el 99% de los usuarios eso ya se resuelve ejecutando
                  Databasus dentro de la red privada o conectando por SSH, así
                  que el agente reinventaba la rueda y complicaba mucho más de
                  lo necesario un problema simple
                </li>
                <li>
                  No podía ejecutarse en bases de datos gestionadas como RDS y
                  Cloud SQL, que prohíben instalar software en el host pero ya
                  exponen el protocolo de replicación, así que de todos modos
                  hacía falta una vía remota
                </li>
                <li>
                  También traía muchos casos límite. Las conexiones rotas, la
                  gestión de las actualizaciones del agente y la recopilación de
                  registros de un proceso separado eran dolorosas, y cuantas
                  menos piezas móviles tiene un sistema, más fiable es en el uso
                  diario
                </li>
              </ul>

              <p>
                <strong>
                  Nos aseguramos de que las copias existentes queden a salvo.
                </strong>{" "}
                Si actualiza desde una versión que aún tiene copias de agente,
                Databasus no lo hará en silencio: le avisa del cambio y le deja
                quedarse en la <strong>versión 3.42.0</strong> soportada o
                eliminar las copias de agente antiguas antes de actualizar. La
                implementación basada en agente sigue disponible hasta la
                versión 3.42.0 y seguirá funcionando durante mucho tiempo, así
                que nada se rompe.
              </p>

              <h2 id="ai-usage">¿Cómo se utiliza la IA en el desarrollo de Databasus?</h2>

              <p>
                La IA nos ayuda a desarrollar Databasus, desde la planificación y la implementación hasta las
                pruebas y la documentación. Las personas definen el rumbo y toman las decisiones finales.{" "}
                <strong>Una persona lee cada línea de código antes de integrar un cambio.</strong>
              </p>

              <p>
                Utilizamos desarrollo basado en especificaciones (SDD) con{" "}
                <a href="https://github.com/databasus/databasus/tree/main/openspec">OpenSpec</a> para acordar el
                comportamiento antes de implementar.
              </p>

              <h3 id="ai-review-hooks">Revisión independiente en cada etapa</h3>

              <p>
                Los hooks automáticos inician sesiones de revisión separadas con acceso de solo lectura al
                repositorio. Cada agente inspecciona el código pertinente y devuelve sus propios hallazgos. El
                agente que implementa no puede aprobar su propio trabajo.
              </p>

              <p>
                La revisión empieza antes de implementar. Agentes de planificación separados buscan soluciones
                más sencillas, comprueban los nombres propuestos y evalúan el diseño. También contrastan los
                requisitos con las tareas de verificación.
              </p>

              <ul>
                <li>
                  <strong>Revisor de código:</strong> Busca errores, casos límite omitidos, fallos en el manejo
                  de errores, condiciones de carrera, fugas de recursos y problemas de seguridad.
                </li>
                <li>
                  <strong>Revisor de requisitos:</strong> Localiza la implementación y las pruebas de cada
                  requisito. Comprueba que las pruebas demuestren el comportamiento prometido.
                </li>
                <li>
                  <strong>Revisor de arquitectura:</strong> Comprueba los límites entre módulos, la dirección de
                  las dependencias, la propiedad de los datos, las migraciones y las reglas de API.
                </li>
                <li>
                  <strong>Revisor de estructura del código:</strong> Comprueba nombres, responsabilidades de
                  archivos y funciones, comentarios y legibilidad.
                </li>
                <li>
                  <strong>Revisor de pruebas:</strong> Evalúa si las pruebas detectarían un comportamiento
                  ausente o una regresión. También comprueba las esperas por condiciones, la limpieza y las
                  evidencias de verificación.
                </li>
                <li>
                  <strong>Revisor de documentación y tareas:</strong> Contrasta la documentación con el código,
                  comprueba que las tareas estén completas y verifica las evidencias de las comprobaciones
                  declaradas.
                </li>
              </ul>

              <p>
                Las revisiones completas de implementación se ejecutan en paralelo. Cada especificación
                modificada tiene su propio revisor de requisitos. Arquitectura, pruebas y documentación tienen
                revisores separados. Un cambio que abarca dos especificaciones recibe, por tanto, cinco agentes
                de revisión completa.
              </p>

              <p>
                <strong>Los hallazgos bloqueantes impiden aceptar el trabajo.</strong> El agente que implementa
                los corrige o explica por qué un hallazgo es incorrecto. Un revisor posterior comprueba esa
                resolución y los cambios posteriores. Una revisión que no pudo ejecutarse no se considera
                aprobada.
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/.agent-hooks/review_gate.py">
                  Implementación de los hooks de revisión
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/tree/main/.agent-hooks">
                  Instrucciones de los revisores
                </a>
              </p>

              <h3 id="ai-linting">Comprobaciones automáticas y restauraciones reales</h3>

              <p>
                La revisión independiente se complementa con comprobaciones automáticas estrictas. golangci-lint
                y nuestro analizador de arquitectura comprueban las dependencias del backend, los límites entre
                módulos y las reglas de migración. Las comprobaciones de API validan los contratos OpenAPI y el
                código generado. El frontend comprueba TypeScript, React, importaciones entre capas, estilos y
                localización.
              </p>

              <p>
                Ruff y ty comprueban el formato, las reglas de código y los tipos de Python. ShellCheck
                comprueba los scripts de shell y actionlint, los flujos de GitHub Actions. Los hooks previos al
                commit comprueban los archivos afectados del backend, frontend y Python sin reescribirlos. CI
                también ejecuta el análisis estático.
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/mise-tasks/backend/lint">
                  Análisis del backend
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/blob/main/frontend/lint/check.sh">
                  Análisis del frontend
                </a>
                {" · "}
                <a href="https://github.com/databasus/databasus/blob/main/scripts/python_quality.py">
                  Comprobaciones de Python
                </a>
              </p>

              <p>
                Las pruebas comprueban comportamientos reales. El backend se prueba mediante la API de la
                aplicación y los motores, con bases de datos reales en contenedores. Los escenarios de extremo a
                extremo crean copias de seguridad, las restauran y comprueban los datos restaurados. Otras
                pruebas cubren actualizaciones, pantallas del frontend y flujos del navegador con comparaciones
                visuales.
              </p>

              <p>
                Las comprobaciones se eligen según el cambio. Las suites obligatorias de publicación se ejecutan
                en amd64 y arm64. El agente que implementa ejecuta los analizadores y las pruebas pertinentes
                antes de la revisión. Los revisores inspeccionan las evidencias sin volver a ejecutar esos
                comandos.
              </p>

              <p>
                <a href="https://github.com/databasus/databasus/blob/main/docs/testing.md">
                  Estrategia de pruebas y comandos
                </a>
              </p>

              <p>
                Antes de publicar, CodeQL y Semgrep analizan el código, Gitleaks busca secretos y las
                dependencias se comprueban para detectar vulnerabilidades. Trivy analiza las imágenes de
                contenedor y los Dockerfile.
              </p>

              <p>
                <strong>
                  Las comprobaciones obligatorias de CI deben superarse antes de publicar una versión.
                </strong>{" "}
                <a href={`${getLocalizedHref("es", "security")}#security-and-reliability-engineering`}>
                  Seguridad y fiabilidad
                </a>
                .
              </p>

              <h2 id="backup-databasus">
                ¿Cómo respaldar el propio Databasus?
              </h2>

              <p>
                Databasus está pensado para poder recuperarse aunque se pierda o se corrompa su propio servidor.
                Puede hacer copias de seguridad de su propia base de
                datos interna: cuentas, ajustes, las credenciales cifradas de
                sus bases de datos, almacenamientos y canales de notificación,
                programaciones y el catálogo de copias. Actívelo en{" "}
                <strong>Databasus settings → Databasus backups</strong>{" "}
                eligiendo uno o varios almacenamientos existentes. Por defecto,
                la copia de seguridad de Databasus se ejecuta cada hora y se
                conserva 30 días. Cada archivo se cifra con la clave maestra.
              </p>

              <p>
                Una copia de seguridad de Databasus no contiene la clave maestra
                ni los archivos de copia de sus bases de datos, que siguen en
                sus propios almacenamientos.{" "}
                <strong>Guarde la clave maestra por separado.</strong> Databasus
                le pide que la descargue en la primera visita, y la clave es
                también el archivo <code>./databasus-data/secret.key</code> del
                host.
              </p>

              <ul>
                <li>
                  <strong>Elija un almacenamiento remoto.</strong> Un
                  almacenamiento local en el mismo servidor se pierde junto con
                  ese servidor.
                </li>
                <li>
                  <strong>Alertas.</strong> Databasus muestra un aviso cuando
                  una copia de seguridad de Databasus falla, se guarda solo en
                  algunos almacenamientos, se detiene o se retrasa, es decir,
                  cuando ninguna copia se ha completado 15 minutos después de
                  una ejecución programada. Las mismas alertas se envían por
                  correo a través del{" "}
                  <a href={`${getLocalizedHref("es", "advanced-config")}#email-smtp`}>
                  servidor de correo de la instancia
                  </a>{" "}
                  si está configurado.
                </li>
              </ul>

              <h3 id="recover-databasus">
                ¿Cómo recuperar Databasus desde una copia de seguridad de
                Databasus?
              </h3>

              <p>
                Necesita dos cosas: un archivo de copia de seguridad de
                Databasus (<code>.dbbackup</code>), descargado del historial o
                tomado directamente del almacenamiento, y la clave maestra de la
                instalación que lo creó.{" "}
                <strong>
                La recuperación solo funciona en exactamente la misma versión de
                Databasus, incluido el parche.
                </strong>{" "}
                Con cualquier otra versión, Databasus se niega e indica qué
                versión arrancar: recupere en esa versión y después actualice
                como de costumbre.
              </p>

              <p>
                <strong>Opción 1: archivos al arrancar.</strong> Úsela en un
                servidor nuevo. Ponga ambos archivos en el directorio de datos
                montado, por ejemplo <code>./databasus-data/recovery/</code>, y
                arranque la imagen de la versión que creó la copia con dos
                ajustes adicionales. Sustituya <code>&lt;version&gt;</code> por
                esa versión:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{startupRecoveryCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={startupRecoveryCommand} />
                </div>
              </div>

              <ul>
                <li>
                  Si ya arrancó Databasus normalmente en este servidor pero no
                  creó ninguna cuenta ni cambió ningún ajuste, no hace falta
                  borrar sus datos. Deténgalo, añada los dos archivos y los dos
                  ajustes y vuelva a arrancarlo.
                </li>
                <li>
                  La recuperación al arrancar nunca reemplaza una instalación
                  que tenga una cuenta (aunque sea solo una), otros datos de
                  usuario o ajustes cambiados. En ese caso Databasus no arranca.
                  En los registros indica que la recuperación no se ejecutó, por
                  qué, que la instalación actual no ha cambiado y qué hacer a
                  continuación. Para arrancar la instalación actual, quite los
                  dos ajustes o use en su lugar la subida desde la interfaz.
                </li>
                <li>
                  Tras una recuperación correcta, los archivos y los ajustes
                  pueden quedarse donde están. Databasus recuerda el SHA-256 del
                  archivo aplicado, así que los reinicios y las actualizaciones
                  posteriores lo omiten y lo indican en los registros, aunque el
                  archivo se haya renombrado. Un archivo distinto en la misma
                  ruta no se omite.
                </li>
              </ul>

              <p>
                <strong>Opción 2: subida desde la interfaz.</strong> Úsela en
                una instalación que ya está en marcha. En una instalación nueva,
                regístrese primero. Abra{" "}
                <strong>Databasus settings → Databasus backups</strong>, elija
                el archivo de copia y pegue o escriba la clave maestra de la
                instalación que lo creó.
                Se reemplaza todo lo que hay en la instalación actual:
                cuentas, ajustes y el catálogo de copias. Después, inicie
                sesión con una cuenta de la copia.
              </p>

              <h2 id="oss-programs">
                ¿Cómo apoyan a Databasus los programas de código abierto de
                Anthropic y OpenAI?
              </h2>

              <p>
                En marzo de 2026, Databasus fue aceptado tanto en{" "}
                <strong>
                  <a
                    href="https://claude.com/contact-sales/claude-for-oss"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Claude for Open Source
                  </a>
                </strong>{" "}
                de Anthropic como en{" "}
                <strong>
                  <a
                    href="https://developers.openai.com/codex/community/codex-for-oss/"
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Codex for Open Source
                  </a>
                </strong>{" "}
                de OpenAI. Para nosotros es muy valioso que el proyecto haya
                sido reconocido como software de código abierto importante para
                la industria por dos de las empresas de IA líderes del mundo,
                sobre todo dados los altos requisitos de elegibilidad de ambos
                programas.
              </p>

              <p>
                ¿Qué significa para los usuarios? Ambos programas evalúan los
                proyectos de forma independiente antes de aceptarlos. Y gracias
                al acceso ilimitado a las últimas IA tenemos una calidad de
                código mayor, revisiones de seguridad más rápidas y un
                desarrollo activo continuo.
              </p>

              <img
                src="/images/faq/anthropic-email.png"
                alt="Databasus aceptado en el programa Claude for Open Source de Anthropic"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />

              <img
                src="/images/faq/openai-email.png"
                alt="Databasus aceptado en el programa Codex for Open Source de OpenAI"
                className="my-6 rounded-lg border border-gray-700 max-w-full sm:max-w-[1000px]"
                loading="lazy"
              />
            </article>
          </div>
        </main>

        {/* Table of Contents */}
        <DocTableOfContentComponent />
      </div>
    </>
  );
}
