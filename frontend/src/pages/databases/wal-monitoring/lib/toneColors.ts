import { MonitoringTone } from '@/pages/databases/wal-monitoring/model/MonitoringTone';
import { THEME_COLOR_VARIABLES } from '@/shared/theme';

interface ToneColors {
  background: string;
  border: string;
  icon: string;
}

export const getToneColors = (tone: MonitoringTone): ToneColors => {
  switch (tone) {
    case MonitoringTone.SUCCESS:
      return {
        background: THEME_COLOR_VARIABLES['success-soft'],
        border: THEME_COLOR_VARIABLES['success-border'],
        icon: THEME_COLOR_VARIABLES['success-fill'],
      };
    case MonitoringTone.WARNING:
      return {
        background: THEME_COLOR_VARIABLES['warning-soft'],
        border: THEME_COLOR_VARIABLES['warning-border'],
        icon: THEME_COLOR_VARIABLES['warning-fill'],
      };
    case MonitoringTone.ERROR:
      return {
        background: THEME_COLOR_VARIABLES['danger-soft'],
        border: THEME_COLOR_VARIABLES['danger-border'],
        icon: THEME_COLOR_VARIABLES['error-fill'],
      };
    default:
      return {
        background: THEME_COLOR_VARIABLES['hover'],
        border: THEME_COLOR_VARIABLES['border'],
        icon: THEME_COLOR_VARIABLES['muted'],
      };
  }
};
