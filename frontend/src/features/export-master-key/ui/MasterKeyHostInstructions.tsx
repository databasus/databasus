import { Trans } from 'react-i18next';

import { InlineCodeComponent } from '@/shared/ui';

// The key can always be copied from the server itself, which needs neither identity proof nor a
// browser download. The data directory is a volume, so its host path depends on the installation.
export function MasterKeyHostInstructions() {
  return (
    <p className="text-sm">
      <Trans i18nKey="masterKey.host.location" components={{ code: <InlineCodeComponent /> }} />
    </p>
  );
}
