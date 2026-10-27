import type { Locale } from "@/app/i18n";

export const VIDEO_PLAY_LABELS: Record<Locale | "en", string> = {
  en: "Play {title}",
  ru: "Play {title}",
  es: "Play {title}",
  pt: "Play {title}",
  zh: "Play {title}",
  fr: "Play {title}",
  id: "Putar {title}",
};

export const COPY_LABELS: Record<Locale | "en", { copy: string; copied: string }> = {
  en: { copy: "Copy", copied: "Copied!" },
  ru: { copy: "Копировать", copied: "Скопировано!" },
  es: { copy: "Copiar", copied: "¡Copiado!" },
  pt: { copy: "Copiar", copied: "Copiado!" },
  zh: { copy: "复制", copied: "已复制！" },
  fr: { copy: "Copier", copied: "Copié !" },
  id: { copy: "Salin", copied: "Tersalin!" },
};
