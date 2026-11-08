import { TextInput, Tooltip } from '@mantine/core';
import { Info } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { TeamsNotifierSettings } from '@/entities/notifiers';
import { getWebsitePageUrl, useLocale } from '@/shared/i18n';
import { toOptionalValue } from '@/shared/lib';

interface Props {
  settings: TeamsNotifierSettings;
  isSaved: boolean;
  onChange: (settings: TeamsNotifierSettings) => void;
}

export function EditTeamsNotifierComponent({ settings, isSaved, onChange }: Props) {
  const { t } = useTranslation();
  const { locale } = useLocale();

  return (
    <>
      <div className="mb-1 max-w-[250px] sm:ml-[150px]" style={{ lineHeight: 1 }}>
        <a
          className="text-xs !text-accent"
          href={getWebsitePageUrl('notifiersTeams', locale)}
          target="_blank"
          rel="noreferrer"
        >
          {t('notifiers.teams.howToConnect')}
        </a>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[150px] sm:shrink-0 sm:pr-2">
          {t('notifiers.teams.powerAutomateUrl')}
        </div>
        <div className="flex w-full min-w-0 items-center">
          <TextInput
            aria-label={t('notifiers.teams.powerAutomateUrl')}
            value={settings.powerAutomateUrl ?? ''}
            onChange={(e) =>
              onChange({
                ...settings,
                powerAutomateUrl: toOptionalValue(e.target.value.trim()),
              })
            }
            size="xs"
            className="w-full max-w-[250px]"
            placeholder={
              isSaved
                ? t('notifiers.secretKeptPlaceholder')
                : 'https://prod-00.westeurope.logic.azure.com:443/workflows/.....'
            }
          />

          <Tooltip className="cursor-pointer" label={t('notifiers.teams.powerAutomateUrlTooltip')}>
            <Info className="ml-2 inline-block shrink-0 align-middle" size="1em" />
          </Tooltip>
        </div>
      </div>
    </>
  );
}
