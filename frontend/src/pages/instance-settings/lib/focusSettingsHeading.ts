// Moving focus to the heading also tells a screen reader where the page went, and the scroll
// respects a preference for reduced motion.
export const focusSettingsHeading = (headingId: string) => {
  const heading = document.getElementById(headingId);
  if (!heading) return;

  const isMotionReduced = window.matchMedia?.('(prefers-reduced-motion: reduce)').matches ?? false;
  heading.scrollIntoView({ behavior: isMotionReduced ? 'auto' : 'smooth', block: 'start' });
  heading.focus({ preventScroll: true });
};
