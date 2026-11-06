import type { InstanceSettings } from '@/entities/users/model/InstanceSettings';
import type { InstanceSettingsUpdate } from '@/entities/users/model/InstanceSettingsUpdate';
import { type ApiComponents, getApiClient, requireData } from '@/shared/api';

type Schemas = ApiComponents['schemas'];

export const settingsApi = {
  async getSettings(): Promise<InstanceSettings> {
    return requireData(await getApiClient().GET('/instance-settings'));
  },

  async updateSettings(settings: InstanceSettingsUpdate): Promise<InstanceSettings> {
    return requireData(await getApiClient().PUT('/instance-settings', { body: settings }));
  },

  async sendTestEmail(): Promise<Schemas['TestEmail']> {
    return requireData(await getApiClient().POST('/test-emails'));
  },
};
