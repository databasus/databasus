import { execFile } from 'node:child_process';
import { createHash } from 'node:crypto';
import { mkdtemp, writeFile, readFile, access, rm } from 'node:fs/promises';
import { resolve, join } from 'node:path';
import { pathToFileURL } from 'node:url';
import { promisify } from 'node:util';

const execute = promisify(execFile);

export const verifyBrowserProbes = async () => {
  const directory = await mkdtemp(resolve('browser/.failure-probes-'));
  const config = join(directory, 'playwright.config.mjs');
  try {
    const applicationConfigUrl = pathToFileURL(resolve('playwright.config.ts')).href;
    await writeFile(
      config,
      `import configuration from ${JSON.stringify(applicationConfigUrl)}; export default { ...configuration, tsconfig: ${JSON.stringify(resolve('tsconfig.node.json'))}, projects: configuration.projects.filter(project => project.name === 'ui').map(project => ({ ...project, testDir: ${JSON.stringify(directory)} })), testDir: ${JSON.stringify(directory)}, testMatch: 'probe.spec.mjs', outputDir: ${JSON.stringify(join(directory, 'artifacts'))}, snapshotPathTemplate: ${JSON.stringify(join(directory, 'references/{arg}{ext}'))} };`,
    );

    const fixtureUrl = pathToFileURL(resolve('browser/browserTest.ts')).href;
    await writeFile(
      join(directory, 'probe.spec.mjs'),
      `
import { expect } from '@playwright/test';
import { test } from ${JSON.stringify(fixtureUrl)};
test('isolated harness failure probe', async ({ page }) => {
  const mode = process.env.PROBE_MODE;
  if (mode === 'scenario') { expect('actual').toBe('deliberate scenario failure'); return; }
  await page.setContent('<p>Reference ' + (mode === 'changed' ? 'changed' : 'stable') + '</p>');
  if (['unexpected', 'unexpected-outside-api', 'unexpected-static-fetch'].includes(mode)) {
    const path = mode === 'unexpected' ? '/api/v1/undeclared-probe' : mode === 'unexpected-static-fetch' ? '/assets/undeclared-probe.js' : '/undeclared-probe';
    await page.evaluate((url) => fetch(url).catch(() => {}), process.env.BROWSER_BASE_URL + path);
  } else if (mode === 'unexpected-xhr') {
    await page.evaluate((url) => new Promise((resolveRequest) => {
      const request = new XMLHttpRequest();
      request.open('GET', url);
      request.onloadend = () => resolveRequest(undefined);
      request.send();
    }), process.env.BROWSER_BASE_URL + '/undeclared-xhr-probe');
  } else if (mode === 'unexpected-popup') {
    const popup = page.waitForEvent('popup');
    await page.evaluate(() => { window.open('https://unexpected-popup.example/undeclared-probe'); });
    const opened = await popup;
    await opened.waitForLoadState();
  } else if (mode === 'popup-error') {
    const popup = page.waitForEvent('popup');
    await page.evaluate((url) => { window.open(url); }, process.env.BROWSER_BASE_URL + '/storages/google-oauth');
    const opened = await popup;
    await opened.waitForLoadState();
    const failure = opened.waitForEvent('pageerror');
    await opened.evaluate(() => queueMicrotask(() => { throw new Error('deliberate popup failure'); }));
    await failure;
  } else if (mode === 'page-error') {
    const failure = page.waitForEvent('pageerror');
    await page.evaluate(() => queueMicrotask(() => { throw new Error('deliberate page failure'); }));
    await failure;
  } else {
    await expect(page).toHaveScreenshot('probe.png');
  }
});`,
    );

    const probe = async ({
      mode,
      isUpdate = false,
      diagnostic,
    }: {
      mode: string;
      isUpdate?: boolean;
      diagnostic?: string;
    }) => {
      try {
        await execute(
          process.execPath,
          [resolve('node_modules/@playwright/test/cli.js'), 'test', '--config', config],
          {
            env: { ...process.env, PROBE_MODE: mode, UPDATE_REFERENCES: isUpdate ? '1' : '0' },
            timeout: 30_000,
            maxBuffer: 2_000_000,
          },
        );
        if (diagnostic) throw new Error(`Harness probe unexpectedly passed: ${mode}`);
      } catch (error) {
        const failure = error as Error & { code?: number; stdout?: string; stderr?: string };
        if (
          !diagnostic ||
          !failure.code ||
          !`${failure.stdout}${failure.stderr}`.includes(diagnostic)
        )
          throw error;
      }
    };

    await probe({ mode: 'scenario', diagnostic: 'deliberate scenario failure' });
    await probe({ mode: 'unexpected', diagnostic: 'Unexpected HTTP request: GET' });
    await probe({
      mode: 'unexpected-outside-api',
      diagnostic: `Unexpected HTTP request: GET ${process.env.BROWSER_BASE_URL}/undeclared-probe`,
    });
    await probe({
      mode: 'unexpected-static-fetch',
      diagnostic: `Unexpected HTTP request: GET ${process.env.BROWSER_BASE_URL}/assets/undeclared-probe.js`,
    });
    await probe({
      mode: 'unexpected-xhr',
      diagnostic: `Unexpected HTTP request: GET ${process.env.BROWSER_BASE_URL}/undeclared-xhr-probe`,
    });
    await probe({
      mode: 'unexpected-popup',
      diagnostic: 'Unexpected HTTP request: GET https://unexpected-popup.example/undeclared-probe',
    });
    await probe({ mode: 'popup-error', diagnostic: 'Page error: deliberate popup failure' });
    await probe({ mode: 'page-error', diagnostic: 'Page error: deliberate page failure' });

    const reference = join(directory, 'references/probe.png');
    await probe({ mode: 'missing', diagnostic: "snapshot doesn't exist" });
    try {
      await access(reference);
      throw new Error('An ordinary missing-reference run wrote the expected image');
    } catch (error) {
      if ((error as NodeJS.ErrnoException).code !== 'ENOENT') throw error;
    }

    await probe({ mode: 'stable', isUpdate: true });
    const digest = createHash('sha256')
      .update(await readFile(reference))
      .digest('hex');

    await probe({ mode: 'changed', diagnostic: 'toHaveScreenshot' });
    if (
      digest !==
      createHash('sha256')
        .update(await readFile(reference))
        .digest('hex')
    )
      throw new Error('An ordinary changed-reference run overwrote the expected image');
    await probe({ mode: 'stable' });

    console.log(
      'Browser harness probes passed: scenario, caught unexpected API/non-API HTTP and XHR, first popup requests, page and popup errors, missing/changed immutable references and following-run isolation.',
    );
  } finally {
    await rm(directory, { recursive: true, force: true });
  }
};
