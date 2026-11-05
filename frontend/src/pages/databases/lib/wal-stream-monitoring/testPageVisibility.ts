import type { PageVisibility } from './walStreamRefreshController';

// Page visibility the tests switch by hand, telling subscribers as the browser would.
export class FakePageVisibility implements PageVisibility {
  private isPageVisible = true;
  private listeners: (() => void)[] = [];

  isVisible = () => this.isPageVisible;

  subscribe = (onVisibilityChange: () => void) => {
    this.listeners.push(onVisibilityChange);

    return () => {
      this.listeners = this.listeners.filter((listener) => listener !== onVisibilityChange);
    };
  };

  setVisible(isPageVisible: boolean) {
    this.isPageVisible = isPageVisible;
    this.listeners.forEach((listener) => listener());
  }
}
