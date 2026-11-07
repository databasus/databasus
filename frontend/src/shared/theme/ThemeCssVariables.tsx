import { useLayoutEffect } from 'react';

import { getThemeCssVariables } from '@/shared/theme/themeTokens';
import { useTheme } from '@/shared/theme/useTheme';

export const ThemeCssVariables = () => {
  const { resolvedTheme } = useTheme();

  useLayoutEffect(() => {
    const root = document.documentElement;
    const properties = getThemeCssVariables(resolvedTheme);
    const previousProperties = Object.keys(properties).map((property) => ({
      property,
      value: root.style.getPropertyValue(property),
      priority: root.style.getPropertyPriority(property),
    }));

    for (const [property, value] of Object.entries(properties)) {
      root.style.setProperty(property, value);
    }

    return () => {
      for (const { property, value, priority } of previousProperties) {
        if (value) root.style.setProperty(property, value, priority);
        else root.style.removeProperty(property);
      }
    };
  }, [resolvedTheme]);

  return null;
};
