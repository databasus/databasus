"use client";

import { getLocalizedHref, type Locale } from "@/app/i18n";
import LanguageSelectorComponent from "./LanguageSelectorComponent";

const NAVBAR_LABELS: Record<Locale | "en", { community: string; starOnGithub: string }> = {
  id: {
    community: "Komunitas",
    starOnGithub: "Beri bintang di GitHub, dukunganmu sangat berarti ❤️",
  },
  en: {
    community: "Community",
    starOnGithub: "Star on GitHub, it's really important ❤️",
  },
  ru: {
    community: "Сообщество",
    starOnGithub: "Поставьте звезду на GitHub, это очень важно ❤️",
  },
  es: {
    community: "Comunidad",
    starOnGithub: "Danos una estrella en GitHub, es muy importante ❤️",
  },
  pt: {
    community: "Comunidade",
    starOnGithub: "Dê uma estrela no GitHub, é muito importante ❤️",
  },
  zh: {
    community: "社区",
    starOnGithub: "在 GitHub 上点个星，这对我们很重要 ❤️",
  },
  fr: {
    community: "Communauté",
    starOnGithub: "Mettez une étoile sur GitHub, c'est très important ❤️",
  },
};

export default function DocsNavbarComponent({ lang = "en" }: { lang?: Locale | "en" }) {
  const labels = NAVBAR_LABELS[lang];
  const homeHref = getLocalizedHref(lang, "");
  return (
    <nav className="flex h-[60px] w-full justify-center border-b border-[#ffffff20] bg-[#0F1115] sm:h-[70px] md:h-[80px]">
      <div className="flex min-w-0 grow items-center px-4 sm:px-6 md:px-10">
        <a href={homeHref} className="flex items-center">
          <img
            src="/logo.svg"
            alt="Databasus logo"
            className="shrink-0 sm:h-[35px] sm:w-[35px] md:h-[35px] md:w-[35px]"
            fetchPriority="high"
            loading="eager"
          />

          <div className="ml-4 select-none text-lg font-bold text-white sm:ml-3 sm:text-xl md:ml-4 md:text-2xl">
            Databasus
          </div>
        </a>

        <div className="ml-auto mr-4 hidden gap-3 sm:mr-6 md:mr-10 lg:flex lg:gap-5">
          <a
            className="text-gray-300 hover:text-white transition-colors"
            href="https://t.me/databasus_community"
            target="_blank"
            rel="noopener noreferrer"
          >
            {labels.community}
          </a>
        </div>

        <div className="ml-auto lg:ml-0 flex items-stretch">
          <a
            href="https://github.com/databasus/databasus"
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-2 hover:opacity-70 rounded-l-lg px-2 md:px-3 py-2 text-[14px] border border-[#ffffff20] bg-[#0C0E13] transition-opacity"
          >
            <svg
              aria-hidden={true}
              width="24"
              height="24"
              viewBox="0 0 20 20"
              fill="none"
              xmlns="http://www.w3.org/2000/svg"
            >
              <g clipPath="url(#clip0_docs_nav)">
                <path
                  fillRule="evenodd"
                  clipRule="evenodd"
                  d="M9.9702 0C4.45694 0 0 4.4898 0 10.0443C0 14.4843 2.85571 18.2427 6.81735 19.5729C7.31265 19.6729 7.49408 19.3567 7.49408 19.0908C7.49408 18.858 7.47775 18.0598 7.47775 17.2282C4.70429 17.8269 4.12673 16.0308 4.12673 16.0308C3.68102 14.8667 3.02061 14.5676 3.02061 14.5676C2.11286 13.9522 3.08673 13.9522 3.08673 13.9522C4.09367 14.0188 4.62204 14.9833 4.62204 14.9833C5.51327 16.5131 6.94939 16.0808 7.52714 15.8147C7.60959 15.1661 7.87388 14.7171 8.15449 14.4678C5.94245 14.2349 3.6151 13.3702 3.6151 9.51204C3.6151 8.41449 4.01102 7.51653 4.63837 6.81816C4.53939 6.56878 4.19265 5.53755 4.73755 4.15735C4.73755 4.15735 5.57939 3.89122 7.47755 5.18837C8.29022 4.9685 9.12832 4.85666 9.9702 4.85571C10.812 4.85571 11.6702 4.97225 12.4627 5.18837C14.361 3.89122 15.2029 4.15735 15.2029 4.15735C15.7478 5.53755 15.4008 6.56878 15.3018 6.81816C15.9457 7.51653 16.3253 8.41449 16.3253 9.51204C16.3253 13.3702 13.998 14.2182 11.7694 14.4678C12.1327 14.7837 12.4461 15.3822 12.4461 16.3302C12.4461 17.6771 12.4298 18.7582 12.4298 19.0906C12.4298 19.3567 12.6114 19.6729 13.1065 19.5731C17.0682 18.2424 19.9239 14.4843 19.9239 10.0443C19.9402 4.4898 15.4669 0 9.9702 0Z"
                  fill="white"
                />
              </g>
              <defs>
                <clipPath id="clip0_docs_nav">
                  <rect width="20" height="20" fill="white" />
                </clipPath>
              </defs>
            </svg>
            <span className="hidden xl:inline">{labels.starOnGithub}</span>
            <span className="inline xl:hidden">GitHub</span>
          </a>

          <LanguageSelectorComponent isSplitEnd />
        </div>
      </div>
    </nav>
  );
}
