import type { DatabaseResourceComposition } from '@/features/databases';

import { BackupStorageSelection } from './BackupStorageSelection';
import { DatabaseTransferDialogComponent } from './DatabaseTransferDialogComponent';
import { EditDatabaseNotifiersComponent } from './EditDatabaseNotifiersComponent';

export const databaseResourceComposition: DatabaseResourceComposition = {
  renderStorageSelection: (options) => <BackupStorageSelection {...options} />,
  renderTransfer: (options) => <DatabaseTransferDialogComponent {...options} />,
  renderNotifiers: (options) => <EditDatabaseNotifiersComponent {...options} />,
};
