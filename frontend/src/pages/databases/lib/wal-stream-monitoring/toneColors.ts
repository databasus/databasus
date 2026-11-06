import type { GlobalToken } from 'antd';

import { MonitoringTone } from '@/pages/databases/model/wal-stream-monitoring/MonitoringTone';

interface ToneColors {
  background: string;
  border: string;
  icon: string;
}

// AntD tokens follow the dark theme by themselves, so the banner and the stages need no colors of
// their own.
export const getToneColors = (token: GlobalToken, tone: MonitoringTone): ToneColors => {
  switch (tone) {
    case MonitoringTone.SUCCESS:
      return {
        background: token.colorSuccessBg,
        border: token.colorSuccessBorder,
        icon: token.colorSuccess,
      };
    case MonitoringTone.WARNING:
      return {
        background: token.colorWarningBg,
        border: token.colorWarningBorder,
        icon: token.colorWarning,
      };
    case MonitoringTone.ERROR:
      return {
        background: token.colorErrorBg,
        border: token.colorErrorBorder,
        icon: token.colorError,
      };
    default:
      return {
        background: token.colorFillTertiary,
        border: token.colorBorder,
        icon: token.colorTextSecondary,
      };
  }
};
