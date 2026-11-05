// @vitest-environment happy-dom
import { useEffect } from 'react';
import { describe, expect, it } from 'vitest';

import { RequestRevision } from '../lib/RequestRevision';
import { renderScreen } from '../lib/test/renderScreen';
import { useRequestRevision } from './useRequestRevision';

describe('useRequestRevision', () => {
  it('returns one revision object for the whole lifetime of the component', () => {
    const seenRevisions: RequestRevision[] = [];
    const Screen = ({ label }: { label: string }) => {
      const revision = useRequestRevision();

      useEffect(() => {
        seenRevisions.push(revision);
      });

      return <output>{label}</output>;
    };

    const view = renderScreen(<Screen label="first" />);
    view.rerender(<Screen label="second" />);

    expect(seenRevisions).toHaveLength(2);
    expect(seenRevisions[0]).toBe(seenRevisions[1]);
    expect(seenRevisions[0]).toBeInstanceOf(RequestRevision);
  });

  it('keeps invalidations across renders', () => {
    const seenRevisions: RequestRevision[] = [];
    const Screen = ({ label }: { label: string }) => {
      const revision = useRequestRevision();

      useEffect(() => {
        seenRevisions.push(revision);
      });

      return <output>{label}</output>;
    };

    const view = renderScreen(<Screen label="first" />);
    const issuedRevision = seenRevisions[0].Invalidate();
    view.rerender(<Screen label="second" />);

    expect(seenRevisions[1].IsCurrent(issuedRevision)).toBe(true);
  });
});
