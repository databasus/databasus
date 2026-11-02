import { useEffect, useEffectEvent, useRef, useState } from 'react';

const EASING_DURATION_MS = 400;

// Shows a value moving from what is on screen to its new target over 400 ms, so a refresh reads as
// a change. With reduced motion the new value appears at once.
export const useEasedTransition = <TValue>(
  target: TValue,
  interpolate: (from: TValue, to: TValue, progress: number) => TValue,
  isReducedMotion: boolean,
): TValue => {
  const [displayed, setDisplayed] = useState(target);

  const displayedRef = useRef(target);
  const targetKey = JSON.stringify(target);

  const getTransition = useEffectEvent(() => ({ to: target, interpolateTo: interpolate }));

  useEffect(() => {
    const { to, interpolateTo } = getTransition();

    if (isReducedMotion) {
      displayedRef.current = to;
      setDisplayed(to);

      return;
    }

    const from = displayedRef.current;
    const startedAt = performance.now();
    let animationFrame = 0;

    const step = (frameTime: number) => {
      const progress = Math.min(1, (frameTime - startedAt) / EASING_DURATION_MS);
      const value = interpolateTo(from, to, progress);

      displayedRef.current = value;
      setDisplayed(value);

      if (progress < 1) {
        animationFrame = requestAnimationFrame(step);
      }
    };

    animationFrame = requestAnimationFrame(step);

    return () => cancelAnimationFrame(animationFrame);
  }, [targetKey, isReducedMotion]);

  return displayed;
};
