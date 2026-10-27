"use client";

import { usePathname } from "next/navigation";
import { useState, useEffect, useRef } from "react";
import { getLocalizedHref, isTranslatedPath, type Locale } from "@/app/i18n";

interface NavItem {
  title: string;
  href: string;
  children?: NavItem[];
}

const navItems: NavItem[] = [
  {
    title: "Installation",
    href: "/installation",
  },
  {
    title: "Restore verification",
    href: "/restore-verification",
  },
  {
    title: "Storages",
    href: "/storages",
    children: [
      { title: "Google Drive", href: "/storages/google-drive" },
      { title: "Cloudflare R2", href: "/storages/cloudflare-r2" },
    ],
  },
  {
    title: "Notifiers",
    href: "/notifiers",
    children: [
      { title: "Slack", href: "/notifiers/slack" },
      { title: "Microsoft Teams", href: "/notifiers/teams" },
    ],
  },
  {
    title: "Access management",
    href: "/access-management",
  },
  {
    title: "Reset password",
    href: "/password",
  },
  {
    title: "Security",
    href: "/security",
  },
  {
    title: "FAQ",
    href: "/faq",
    children: [
      { title: "How to backup localhost", href: "/faq/localhost" },
      { title: "How to backup Supabase", href: "/faq/supabase" },
    ],
  },
  {
    title: "Contribute",
    href: "/contribute",
    children: [
      { title: "How to add storage", href: "/contribute/how-to-add-storage" },
      { title: "How to add notifier", href: "/contribute/how-to-add-notifier" },
    ],
  },

  {
    title: "Comparisons",
    href: "/pgdump-alternative",
    children: [
      { title: "pg_dump alternative", href: "/pgdump-alternative" },
      { title: "Databasus vs Barman", href: "/databasus-vs-barman" },
      { title: "Databasus vs PgBackWeb", href: "/databasus-vs-pgbackweb" },
      { title: "Databasus vs pgBackRest", href: "/databasus-vs-pgbackrest" },
      { title: "Databasus vs WAL-G", href: "/databasus-vs-wal-g" },
    ],
  },
  {
    title: "Manual recovery from backup without Databasus",
    href: "/how-to-recover-without-databasus",
  },
  {
    title: "Advanced config",
    href: "/advanced-config",
  },
];

// Keyed by the English titles used in navItems.
const SIDEBAR_TITLES: Record<Locale, Record<string, string>> = {
  id: {
    Installation: "Instalasi",
    "Restore verification": "Verifikasi restore",
    Storages: "Penyimpanan",
    Notifiers: "Kanal notifikasi",
    "Access management": "Pengelolaan akses",
    "Reset password": "Atur ulang kata sandi",
    Security: "Keamanan",
    FAQ: "Pertanyaan umum",
    "How to backup localhost": "Cara backup localhost",
    "How to backup Supabase": "Cara backup Supabase",
    Contribute: "Berkontribusi",
    "How to add storage": "Cara menambahkan penyimpanan",
    "How to add notifier": "Cara menambahkan kanal notifikasi",
    Comparisons: "Perbandingan",
    "pg_dump alternative": "Alternatif pg_dump",
    "Manual recovery from backup without Databasus": "Restore manual dari backup tanpa Databasus",
    "Advanced config": "Konfigurasi lanjutan",
  },
  ru: {
    Installation: "Установка",
    "Restore verification": "Проверка восстановления",
    Storages: "Хранилища",
    Notifiers: "Уведомления",
    "Access management": "Управление доступом",
    "Reset password": "Сброс пароля",
    Security: "Безопасность",
    FAQ: "FAQ",
    "How to backup localhost": "Как бекапить localhost",
    "How to backup Supabase": "Как бекапить Supabase",
    Contribute: "Участие в разработке",
    "How to add storage": "Как добавить хранилище",
    "How to add notifier": "Как добавить уведомления",
    Comparisons: "Сравнения",
    "pg_dump alternative": "Альтернатива pg_dump",
    "Manual recovery from backup without Databasus":
      "Ручное восстановление из бекапа без Databasus",
    "Advanced config": "Расширенная настройка",
  },
  es: {
    Installation: "Instalación",
    "Restore verification": "Verificación de restauración",
    Storages: "Almacenamientos",
    Notifiers: "Notificadores",
    "Access management": "Gestión de acceso",
    "Reset password": "Restablecer contraseña",
    Security: "Seguridad",
    FAQ: "Preguntas frecuentes",
    "How to backup localhost": "Cómo respaldar localhost",
    "How to backup Supabase": "Cómo respaldar Supabase",
    Contribute: "Contribuir",
    "How to add storage": "Cómo añadir un almacenamiento",
    "How to add notifier": "Cómo añadir un notificador",
    Comparisons: "Comparaciones",
    "pg_dump alternative": "Alternativa a pg_dump",
    "Manual recovery from backup without Databasus":
      "Recuperación manual desde un respaldo sin Databasus",
    "Advanced config": "Configuración avanzada",
  },
  pt: {
    Installation: "Instalação",
    "Restore verification": "Verificação de restauração",
    Storages: "Armazenamentos",
    Notifiers: "Notificadores",
    "Access management": "Gestão de acesso",
    "Reset password": "Redefinir senha",
    Security: "Segurança",
    FAQ: "Perguntas frequentes",
    "How to backup localhost": "Como fazer backup do localhost",
    "How to backup Supabase": "Como fazer backup do Supabase",
    Contribute: "Contribuir",
    "How to add storage": "Como adicionar um armazenamento",
    "How to add notifier": "Como adicionar um notificador",
    Comparisons: "Comparações",
    "pg_dump alternative": "Alternativa ao pg_dump",
    "Manual recovery from backup without Databasus": "Recuperação manual do backup sem Databasus",
    "Advanced config": "Configuração avançada",
  },
  zh: {
    Installation: "安装",
    "Restore verification": "恢复验证",
    Storages: "存储",
    Notifiers: "通知",
    "Access management": "访问管理",
    "Reset password": "重置密码",
    Security: "安全",
    FAQ: "常见问题",
    "How to backup localhost": "如何备份 localhost",
    "How to backup Supabase": "如何备份 Supabase",
    Contribute: "参与贡献",
    "How to add storage": "如何添加存储",
    "How to add notifier": "如何添加通知器",
    Comparisons: "对比",
    "pg_dump alternative": "pg_dump 替代方案",
    "Manual recovery from backup without Databasus": "不使用 Databasus 手动恢复备份",
    "Advanced config": "高级配置",
  },
  fr: {
    Installation: "Installation",
    "Restore verification": "Vérification de restauration",
    Storages: "Stockages",
    Notifiers: "Notificateurs",
    "Access management": "Gestion des accès",
    "Reset password": "Réinitialiser le mot de passe",
    Security: "Sécurité",
    FAQ: "FAQ",
    "How to backup localhost": "Sauvegarder localhost",
    "How to backup Supabase": "Sauvegarder Supabase",
    Contribute: "Contribuer",
    "How to add storage": "Ajouter un stockage",
    "How to add notifier": "Ajouter un notificateur",
    Comparisons: "Comparaisons",
    "pg_dump alternative": "Alternative à pg_dump",
    "Manual recovery from backup without Databasus":
      "Restauration manuelle d'une sauvegarde sans Databasus",
    "Advanced config": "Configuration avancée",
  },
};

const NAVIGATION_HEADINGS: Record<Locale | "en", string> = {
  en: "Navigation",
  ru: "Навигация",
  es: "Navegación",
  pt: "Navegação",
  zh: "导航",
  fr: "Navigation",
  id: "Navigasi",
};

const NAVIGATION_ACTIONS: Record<
  Locale | "en",
  { toggleMenu: string; closeMenu: string; toggleSection: string }
> = {
  en: {
    toggleMenu: "Toggle navigation menu",
    closeMenu: "Close menu",
    toggleSection: "Toggle {title} section",
  },
  ru: {
    toggleMenu: "Toggle navigation menu",
    closeMenu: "Close menu",
    toggleSection: "Toggle {title} section",
  },
  es: {
    toggleMenu: "Toggle navigation menu",
    closeMenu: "Close menu",
    toggleSection: "Toggle {title} section",
  },
  pt: {
    toggleMenu: "Toggle navigation menu",
    closeMenu: "Close menu",
    toggleSection: "Toggle {title} section",
  },
  zh: {
    toggleMenu: "Toggle navigation menu",
    closeMenu: "Close menu",
    toggleSection: "Toggle {title} section",
  },
  fr: {
    toggleMenu: "Toggle navigation menu",
    closeMenu: "Close menu",
    toggleSection: "Toggle {title} section",
  },
  id: {
    toggleMenu: "Buka/tutup menu navigasi",
    closeMenu: "Tutup menu",
    toggleSection: "Buka/tutup bagian {title}",
  },
};

export default function DocsSidebarComponent({ lang = "en" }: { lang?: Locale | "en" }) {
  const pathname = usePathname();

  const localizeHref = (href: string) => {
    if (lang === "en") return href;
    const path = href.replace(/^\//, "");
    // Untranslated pages (e.g. Contribute) keep their English URLs in every locale.
    return isTranslatedPath(path) ? getLocalizedHref(lang, path) : href;
  };
  const localizeTitle = (title: string) =>
    lang === "en" ? title : (SIDEBAR_TITLES[lang][title] ?? title);
  const localizedNavItems = navItems.map((item) => ({
    ...item,
    title: localizeTitle(item.title),
    href: localizeHref(item.href),
    children: item.children?.map((child) => ({
      ...child,
      title: localizeTitle(child.title),
      href: localizeHref(child.href),
    })),
  }));
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false);
  const menuButtonRef = useRef<HTMLButtonElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const [manuallyToggledSections, setManuallyToggledSections] = useState<Set<string>>(new Set());

  const isActive = (href: string) => {
    // Normalize paths by removing trailing slashes for comparison
    const normalizedPathname =
      pathname.endsWith("/") && pathname !== "/" ? pathname.slice(0, -1) : pathname;
    const normalizedHref = href.endsWith("/") && href !== "/" ? href.slice(0, -1) : href;
    return normalizedPathname === normalizedHref;
  };

  const isParentActive = (item: NavItem) => {
    if (item.children) {
      const normalizedPathname =
        pathname.endsWith("/") && pathname !== "/" ? pathname.slice(0, -1) : pathname;
      return item.children.some((child) => {
        const normalizedChildHref =
          child.href.endsWith("/") && child.href !== "/" ? child.href.slice(0, -1) : child.href;
        return normalizedPathname === normalizedChildHref;
      });
    }
    return false;
  };

  // Determine if a section should be expanded
  const isSectionExpanded = (href: string) => {
    // If manually toggled, respect that
    if (manuallyToggledSections.has(href)) {
      return true;
    }
    // Auto-expand if a child page is active
    const item = localizedNavItems.find((i) => i.href === href);
    if (item && isParentActive(item)) {
      return true;
    }
    return false;
  };

  // Manage body overflow when mobile menu is open
  useEffect(() => {
    if (isMobileMenuOpen) {
      document.body.style.overflow = "hidden";
      closeButtonRef.current?.focus();
    } else {
      document.body.style.overflow = "";
    }

    // Cleanup on unmount
    return () => {
      document.body.style.overflow = "";
    };
  }, [isMobileMenuOpen]);

  const closeMenu = () => {
    setIsMobileMenuOpen(false);
    menuButtonRef.current?.focus();
  };

  useEffect(() => {
    if (!isMobileMenuOpen) return;
    const dismissOnEscape = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      event.preventDefault();
      setIsMobileMenuOpen(false);
      menuButtonRef.current?.focus();
    };
    document.addEventListener("keydown", dismissOnEscape);
    return () => document.removeEventListener("keydown", dismissOnEscape);
  }, [isMobileMenuOpen]);

  const toggleSection = (href: string, hasChildren: boolean) => {
    if (!hasChildren) return;

    setManuallyToggledSections((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(href)) {
        newSet.delete(href);
      } else {
        newSet.add(href);
      }
      return newSet;
    });
  };

  const renderSidebarContent = () => (
    <nav className="space-y-0.5">
      {localizedNavItems.map((item) => (
        <div key={item.href}>
          <div className="flex items-center">
            <a
              href={item.href}
              onClick={() => setIsMobileMenuOpen(false)}
              className={`flex-1 rounded-md px-2 py-1.5 text-sm transition-colors relative pl-4 ${
                isActive(item.href) ? "text-white font-medium" : "text-gray-400 hover:text-white"
              }`}
            >
              <span
                className={`absolute left-0 top-0 bottom-0 w-0.5 rounded-full transition-all duration-200 ${
                  isActive(item.href) ? "bg-blue-600 opacity-100" : "bg-transparent opacity-0"
                }`}
              />
              {item.title}
            </a>
            {item.children && (
              <button
                onClick={() => toggleSection(item.href, !!item.children)}
                className={`ml-1 rounded-md p-1 transition-all duration-200 ${
                  isActive(item.href) ? "text-white" : "text-gray-500 hover:text-gray-300"
                }`}
                aria-label={NAVIGATION_ACTIONS[lang].toggleSection.replace("{title}", item.title)}
                aria-expanded={isSectionExpanded(item.href)}
              >
                <svg
                  className={`h-3.5 w-3.5 transition-transform duration-200 ${
                    isSectionExpanded(item.href) ? "rotate-90" : ""
                  }`}
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                  strokeWidth={2}
                >
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 5l7 7-7 7" />
                </svg>
              </button>
            )}
          </div>
          {item.children && (
            <div
              inert={!isSectionExpanded(item.href)}
              className={`overflow-hidden transition-all duration-200 ease-in-out ${
                isSectionExpanded(item.href) ? "max-h-96 opacity-100" : "max-h-0 opacity-0"
              }`}
            >
              <div className="ml-3 mt-0.5 space-y-0.5 border-l border-[#ffffff20] pl-3">
                {item.children.map((child) => (
                  <a
                    key={child.href}
                    href={child.href}
                    onClick={() => setIsMobileMenuOpen(false)}
                    className={`block rounded-md px-2 py-1.5 text-sm transition-colors relative pl-4 ${
                      isActive(child.href)
                        ? "text-white font-medium"
                        : "text-gray-400 hover:text-white"
                    }`}
                  >
                    <span
                      className={`absolute left-0 top-0 bottom-0 w-0.5 rounded-full transition-all duration-200 ${
                        isActive(child.href)
                          ? "bg-blue-500 opacity-100"
                          : "bg-transparent opacity-0"
                      }`}
                    />
                    {child.title}
                  </a>
                ))}
              </div>
            </div>
          )}
        </div>
      ))}
    </nav>
  );

  return (
    <>
      {/* Mobile Menu Button */}
      <button
        ref={menuButtonRef}
        onClick={() => setIsMobileMenuOpen(!isMobileMenuOpen)}
        className="docs-mobile-navigation mx-4 my-3 flex h-11 w-11 items-center justify-center rounded-lg bg-blue-600 text-white hover:bg-blue-700 lg:hidden"
        aria-label={NAVIGATION_ACTIONS[lang].toggleMenu}
        aria-expanded={isMobileMenuOpen}
      >
        {isMobileMenuOpen ? (
          <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        ) : (
          <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M4 6h16M4 12h16M4 18h16"
            />
          </svg>
        )}
      </button>

      {/* Mobile Menu Overlay */}
      {isMobileMenuOpen && (
        <div
          className="fixed inset-0 z-40 backdrop-blur-md bg-black/50 lg:hidden"
          onClick={closeMenu}
        />
      )}

      {/* Mobile Menu */}
      <aside
        inert={!isMobileMenuOpen}
        className={`fixed bottom-0 left-0 right-0 z-40 max-h-[80vh] overflow-y-auto rounded-t-2xl border-t border-[#ffffff20] bg-[#0F1115] p-6 shadow-2xl transition-transform duration-300 lg:hidden ${
          isMobileMenuOpen ? "translate-y-0" : "translate-y-full"
        }`}
      >
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-white">{NAVIGATION_HEADINGS[lang]}</h2>
          <button
            ref={closeButtonRef}
            onClick={closeMenu}
            className="flex h-11 w-11 items-center justify-center rounded-lg text-gray-400 hover:bg-[#1f2937]"
            aria-label={NAVIGATION_ACTIONS[lang].closeMenu}
          >
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>
        {renderSidebarContent()}
      </aside>

      {/* Desktop Sidebar */}
      <aside className="hidden w-64 border-r border-[#ffffff20] bg-[#0F1115] lg:block">
        <div className="sticky top-0 h-screen overflow-y-auto p-6">{renderSidebarContent()}</div>
      </aside>
    </>
  );
}
