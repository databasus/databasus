// @vitest-environment happy-dom
import { type RefObject, useEffect, useLayoutEffect } from 'react';
import { describe, expect, it } from 'vitest';

import { useLatestRef } from '@/shared/hooks/useLatestRef';
import { renderScreen } from '@/shared/lib/test/renderScreen';

interface Props {
  value: string;
  onEffect: (latestValue: string) => void;
}

const LatestValueScreen = ({ value, onEffect }: Props) => {
  const latestValue = useLatestRef(value);

  useEffect(() => {
    onEffect(latestValue.current);
  });

  return null;
};

describe('useLatestRef', () => {
  it('holds the value of the first render when the first effect runs', () => {
    const seenValues: string[] = [];

    renderScreen(<LatestValueScreen value="first" onEffect={(value) => seenValues.push(value)} />);

    expect(seenValues).toEqual(['first']);
  });

  it('holds the new value when an effect of the same commit runs', () => {
    const seenValues: string[] = [];
    const onEffect = (value: string) => seenValues.push(value);

    const view = renderScreen(<LatestValueScreen value="first" onEffect={onEffect} />);
    view.rerender(<LatestValueScreen value="second" onEffect={onEffect} />);

    expect(seenValues).toEqual(['first', 'second']);
  });

  it('holds the new value when a child effect of the same commit runs', () => {
    const seenValues: string[] = [];
    const Child = ({ latestValue }: { latestValue: RefObject<string> }) => {
      useEffect(() => {
        seenValues.push(latestValue.current);
      });

      return null;
    };
    const Parent = ({ value }: { value: string }) => {
      const latestValue = useLatestRef(value);

      return <Child latestValue={latestValue} />;
    };

    const view = renderScreen(<Parent value="first" />);
    view.rerender(<Parent value="second" />);

    expect(seenValues).toEqual(['first', 'second']);
  });

  it('keeps one reference across renders with the value of the latest commit', () => {
    const seenReferences: { current: string }[] = [];
    const Screen = ({ value }: { value: string }) => {
      const latestValue = useLatestRef(value);

      useLayoutEffect(() => {
        seenReferences.push(latestValue);
      });

      return null;
    };

    const view = renderScreen(<Screen value="first" />);
    view.rerender(<Screen value="second" />);

    expect(seenReferences[0]).toBe(seenReferences[1]);
    expect(seenReferences[1].current).toBe('second');
  });
});
