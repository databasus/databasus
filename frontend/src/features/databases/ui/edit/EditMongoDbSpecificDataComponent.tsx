import { CopyOutlined, DownOutlined, InfoCircleOutlined, UpOutlined } from '@ant-design/icons';
import { App, Button, Input, InputNumber, Switch, Tooltip } from 'antd';
import { useEffect, useState } from 'react';

import { type Database, databaseApi } from '../../../../entity/databases';
import { MongodbConnectionStringParser } from '../../../../entity/databases/model/mongodb/MongodbConnectionStringParser';
import { ToastHelper } from '../../../../shared/toast';

interface Props {
  database: Database;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  isShowBackButton: boolean;
  onBack: () => void;

  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (database: Database) => void;

  isShowDbName?: boolean;
}

export const EditMongoDbSpecificDataComponent = ({
  database,

  isShowCancelButton,
  onCancel,

  isShowBackButton,
  onBack,

  saveButtonText,
  isSaveToApi,
  onSaved,
  isShowDbName = true,
}: Props) => {
  const { message } = App.useApp();

  const [editingDatabase, setEditingDatabase] = useState<Database>();
  const [isSaving, setIsSaving] = useState(false);

  const [isConnectionTested, setIsConnectionTested] = useState(false);
  const [isTestingConnection, setIsTestingConnection] = useState(false);
  const [isConnectionFailed, setIsConnectionFailed] = useState(false);

  const hasAdvancedValues = !!database.mongodb?.authDatabase;
  const [isShowAdvanced, setShowAdvanced] = useState(hasAdvancedValues);

  const parseFromClipboard = async () => {
    try {
      const text = await navigator.clipboard.readText();
      const trimmedText = text.trim();

      if (!trimmedText) {
        message.error('Clipboard is empty');
        return;
      }

      const result = MongodbConnectionStringParser.parse(trimmedText);

      if ('error' in result) {
        message.error(result.error);
        return;
      }

      if (!editingDatabase?.mongodb) return;

      const updatedDatabase: Database = {
        ...editingDatabase,
        mongodb: {
          ...editingDatabase.mongodb,
          host: result.host,
          port: result.port,
          username: result.username,
          password: result.password,
          database: result.database,
          authDatabase: result.authDatabase,
          isHttps: result.useTls,
          cpuCount: 1,
        },
      };

      setEditingDatabase(updatedDatabase);
      setIsConnectionTested(false);
      message.success('Connection string parsed successfully');
    } catch {
      message.error('Failed to read clipboard. Please check browser permissions.');
    }
  };

  const testConnection = async () => {
    if (!editingDatabase?.mongodb) return;
    setIsTestingConnection(true);
    setIsConnectionFailed(false);

    const trimmedDatabase = {
      ...editingDatabase,
      mongodb: {
        ...editingDatabase.mongodb,
        password: editingDatabase.mongodb.password?.trim(),
      },
    };

    try {
      await databaseApi.testDatabaseConnectionDirect(trimmedDatabase);
      setIsConnectionTested(true);
      ToastHelper.showToast({
        title: 'Connection test passed',
        description: 'You can continue with the next step',
      });
    } catch (e) {
      setIsConnectionFailed(true);
      alert((e as Error).message);
    }

    setIsTestingConnection(false);
  };

  const saveDatabase = async () => {
    if (!editingDatabase?.mongodb) return;

    const trimmedDatabase = {
      ...editingDatabase,
      mongodb: {
        ...editingDatabase.mongodb,
        password: editingDatabase.mongodb.password?.trim(),
      },
    };

    if (isSaveToApi) {
      setIsSaving(true);

      try {
        await databaseApi.updateDatabase(trimmedDatabase);
      } catch (e) {
        alert((e as Error).message);
      }

      setIsSaving(false);
    }

    onSaved(trimmedDatabase);
  };

  useEffect(() => {
    setIsSaving(false);
    setIsConnectionTested(false);
    setIsTestingConnection(false);
    setIsConnectionFailed(false);

    setEditingDatabase({ ...database });
  }, [database]);

  if (!editingDatabase) return null;

  let isAllFieldsFilled = true;
  if (!editingDatabase.mongodb?.host) isAllFieldsFilled = false;
  if (!editingDatabase.mongodb?.port) isAllFieldsFilled = false;
  if (!editingDatabase.mongodb?.username) isAllFieldsFilled = false;
  if (!editingDatabase.id && !editingDatabase.mongodb?.password) isAllFieldsFilled = false;
  if (!editingDatabase.mongodb?.database) isAllFieldsFilled = false;

  const isLocalhostDb =
    editingDatabase.mongodb?.host?.includes('localhost') ||
    editingDatabase.mongodb?.host?.includes('127.0.0.1');

  return (
    <div>
      <div className="mb-3 flex">
        <div className="min-w-[150px]" />
        <div
          className="cursor-pointer text-sm text-gray-600 transition-colors hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-200"
          onClick={parseFromClipboard}
        >
          <CopyOutlined className="mr-1" />
          Parse from clipboard
        </div>
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Host</div>
        <Input
          value={editingDatabase.mongodb?.host}
          onChange={(e) => {
            if (!editingDatabase.mongodb) return;

            setEditingDatabase({
              ...editingDatabase,
              mongodb: {
                ...editingDatabase.mongodb,
                host: e.target.value.trim().replace('https://', '').replace('http://', ''),
              },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder="Enter MongoDB host"
        />
      </div>

      {isLocalhostDb && (
        <div className="mb-1 flex">
          <div className="min-w-[150px]" />
          <div className="max-w-[200px] text-xs text-gray-500 dark:text-gray-400">
            Please{' '}
            <a
              href="https://databasus.com/faq/localhost"
              target="_blank"
              rel="noreferrer"
              className="!text-blue-600 dark:!text-blue-400"
            >
              read this document
            </a>{' '}
            to study how to backup local database
          </div>
        </div>
      )}

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Port</div>
        <InputNumber
          type="number"
          value={editingDatabase.mongodb?.port}
          onChange={(e) => {
            if (!editingDatabase.mongodb || e === null) return;

            setEditingDatabase({
              ...editingDatabase,
              mongodb: { ...editingDatabase.mongodb, port: e },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder="27017"
        />
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Username</div>
        <Input
          value={editingDatabase.mongodb?.username}
          onChange={(e) => {
            if (!editingDatabase.mongodb) return;

            setEditingDatabase({
              ...editingDatabase,
              mongodb: { ...editingDatabase.mongodb, username: e.target.value.trim() },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder="Enter MongoDB username"
        />
      </div>

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Password</div>
        <Input.Password
          value={editingDatabase.mongodb?.password}
          onChange={(e) => {
            if (!editingDatabase.mongodb) return;

            setEditingDatabase({
              ...editingDatabase,
              mongodb: { ...editingDatabase.mongodb, password: e.target.value },
            });
            setIsConnectionTested(false);
          }}
          size="small"
          className="max-w-[200px] grow"
          placeholder="Enter MongoDB password"
          autoComplete="off"
          data-1p-ignore
          data-lpignore="true"
          data-form-type="other"
        />
      </div>

      {isShowDbName && (
        <div className="mb-1 flex w-full items-center">
          <div className="min-w-[150px]">DB name</div>
          <Input
            value={editingDatabase.mongodb?.database}
            onChange={(e) => {
              if (!editingDatabase.mongodb) return;

              setEditingDatabase({
                ...editingDatabase,
                mongodb: { ...editingDatabase.mongodb, database: e.target.value.trim() },
              });
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
            placeholder="Enter MongoDB database name"
          />
        </div>
      )}

      <div className="mb-1 flex w-full items-center">
        <div className="min-w-[150px]">Use HTTPS</div>
        <Switch
          checked={editingDatabase.mongodb?.isHttps}
          onChange={(checked) => {
            if (!editingDatabase.mongodb) return;

            setEditingDatabase({
              ...editingDatabase,
              mongodb: { ...editingDatabase.mongodb, isHttps: checked },
            });
            setIsConnectionTested(false);
          }}
          size="small"
        />
      </div>

      <div className="mb-5 flex w-full items-center">
        <div className="min-w-[150px]">CPU count</div>
        <div className="flex items-center">
          <InputNumber
            min={1}
            max={16}
            value={editingDatabase.mongodb?.cpuCount}
            onChange={(value) => {
              if (!editingDatabase.mongodb) return;

              setEditingDatabase({
                ...editingDatabase,
                mongodb: { ...editingDatabase.mongodb, cpuCount: value || 1 },
              });
              setIsConnectionTested(false);
            }}
            size="small"
            className="max-w-[200px] grow"
          />

          <Tooltip
            className="cursor-pointer"
            title="Number of CPU cores to use for backup and restore operations. Higher values may speed up operations but use more resources."
          >
            <InfoCircleOutlined className="ml-2" style={{ color: 'gray' }} />
          </Tooltip>
        </div>
      </div>

      <div className="mt-4 mb-1 flex items-center">
        <div
          className="flex cursor-pointer items-center text-sm text-blue-600 hover:text-blue-800"
          onClick={() => setShowAdvanced(!isShowAdvanced)}
        >
          <span className="mr-2">Advanced settings</span>

          {isShowAdvanced ? (
            <UpOutlined style={{ fontSize: '12px' }} />
          ) : (
            <DownOutlined style={{ fontSize: '12px' }} />
          )}
        </div>
      </div>

      {isShowAdvanced && (
        <>
          <div className="mb-1 flex w-full items-center">
            <div className="min-w-[150px]">Auth database</div>
            <Input
              value={editingDatabase.mongodb?.authDatabase}
              onChange={(e) => {
                if (!editingDatabase.mongodb) return;

                setEditingDatabase({
                  ...editingDatabase,
                  mongodb: { ...editingDatabase.mongodb, authDatabase: e.target.value.trim() },
                });
                setIsConnectionTested(false);
              }}
              size="small"
              className="max-w-[200px] grow"
              placeholder="admin"
            />
          </div>

          <div className="mt-4 mb-3 text-sm font-semibold text-gray-700 dark:text-gray-300">
            TLS Certificates (Optional)
          </div>

          <div className="mb-1 flex w-full items-start">
            <div className="min-w-[150px] pt-1">CA Certificate</div>
            <div className="max-w-[400px] grow">
              <Input.TextArea
                value={editingDatabase.mongodb?.tlsCaFile}
                onChange={(e) => {
                  if (!editingDatabase.mongodb) return;

                  setEditingDatabase({
                    ...editingDatabase,
                    mongodb: { ...editingDatabase.mongodb, tlsCaFile: e.target.value },
                  });
                  setIsConnectionTested(false);
                }}
                size="small"
                rows={3}
                placeholder="Paste PEM-encoded CA certificate"
              />
              <div className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Required for server certificate verification
              </div>
            </div>
          </div>

          <div className="mb-1 flex w-full items-start">
            <div className="min-w-[150px] pt-1">Client Certificate</div>
            <div className="max-w-[400px] grow">
              <Input.TextArea
                value={editingDatabase.mongodb?.tlsCertFile}
                onChange={(e) => {
                  if (!editingDatabase.mongodb) return;

                  setEditingDatabase({
                    ...editingDatabase,
                    mongodb: { ...editingDatabase.mongodb, tlsCertFile: e.target.value },
                  });
                  setIsConnectionTested(false);
                }}
                size="small"
                rows={3}
                placeholder="Paste PEM-encoded client certificate"
              />
              <div className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Required for mutual TLS authentication
              </div>
            </div>
          </div>

          <div className="mb-1 flex w-full items-start">
            <div className="min-w-[150px] pt-1">Client Key</div>
            <div className="max-w-[400px] grow">
              <Input.TextArea
                value={editingDatabase.mongodb?.tlsCertKeyFile}
                onChange={(e) => {
                  if (!editingDatabase.mongodb) return;

                  setEditingDatabase({
                    ...editingDatabase,
                    mongodb: { ...editingDatabase.mongodb, tlsCertKeyFile: e.target.value },
                  });
                  setIsConnectionTested(false);
                }}
                size="small"
                rows={3}
                placeholder="Paste PEM-encoded private key"
              />
              <div className="mt-1 text-xs text-gray-500 dark:text-gray-400">
                Required for mutual TLS authentication
              </div>
            </div>
          </div>
        </>
      )}

      <div className="mt-5 flex">
        {isShowCancelButton && (
          <Button className="mr-1" danger ghost onClick={() => onCancel()}>
            Cancel
          </Button>
        )}

        {isShowBackButton && (
          <Button className="mr-auto" type="primary" ghost onClick={() => onBack()}>
            Back
          </Button>
        )}

        {!isConnectionTested && (
          <Button
            type="primary"
            onClick={() => testConnection()}
            loading={isTestingConnection}
            disabled={!isAllFieldsFilled}
            className="mr-5"
          >
            Test connection
          </Button>
        )}

        {isConnectionTested && (
          <Button
            type="primary"
            onClick={() => saveDatabase()}
            loading={isSaving}
            disabled={!isAllFieldsFilled}
            className="mr-5"
          >
            {saveButtonText || 'Save'}
          </Button>
        )}
      </div>

      {isConnectionFailed && (
        <div className="mt-3 text-sm text-gray-500 dark:text-gray-400">
          If your database uses IP whitelist, make sure Databasus server IP is added to the allowed
          list.
        </div>
      )}
    </div>
  );
};
