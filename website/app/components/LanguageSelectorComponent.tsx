"use client";

import { usePathname } from "next/navigation";
import { useEffect, useRef, useState } from "react";
import { LANGUAGE_SELECTOR_LABELS } from "./landingLabels";
import {
  LOCALES,
  LOCALE_NATIVE_NAMES,
  getLocalizedHref,
  isTranslatedPath,
  splitLocaleFromPathname,
  type Locale,
} from "@/app/i18n";

const SELECTABLE_LOCALES: (Locale | "en")[] = ["en", ...LOCALES];

function getLocaleHref(locale: Locale | "en", path: string): string {
  return getLocalizedHref(locale, isTranslatedPath(path) ? path : "");
}

export default function LanguageSelectorComponent({
  isSplitEnd = false,
}: {
  isSplitEnd?: boolean;
}) {
  const pathname = usePathname();
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);
  const buttonRef = useRef<HTMLButtonElement>(null);

  const { locale: currentLocale, path } = splitLocaleFromPathname(pathname);

  useEffect(() => {
    if (!isOpen) return;
    const closeOnOutsideClick = (event: MouseEvent) => {
      if (!dropdownRef.current?.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener("mousedown", closeOnOutsideClick);
    const dismissOnEscape = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      setIsOpen(false);
      buttonRef.current?.focus();
    };
    document.addEventListener("keydown", dismissOnEscape);
    return () => {
      document.removeEventListener("mousedown", closeOnOutsideClick);
      document.removeEventListener("keydown", dismissOnEscape);
    };
  }, [isOpen]);

  return (
    <div
      className={`relative ${isSplitEnd ? "flex" : "ml-2 md:ml-3"}`}
      ref={dropdownRef}
      onMouseEnter={() => setIsOpen(true)}
      onMouseLeave={() => setIsOpen(false)}
    >
      <button
        ref={buttonRef}
        onClick={() => setIsOpen(!isOpen)}
        className={`flex items-center border border-[#ffffff20] bg-[#0C0E13] px-2 py-2 text-[14px] text-gray-300 transition-opacity hover:opacity-70 md:px-3 ${
          isSplitEnd ? "h-full rounded-r-lg border-l-0" : "rounded-lg"
        }`}
        aria-label={LANGUAGE_SELECTOR_LABELS[currentLocale]}
        aria-expanded={isOpen}
      >
        {currentLocale.toUpperCase()}
      </button>

      {isOpen && (
        // pt-2 instead of a margin keeps the hover area contiguous, so moving
        // the cursor from the button to the list does not close the dropdown
        <div className="absolute right-0 top-full z-50 pt-2">
          <div className="w-40 overflow-hidden rounded-lg border border-[#ffffff20] bg-[#0F1115] py-1 shadow-2xl">
            {SELECTABLE_LOCALES.map((locale) => (
              <a
                key={locale}
                href={getLocaleHref(locale, path)}
                className={`block px-3 py-2 text-sm transition-colors ${
                  locale === currentLocale
                    ? "font-medium text-white"
                    : "text-gray-400 hover:bg-[#1f2937] hover:text-white"
                }`}
              >
                {LOCALE_NATIVE_NAMES[locale]}
              </a>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
