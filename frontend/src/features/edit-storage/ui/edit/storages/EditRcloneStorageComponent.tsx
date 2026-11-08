import { TextInput, Textarea, Tooltip } from '@mantine/core';
import { Info } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import type { RcloneStorageSettings } from '@/entities/storages';
import { toOptionalValue } from '@/shared/lib';

interface Props {
  settings: RcloneStorageSettings;
  isSaved: boolean;
  onChange: (settings: RcloneStorageSettings) => void;
}

export function EditRcloneStorageComponent({ settings, isSaved, onChange }: Props) {
  const { t } = useTranslation();

  return (
    <>
      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-start">
        <div className="mb-1 min-w-[110px] pt-1 sm:mb-0 sm:pr-2">{t('storages.fields.config')}</div>
        <div className="flex w-full flex-col">
          <div className="flex items-start">
            <Textarea
              aria-label={t('storages.fields.config')}
              value={settings.configContent || ''}
              onChange={(e) => {
                onChange({ ...settings, configContent: toOptionalValue(e.target.value) });
              }}
              className="w-full max-w-[400px] font-mono text-xs"
              // eslint-disable-next-line i18next/no-literal-string -- rclone.conf example, not copy
              placeholder={`[myremote]
type = s3
provider = AWS
access_key_id = YOUR_ACCESS_KEY
secret_access_key = YOUR_SECRET_KEY
region = us-east-1`}
              rows={8}
              style={{ resize: 'vertical' }}
            />

            <Tooltip className="cursor-pointer" label={t('storages.edit.rclone.configTooltip')}>
              <Info className="mt-2 ml-2 inline-block shrink-0 align-middle" size="1em" />
            </Tooltip>
          </div>
        </div>
      </div>

      {!isSaved && (
        <div className="mb-2 flex items-center">
          <div className="hidden min-w-[110px] sm:block" />

          <div className="max-w-[300px] text-xs text-muted">
            {t('storages.edit.rclone.configHiddenNote')}
          </div>
        </div>
      )}

      <div className="mb-2 flex items-center">
        <div className="hidden min-w-[110px] sm:block" />

        <div className="text-xs text-accent">
          <a href="https://rclone.org/docs/" target="_blank" rel="noreferrer">
            {t('storages.edit.rclone.docsLink')}
          </a>
        </div>
      </div>

      <div className="mb-1 flex w-full flex-col items-start sm:flex-row sm:items-center">
        <div className="mb-1 min-w-0 wrap-anywhere sm:mb-0 sm:w-[110px] sm:shrink-0 sm:pr-2">
          {t('storages.fields.remotePath')}
        </div>
        <div className="flex w-full min-w-0 items-center">
          <TextInput
            aria-label={t('storages.fields.remotePath')}
            value={settings.remotePath || ''}
            onChange={(e) => {
              onChange({ ...settings, remotePath: toOptionalValue(e.target.value.trim()) });
            }}
            size="xs"
            className="w-full max-w-[250px]"
            placeholder={t('storages.edit.optionalPlaceholder', { example: '/backups' })}
          />

          <Tooltip className="cursor-pointer" label={t('storages.edit.rclone.remotePathTooltip')}>
            <Info className="ml-2 inline-block shrink-0 align-middle" size="1em" />
          </Tooltip>
        </div>
      </div>
    </>
  );
}
