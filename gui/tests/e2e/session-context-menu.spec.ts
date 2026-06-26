import { test, expect } from 'playwright/test';

test('session context menu stays visible after right-click', async ({ page }) => {
  await page.goto('/');

  await page.waitForSelector('text=Sessions');
  const sessionItem = page.locator('div').filter({ hasText: /Session/i }).first();
  await expect(sessionItem).toBeVisible();

  await sessionItem.click({ button: 'right' });
  await expect(page.locator('.session-context-menu')).toBeVisible();
});
