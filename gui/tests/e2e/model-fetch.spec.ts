import { test, expect } from 'playwright/test';

test('model config initialize triggers fetch endpoint', async ({ page }) => {
  await page.goto('/');

  await page.route('**/api/agent/models/fetch', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        code: 0,
        data: {
          models: [
            {
              id: 'glm-4',
              name: 'GLM-4',
              provider: 'zhipu',
              description: 'Discovered from zhipu provider',
              max_tokens: 8192,
              supports_streaming: true,
            },
          ],
        },
      }),
    });
  });

  await page.getByRole('button', { name: /setting/i }).click();
  await expect(page.getByText('Model Configuration')).toBeVisible();

  await page.getByLabel('API Key').fill('test-api-key');
  await page.getByRole('button', { name: /Initialize Models/i }).click();

  await expect(page.getByText(/Fetched 1 model\(s\) from Zhipu AI/i)).toBeVisible();
});
