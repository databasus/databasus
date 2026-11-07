/* eslint-disable i18next/no-literal-string -- MB and GB are technical unit symbols */
type NumberFormatter = (value: number, options?: Intl.NumberFormatOptions) => string;

export const formatCompactSizeMb = (sizeMb: number, formatNumber: NumberFormatter): string => {
  if (sizeMb >= 1024) {
    return `${formatNumber(sizeMb / 1024, { maximumFractionDigits: 1 })} GB`;
  }

  return `${formatNumber(sizeMb, { maximumFractionDigits: 0 })} MB`;
};
