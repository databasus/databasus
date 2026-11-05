import type { DiskUsage } from '@/entities/disk';

const BYTES_IN_GB = 1024 ** 3;
const ONE_DECIMAL: Intl.NumberFormatOptions = {
  minimumFractionDigits: 1,
  maximumFractionDigits: 1,
};

// Interpolation values for the disk usage messages, with digits grouped by the selected language
export const getDiskUsageValues = (
  diskUsage: DiskUsage,
  formatNumber: (value: number, options?: Intl.NumberFormatOptions) => string,
) => ({
  used: formatNumber(diskUsage.usedSpaceBytes / BYTES_IN_GB, ONE_DECIMAL),
  total: formatNumber(diskUsage.totalSpaceBytes / BYTES_IN_GB, ONE_DECIMAL),
  percent: formatNumber((diskUsage.usedSpaceBytes / diskUsage.totalSpaceBytes) * 100, ONE_DECIMAL),
});
