import { LanguageSelectorComponent } from '@/shared/ui/LanguageSelectorComponent';
import { ThemeToggleComponent } from '@/shared/ui/ThemeToggleComponent';

// Language and theme are both per-browser display preferences, so they share one split button:
// language on the left, theme on the right.
export function LanguageThemeControlComponent() {
  return (
    <div className="flex items-stretch">
      <LanguageSelectorComponent isSplitStart />
      <ThemeToggleComponent isSplitEnd />
    </div>
  );
}
