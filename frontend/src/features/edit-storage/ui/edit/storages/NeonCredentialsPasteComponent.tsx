import { useTranslation } from 'react-i18next';

import {
  type NeonObjectStorageCredentials,
  S3_COMPATIBLE_SERVICES,
  parseNeonObjectStorageCredentials,
} from '@/entities/storages';
import { NEON_CREDENTIALS_IMPORT_MESSAGES } from '@/features/edit-storage/lib/neonCredentialsImportMessages';
import { useClipboardTextImport } from '@/shared/lib/useClipboardTextImport';
import { ClipboardPasteBlockComponent } from '@/shared/ui';

interface Props {
  onParsed: (credentials: NeonObjectStorageCredentials) => void;
}

export const NeonCredentialsPasteComponent = ({ onParsed }: Props) => {
  const { t } = useTranslation();
  const { startImport, pasteModal } = useClipboardTextImport({
    parse: parseNeonObjectStorageCredentials,
    isIncomplete: (credentials) => credentials.missingKeys.length > 0,
    onParsed,
    messages: NEON_CREDENTIALS_IMPORT_MESSAGES,
  });

  const { name: serviceName, credentialsPaste } = S3_COMPATIBLE_SERVICES.neon;

  return (
    <ClipboardPasteBlockComponent
      title={t('storages.edit.s3.neonPaste.title', { service: serviceName })}
      hint={t(credentialsPaste.hintKey)}
      example={credentialsPaste.example}
      pasteLabel={t('storages.edit.s3.neonPaste.pasteFromClipboard')}
      dividerLabel={t('storages.edit.s3.neonPaste.orEnterManually')}
      onPaste={() => void startImport()}
      pasteModal={pasteModal}
    />
  );
};
