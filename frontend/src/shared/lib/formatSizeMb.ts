const MB_IN_GB = 1024;

export const formatSizeMb = (
  sizeMb: number | undefined,
  formatNumber: (value: number) => string,
): string => {
  const size = sizeMb ?? 0;

  if (size >= MB_IN_GB) {
    return `${formatNumber(Number((size / MB_IN_GB).toFixed(2)))} GB`;
  }

  return `${formatNumber(Number(size.toFixed(2)))} MB`;
};
