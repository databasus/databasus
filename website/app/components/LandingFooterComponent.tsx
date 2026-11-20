import { getLocalizedHref, type Locale } from "@/app/i18n";
import {
  COMMUNITY_LINKS,
  SOCIAL_MEDIA_LINKS,
  type SiteLink,
} from "./communityLinks";
import { LANDING_LABELS } from "./landingLabels";

const LINK_CLASS_NAME = "text-gray-300 hover:text-white transition-colors";

function FooterColumn({ title, links }: { title: string; links: SiteLink[] }) {
  return (
    <div>
      <p className="mb-3 text-sm md:text-base font-semibold text-white">{title}</p>

      <ul className="flex flex-col gap-2 text-sm md:text-base">
        {links.map((link) => (
          <li key={link.href}>
            <a
              href={link.href}
              className={LINK_CLASS_NAME}
              {...(link.isExternal
                ? { target: "_blank", rel: "noopener noreferrer" }
                : {})}
            >
              {link.label}
            </a>
          </li>
        ))}
      </ul>
    </div>
  );
}

export default function LandingFooterComponent({
  lang = "en",
}: {
  lang?: Locale | "en";
}) {
  const labels = LANDING_LABELS[lang];
  const homeHref = getLocalizedHref(lang, "");

  const backupLinks: SiteLink[] = [
    { label: labels.postgresqlBackup, href: homeHref, isExternal: false },
    {
      label: labels.mysqlBackup,
      href: getLocalizedHref(lang, "mysql-backup"),
      isExternal: false,
    },
    {
      label: labels.mongodbBackup,
      href: getLocalizedHref(lang, "mongodb-backup"),
      isExternal: false,
    },
  ];
  const resourceLinks: SiteLink[] = [
    {
      label: labels.documentation,
      href: getLocalizedHref(lang, "installation"),
      isExternal: false,
    },
    {
      label: "GitHub",
      href: "https://github.com/databasus/databasus",
      isExternal: true,
    },
    {
      label: labels.developer,
      href: "https://rostislav-dugin.com",
      isExternal: true,
    },
  ];

  return (
    <footer className="py-10 md:py-14 border-t border-[#ffffff20] px-4 md:px-6 lg:px-0">
      <div className="mx-auto w-full max-w-[1000px] 2xl:max-w-[1200px]">
        <div className="grid grid-cols-2 gap-x-6 gap-y-8 md:grid-cols-[1.5fr_repeat(4,1fr)]">
          <div className="col-span-2 md:col-span-1">
            <a href={homeHref} className="mb-4 flex items-center gap-2.5">
              <img
                src="/logo.svg"
                alt={labels.logoAlt}
                width={32}
                height={32}
                className="h-7 w-7 md:h-8 md:w-8"
              />

              <span className="text-base md:text-lg font-semibold">
                Databasus
              </span>
            </a>

            <a
              href="mailto:info@databasus.com"
              className={`text-sm md:text-base ${LINK_CLASS_NAME}`}
            >
              info@databasus.com
            </a>
          </div>

          <FooterColumn title={labels.backups} links={backupLinks} />
          <FooterColumn title={labels.resources} links={resourceLinks} />
          <FooterColumn title={labels.community} links={COMMUNITY_LINKS} />
          <FooterColumn title={labels.socialMedia} links={SOCIAL_MEDIA_LINKS} />
        </div>

        <div className="mt-10 flex flex-col gap-3 border-t border-[#ffffff20] pt-6 text-sm md:text-base text-gray-400 md:flex-row md:items-center md:justify-between">
          <p>{labels.copyright}</p>

          <div className="flex flex-wrap gap-4 md:gap-6">
            <a href="/privacy" className="hover:text-white transition-colors">
              {labels.privacy}
            </a>
            <a
              href="/terms-of-use"
              className="hover:text-white transition-colors"
            >
              {labels.termsOfUse}
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
}
