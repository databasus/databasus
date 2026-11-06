import { onTestFinished } from 'vitest';

const overrideProperty = (target: object, property: string, value: unknown) => {
  const original = Object.getOwnPropertyDescriptor(target, property);
  Object.defineProperty(target, property, { configurable: true, value });
  onTestFinished(() => {
    if (original) Object.defineProperty(target, property, original);
    else Reflect.deleteProperty(target, property);
  });
};

// Call this after renderScreen: its user-event setup installs a clipboard of its own over this one.
export const stubClipboard = (readText: () => Promise<string>) => {
  overrideProperty(window, 'isSecureContext', true);
  overrideProperty(navigator, 'clipboard', { readText });
};

export const stubClipboardWrite = (writeText: (text: string) => Promise<void>) => {
  overrideProperty(window, 'isSecureContext', true);
  overrideProperty(navigator, 'clipboard', { writeText });
};

export const stubClipboardUnavailable = () => {
  overrideProperty(window, 'isSecureContext', false);
};

export const stubClipboardCommand = (execCommand: Document['execCommand']) => {
  overrideProperty(document, 'execCommand', execCommand);
};
