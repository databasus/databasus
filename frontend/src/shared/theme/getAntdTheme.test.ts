import { theme } from 'antd';
import { describe, expect, it } from 'vitest';

import { getAntdTheme } from './getAntdTheme';

const getLuminance = ({ color, surface = '#ffffff' }: { color: string; surface?: string }) => {
  const background = surface.match(/[\da-f]{2}/gi)!.map((channel) => parseInt(channel, 16));
  const rgba = color
    .match(/^rgba?\(([^)]+)\)/)?.[1]
    .split(',')
    .map(Number);
  const channels = rgba
    ? rgba
        .slice(0, 3)
        .map(
          (channel, index) =>
            (channel * (rgba[3] ?? 1) + background[index] * (1 - (rgba[3] ?? 1))) / 255,
        )
    : color.match(/[\da-f]{2}/gi)!.map((channel) => parseInt(channel, 16) / 255);
  const linear = channels.map((channel) =>
    channel <= 0.04045 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2.4,
  );
  return linear[0] * 0.2126 + linear[1] * 0.7152 + linear[2] * 0.0722;
};

describe('getAntdTheme', () => {
  it.each(['light', 'dark'] as const)(
    'keeps links and focus distinguishable when surfaces use the %s theme',
    (mode) => {
      const tokens = theme.getDesignToken(getAntdTheme(mode));
      const background = getLuminance({ color: tokens.colorBgContainer });

      for (const color of [
        tokens.colorText,
        tokens.colorTextSecondary,
        tokens.colorTextPlaceholder,
        tokens.colorLink,
        tokens.colorTextDescription,
        tokens.colorErrorText,
        tokens.colorSuccessText,
        tokens.colorWarningText,
      ]) {
        const foreground = getLuminance({ color: color, surface: tokens.colorBgContainer });
        expect(
          (Math.max(background, foreground) + 0.05) / (Math.min(background, foreground) + 0.05),
        ).toBeGreaterThanOrEqual(4.5);
      }

      const border = getLuminance({ color: tokens.colorBorder });
      expect(
        (Math.max(background, border) + 0.05) / (Math.min(background, border) + 0.05),
      ).toBeGreaterThanOrEqual(3);
    },
  );
});
