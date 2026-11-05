import { theme } from 'antd';
import { useLayoutEffect } from 'react';

export const ThemeCssVariables = () => {
  const { token } = theme.useToken();

  useLayoutEffect(() => {
    const root = document.documentElement;
    const properties = {
      '--app-page': token.colorBgLayout,
      '--app-surface': token.colorBgContainer,
      '--app-raised': token.colorBgElevated,
      '--app-content': token.colorText,
      '--app-muted': token.colorTextSecondary,
      '--app-border': token.colorBorder,
      '--app-hover': token.colorFillTertiary,
      '--app-selected': token.controlItemBgActive,
      '--app-accent': token.colorLink,
      '--app-danger': token.colorErrorText,
      '--app-success': token.colorSuccessText,
      '--app-success-fill': token.colorSuccess,
      '--app-on-success-fill': token.colorTextLightSolid,
      '--app-warning': token.colorWarningText,
      '--app-focus': token.colorLink,
      '--app-code': token.colorFillTertiary,
      '--color-scrollbar-track': token.colorBgLayout,
      '--color-scrollbar-thumb': token.colorTextQuaternary,
      '--color-scrollbar-thumb-hover': token.colorTextSecondary,
    };
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
  }, [token]);

  return null;
};
