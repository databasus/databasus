const MB_IN_GB = 1024;

export const formatBackupSize = (
  sizeMb: number,
  formatNumber: (value: number) => string,
): string => {
  if (sizeMb >= MB_IN_GB) {
    return `${formatNumber(Number((sizeMb / MB_IN_GB).toFixed(2)))} GB`;
  }

  return `${formatNumber(Number(sizeMb.toFixed(2)))} MB`;
};
