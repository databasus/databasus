if (typeof window !== 'undefined' && typeof window.alert !== 'function') {
  window.alert = () => {};
}

if (typeof document !== 'undefined' && !document.fonts) {
  Object.defineProperty(document, 'fonts', {
    configurable: true,
    value: Object.assign(new EventTarget(), { ready: Promise.resolve() }),
  });
}
