"use client";

import { useState } from "react";
import { getLocalizedHref, type Locale } from "@/app/i18n";
import { COPY_LABELS, VIDEO_PLAY_LABELS } from "./copyLabels";
import LiteYouTubeEmbed from "./LiteYouTubeEmbed";

type InstallationTexts = {
  automatedLabel: string;
  automatedTitle: string;
  automatedDescription: string;
  dockerDescription: string;
  composeDescription: string;
  helmDescription: string;
  withSudo: string;
  withoutSudo: string;
  helmClusterIP: string;
  helmLoadBalancer: string;
  helmIngress: string;
  readMore: string;
  videoTitle: string;
};

const INSTALLATION_TEXTS: Record<Locale | "en", InstallationTexts> = {
  id: {
    automatedLabel: "Skrip otomatis",
    automatedTitle: "Skrip otomatis (disarankan)",
    automatedDescription:
      "Skrip instalasi akan memasang Docker beserta Docker Compose (jika belum terpasang), menyiapkan Databasus, dan mengaturnya agar berjalan otomatis saat sistem dinyalakan ulang.",
    dockerDescription:
      "Cara termudah menjalankan Databasus. Satu perintah ini akan menjalankan Databasus, menyimpan semua data di direktori ./databasus-data, dan mengaturnya agar berjalan otomatis saat sistem dinyalakan ulang.",
    composeDescription:
      "Buat file docker-compose.yml dengan konfigurasi berikut, lalu jalankan: docker compose up -d",
    helmDescription:
      "Untuk deployment Kubernetes, instal langsung dari registry OCI. Pilih cara akses: ClusterIP dengan port-forward untuk pengembangan, LoadBalancer untuk lingkungan cloud, atau Ingress untuk akses melalui domain.",
    withSudo: "dengan sudo",
    withoutSudo: "tanpa sudo",
    helmClusterIP: "Dengan ClusterIP + port-forward (pengembangan)",
    helmLoadBalancer: "Dengan LoadBalancer (lingkungan cloud)",
    helmIngress: "Dengan Ingress (akses melalui domain)",
    readMore: "Selengkapnya tentang instalasi",
    videoTitle: "Cara menginstal Databasus",
  },
  en: {
    automatedLabel: "Automated script",
    automatedTitle: "Automated script (recommended)",
    automatedDescription:
      "The installation script will install Docker with Docker Compose (if not already installed), set up Databasus and configure automatic startup on system reboot.",
    dockerDescription:
      "The easiest way to run Databasus. This single command will start Databasus, store all data in ./databasus-data directory and automatically restart on system reboot.",
    composeDescription:
      "Create a docker-compose.yml file with the following configuration, then run: docker compose up -d",
    helmDescription:
      "For Kubernetes deployments, install directly from the OCI registry. Choose your preferred access method: ClusterIP with port-forward for development, LoadBalancer for cloud environments, or Ingress for domain-based access.",
    withSudo: "with sudo",
    withoutSudo: "without sudo",
    helmClusterIP: "With ClusterIP + port-forward (development)",
    helmLoadBalancer: "With LoadBalancer (cloud environments)",
    helmIngress: "With Ingress (domain-based access)",
    readMore: "Read more about installation",
    videoTitle: "How to install Databasus",
  },
  ru: {
    automatedLabel: "Автоматический скрипт",
    automatedTitle: "Автоматический скрипт (рекомендуется)",
    automatedDescription:
      "Скрипт установки поставит Docker с Docker Compose (если они ещё не установлены), настроит Databasus и включит автозапуск после перезагрузки системы.",
    dockerDescription:
      "Самый простой способ запустить Databasus. Одна команда запустит Databasus, сохранит все данные в каталоге ./databasus-data и включит автоперезапуск после перезагрузки.",
    composeDescription:
      "Создайте файл docker-compose.yml со следующей конфигурацией, затем выполните: docker compose up -d",
    helmDescription:
      "Для Kubernetes устанавливайте напрямую из OCI-реестра. Выберите способ доступа: ClusterIP с port-forward для разработки, LoadBalancer для облаков или Ingress для доступа по домену.",
    withSudo: "с sudo",
    withoutSudo: "без sudo",
    helmClusterIP: "ClusterIP + port-forward (для разработки)",
    helmLoadBalancer: "LoadBalancer (облачные окружения)",
    helmIngress: "Ingress (доступ по домену)",
    readMore: "Подробнее об установке",
    videoTitle: "Как установить Databasus",
  },
  es: {
    automatedLabel: "Script automático",
    automatedTitle: "Script automático (recomendado)",
    automatedDescription:
      "El script de instalación instalará Docker con Docker Compose (si aún no están instalados), configurará Databasus y activará el arranque automático al reiniciar el sistema.",
    dockerDescription:
      "La forma más sencilla de ejecutar Databasus. Este único comando iniciará Databasus, guardará todos los datos en el directorio ./databasus-data y se reiniciará automáticamente al reiniciar el sistema.",
    composeDescription:
      "Cree un archivo docker-compose.yml con la siguiente configuración y luego ejecute: docker compose up -d",
    helmDescription:
      "Para despliegues en Kubernetes, instale directamente desde el registro OCI. Elija su método de acceso: ClusterIP con port-forward para desarrollo, LoadBalancer para entornos en la nube o Ingress para acceso por dominio.",
    withSudo: "con sudo",
    withoutSudo: "sin sudo",
    helmClusterIP: "Con ClusterIP + port-forward (desarrollo)",
    helmLoadBalancer: "Con LoadBalancer (entornos en la nube)",
    helmIngress: "Con Ingress (acceso por dominio)",
    readMore: "Más sobre la instalación",
    videoTitle: "Cómo instalar Databasus",
  },
  pt: {
    automatedLabel: "Script automático",
    automatedTitle: "Script automático (recomendado)",
    automatedDescription:
      "O script de instalação instala o Docker com Docker Compose (se ainda não estiverem instalados), configura o Databasus e ativa o arranque automático após reiniciar o sistema.",
    dockerDescription:
      "A forma mais simples de executar o Databasus. Este único comando inicia o Databasus, guarda todos os dados no diretório ./databasus-data e reinicia automaticamente após o reinício do sistema.",
    composeDescription:
      "Crie um arquivo docker-compose.yml com a seguinte configuração e depois execute: docker compose up -d",
    helmDescription:
      "Para Kubernetes, instale diretamente do registro OCI. Escolha o método de acesso: ClusterIP com port-forward para desenvolvimento, LoadBalancer para nuvem ou Ingress para acesso por domínio.",
    withSudo: "com sudo",
    withoutSudo: "sem sudo",
    helmClusterIP: "Com ClusterIP + port-forward (desenvolvimento)",
    helmLoadBalancer: "Com LoadBalancer (ambientes de nuvem)",
    helmIngress: "Com Ingress (acesso por domínio)",
    readMore: "Mais sobre a instalação",
    videoTitle: "Como instalar o Databasus",
  },
  zh: {
    automatedLabel: "自动安装脚本",
    automatedTitle: "自动安装脚本（推荐）",
    automatedDescription:
      "安装脚本会安装 Docker 和 Docker Compose（如果尚未安装），配置 Databasus 并设置系统重启后自动启动。",
    dockerDescription:
      "运行 Databasus 最简单的方式。这一条命令会启动 Databasus，将所有数据保存在 ./databasus-data 目录中，并在系统重启后自动重启。",
    composeDescription:
      "创建包含以下配置的 docker-compose.yml 文件，然后运行：docker compose up -d",
    helmDescription:
      "在 Kubernetes 中直接从 OCI 仓库安装。选择访问方式：开发用 ClusterIP 加 port-forward，云环境用 LoadBalancer，或用 Ingress 按域名访问。",
    withSudo: "使用 sudo",
    withoutSudo: "不使用 sudo",
    helmClusterIP: "ClusterIP + port-forward（开发环境）",
    helmLoadBalancer: "LoadBalancer（云环境）",
    helmIngress: "Ingress（按域名访问）",
    readMore: "了解更多安装方式",
    videoTitle: "如何安装 Databasus",
  },
  fr: {
    automatedLabel: "Script automatique",
    automatedTitle: "Script automatique (recommandé)",
    automatedDescription:
      "Le script d'installation installe Docker avec Docker Compose (s'ils ne sont pas déjà installés), configure Databasus et active le démarrage automatique au redémarrage du système.",
    dockerDescription:
      "La façon la plus simple de lancer Databasus. Cette seule commande démarre Databasus, stocke toutes les données dans le répertoire ./databasus-data et redémarre automatiquement au redémarrage du système.",
    composeDescription:
      "Créez un fichier docker-compose.yml avec la configuration suivante, puis exécutez : docker compose up -d",
    helmDescription:
      "Pour Kubernetes, installez directement depuis le registre OCI. Choisissez votre méthode d'accès : ClusterIP avec port-forward pour le développement, LoadBalancer pour le cloud ou Ingress pour un accès par domaine.",
    withSudo: "avec sudo",
    withoutSudo: "sans sudo",
    helmClusterIP: "Avec ClusterIP + port-forward (développement)",
    helmLoadBalancer: "Avec LoadBalancer (environnements cloud)",
    helmIngress: "Avec Ingress (accès par domaine)",
    readMore: "En savoir plus sur l'installation",
    videoTitle: "Comment installer Databasus",
  },
};

type InstallMethod = "Automated Script" | "Docker Run" | "Docker Compose" | "Helm";

type ScriptVariant = {
  label: string;
  code: string;
};

type CodeBlock = {
  label: string;
  code: string;
};

type Installation = {
  label: string;
  title: string;
  code: string | ScriptVariant[];
  codeBlocks?: CodeBlock[];
  language: string;
  description: string;
};

function getInstallationMethods(texts: InstallationTexts): Record<InstallMethod, Installation> {
  return {
    "Automated Script": {
      label: texts.automatedLabel,
      title: texts.automatedTitle,
      language: "bash",
      description: texts.automatedDescription,
      code: [
        {
          label: texts.withSudo,
          code: `sudo apt-get install -y curl && \\
sudo curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh | sudo bash`,
        },
        {
          label: texts.withoutSudo,
          code: `apt-get install -y curl && \\
curl -sSL https://raw.githubusercontent.com/databasus/databasus/refs/heads/main/install-databasus.sh | bash`,
        },
      ],
    },
    "Docker Run": {
      label: "Docker",
      title: "Docker",
      language: "bash",
      description: texts.dockerDescription,
      code: `docker run -d \\
  --name databasus \\
  -p 4005:4005 \\
  -v ./databasus-data:/databasus-data \\
  --restart unless-stopped \\
  databasus/databasus:latest`,
    },
    "Docker Compose": {
      label: "Docker Compose",
      title: "Docker Compose",
      language: "yaml",
      description: texts.composeDescription,
      code: `services:
  databasus:
    container_name: databasus
    image: databasus/databasus:latest
    ports:
      - "4005:4005"
    volumes:
      - ./databasus-data:/databasus-data
    restart: unless-stopped`,
    },
    Helm: {
      label: "Helm (Kubernetes)",
      title: "Helm (Kubernetes)",
      language: "bash",
      description: texts.helmDescription,
      code: "",
      codeBlocks: [
        {
          label: texts.helmClusterIP,
          code: `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace

kubectl port-forward svc/databasus-service 4005:4005 -n databasus
# Access at http://localhost:4005`,
        },
        {
          label: texts.helmLoadBalancer,
          code: `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace \\
  --set service.type=LoadBalancer

kubectl get svc databasus-service -n databasus
# Access at http://<EXTERNAL-IP>:4005`,
        },
        {
          label: texts.helmIngress,
          code: `helm install databasus oci://ghcr.io/databasus/charts/databasus \\
  -n databasus --create-namespace \\
  --set ingress.enabled=true \\
  --set ingress.hosts[0].host=backup.example.com`,
        },
      ],
    },
  };
}

const methods: InstallMethod[] = ["Automated Script", "Docker Run", "Docker Compose", "Helm"];

export default function InstallationComponent({ lang = "en" }: { lang?: Locale | "en" }) {
  const [selectedMethod, setSelectedMethod] = useState<InstallMethod>("Automated Script");
  const texts = INSTALLATION_TEXTS[lang];
  const installationMethods = getInstallationMethods(texts);
  const [selectedVariant, setSelectedVariant] = useState(0);
  const [isCopied, setIsCopied] = useState(false);
  const [copiedBlockIndex, setCopiedBlockIndex] = useState<number | null>(null);

  const currentInstallation = installationMethods[selectedMethod];
  const hasVariants =
    Array.isArray(currentInstallation.code) && currentInstallation.code.length > 0;
  const hasCodeBlocks = currentInstallation.codeBlocks && currentInstallation.codeBlocks.length > 0;

  const handleMethodChange = (method: InstallMethod) => {
    setSelectedMethod(method);
    setSelectedVariant(0);
    setIsCopied(false);
    setCopiedBlockIndex(null);
  };

  const handleVariantChange = (index: number) => {
    setSelectedVariant(index);
    setIsCopied(false);
  };

  const getCurrentCode = () => {
    if (hasVariants) {
      return (currentInstallation.code as ScriptVariant[])[selectedVariant].code;
    }
    return currentInstallation.code as string;
  };

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(getCurrentCode());
      setIsCopied(true);
      setTimeout(() => setIsCopied(false), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  const handleCopyBlock = async (code: string, index: number) => {
    try {
      await navigator.clipboard.writeText(code);
      setCopiedBlockIndex(index);
      setTimeout(() => setCopiedBlockIndex(null), 2000);
    } catch (err) {
      console.error("Failed to copy:", err);
    }
  };

  return (
    <div className="mx-auto w-full">
      {/* Installation methods tabs */}
      <div className="mb-6 flex flex-wrap gap-2 justify-center">
        {methods.map((method) => (
          <button
            key={method}
            onClick={() => handleMethodChange(method)}
            className={`cursor-pointer rounded-lg px-3 py-1.5 text-sm font-medium transition-colors sm:px-4 sm:py-2 md:px-6 md:py-2 md:text-base ${
              selectedMethod === method
                ? "bg-blue-600 text-white border border-[#155dfc]"
                : "border border-[#ffffff20] hover:border-[#155dfc] hover:bg-blue-600 hover:text-white"
            }`}
          >
            {installationMethods[method].label}
          </button>
        ))}
      </div>

      <div className="flex flex-col lg:flex-row gap-8 lg:gap-10 mt-8 lg:mt-20">
        <div className="w-full lg:w-[50%]">
          <div className="text-xl md:text-2xl font-bold mb-3 md:mb-4">
            {currentInstallation.title}
          </div>

          {/* Description */}
          <div className="mb-4 md:mb-5 max-w-[550px] text-gray-400 text-sm md:text-base">
            {currentInstallation.description}
          </div>

          {/* Script variants tabs (only for Automated Script) */}
          {hasVariants && (
            <div className="mb-4 flex flex-wrap gap-2">
              {(currentInstallation.code as ScriptVariant[]).map((variant, index) => (
                <button
                  key={index}
                  onClick={() => handleVariantChange(index)}
                  className={`cursor-pointer rounded-lg px-3 py-1.5 text-sm sm:px-4 sm:py-2 font-medium transition-colors ${
                    selectedVariant === index
                      ? "bg-[#2C2F35] text-white border border-[#2C2F35]"
                      : "border border-[#ffffff20] hover:border-[#2C2F35] hover:bg-[#2C2F35] hover:text-white"
                  }`}
                >
                  {variant.label}
                </button>
              ))}
            </div>
          )}

          {/* Multiple code blocks (for Helm) */}
          {hasCodeBlocks ? (
            <div className="space-y-4">
              {currentInstallation.codeBlocks!.map((block, index) => (
                <div key={index}>
                  <p className="mb-2 text-gray-400 text-sm md:text-base">{block.label}</p>
                  <div className="relative">
                    <pre className="rounded-lg p-3 md:p-4 sm:pr-14 md:pr-16 text-sm border border-[#ffffff20] overflow-x-auto">
                      <code className="block whitespace-pre-wrap wrap-break-word">
                        {block.code}
                      </code>
                    </pre>
                    <button
                      onClick={() => handleCopyBlock(block.code, index)}
                      className={`absolute right-2 top-2 rounded px-2 py-1 text-sm text-white transition-colors ${
                        copiedBlockIndex === index
                          ? "bg-green-500"
                          : "bg-blue-600 hover:bg-blue-700"
                      }`}
                    >
                      {copiedBlockIndex === index
                        ? COPY_LABELS[lang].copied
                        : COPY_LABELS[lang].copy}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            /* Single code block with copy button */
            <div className="relative border border-[#ffffff20] max-w-full lg:max-w-[530px] rounded-lg p-2 sm:pr-14 md:pr-16">
              <pre className="rounded-lg p-3 md:p-4 sm:pr-14 md:pr-16 text-sm overflow-x-auto">
                <code className="block whitespace-pre-wrap wrap-break-word">
                  {getCurrentCode()}
                </code>
              </pre>

              <button
                onClick={handleCopy}
                className={`absolute right-2 top-2 rounded px-2 py-1 text-sm text-white transition-colors border border-[#ffffff20] ${
                  isCopied ? "bg-green-500" : "bg-blue-600 hover:bg-blue-700"
                }`}
              >
                {isCopied ? COPY_LABELS[lang].copied : COPY_LABELS[lang].copy}
              </button>
            </div>
          )}

          <a
            href={getLocalizedHref(lang, "installation")}
            className="inline-flex items-center gap-1 mt-4 md:mt-5 text-blue-400 hover:text-blue-600 text-sm md:text-base"
          >
            {texts.readMore}
            <svg
              width="18"
              height="18"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
              strokeLinecap="round"
              strokeLinejoin="round"
            >
              <path d="M5 12h14M12 5l7 7-7 7" />
            </svg>
          </a>
        </div>

        <div className="w-full lg:w-[50%]">
          <div className="flex-1 relative rounded-lg overflow-hidden shadow-lg border border-[#ffffff20]">
            <LiteYouTubeEmbed
              videoId="KaNLPkuu03M"
              title={texts.videoTitle}
              playLabel={VIDEO_PLAY_LABELS[lang].replace("{title}", texts.videoTitle)}
              thumbnailSrc="/images/index/how-to-install-preview.svg"
            />
          </div>
        </div>
      </div>
    </div>
  );
}
