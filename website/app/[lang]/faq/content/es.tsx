import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

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
                  text: "Databasus ejecuta las copias físicas de forma remota desde su propio host, conectándose a su PostgreSQL por el protocolo de replicación estándar, así que no hace falta instalar nada en el servidor de la base de datos. A las bases de datos en redes cerradas se llega mediante un túnel SSH. Las copias físicas usan la pila nativa de PostgreSQL 17: copias completas con pg_basebackup, incrementales a nivel de bloque con pg_basebackup --incremental basadas en los resúmenes de WAL del servidor (summarize_wal = on) y streaming continuo de WAL con pg_receivewal. Las copias físicas requieren PostgreSQL 17 o superior; en versiones anteriores se usan las copias lógicas con pg_dump. Para restaurar a un punto en el tiempo, pg_combinebackup reconstruye un directorio de datos ejecutable a partir de la copia completa y su cadena incremental, y PostgreSQL luego reproduce el WAL hasta el momento objetivo que elija, de modo que puede restaurar a cualquier segundo entre copias. La interfaz de Databasus da instrucciones paso a paso para restaurar a un host o a una base de datos en Docker, ya sea mediante un script listo que convierte la restauración en un solo comando o descargando las copias y reconstruyendo la cadena de partes completas, incrementales y WAL. Las incrementales y el WAL son opcionales: puede tomar solo una copia completa, y el WAL no es obligatorio. Usamos las copias nativas de PostgreSQL 17 porque reutilizan la maquinaria de respaldo probada de PostgreSQL en lugar de reinventarla, funcionan con bases de datos remotas incluidos servicios gestionados como RDS y Cloud SQL, y dan una pérdida de datos casi nula.",
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
                  text: "Para respaldar Databasus, vaya a /opt/databasus (o la carpeta donde lo instaló) y entre en el directorio databasus-data. Necesita respaldar el archivo secret.key (clave de cifrado de las credenciales) y la carpeta /pgdata (base de datos interna con las configuraciones y los metadatos de las copias). Hay dos escenarios de recuperación: 1) puede recuperar las copias de las bases de datos usando solo secret.key sin la interfaz de Databasus (vea la guía de recuperación manual), 2) para restaurar la interfaz de Databasus con todas las configuraciones y el historial, necesita tanto secret.key como la carpeta /pgdata. Para restaurar, recree esa estructura de carpetas en otro servidor.",
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
                  Funcionan con bases de datos remotas, incluidos servicios
                  gestionados como Amazon RDS y Google Cloud SQL, que exponen el
                  protocolo de replicación pero prohíben instalar software en el
                  host
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

              <h2 id="backup-databasus">
                ¿Cómo respaldar el propio Databasus?
              </h2>

              <p>
                Si quiere respaldar su instancia de Databasus (incluidas todas
                las configuraciones, bases de datos y credenciales), siga estos
                pasos:
              </p>

              <ol>
                <li>
                  Vaya a <code>/opt/databasus</code> (o la carpeta donde instaló
                  Databasus)
                </li>
                <li>
                  Entre en el directorio <code>databasus-data</code>
                </li>
              </ol>

              <p>
                <strong>Necesita respaldar:</strong>
              </p>

              <ul>
                <li>
                  <code>secret.key</code> — clave de cifrado de sus credenciales
                </li>
                <li>
                  <code>/pgdata</code> — base de datos PostgreSQL interna de
                  Databasus que contiene todas sus configuraciones y los
                  metadatos de las copias
                </li>
              </ul>

              <p>
                Si usa almacenamiento local para las copias, también puede
                respaldar la carpeta <code>backups</code>.
              </p>

              <p>
                <strong>Importante:</strong> hay dos escenarios de recuperación
                distintos:
              </p>

              <ul>
                <li>
                  <strong>
                    Recuperar copias sin la interfaz de Databasus:
                  </strong>{" "}
                  puede recuperar las copias de sus bases de datos usando solo
                  el archivo <code>secret.key</code>, sin necesitar Databasus ni
                  sus datos internos. Vea la{" "}
                  <a href="/es/how-to-recover-without-databasus">
                    guía de recuperación manual
                  </a>{" "}
                  para instrucciones detalladas.
                </li>
                <li>
                  <strong>
                    Restaurar la interfaz de Databasus y todas las
                    configuraciones:
                  </strong>{" "}
                  si quiere restaurar la interfaz de Databasus con todas sus
                  configuraciones, copias programadas e historial, necesita
                  respaldar tanto <code>secret.key</code> como la carpeta{" "}
                  <code>/pgdata</code> (que contiene los metadatos de cifrado y
                  todas las configuraciones de Databasus).
                </li>
              </ul>

              <p>
                <strong>Para restaurar Databasus en otro servidor:</strong>{" "}
                simplemente recree la estructura de la carpeta{" "}
                <code>databasus-data</code> con los archivos respaldados y
                arranque Databasus.
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
