import { API_BASE_PATH } from '../../api';

export const getApiMockPath = (path: string) =>
  `${API_BASE_PATH}${path.replace(/\{([^}]+)\}/g, ':$1')}`;
