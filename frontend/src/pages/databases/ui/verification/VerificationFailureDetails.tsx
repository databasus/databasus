import { CopyOutlined } from '@ant-design/icons';
import { Alert, Button, Collapse } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import type { VerificationDetails } from '@/pages/databases/model/verification/VerificationDetails';
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
    <Collapse
      size="small"
      items={[
        {
          key: 'diagnostic',
          label: t('verification.runs.details.diagnostic.title'),
          children: (
            <div className="min-w-0 space-y-3">
              {diagnostic.isTruncated && (
                <Alert
                  type="warning"
                  showIcon
                  message={t('verification.runs.details.diagnostic.truncated')}
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
                icon={<CopyOutlined aria-hidden />}
                loading={isCopying}
                onClick={() => void copyDiagnostic()}
              >
                {t('common.actions.copy')}
              </Button>
              {copyOutcome !== 'idle' && (
                <Alert
                  role="status"
                  type={copyOutcome === 'success' ? 'success' : 'error'}
                  showIcon
                  message={
                    copyOutcome === 'success'
                      ? t('verification.runs.details.diagnostic.copied')
                      : t('verification.runs.details.diagnostic.copyFailed')
                  }
                />
              )}
            </div>
          ),
        },
      ]}
    />
  );
};
