import { MetadataRoute } from "next";
import {
  BASE_URL,
  LOCALES,
  TRANSLATED_PATHS,
  getLanguageAlternates,
  getLocalizedUrl,
  type TranslatedPath,
} from "./i18n";

export const dynamic = "force-static";

// The manual recovery guide is outdated: Databasus backups replace it. Its pages stay reachable for
// existing links but are not offered to search engines.
const PATHS_OUTSIDE_SITEMAP = ["how-to-recover-without-databasus"] as const;

type SitemapPath = Exclude<TranslatedPath, (typeof PATHS_OUTSIDE_SITEMAP)[number]>;

function isSitemapPath(path: TranslatedPath): path is SitemapPath {
  return !(PATHS_OUTSIDE_SITEMAP as readonly string[]).includes(path);
}

const CORE_PAGE_SETTINGS: Record<
  SitemapPath,
  { priority: number; changeFrequency: "weekly" | "monthly" }
> = {
  "": { priority: 1.0, changeFrequency: "weekly" },
  installation: { priority: 0.9, changeFrequency: "weekly" },
  "advanced-config": { priority: 0.8, changeFrequency: "monthly" },
  "access-management": { priority: 0.8, changeFrequency: "monthly" },
  password: { priority: 0.8, changeFrequency: "monthly" },
  security: { priority: 0.9, changeFrequency: "monthly" },
  faq: { priority: 0.9, changeFrequency: "monthly" },
  "faq/localhost": { priority: 0.8, changeFrequency: "monthly" },
  "faq/supabase": { priority: 0.8, changeFrequency: "monthly" },
  "faq/neon": { priority: 0.8, changeFrequency: "monthly" },
  storages: { priority: 0.8, changeFrequency: "monthly" },
  "storages/cloudflare-r2": { priority: 0.7, changeFrequency: "monthly" },
  notifiers: { priority: 0.8, changeFrequency: "monthly" },
  "notifiers/slack": { priority: 0.7, changeFrequency: "monthly" },
  "notifiers/teams": { priority: 0.7, changeFrequency: "monthly" },
  "pgdump-alternative": { priority: 0.8, changeFrequency: "monthly" },
  "databasus-vs-pgbackweb": { priority: 0.8, changeFrequency: "monthly" },
  "databasus-vs-pgbackrest": { priority: 0.8, changeFrequency: "monthly" },
  "databasus-vs-barman": { priority: 0.8, changeFrequency: "monthly" },
  "databasus-vs-wal-g": { priority: 0.8, changeFrequency: "monthly" },
  "mysql-backup": { priority: 0.9, changeFrequency: "weekly" },
  "mongodb-backup": { priority: 0.9, changeFrequency: "weekly" },
  "restore-verification": { priority: 0.9, changeFrequency: "monthly" },
};

const UNTRANSLATED_PAGE_SETTINGS: Record<
  string,
  { priority: number; changeFrequency: "monthly" }
> = {
  privacy: { priority: 0.8, changeFrequency: "monthly" },
  "terms-of-use": { priority: 0.6, changeFrequency: "monthly" },
};

export default function sitemap(): MetadataRoute.Sitemap {
  const currentDate = new Date().toISOString();

  const coreEntries = TRANSLATED_PATHS.filter(isSitemapPath).flatMap((path) => {
    const { priority, changeFrequency } = CORE_PAGE_SETTINGS[path];
    const languages = getLanguageAlternates(path);
    return (["en", ...LOCALES] as const).map((locale) => ({
      url: getLocalizedUrl(locale, path),
      lastModified: currentDate,
      changeFrequency,
      priority,
      alternates: { languages },
    }));
  });

  const untranslatedEntries = Object.entries(UNTRANSLATED_PAGE_SETTINGS).map(
    ([path, { priority, changeFrequency }]) => ({
      url: `${BASE_URL}/${path}/`,
      lastModified: currentDate,
      changeFrequency,
      priority,
    }),
  );

  return [...coreEntries, ...untranslatedEntries];
}
