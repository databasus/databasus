<div align="center">
  <img src="../logo.svg" alt="Databasus Logo" width="250"/>

  <h3>Herramienta de copia de seguridad de PostgreSQL</h3>
  <p>Databasus es una herramienta gratuita, de código abierto y autoalojada para respaldar PostgreSQL. Cree copias de seguridad en distintos almacenamientos (S3, Google Drive, FTP, etc.) con notificaciones sobre el progreso (Slack, Discord, Telegram, etc.). Con un enfoque en Point-in-Time Recovery con RPO/RTO bajos</p>
  
  <!-- Badges -->
   [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-336791?logo=postgresql&logoColor=white)](https://www.postgresql.org/)
  [![MySQL](https://img.shields.io/badge/MySQL-4479A1?logo=mysql&logoColor=white)](https://www.mysql.com/)
  [![MariaDB](https://img.shields.io/badge/MariaDB-003545?logo=mariadb&logoColor=white)](https://mariadb.org/)
  [![MongoDB](https://img.shields.io/badge/MongoDB-47A248?logo=mongodb&logoColor=white)](https://www.mongodb.com/)
  <br />
  [![Apache 2.0 License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](../../LICENSE)
  [![Docker Pulls](https://img.shields.io/docker/pulls/databasus/databasus?color=brightgreen)](https://hub.docker.com/r/databasus/databasus)
  [![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey)](https://github.com/databasus/databasus)
  [![Self Hosted](https://img.shields.io/badge/self--hosted-yes-brightgreen)](https://github.com/databasus/databasus)
  [![Open Source](https://img.shields.io/badge/open%20source-❤️-red)](https://github.com/databasus/databasus)

  <p>
    <a href="../../README.md">English</a> •
    <a href="README.ru.md">Русский</a> •
    <b>Español</b> •
    <a href="README.pt.md">Português</a> •
    <a href="README.zh.md">中文</a> •
    <a href="README.fr.md">Français</a>
  </p>

  <p>
    <a href="#-características">Características</a> •
    <a href="#-instalación">Instalación</a> •
    <a href="#-uso">Uso</a> •
    <a href="#-licencia">Licencia</a> •
    <a href="#-contribuir">Contribuir</a>
  </p>

  <p style="margin-top: 20px; margin-bottom: 20px; font-size: 1.2em;">
    <a href="https://databasus.com/es/" target="_blank"><strong>🌐 Sitio web de Databasus</strong></a>
  </p>
  
  <img src="../dashboard-dark.svg" alt="Databasus Dark Dashboard" width="800" style="margin-bottom: 10px;"/>

  <img src="../dashboard.svg" alt="Databasus Dashboard" width="800"/>
</div>

---

## ✨ Características

### 📦 **Tipos de copia de seguridad**

- **Física**: copia a nivel de archivos de todo el clúster de la base de datos, sobre el mecanismo nativo de copias incrementales de PostgreSQL (leer más)
  - **Completa**: una copia íntegra y autocontenida del clúster
  - **Incremental**: guarda solo lo que cambió desde la última copia completa, así los respaldos se mantienen pequeños y rápidos
  - **Streaming de WAL**: captura de forma continua el flujo de escritura de la base de datos y habilita la recuperación a un punto en el tiempo (PITR). Pensado para la recuperación ante desastres y una pérdida de datos casi nula
- **Lógica**: volcado nativo de la base de datos en el formato binario propio del motor (comprimido y apto para restauración en paralelo)

### 🔄 **Copias de seguridad programadas**

- **Programación flexible**: cada hora, diaria, semanal, mensual o cron
- **Horarios precisos**: ejecute los respaldos a la hora que quiera (por ejemplo, a las 4 AM, cuando baja el tráfico)
- **Compresión inteligente**: archivos de 4 a 8 veces más pequeños con compresión equilibrada (~20% de sobrecarga)

### 🧪 **Verificación de restauración** <a href="https://databasus.com/es/restore-verification/">(documentación)</a>

Databasus ejecuta una restauración real para confirmar que las copias sirven, en vez de comprobar solo que el archivo está intacto en disco o que cuadra la suma de verificación.

- **Disparadores**: después de cada copia o según una programación flexible (cada hora, diaria, semanal, mensual o cron)
- **Restauración real**: levanta un contenedor de base de datos, ejecuta la restauración y compara el tamaño restaurado con el de la copia
- **Informe**: enumera cada tabla con su recuento de filas
- **Notificaciones opcionales**: envíe el informe, o solo los avisos de fallo, por cualquier notificador configurado

### 🗑️ **Políticas de retención**

- **Periodo de tiempo**: conserve las copias durante un plazo fijo (por ejemplo, 7 días, 3 meses, 1 año)
- **Cantidad**: conserve un número fijo de las copias más recientes (por ejemplo, las últimas 30)
- **GFS (Grandfather-Father-Son)**: retención por capas. Conserve copias horarias, diarias, semanales, mensuales y anuales de forma independiente para tener un histórico detallado a largo plazo (requisito habitual en empresas)
- **Límites de tamaño**: fije topes por copia y de almacenamiento total para controlar el espacio ocupado

### 🗄️ **Múltiples destinos de almacenamiento** <a href="https://databasus.com/es/storages/">(ver compatibles)</a>

- **Almacenamiento local**: guarde las copias en su VPS o servidor
- **Almacenamiento en la nube**: S3, Cloudflare R2, Google Drive, NAS, Dropbox, SFTP, Rclone y más
- **Seguro**: todos los datos quedan bajo su control

### 📱 **Notificaciones** <a href="https://databasus.com/es/notifiers/">(ver compatibles)</a>

- **Varios canales**: correo, Telegram, Slack, Discord, Teams, Mattermost, webhooks
- **Avisos en tiempo real**: notificaciones de éxito y de fallo
- **Integración con el equipo**: encaja bien en los flujos de trabajo DevOps

### 🔒 **Seguridad de nivel empresarial** <a href="https://databasus.com/es/security/">(documentación)</a>

- **Cifrado AES-256-GCM**: protección de nivel empresarial para los archivos de respaldo
- **Almacenamiento de confianza cero**: las copias están cifradas y resultan inútiles para un atacante, así que puede guardarlas sin riesgo en almacenamientos compartidos como S3, Azure Blob Storage, etc.
- **Cifrado de los secretos**: todo dato sensible se cifra y nunca se expone, ni siquiera en los registros o en los mensajes de error
- **Usuario de solo lectura**: Databasus usa por defecto un usuario de solo lectura para los respaldos y nunca guarda nada que pueda modificar sus datos
- **Autenticación de dos factores**: el inicio de sesión con contraseña puede exigir un código de seis dígitos enviado al correo de la cuenta. Entrar con Google o GitHub sigue apoyándose en las comprobaciones de esos proveedores

### 👥 **Adecuado para equipos** <a href="https://databasus.com/es/access-management/">(documentación)</a>

- **Espacios de trabajo**: agrupe bases de datos, notificadores y almacenamientos por proyecto o por equipo
- **Gestión de accesos**: decida quién puede ver o administrar cada base de datos con permisos basados en roles
- **Registros de auditoría**: siga toda la actividad del sistema y los cambios hechos por los usuarios
- **Roles de usuario**: asigne los roles de lector, miembro, administrador o propietario dentro de cada espacio de trabajo
- **Registros OpenTelemetry**: exporte los registros de la aplicación y de auditoría a un sistema externo (por defecto también se escriben en un archivo local)

### 🎨 **Cómodo de usar**

- **Interfaz cuidada por un diseñador**: limpia, intuitiva y trabajada al detalle
- **Temas claro y oscuro**: elija el aspecto que le convenga
- **Adaptada a móvil**: revise sus copias de seguridad desde cualquier sitio y cualquier dispositivo

### 💾 **Bases de datos compatibles**

- **PostgreSQL**: 14, 15, 16, 17 y 18 (física y lógica)
- **MySQL**: 5.7, 8.0, 8.4, 9 y 26 (solo lógica)
- **MariaDB**: 5.5, 10, 11, 12 y 13 (solo lógica)
- **MongoDB**: 4.2+, 5, 6, 7, 8 y 9 (solo lógica)

### 🐳 **Autoalojado y seguro**

- **Basado en Docker**: fácil de desplegar y de mantener
- **La privacidad primero**: todos sus datos se quedan en su infraestructura
- **Código abierto**: licencia Apache 2.0, puede revisar cada línea de código
- **SSH integrado**: conéctese a su Databasus por un túnel SSH

### 📦 Instalación <a href="https://databasus.com/es/installation/">(documentación)</a>

Tiene cuatro formas de instalar Databasus:

- Script automatizado (recomendado)
- Docker run sencillo
- Configuración con Docker Compose
- Kubernetes con Helm

<img src="../healthchecks.svg" alt="Databasus Dashboard" width="800"/>

---

## 📦 Instalación

Tiene cuatro formas de instalar Databasus: script automatizado (recomendado), Docker run sencillo o configuración con Docker Compose.

### Opción 1: script de instalación automatizado (recomendado, solo Linux)

El script de instalación se encarga de:

- ✅ Instalar Docker con Docker Compose (si aún no están instalados)
- ✅ Configurar Databasus
- ✅ Dejar el arranque automático listo para cuando se reinicie el sistema

```bash
sudo apt-get install -y curl && \
sudo curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh \
| sudo bash
```

### Opción 2: Docker run sencillo

La forma más fácil de ejecutar Databasus:

```bash
docker run -d \
  --name databasus \
  -p 4005:4005 \
  -v ./databasus-data:/databasus-data \
  --restart unless-stopped \
  databasus/databasus:latest
```

_La misma imagen está en el registro de GitHub: use `ghcr.io/databasus/databasus:latest` si Docker Hub le limita la descarga._

Este único comando hace lo siguiente:

- ✅ Arranca Databasus
- ✅ Guarda todos los datos en el directorio `./databasus-data`
- ✅ Vuelve a arrancar solo cuando se reinicia el sistema

### Opción 3: configuración con Docker Compose

Cree un archivo `docker-compose.yml` con esta configuración:

```yaml
services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "databasus", "healthcheck"]
      interval: 30s
      timeout: 5s
      retries: 3
      start_period: 60s
```

Después ejecute:

```bash
docker compose up -d
```

### Opción 4: Kubernetes con Helm

Para despliegues en Kubernetes, instale directamente desde el registro OCI.

_Añada `--set image.repository=ghcr.io/databasus/databasus` a cualquiera de los comandos siguientes para descargar la imagen de GHCR en lugar de Docker Hub._

**Con ClusterIP + port-forward (desarrollo y pruebas):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace
```

```bash
kubectl port-forward svc/databasus-service 4005:4005 -n databasus
# Access at http://localhost:4005
```

**Con LoadBalancer (entornos en la nube):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set service.type=LoadBalancer
```

```bash
kubectl get svc databasus-service -n databasus
# Access at http://<EXTERNAL-IP>:4005
```

**Con Ingress (acceso por dominio):**

```bash
helm install databasus oci://ghcr.io/databasus/charts/databasus \
  -n databasus --create-namespace \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=backup.example.com
```

Para más opciones (NodePort, TLS, HTTPRoute para Gateway API), consulte el [README del chart de Helm](../../deploy/helm/README.md).

---

## 🚀 Uso

1. **Cree la primera cuenta**: abra `http://localhost:4005` y regístrese. La primera cuenta creada en una instancia es la que la administra
2. **Añada su primera base de datos para respaldar**: pulse "New Database" y siga el asistente
3. **Configure la programación**: elija entre intervalos horarios, diarios, semanales, mensuales o cron
4. **Indique la conexión a la base de datos**: introduzca sus credenciales y los datos de conexión
5. **Elija el almacenamiento**: seleccione dónde guardar las copias (local, S3, Google Drive, etc.)
6. **Configure la política de retención**: elija periodo de tiempo, cantidad o GFS para controlar cuánto se conservan las copias
7. **Añada notificaciones** (opcional): configure avisos por correo, Telegram, Slack, Mattermost o webhook
8. **Guarde y arranque**: Databasus validará la configuración y pondrá en marcha la programación de respaldos

### 🔑 Restablecer la contraseña <a href="https://databasus.com/es/password/">(documentación)</a>

Si necesita restablecer la contraseña, use el comando integrado:

```bash
docker exec -it databasus ./main --new-password="YourNewSecurePassword123" --email="owner@example.com"
```

Sustituya `owner@example.com` por la dirección de correo real de la cuenta cuya contraseña quiere restablecer.

Si no recuerda qué dirección administra la instancia, liste las cuentas de administrador:

```bash
docker exec -it databasus ./main --list-admins
```

La salida nombra cada cuenta de administrador con su correo electrónico, nombre visible, fecha de creación y estado de actividad, y marca la que la instancia reconoce como su administrador. No imprime ninguna contraseña ni su hash.

Para dar de alta una cuenta sin registrarse en el navegador, por ejemplo desde un script de despliegue, créela desde la consola:

```bash
docker exec -it databasus ./main --create-user --email="ops@example.com" --password="YourSecurePassword123" --role=admin
```

`--role` admite `admin` o `member`, la contraseña necesita al menos 8 caracteres y `--name` fija, si lo desea, el nombre visible. En una instancia que todavía no tiene ninguna cuenta, la primera debe usar `--role=admin` y pasa a administrar la instancia. Repetir el comando para la misma cuenta no cambia nada, así que puede incluirlo en cada despliegue; para cambiar la contraseña, use `--new-password`.

### 💾 Respaldar el propio Databasus

Después de instalarlo, conviene también <a href="https://databasus.com/es/faq/#backup-databasus">respaldar su propio Databasus</a> o, al menos, copiar la clave secreta que se usa para el cifrado (bastan 30 segundos). Así podrá restaurar desde sus copias cifradas si pierde el acceso al servidor con Databasus o si este se corrompe.

---

## 🛡️ Ingeniería de seguridad y fiabilidad

Databasus trabaja con datos sensibles, así que prevenir vulnerabilidades, accesos no autorizados y fugas de datos es una preocupación primordial. Invertimos en ello en ambos lados del sistema: en el propio código (comprobaciones de permisos, cifrado, manejo cuidadoso de los secretos) y en la infraestructura que lo rodea (análisis de dependencias, respuesta a CVE, prácticas DevSecOps). La canalización que se describe a continuación se ejecuta automáticamente en cada commit y PR. Ninguna capa basta por sí sola, pero juntas reducen la probabilidad de que código vulnerable, dependencias inseguras, imágenes rotas o copias no restaurables lleguen a una versión publicada.

Para el análisis estático combinamos varias pasadas independientes. CodeQL escanea todo el código en busca de problemas de seguridad. En cada PR, gitleaks busca secretos filtrados y semgrep revisa el código con reglas de seguridad. Los Dockerfiles y los flujos de CI tienen reglas adicionales propias (referencias de acciones fijadas, permisos de mínimo privilegio, imágenes base sospechosas), de modo que los patrones inseguros se señalan antes de fusionarse. Además de estas comprobaciones por PR, Codex Security de OpenAI realiza auditorías periódicas y más profundas de todo el código. Es un programa aparte que detecta problemas arquitectónicos y transversales que los escaneos limitados al momento del PR pasan por alto.

En cuanto a las dependencias, Dependabot vigila todas las nuestras contra la GitHub Advisory Database y detecta los CVE a los pocos minutos de su publicación. Las actualizaciones pasan por un periodo de espera para que las versiones recién publicadas maduren antes de que las adoptemos. Es una defensa deliberada contra incidentes de paquetes comprometidos, como los ataques a la cadena de suministro. La Dependency Review Action bloquea de plano cualquier PR que introduzca un CVE nuevo de nivel HIGH o CRITICAL. En el código Go, govulncheck rechaza un PR cuando nuestro código llama a una función con una vulnerabilidad conocida, y `go mod verify` compara cada módulo descargado con su suma de verificación registrada.

Las imágenes de contenedor se escanean con Trivy en cada build. Una pasada aparte de Trivy sobre el Dockerfile detecta configuraciones incorrectas antes de que lleguen a una imagen. Todas las GitHub Actions están fijadas a SHA de commit completos en lugar de etiquetas flotantes como `@v4` o `@main`, que han sido un vector de ataque activo en 2025. Los flujos de trabajo usan por defecto permisos de mínimo privilegio y solo los elevan por trabajo cuando hace falta de verdad.

Las rutas críticas están cubiertas por pruebas unitarias y de integración, ejecutadas contra contenedores de bases de datos reales para cada motor y versión mayor compatibles. La restauración es la ruta que más importa en una herramienta de respaldo, así que la probamos explícitamente: cada PR ejecuta ciclos completos de copia y restauración contra esos mismos contenedores reales y verifica que las copias realmente pueden restaurarse de extremo a extremo, no solo escribirse con éxito. El resto de la canalización de CI/CD ejecuta lint, comprobación de tipos, la suite de pruebas completa, pruebas de humo de las imágenes y builds multiarquitectura en cada PR. Una versión solo se publica si todo pasa.

¿Ha encontrado una vulnerabilidad? Infórmela a través de la pestaña Security de GitHub. Consulte [SECURITY.md](https://github.com/databasus/databasus?tab=security-ov-file#readme). Los informes de seguridad se atienden con la máxima prioridad. Para la seguridad de la aplicación en ejecución (AES-256-GCM en reposo, almacenamiento de confianza cero, secretos cifrados, usuario de base de datos de solo lectura por defecto), consulte [Seguridad de nivel empresarial](#-seguridad-de-nivel-empresarial-documentación) en la sección de características de arriba.

---

## 📝 Licencia

Este proyecto está publicado bajo la licencia Apache 2.0. Consulte el archivo [LICENSE](../../LICENSE) para los detalles

## 🤝 Contribuir

¡Las contribuciones son bienvenidas! Lea la <a href="https://databasus.com/contribute">guía de contribución</a> para conocer los detalles, las prioridades y las reglas. Si quiere contribuir pero no sabe por dónde empezar, escríbame por Telegram [@rostislav_dugin](https://t.me/rostislav_dugin)

También puede unirse a nuestra gran comunidad de desarrolladores, DBA e ingenieros DevOps en Telegram [@databasus_community](https://t.me/databasus_community).
