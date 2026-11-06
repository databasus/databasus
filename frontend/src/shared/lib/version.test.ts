import { describe, expect, it } from 'vitest';

import { compareVersions, parseGitHubTag } from '@/shared/lib/version';

describe('parseGitHubTag', () => {
  it('strips a leading v from a release tag', () => {
    expect(parseGitHubTag('v3.40.0')).toBe('3.40.0');
  });

  it('leaves a tag without a v prefix unchanged', () => {
    expect(parseGitHubTag('3.40.0')).toBe('3.40.0');
  });

  it('trims surrounding whitespace', () => {
    expect(parseGitHubTag('  v3.40.0  ')).toBe('3.40.0');
  });
});

describe('compareVersions', () => {
  it('returns 0 for equal versions', () => {
    expect(compareVersions('3.26.0', '3.26.0')).toBe(0);
  });

  it('returns a positive number when the first version is newer', () => {
    expect(compareVersions('3.40.0', '3.26.0')).toBeGreaterThan(0);
  });

  it('returns a negative number when the first version is older', () => {
    expect(compareVersions('3.26.0', '3.40.0')).toBeLessThan(0);
  });

  it('compares the minor segment when majors match', () => {
    expect(compareVersions('3.39.0', '3.40.0')).toBeLessThan(0);
  });

  it('compares the patch segment when major and minor match', () => {
    expect(compareVersions('3.35.1', '3.35.0')).toBeGreaterThan(0);
  });

  it('treats missing segments as zero', () => {
    expect(compareVersions('3.40', '3.40.0')).toBe(0);
  });

  it('treats non-numeric segments as zero', () => {
    expect(compareVersions('3.40.x', '3.40.0')).toBe(0);
  });

  describe('against real databasus release tags', () => {
    const releaseTags = [
      'v3.40.0',
      'v3.39.2',
      'v3.39.1',
      'v3.39.0',
      'v3.38.0',
      'v3.35.1',
      'v3.35.0',
      'v3.32.2',
      'v3.28.1',
      'v3.28.0',
      'v3.27.0',
      'v3.26.0',
    ].map(parseGitHubTag);

    it('orders the published tags newest-first', () => {
      for (let i = 0; i < releaseTags.length - 1; i++) {
        expect(compareVersions(releaseTags[i], releaseTags[i + 1])).toBeGreaterThan(0);
      }
    });

    it('reports an update when the bundled version is behind the latest release', () => {
      const latestRelease = releaseTags[0];

      expect(compareVersions(latestRelease, '3.26.0')).toBeGreaterThan(0);
    });

    it('reports no update when the bundled version matches the latest release', () => {
      const latestRelease = releaseTags[0];

      expect(compareVersions(latestRelease, parseGitHubTag('v3.40.0'))).toBe(0);
    });
  });
});
