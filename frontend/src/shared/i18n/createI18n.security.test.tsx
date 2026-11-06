import type { i18n as I18nInstance } from 'i18next';
import { renderToStaticMarkup } from 'react-dom/server';
import { I18nextProvider, Trans } from 'react-i18next';
import { beforeAll, describe, expect, it } from 'vitest';

import { ApiError } from '@/shared/api';
import { createI18n } from '@/shared/i18n/createI18n';
import { translateApiError } from '@/shared/i18n/translateApiError';

// Names a user could give a workspace, a database or themselves. Each must reach the page as the
// exact text entered, never as markup.
const HOSTILE_NAMES = ['<img src=x onerror=alert(1)>', '<strong>bold</strong>', 'a {{name}} b'];

const escapeHtml = (text: string) =>
  text.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;');

const getElementNames = (markup: string) =>
  [...markup.matchAll(/<([a-z][a-z0-9]*)/g)].map((match) => match[1]);

let i18n: I18nInstance;

const render = (node: React.ReactNode) =>
  renderToStaticMarkup(<I18nextProvider i18n={i18n}>{node}</I18nextProvider>);

beforeAll(() => {
  i18n = createI18n('en');
  i18n.addResourceBundle('en', 'translation', {
    securityTest: {
      plain: 'Delete {{name}}?',
      withValues: 'Delete <strong>{{name}}</strong>?',
      withElement: 'Delete <workspaceName/>? This cannot be <bold>undone</bold>.',
      unsuppliedMarkup: 'Run <strong>now</strong> on <host>:<port>',
    },
  });
});

describe('translated text renders as text', () => {
  it.each(['quota_exceeded', 'constructor', undefined])(
    'renders a general error without server text with code %s',
    (code) => {
      const markup = render(
        <p>
          {translateApiError(
            new ApiError({ code, message: 'https://private.example/secret <img src=x>' }),
            i18n.t,
          )}
        </p>,
      );
      expect(markup).toBe(`<p>${i18n.t('errors.unknown')}</p>`);
      expect(markup).not.toContain('private.example');
      expect(markup).not.toContain('secret');
    },
  );

  it.each(['database_connection_failed', 'insufficient_disk_space'])(
    'renders supported diagnostics as plain text with code %s',
    (code) => {
      const markup = render(
        <p>
          {translateApiError(new ApiError({ code, message: '<img src=x> 1024 bytes' }), i18n.t)}
        </p>,
      );
      expect(markup).toContain('&lt;img src=x&gt; 1024 bytes');
      expect(getElementNames(markup)).toEqual(['p']);
    },
  );

  it.each(HOSTILE_NAMES)('shows %s literally when t() output is rendered', (name) => {
    const markup = render(<p>{i18n.t('securityTest.plain' as never, { name })}</p>);

    expect(markup).toBe(`<p>Delete ${escapeHtml(name)}?</p>`);
  });

  // Trans interpolates values before it parses tags, so a value that spells a supplied tag would
  // become that element without the escaping configured in createI18n.
  it.each(HOSTILE_NAMES)('creates no element from %s passed through Trans values', (name) => {
    const markup = render(
      <p>
        <Trans
          i18nKey={'securityTest.withValues' as never}
          values={{ name }}
          components={{ strong: <strong /> }}
        />
      </p>,
    );

    expect(getElementNames(markup)).toEqual(['p', 'strong']);
    expect(markup).not.toContain('<img');

    // Trans expands placeholder syntax inside a value a second time, which is why user-entered
    // values go through a supplied element instead (next test). Other names come out verbatim.
    if (!name.includes('{{')) {
      expect(markup).toBe(`<p>Delete <strong>${escapeHtml(name)}</strong>?</p>`);
    }
  });

  it.each(HOSTILE_NAMES)('shows %s exactly as entered through a supplied element', (name) => {
    const markup = render(
      <p>
        <Trans
          i18nKey={'securityTest.withElement' as never}
          components={{ workspaceName: <strong>{name}</strong>, bold: <b /> }}
        />
      </p>,
    );

    expect(markup).toBe(
      `<p>Delete <strong>${escapeHtml(name)}</strong>? This cannot be <b>undone</b>.</p>`,
    );
  });

  it('shows tags the component does not supply as text', () => {
    const markup = render(
      <p>
        <Trans i18nKey={'securityTest.unsuppliedMarkup' as never} />
      </p>,
    );

    expect(getElementNames(markup)).toEqual(['p']);
    expect(markup).toBe('<p>Run &lt;strong&gt;now&lt;/strong&gt; on &lt;host&gt;:&lt;port&gt;</p>');
  });
});
