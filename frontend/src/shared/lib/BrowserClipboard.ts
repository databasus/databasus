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
    try {
      textarea.select();
      if (!document.execCommand('copy')) throw new Error('Clipboard copy was refused');
    } finally {
      textarea.remove();
    }
  }

  static async readFromClipboard(): Promise<string> {
    const text = await navigator.clipboard.readText();
    return text;
  }
}
