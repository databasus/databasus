import type { ReactNode } from 'react';

import type { Database } from '@/entities/databases';
import type { StorageSelectionOptions } from '@/entities/storages';

export interface DatabaseNotifierEditorOptions {
  database: Database;
  workspaceId: string;

  isShowCancelButton?: boolean;
  onCancel: () => void;

  isShowBackButton: boolean;
  onBack: () => void;

  isShowSaveOnlyForUnsaved: boolean;
  saveButtonText?: string;
  isSaveToApi: boolean;
  onSaved: (database: Database) => void;
}

export interface DatabaseResourceComposition {
  renderStorageSelection: (options: StorageSelectionOptions) => ReactNode;
  renderNotifiers: (options: DatabaseNotifierEditorOptions) => ReactNode;
  renderTransfer: (options: {
    database: Database;
    currentStorageId?: string;
    onClose: () => void;
    onTransferred: () => void;
  }) => ReactNode;
}
