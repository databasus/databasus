import { App, Button, Modal, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';

import {
  DATABASE_TYPE_LABEL_KEYS,
  type Database,
  DatabaseType,
  type LimitedUserCheck,
  databaseApi,
} from '../../../entities/databases';
import {
  TransByKey,
  type TranslationKey,
  getWebsitePageUrl,
  translateApiError,
  useLocale,
} from '../../../shared/i18n';
import { toDatabaseSettingsTarget } from './toDatabaseSettingsTarget';

interface Props {
  database: Database;
  onReadOnlyUserUpdated: (database: Database) => void;

  onGoBack: () => void;
  onSkipped: () => void;
  onReadOnlyUserNotSuggested: () => void;
}

const PRIVILEGES_TRUNCATE_LENGTH = 50;

const FORCED_WAL_ROTATION_WARNING_SECONDS = 15;

// Physical PostgreSQL gets a replication-only user and every other engine a read-only one. The two
// nouns inflect differently in other languages, so each kind has its own sentences.
interface RestrictedUserCopyKeys {
  checking: TranslationKey;
  title: TranslationKey;
  description: TranslationKey;
  noWriteCredentials: TranslationKey;
  create: TranslationKey;
  skipTitle: TranslationKey;
  skipQuestion: TranslationKey;
}

const READ_ONLY_USER_COPY_KEYS: RestrictedUserCopyKeys = {
  checking: 'databases.readOnlyUser.readOnly.checking',
  title: 'databases.readOnlyUser.readOnly.title',
  description: 'databases.readOnlyUser.readOnly.description',
  noWriteCredentials: 'databases.readOnlyUser.readOnly.noWriteCredentials',
  create: 'databases.readOnlyUser.readOnly.create',
  skipTitle: 'databases.readOnlyUser.readOnly.skipTitle',
  skipQuestion: 'databases.readOnlyUser.readOnly.skipQuestion',
};

const REPLICATION_ONLY_USER_COPY_KEYS: RestrictedUserCopyKeys = {
  checking: 'databases.readOnlyUser.replicationOnly.checking',
  title: 'databases.readOnlyUser.replicationOnly.title',
  description: 'databases.readOnlyUser.replicationOnly.description',
  noWriteCredentials: 'databases.readOnlyUser.replicationOnly.noWriteCredentials',
  create: 'databases.readOnlyUser.replicationOnly.create',
  skipTitle: 'databases.readOnlyUser.replicationOnly.skipTitle',
  skipQuestion: 'databases.readOnlyUser.replicationOnly.skipQuestion',
};

export const CreateReadOnlyComponent = ({
  database,
  onReadOnlyUserUpdated,
  onGoBack,
  onSkipped,
  onReadOnlyUserNotSuggested,
}: Props) => {
  const { t } = useTranslation();
  const { locale } = useLocale();
  const { message } = App.useApp();

  const [isCheckingReadOnlyUserSuggestion, setIsCheckingReadOnlyUserSuggestion] = useState(false);
  const [isCreatingReadOnlyUser, setIsCreatingReadOnlyUser] = useState(false);
  const [isShowSkipConfirmation, setShowSkipConfirmation] = useState(false);
  const [privileges, setPrivileges] = useState<string[]>([]);
  const [isPrivilegesExpanded, setIsPrivilegesExpanded] = useState(false);

  const isPhysicalPostgres = database.settings.type === DatabaseType.POSTGRES_PHYSICAL;
  const isMongodb = database.settings.type === DatabaseType.MONGODB;
  const databaseTypeName = t(DATABASE_TYPE_LABEL_KEYS[database.settings.type]);
  const settingsTarget = toDatabaseSettingsTarget(database, database.settings);
  const userKindKeys = isPhysicalPostgres
    ? REPLICATION_ONLY_USER_COPY_KEYS
    : READ_ONLY_USER_COPY_KEYS;

  const fetchLimitedUserCheck = async (): Promise<LimitedUserCheck | null> => {
    try {
      return await databaseApi.checkLimitedUser(settingsTarget);
    } catch (e) {
      message.error(translateApiError(e, t));
      return null;
    }
  };

  const getPrivilegesDisplay = () => {
    const fullText = privileges.join(', ');
    if (isPrivilegesExpanded || fullText.length <= PRIVILEGES_TRUNCATE_LENGTH) {
      return fullText;
    }

    return fullText.substring(0, PRIVILEGES_TRUNCATE_LENGTH) + '...';
  };

  const shouldShowExpandToggle = () => {
    const fullText = privileges.join(', ');
    return fullText.length > PRIVILEGES_TRUNCATE_LENGTH;
  };

  const provisionRestrictedUser = async () => {
    setIsCreatingReadOnlyUser(true);

    try {
      const limitedUser = await databaseApi.createLimitedUser(settingsTarget);

      if (isPhysicalPostgres && !limitedUser.isForcedWalRotationAvailable) {
        message.warning(
          t('databases.readOnlyUser.forcedWalRotationUnavailable'),
          FORCED_WAL_ROTATION_WARNING_SECONDS,
        );
      }

      onReadOnlyUserUpdated({
        ...database,
        settings: {
          ...database.settings,
          username: limitedUser.username,
          password: limitedUser.password,
        },
      });
    } catch (e) {
      message.error(translateApiError(e, t));
    }

    setIsCreatingReadOnlyUser(false);
  };

  const handleSkip = () => {
    setShowSkipConfirmation(true);
  };

  const handleSkipConfirmed = () => {
    setShowSkipConfirmation(false);
    onSkipped();
  };

  useEffect(() => {
    const run = async () => {
      setIsCheckingReadOnlyUserSuggestion(true);

      const limitedUserCheck = await fetchLimitedUserCheck();
      setPrivileges(limitedUserCheck?.privileges || []);

      // A failed check must not silently advance the wizard - keep the screen so the user
      // still gets the choice.
      if (limitedUserCheck && !limitedUserCheck.isAdvisable) {
        onReadOnlyUserNotSuggested();
      }

      setIsCheckingReadOnlyUserSuggestion(false);
    };
    run();
  }, []);

  if (isCheckingReadOnlyUserSuggestion) {
    return (
      <div className="flex items-center">
        <Spin />
        <span className="ml-3">{t(userKindKeys.checking)}</span>
      </div>
    );
  }

  return (
    <div>
      <div className="mb-5">
        <p className="mb-3 text-lg font-bold">{t(userKindKeys.title)}</p>

        <p className="mb-2">{t(userKindKeys.description, { databaseType: databaseTypeName })}</p>

        <ul className="mb-2 ml-5 list-disc">
          <li>{t('databases.readOnlyUser.reasons.preventsModifications')}</li>
          <li>{t('databases.readOnlyUser.reasons.leastPrivilege')}</li>
          <li>{t('databases.readOnlyUser.reasons.bestPractice')}</li>
        </ul>

        <p className="mb-2">
          <Trans
            i18nKey="databases.readOnlyUser.securityNote"
            components={{
              docsLink: (
                <a
                  href={getWebsitePageUrl('security', locale)}
                  target="_blank"
                  rel="noreferrer"
                  className="!text-blue-600 dark:!text-blue-400"
                />
              ),
            }}
          />
        </p>

        <p className="mt-3">
          <TransByKey i18nKey={userKindKeys.noWriteCredentials} components={{ bold: <b /> }} />
        </p>

        <p className="mt-3">
          {privileges.length === 0 ? (
            <Trans
              i18nKey={
                isMongodb
                  ? 'databases.readOnlyUser.noWriteRoles'
                  : 'databases.readOnlyUser.noWritePrivileges'
              }
              components={{ bold: <b /> }}
            />
          ) : (
            <>
              {isMongodb
                ? t('databases.readOnlyUser.writeRoles')
                : t('databases.readOnlyUser.writePrivileges')}{' '}
              <span
                className={shouldShowExpandToggle() ? 'cursor-pointer hover:opacity-80' : ''}
                onClick={() =>
                  shouldShowExpandToggle() && setIsPrivilegesExpanded(!isPrivilegesExpanded)
                }
              >
                {getPrivilegesDisplay()}
                {shouldShowExpandToggle() && (
                  <span className="ml-1 text-xs text-blue-600 hover:opacity-80">
                    {isPrivilegesExpanded
                      ? t('databases.readOnlyUser.collapse')
                      : t('databases.readOnlyUser.expand')}
                  </span>
                )}
              </span>
            </>
          )}
        </p>
      </div>

      <div className="mt-5 flex">
        <Button className="mr-auto" type="primary" ghost onClick={() => onGoBack()}>
          {t('common.actions.back')}
        </Button>

        <Button className="mr-2 ml-auto" danger ghost onClick={handleSkip}>
          {t('databases.readOnlyUser.skip')}
        </Button>

        <Button
          type="primary"
          onClick={provisionRestrictedUser}
          loading={isCreatingReadOnlyUser}
          disabled={isCreatingReadOnlyUser}
        >
          {t(userKindKeys.create)}
        </Button>
      </div>

      <Modal
        title={t(userKindKeys.skipTitle)}
        open={isShowSkipConfirmation}
        onCancel={() => setShowSkipConfirmation(false)}
        footer={null}
        width={450}
      >
        <div className="mb-5">
          <p className="mb-2">{t(userKindKeys.skipQuestion)}</p>

          <p className="mb-2">{t('databases.readOnlyUser.skipRisk')}</p>

          <p>{t('databases.readOnlyUser.skipAdvice')}</p>
        </div>

        <div className="flex justify-end">
          <Button className="mr-2" danger ghost onClick={handleSkipConfirmed}>
            {t('databases.readOnlyUser.acceptRisks')}
          </Button>

          <Button type="primary" onClick={() => setShowSkipConfirmation(false)}>
            {t('databases.readOnlyUser.continueSecurely')}
          </Button>
        </div>
      </Modal>
    </div>
  );
};
