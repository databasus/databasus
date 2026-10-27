export const BASE_URL = "https://databasus.com";

export const LOCALES = ["ru", "es", "pt", "zh", "fr", "id"] as const;

export type Locale = (typeof LOCALES)[number];

export const HREFLANG_CODES: Record<Locale, string> = {
  ru: "ru",
  es: "es",
  pt: "pt",
  zh: "zh-CN",
  fr: "fr",
  id: "id",
};

export const HTML_LANG_CODES: Record<Locale, string> = {
  ru: "ru",
  es: "es",
  pt: "pt",
  zh: "zh-CN",
  fr: "fr",
  id: "id",
};

export const OG_LOCALES: Record<Locale, string> = {
  ru: "ru_RU",
  es: "es_ES",
  pt: "pt_BR",
  zh: "zh_CN",
  fr: "fr_FR",
  id: "id_ID",
};

export const LOCALE_NATIVE_NAMES: Record<Locale | "en", string> = {
  en: "English",
  ru: "Русский",
  es: "Español",
  pt: "Português",
  zh: "中文",
  fr: "Français",
  id: "Bahasa Indonesia",
};

// Paths of the translated core, without leading slash; "" is the home page.
export const TRANSLATED_PATHS = [
  "",
  "installation",
  "installation/agent",
  "advanced-config",
  "access-management",
  "password",
  "security",
  "faq",
  "faq/localhost",
  "faq/supabase",
  "storages",
  "storages/google-drive",
  "storages/cloudflare-r2",
  "notifiers",
  "notifiers/slack",
  "notifiers/teams",
  "pgdump-alternative",
  "databasus-vs-pgbackweb",
  "databasus-vs-pgbackrest",
  "databasus-vs-barman",
  "databasus-vs-wal-g",
  "mysql-backup",
  "mongodb-backup",
  "how-to-recover-without-databasus",
  "restore-verification",
] as const;

export type TranslatedPath = (typeof TRANSLATED_PATHS)[number];

export function isLocale(value: string): value is Locale {
  return (LOCALES as readonly string[]).includes(value);
}

export function isTranslatedPath(path: string): path is TranslatedPath {
  return (TRANSLATED_PATHS as readonly string[]).includes(path);
}

// URL form is WITH trailing slash: Next renders metadata URLs that way under
// `trailingSlash: true`, and canonical, hreflang and sitemap must all match.
export function getLocalizedUrl(locale: Locale | "en", path: string): string {
  const suffix = path === "" ? "" : `/${path}`;
  if (locale === "en") {
    return `${BASE_URL}${suffix}/`;
  }
  return `${BASE_URL}/${locale}${suffix}/`;
}

export function getLanguageAlternates(path: string): Record<string, string> {
  const alternates: Record<string, string> = {
    en: getLocalizedUrl("en", path),
  };
  for (const locale of LOCALES) {
    alternates[HREFLANG_CODES[locale]] = getLocalizedUrl(locale, path);
  }
  alternates["x-default"] = getLocalizedUrl("en", path);
  return alternates;
}

// Relative counterpart of getLocalizedUrl, for in-page hrefs.
export function getLocalizedHref(locale: Locale | "en", path: string): string {
  const suffix = path === "" ? "" : `/${path}`;
  if (locale === "en") {
    return `${suffix}/`;
  }
  return `/${locale}${suffix}/`;
}

export interface LocalizedPathname {
  locale: Locale | "en";
  path: string;
}

export function splitLocaleFromPathname(pathname: string): LocalizedPathname {
  const segments = pathname.split("/").filter(Boolean);
  if (segments.length > 0 && isLocale(segments[0])) {
    return { locale: segments[0], path: segments.slice(1).join("/") };
  }
  return { locale: "en", path: segments.join("/") };
}
