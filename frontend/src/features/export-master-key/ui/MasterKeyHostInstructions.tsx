import { useTranslation } from 'react-i18next';

import { InlineCodeComponent } from '@/shared/ui';

interface Props {
  masterKeyPath: string;
}

// The key can always be copied from the server itself, which needs neither identity proof nor a
// browser download.
export function MasterKeyHostInstructions({ masterKeyPath }: Props) {
  const { t } = useTranslation();
  // eslint-disable-next-line i18next/no-literal-string -- shell command
  const copyCommand = `docker cp databasus:${masterKeyPath} ./databasus-master-key.txt`;

  return (
    <div className="text-sm">
      <p>{t('masterKey.host.description')}</p>
      <p className="mt-1">
        {t('masterKey.host.path')} <InlineCodeComponent>{masterKeyPath}</InlineCodeComponent>
      </p>
      <pre className="mt-2 rounded-md bg-code p-3 font-mono text-xs break-all whitespace-pre-wrap">
        {copyCommand}
      </pre>
      <p className="mt-1 text-xs text-muted">{t('masterKey.host.containerName')}</p>
    </div>
  );
}
