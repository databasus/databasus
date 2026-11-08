import { Accordion, Alert, Button } from '@mantine/core';
import { Copy, Info, TriangleAlert } from 'lucide-react';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { VerificationDetails } from '@/pages/databases/verification/model/verificationContract';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';
import { RequestRevision } from '@/shared/lib/RequestRevision';

interface Props {
  diagnostic: NonNullable<VerificationDetails['failureDiagnostic']>;
}

export const VerificationFailureDetails = ({ diagnostic }: Props) => {
  const { t } = useTranslation();
  const [copyRevision] = useState(() => new RequestRevision());
  const [isCopying, setIsCopying] = useState(false);
  const [copyOutcome, setCopyOutcome] = useState<'idle' | 'success' | 'failure'>('idle');

  useEffect(
    () => () => {
      copyRevision.Invalidate();
    },
    [copyRevision],
  );

  const copyDiagnostic = async () => {
    if (isCopying) return;

    const revision = copyRevision.Invalidate();
    setIsCopying(true);
    setCopyOutcome('idle');

    try {
      await BrowserClipboard.copyToClipboard(diagnostic.output);
      if (copyRevision.IsCurrent(revision)) setCopyOutcome('success');
    } catch {
      if (copyRevision.IsCurrent(revision)) setCopyOutcome('failure');
    }

    if (copyRevision.IsCurrent(revision)) setIsCopying(false);
  };

  return (
    <Accordion variant="contained">
      <Accordion.Item value="diagnostic">
        <Accordion.Control>{t('verification.runs.details.diagnostic.title')}</Accordion.Control>
        <Accordion.Panel>
          <div className="min-w-0 space-y-3">
            {diagnostic.isTruncated && (
              <Alert
                color="warning"
                icon={
                  <TriangleAlert
                    aria-hidden
                    size={16}
                    className="inline-block shrink-0 align-middle"
                  />
                }
                title={t('verification.runs.details.diagnostic.truncated')}
              />
            )}
            <pre
              tabIndex={0}
              aria-label={t('verification.runs.details.diagnostic.title')}
              className="min-w-0 overflow-x-auto font-mono text-xs whitespace-pre"
            >
              {diagnostic.output}
            </pre>
            <Button
              className="!h-auto min-h-8 max-w-full !whitespace-normal"
              leftSection={
                <Copy aria-hidden size="1em" className="inline-block shrink-0 align-middle" />
              }
              loading={isCopying}
              onClick={() => void copyDiagnostic()}
              aria-busy={isCopying}
            >
              {t('common.actions.copy')}
            </Button>
            {copyOutcome !== 'idle' && (
              <Alert
                role="status"
                color={copyOutcome === 'success' ? 'success' : 'danger'}
                icon={<Info aria-hidden size={16} className="inline-block shrink-0 align-middle" />}
                title={
                  copyOutcome === 'success'
                    ? t('verification.runs.details.diagnostic.copied')
                    : t('verification.runs.details.diagnostic.copyFailed')
                }
              />
            )}
          </div>
        </Accordion.Panel>
      </Accordion.Item>
    </Accordion>
  );
};
