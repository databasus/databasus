import type { Metadata } from "next";
import { OG_LOCALES, getLanguageAlternates, getLocalizedUrl } from "@/app/i18n";
import { CopyButton } from "@/app/components/CopyButton";
import DocsNavbarComponent from "@/app/components/DocsNavbarComponent";
import DocsSidebarComponent from "@/app/components/DocsSidebarComponent";
import DocTableOfContentComponent from "@/app/components/DocTableOfContentComponent";

export const metadata: Metadata = {
  title: "Restablecer contraseña - Documentación de Databasus",
  description:
    "Aprenda a restablecer las contraseñas de usuario, listar los administradores y crear cuentas en Databasus con la herramienta de línea de comandos integrada. Recuperación de contraseña rápida y segura para su sistema de copias de seguridad de PostgreSQL.",
  keywords: [
    "restablecer contraseña Databasus",
    "restablecer contraseña de usuario",
    "contraseña de copia de seguridad de PostgreSQL",
    "recuperación de contraseña en Docker",
    "recuperación de contraseña",
    "autenticación de Databasus",
  ],
  openGraph: {
    title: "Restablecer contraseña - Documentación de Databasus",
    description:
      "Aprenda a restablecer las contraseñas de usuario, listar los administradores y crear cuentas en Databasus con la herramienta de línea de comandos integrada. Recuperación de contraseña rápida y segura para su sistema de copias de seguridad de PostgreSQL.",
    type: "article",
    url: getLocalizedUrl("es", "password"),
    locale: OG_LOCALES.es,
  },
  twitter: {
    card: "summary",
    title: "Restablecer contraseña - Documentación de Databasus",
    description:
      "Aprenda a restablecer las contraseñas de usuario, listar los administradores y crear cuentas en Databasus con la herramienta de línea de comandos integrada. Recuperación de contraseña rápida y segura para su sistema de copias de seguridad de PostgreSQL.",
  },
  alternates: {
    canonical: getLocalizedUrl("es", "password"),
    languages: getLanguageAlternates("password"),
  },
  robots: "index, follow",
};

export default function PasswordResetPage() {
  const resetPasswordCommand = `docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com"`;
  const listAdminsCommand = `docker exec -it databasus ./main --list-admins`;
  const createUserCommand = `docker exec -it databasus ./main --create-user --email="ops@example.com" --password="YourSecurePassword123" --role=admin`;
  const disableTwoFactorCommand = `docker exec -it databasus ./main --disable-2fa`;
  const resetTwoFactorCommand = `docker exec -it databasus ./main --reset-2fa --email="owner@example.com"`;

  return (
    <>
      {/* JSON-LD Structured Data */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "TechArticle",
            headline: "Restablecer contraseña - Documentación de Databasus",
            description:
              "Aprenda a restablecer las contraseñas de usuario en Databasus con la herramienta de línea de comandos integrada.",
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
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{
          __html: JSON.stringify({
            "@context": "https://schema.org",
            "@type": "HowTo",
            name: "Cómo restablecer la contraseña de un usuario de Databasus",
            description:
              "Guía paso a paso para restablecer las contraseñas de usuario en Databasus",
            step: [
              {
                "@type": "HowToStep",
                name: "Ejecute el comando de restablecimiento de contraseña",
                text: "Ejecute el comando docker exec con la nueva contraseña y el correo electrónico del usuario.",
                itemListElement: [
                  {
                    "@type": "HowToDirection",
                    text: "Use docker exec para ejecutar el comando de restablecimiento de contraseña dentro del contenedor de Databasus",
                  },
                ],
              },
              {
                "@type": "HowToStep",
                name: "Verifique el cambio de contraseña",
                text: "Inicie sesión en Databasus con la nueva contraseña para confirmar que el cambio se aplicó correctamente.",
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
              <h1 id="reset-password">Restablecer la contraseña de usuario</h1>

              <h2 id="reset-password-command">
                Comando de restablecimiento de contraseña
              </h2>

              <p>
                Para restablecer la contraseña de un usuario, use el siguiente
                comando en el servidor donde se ejecuta Databasus:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetPasswordCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={resetPasswordCommand} />
                </div>
              </div>

              <h2 id="parameters">Parámetros</h2>

              <p>El comando acepta los siguientes parámetros:</p>

              <ul>
                <li>
                  <strong>--new-password</strong>: la nueva contraseña, de al
                  menos 8 caracteres. Asegúrese de que sea segura y combine
                  letras, números y caracteres especiales.
                </li>
                <li>
                  <strong>--email</strong>: el correo electrónico de la cuenta
                  cuya contraseña quiere restablecer (por ejemplo,{" "}
                  <code>owner@example.com</code>). Una instancia creada antes de
                  que la primera cuenta la administrara todavía lleva la
                  dirección provisional <code>admin</code> hasta que su
                  propietario la reemplace, así que allí indique{" "}
                  <code>admin</code>.
                </li>
              </ul>

              <h2 id="list-admins">Listar las cuentas de administrador</h2>

              <p>
                Si no recuerda qué dirección administra la instancia, liste las
                cuentas de administrador en el servidor donde se ejecuta
                Databasus:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{listAdminsCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={listAdminsCommand} lang="es" />
                </div>
              </div>

              <p>
                La salida nombra cada cuenta de administrador con su correo
                electrónico, nombre visible, fecha de creación y estado de
                actividad, y marca la que la instancia reconoce como su
                administrador. No imprime ninguna contraseña, ni su hash, ni
                ningún token. En una instancia donde todavía nadie ha creado una
                cuenta, informa de que la instancia no tiene administrador.
              </p>

              <h2 id="create-user">Crear una cuenta desde la consola</h2>

              <p>
                Para dar de alta una cuenta sin registrarse en el navegador, por
                ejemplo desde un script de despliegue, créela en el servidor
                donde se ejecuta Databasus:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{createUserCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton lang="es" text={createUserCommand} />
                </div>
              </div>

              <p>El comando acepta los siguientes parámetros:</p>

              <ul>
                <li>
                  <strong>--email</strong>: el correo electrónico de la nueva
                  cuenta. Debe ser una dirección válida.
                </li>
                <li>
                  <strong>--password</strong>: la contraseña de la nueva cuenta,
                  de al menos 8 caracteres.
                </li>
                <li>
                  <strong>--role</strong>: <code>admin</code> o{" "}
                  <code>member</code>.
                </li>
                <li>
                  <strong>--name</strong>: un nombre visible opcional. Si no lo
                  indica, la cuenta recibe como nombre la parte de la dirección
                  anterior a la <code>@</code>.
                </li>
              </ul>

              <p>
                La cuenta queda activa de inmediato y entra con esa dirección y
                esa contraseña, esté o no abierto el registro. En una instancia
                que todavía no tiene ninguna cuenta, la cuenta que cree pasa a
                administrar la instancia, así que la primera debe usar{" "}
                <code>--role=admin</code>. Un administrador creado más tarde de
                esta forma es un administrador corriente.
              </p>

              <p>
                Si vuelve a ejecutar el comando para una cuenta que ya existe con
                el mismo rol, termina sin error y no cambia nada, así que puede
                incluirlo en cada despliegue. En ese caso la contraseña no se
                toca: para cambiarla, use el comando de restablecimiento{" "}
                <code>--new-password</code> descrito más arriba. Si la dirección
                pertenece a una cuenta con otro rol, a una cuenta desactivada o
                invitada, o a una cuenta que inicia sesión con Google o GitHub,
                el comando falla y no cambia nada.
              </p>
              <h2 id="disable-two-factor">Dejar de exigir 2FA</h2>

              <p>
                Una instancia puede exigir, además de la contraseña, un código
                de seis dígitos enviado por correo al iniciar sesión. Si el
                servidor de correo deja de entregar esos códigos, nadie entra
                con contraseña, y restablecerla no basta. Desactive el segundo
                factor en el servidor donde se ejecuta Databasus:
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
                El comando informa de si ha cambiado algo, termina sin quejarse
                cuando el segundo factor ya está desactivado y registra el
                cambio en el historial de auditoría. El siguiente inicio de
                sesión con contraseña ya no pide código, y un administrador
                puede volver a activar el ajuste cuando el correo funcione.
              </p>

              <h2 id="reset-authenticator-app">Restablecer la aplicación de autenticación de una cuenta</h2>

              <p>
                Si alguien perdió el teléfono con su aplicación de autenticación, quite la aplicación de su cuenta en el servidor donde se ejecuta Databasus:
              </p>

              <div className="relative my-6">
                <pre className="overflow-x-auto rounded-lg bg-gray-900 p-4 text-sm text-gray-100">
                  <code>{resetTwoFactorCommand}</code>
                </pre>
                <div className="absolute right-2 top-2">
                  <CopyButton text={resetTwoFactorCommand} />
                </div>
              </div>

              <p>
                El siguiente inicio de sesión de esa cuenta sigue el ajuste de 2FA de la instancia. Un administrador también puede restablecer la aplicación de un usuario desde la lista de usuarios.
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
