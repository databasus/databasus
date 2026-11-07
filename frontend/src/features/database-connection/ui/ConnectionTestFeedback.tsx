import { Alert, Button, Tooltip } from '@mantine/core';
import { CircleAlert, Copy } from 'lucide-react';
import { useTranslation } from 'react-i18next';

import { type ConnectionErrorCode, physicalConnectionErrorContent } from '@/entities/databases';
import type { DatabaseConnectionTestFailure } from '@/features/database-connection/model/useDatabaseConnectionTest';
import { TransByKey, translateApiError, translateLocalizedText } from '@/shared/i18n';
import { BrowserClipboard } from '@/shared/lib/BrowserClipboard';
import { useFeedback } from '@/shared/lib/feedback';

interface Props {
  failure: DatabaseConnectionTestFailure;
  isPhysical: boolean;
  username: string;
}

export const ConnectionTestFeedback = ({ failure, isPhysical, username }: Props) => {
  const { t } = useTranslation();
  const notify = useFeedback();
  const copyCommand = async (command: string) => {
    try {
      await BrowserClipboard.copyToClipboard(command);
      notify({ kind: 'success', title: t('common.messages.copiedToClipboard') });
    } catch {
      notify({ kind: 'error', title: t('databases.edit.copyFailed') });
    }
  };

  const renderCopyableCommand = (command: string) => (
    <div className="relative mt-2">
      <pre className="rounded-md bg-code p-3 pr-10 font-mono text-xs break-all whitespace-pre-wrap text-content">
        {command}
      </pre>
      <Tooltip label={t('common.actions.copy')}>
        <Button
          variant="subtle"
          size="xs"
          className="!absolute top-2 right-2"
          aria-label={t('common.actions.copy')}
          leftSection={<Copy aria-hidden size={14} />}
          onClick={() => copyCommand(command)}
        />
      </Tooltip>
    </div>
  );

  const renderClassifiedFailure = (reason: ConnectionErrorCode, serverError?: string) => {
    const content = physicalConnectionErrorContent[reason];
    const commandContext = { username };
    const steps = content.buildSteps?.(commandContext) ?? [];
    const managedNote = content.buildManagedNote?.(commandContext);

    return (
      <Alert
        color="danger"
        className="mt-3"
        title={<span className="text-sm font-bold">{t(content.titleKey)}</span>}
      >
        <div>
          <div>{t(content.summaryKey)}</div>
          {steps.map((step, index) =>
            step.type === 'command' ? (
              <div key={index}>{renderCopyableCommand(step.command)}</div>
            ) : (
              <p key={index} className="mt-2 text-sm">
                <TransByKey i18nKey={step.key} components={{ bold: <strong /> }} />
              </p>
            ),
          )}
          {managedNote && (
            <div className="mt-2 text-sm">{translateLocalizedText(managedNote, t)}</div>
          )}
          {serverError && <div className="mt-2 font-mono text-xs break-all">{serverError}</div>}
        </div>
      </Alert>
    );
  };

  if (failure.kind === 'result' && failure.result.reason) {
    return renderClassifiedFailure(failure.result.reason, failure.result.error);
  }

  const failureText =
    failure.kind === 'api'
      ? translateApiError(failure.error, t)
      : t('databases.edit.connectionTestFailed', { error: failure.result.error });

  return (
    <>
      <Alert
        color="danger"
        icon={<CircleAlert aria-hidden size={16} />}
        className="mt-3"
        title={failureText}
      />

      {!isPhysical && (
        <div className="mt-3 text-sm text-muted">{t('databases.edit.ipWhitelistHint')}</div>
      )}
    </>
  );
};
