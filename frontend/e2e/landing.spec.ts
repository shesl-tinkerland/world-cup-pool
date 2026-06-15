import { expect, test } from '@playwright/test';

test('anonymous root shows the public landing page', async ({ page }) => {
	await page.goto('/');

	await expect(page).toHaveURL(/\/$/);
	await expect(page.getByRole('heading', { name: /logg inn og slå (kollegaen|kompisen)|log in and beat your/i })).toBeVisible();
	await expect(page.getByRole('button', { name: /logg inn med google|log in with google/i })).toBeVisible();
	await expect(page.getByText(/gruppetabell|group table/i)).toBeVisible();
	await expect(page.getByText(/poengsystem|points system/i)).toBeVisible();
	await expect(page.getByText(/liga-chat|league chat/i)).toBeVisible();
});

test('anonymous protected routes still redirect to login', async ({ page }) => {
	await page.goto('/tips');

	await expect(page).toHaveURL(/\/login(?:\?|$)/);
	await expect(page.getByRole('button', { name: /logg inn|log in/i })).toBeVisible();
});
