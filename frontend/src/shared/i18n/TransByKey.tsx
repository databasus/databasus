import type { ReactElement } from 'react';
import { Trans } from 'react-i18next';

import type { TranslationKey } from '@/shared/i18n/TranslationKey';

interface Props {
  i18nKey: TranslationKey;
  components?: Record<string, ReactElement>;
  values?: Record<string, string | number>;
}

// Renders a key held as data (a label table entry, a parser result) through Trans. Trans cannot
// type-check an i18nKey typed as the whole TranslationKey union: TypeScript gives up with TS2590
// because the union is too large. The key is checked here instead, where it is typed.
export function TransByKey({ i18nKey, components, values }: Props) {
  return <Trans i18nKey={i18nKey as never} components={components} values={values} />;
}
