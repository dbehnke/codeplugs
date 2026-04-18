import { chromium } from 'playwright';
import { resolve } from 'path';

const TALKGROUPS_ARG = '--talkgroups';
const DEFAULT_TALKGROUPS = '31261,31266,3126,313136';

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

async function waitForButtonEnabled(page, text: string, timeout: number) {
  const button = page.locator('button', { hasText: text });
  const deadline = Date.now() + timeout;
  while (Date.now() < deadline) {
    if (await button.isEnabled()) return true;
    await page.waitForTimeout(500);
  }
  return false;
}

async function main() {
  const { talkgroups, outputPath } = parseArgs();
  console.log(`Fetching BrandMeister contacts for talkgroups: ${talkgroups}`);
  console.log(`Output: ${outputPath}`);

  await withRetry(async () => {
    const browser = await chromium.launch({ headless: true });
    const context = await browser.newContext({
      userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36',
    });
    const page = await context.newPage();

    const downloadPromise = page.waitForEvent('download', { timeout: 120000 });

    console.log('Navigating to BrandMeister contact export page...');
    await page.goto('https://brandmeister.network/?page=contactsexport', { waitUntil: 'networkidle' });

    console.log('Waiting for talkgroups input field...');
    await page.waitForSelector('input[type="text"]', { timeout: 10000 });

    await page.locator('input[type="text"]').fill(talkgroups);
    console.log(`Filled talkgroups: ${talkgroups}`);

    console.log('Clicking Run button...');
    await page.locator('button', { hasText: 'Run' }).click();

    console.log('Waiting for results to load...');
    const ready = await waitForButtonEnabled(page, 'CSV', 90000);
    if (!ready) {
      console.error('Timed out waiting for results (CSV button did not enable).');
      await browser.close();
      process.exit(1);
    }
    console.log('Results loaded.');

    console.log('Clicking CSV button...');
    await page.locator('button', { hasText: 'CSV' }).click();

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

    await browser.close();
    console.log('Done.');
  });
}

main().catch((err) => {
  console.error('Error:', err.message);
  process.exit(1);
});
