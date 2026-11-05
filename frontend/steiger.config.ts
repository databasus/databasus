import fsd from '@feature-sliced/steiger-plugin';
import { defineConfig } from 'steiger';

export default defineConfig([
  fsd.plugin,
  {
    rules: {
      'fsd/forbidden-imports': 'error',
      'fsd/public-api': 'error',
      'fsd/no-layer-public-api': 'error',
      'fsd/no-segmentless-slices': 'error',
      'fsd/no-segments-on-sliced-layers': 'error',
      'fsd/no-processes': 'error',
    },
  },
]);
