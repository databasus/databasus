import { getLocalizedHref, type Locale } from "@/app/i18n";
import CommunityDropdownComponent from "./CommunityDropdownComponent";
import LanguageSelectorComponent from "./LanguageSelectorComponent";
import { LANDING_LABELS } from "./landingLabels";

export interface LandingSectionLink {
  href: string;
  label: string;
}

// Fixed top navbar of the landing pages. sectionLinks are the page's own
// in-page anchors, shown before the links every landing page shares.
export default function LandingHeaderComponent({
  lang = "en",
  sectionLinks,
}: {
  lang?: Locale | "en";
  sectionLinks: LandingSectionLink[];
}) {
  const labels = LANDING_LABELS[lang];
  const homeHref = getLocalizedHref(lang, "");

  return (
    <header className="fixed top-0 left-0 right-0 z-50 flex justify-center pt-3 md:pt-5 px-4 md:px-0">
      <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
        <nav className="relative flex items-center justify-between border backdrop-blur-md bg-[#0C0E13]/80 md:bg-[#0C0E13]/20 border-[#ffffff20] px-3 py-2 rounded-xl">
          <a href={homeHref} className="flex items-center gap-2.5">
            <img
              src="/logo.svg"
              alt={labels.logoAlt}
              width={32}
              height={32}
              className="h-7 w-7 md:h-8 md:w-8"
              fetchPriority="high"
              loading="eager"
            />

            <span className="text-base md:text-lg font-semibold pl-1">
              Databasus
            </span>
          </a>

          {/* Desktop Navigation */}
          <div className="absolute left-1/2 -translate-x-1/2 hidden lg:flex items-center gap-3">
            {sectionLinks.map((link) => (
              <a
                key={link.href}
                href={link.href}
                className="py-2 hover:text-gray-300 transition-colors"
              >
                {link.label}
              </a>
            ))}

            <a
              href={getLocalizedHref(lang, "installation")}
              className="py-2 hover:text-gray-300 transition-colors"
            >
              {labels.docs}
            </a>

            <CommunityDropdownComponent
              label={labels.community}
              triggerClassName="py-2 hover:text-gray-300 transition-colors"
            />
          </div>

          {/* GitHub + language split button */}
          <div className="flex items-stretch">
            <a
              href="https://github.com/databasus/databasus"
              target="_blank"
              rel="noopener noreferrer"
              className="flex items-center gap-2 hover:opacity-70 rounded-l-lg px-2 md:px-3 py-2 text-[14px] border border-[#ffffff20] bg-[#0C0E13] transition-colors"
            >
              <svg
                aria-hidden={true}
                width="24"
                height="24"
                viewBox="0 0 20 20"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <g clipPath="url(#clip0_1_2459)">
                  <path
                    fillRule="evenodd"
                    clipRule="evenodd"
                    d="M9.9702 0C4.45694 0 0 4.4898 0 10.0443C0 14.4843 2.85571 18.2427 6.81735 19.5729C7.31265 19.6729 7.49408 19.3567 7.49408 19.0908C7.49408 18.858 7.47775 18.0598 7.47775 17.2282C4.70429 17.8269 4.12673 16.0308 4.12673 16.0308C3.68102 14.8667 3.02061 14.5676 3.02061 14.5676C2.11286 13.9522 3.08673 13.9522 3.08673 13.9522C4.09367 14.0188 4.62204 14.9833 4.62204 14.9833C5.51327 16.5131 6.94939 16.0808 7.52714 15.8147C7.60959 15.1661 7.87388 14.7171 8.15449 14.4678C5.94245 14.2349 3.6151 13.3702 3.6151 9.51204C3.6151 8.41449 4.01102 7.51653 4.63837 6.81816C4.53939 6.56878 4.19265 5.53755 4.73755 4.15735C4.73755 4.15735 5.57939 3.89122 7.47755 5.18837C8.29022 4.9685 9.12832 4.85666 9.9702 4.85571C10.812 4.85571 11.6702 4.97225 12.4627 5.18837C14.361 3.89122 15.2029 4.15735 15.2029 4.15735C15.7478 5.53755 15.4008 6.56878 15.3018 6.81816C15.9457 7.51653 16.3253 8.41449 16.3253 9.51204C16.3253 13.3702 13.998 14.2182 11.7694 14.4678C12.1327 14.7837 12.4461 15.3822 12.4461 16.3302C12.4461 17.6771 12.4298 18.7582 12.4298 19.0906C12.4298 19.3567 12.6114 19.6729 13.1065 19.5731C17.0682 18.2424 19.9239 14.4843 19.9239 10.0443C19.9402 4.4898 15.4669 0 9.9702 0Z"
                    fill="white"
                  />
                </g>
                <defs>
                  <clipPath id="clip0_1_2459">
                    <rect width="20" height="20" fill="white" />
                  </clipPath>
                </defs>
              </svg>
              <span className="hidden 2xl:inline">{labels.starOnGithub}</span>
              <span className="inline 2xl:hidden">GitHub</span>
            </a>

            <LanguageSelectorComponent isSplitEnd />
          </div>
        </nav>
      </div>
    </header>
  );
}
