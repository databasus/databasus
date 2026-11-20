import { ShieldAlert, ShieldCheck, ShieldEllipsis, ShieldOff, ShieldX } from 'lucide-react';
import { describe, expect, it } from 'vitest';

import { InstanceBackupHealth } from '@/entities/instance-backups';
import { getInstanceBackupHealthPresentation } from '@/pages/instance-settings/model/getInstanceBackupHealthPresentation';
import { DICTIONARIES } from '@/shared/i18n';

const readEnglish = (key: string): unknown =>
  key
    .split('.')
    .reduce<unknown>(
      (node, part) => (node as Record<string, unknown> | undefined)?.[part],
      DICTIONARIES.en,
    );

describe('getInstanceBackupHealthPresentation', () => {
  it('gives every health its own English title', () => {
    const titles = Object.values(InstanceBackupHealth).map((health) =>
      readEnglish(getInstanceBackupHealthPresentation(health).titleKey),
    );

    expect(titles.every((title) => typeof title === 'string')).toBe(true);
    expect(new Set(titles).size).toBe(titles.length);
  });

  it.each([
    { health: 'NOT_CONFIGURED', color: 'warning', Icon: ShieldOff },
    { health: 'INITIALIZING', color: 'accent', Icon: ShieldEllipsis },
    { health: 'HEALTHY', color: 'success', Icon: ShieldCheck },
    { health: 'PARTIAL', color: 'warning', Icon: ShieldAlert },
    { health: 'FAILED', color: 'danger', Icon: ShieldX },
    { health: 'OVERDUE', color: 'warning', Icon: ShieldAlert },
    { health: 'STOPPED', color: 'danger', Icon: ShieldOff },
    { health: 'RESUMING', color: 'accent', Icon: ShieldEllipsis },
  ] as const)(
    'marks $health with the $color color and its shield icon',
    ({ health, color, Icon }) => {
      const presentation = getInstanceBackupHealthPresentation(health);

      expect(presentation.color).toBe(color);
      expect(presentation.Icon).toBe(Icon);
    },
  );
});
