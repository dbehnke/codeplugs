import { chromium } from 'playwright';
import { resolve } from 'path';

const TALKGROUPS_ARG = '--talkgroups';
const DEFAULT_TALKGROUPS = '31261,31266,3126,313136';
const CONTACTS_EXPORT_URL = 'https://brandmeister.network/#/contactsexport';
const QUERY_TIMEOUT_MS = 120000;

function parseArgs() {
  const args = process.argv.slice(2);
  let talkgroups = DEFAULT_TALKGROUPS;
  let outputPath = 'filters/filter-brandmeister.csv';

  for (let i = 0; i < args.length; i++) {
    if (args[i] === TALKGROUPS_ARG && args[i + 1]) {
      talkgroups = args[++i];
    } else if (args[i].startsWith('--talkgroups=')) {
      talkgroups = args[i].split('=')[1];
    } else if (args[i] === '--output' && args[i + 1]) {
      outputPath = args[++i];
    } else if (args[i].startsWith('--output=')) {
      outputPath = args[i].split('=')[1];
    }
  }

  return { talkgroups, outputPath };
}

async function withRetry<T>(fn: () => Promise<T>, maxRetries = 1, delayMs = 5000): Promise<T> {
  let lastError: Error | undefined;
  for (let attempt = 0; attempt <= maxRetries; attempt++) {
    try {
      return await fn();
    } catch (err) {
      lastError = err as Error;
      if (attempt < maxRetries) {
        console.log(`Attempt ${attempt + 1} failed, retrying in ${delayMs}ms...`);
        await new Promise((r) => setTimeout(r, delayMs));
      }
    }
  }
  throw lastError;
}

async function main() {
  const { talkgroups, outputPath } = parseArgs();
  console.log(`Fetching BrandMeister contacts for talkgroups: ${talkgroups}`);
  console.log(`Output: ${outputPath}`);

  await withRetry(async () => {
    const browser = await chromium.launch({ headless: true });
    try {
      const context = await browser.newContext({
        userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
        viewport: { width: 1920, height: 1080 },
        deviceScaleFactor: 2,
      });
      const page = await context.newPage();

      console.log('Navigating to BrandMeister contact export page...');
      await page.goto(CONTACTS_EXPORT_URL, { waitUntil: 'networkidle' });

      const talkgroupsInput = page.locator('#talkgroups');
      const runButton = page.locator('#addTalkgroup');

      console.log('Waiting for talkgroups input field...');
      await talkgroupsInput.waitFor({ state: 'visible', timeout: 15000 });

      await talkgroupsInput.fill(talkgroups);
      console.log(`Filled talkgroups: ${talkgroups}`);

      console.log('Running contact query...');
      await runButton.click();

      // The results table and CSV button exist before the query starts. Wait for
      // a real data cell to replace DataTables' `.dt-empty` placeholder before
      // exporting, otherwise CSV can download an empty file.
      await page.waitForSelector('#userTable tbody td:not(.dt-empty)', {
        state: 'visible',
        timeout: QUERY_TIMEOUT_MS,
      });
      console.log('Results loaded.');

      const csvButton = page.getByRole('button', { name: 'CSV', exact: true });
      await csvButton.waitFor({ state: 'visible', timeout: 5000 });

      console.log('Clicking CSV button...');
      const downloadPromise = page.waitForEvent('download', { timeout: 30000 });
      await csvButton.click();

      console.log('Waiting for download to complete...');
      const download = await downloadPromise;
      const tempPath = await download.path();
      if (!tempPath) {
        throw new Error('Download path is null');
      }

      console.log(`Downloaded file: ${download.suggestedFilename()}`);

      const fs = await import('fs');
      const downloadedContent = fs.readFileSync(tempPath, 'utf-8');
      const absoluteOutputPath = resolve(outputPath);
      fs.writeFileSync(absoluteOutputPath, downloadedContent, 'utf-8');

      const lines = downloadedContent.split('\n').filter((l) => l.trim()).length;
      console.log(`Saved ${lines} lines to ${absoluteOutputPath}`);
      console.log('Done.');
    } finally {
      await browser.close();
    }
  });
}

main().catch((err) => {
  console.error('Error:', err.message);
  process.exit(1);
});
