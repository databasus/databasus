export class BrowserClipboard {
  static isClipboardApiAvailable(): boolean {
    return !!(navigator.clipboard && window.isSecureContext);
  }

  static async copyToClipboard(text: string): Promise<void> {
    if (this.isClipboardApiAvailable()) {
      await navigator.clipboard.writeText(text);
      return;
    }

    const textarea = document.createElement('textarea');
    textarea.value = text;
    textarea.style.position = 'fixed';
    textarea.style.opacity = '0';
    document.body.appendChild(textarea);
    textarea.select();
    document.execCommand('copy');
    document.body.removeChild(textarea);
  }

  static async readFromClipboard(): Promise<string> {
    const text = await navigator.clipboard.readText();
    return text;
  }
}
