import type { ApiComponents } from '../../../shared/api';

// Settings that are not saved yet. With databaseId, the stored secrets of that database fill the
// secrets the settings leave out.
export type DatabaseSettingsTarget = ApiComponents['schemas']['DatabaseSettingsTarget'];
