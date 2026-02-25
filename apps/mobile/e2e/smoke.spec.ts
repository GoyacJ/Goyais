import { expect, test } from "@playwright/test";

const viewports = [
  { width: 390, height: 844 },
  { width: 430, height: 932 },
  { width: 768, height: 1024 }
];

for (const viewport of viewports) {
  test(`mobile shell renders (${viewport.width}x${viewport.height})`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await page.goto("/main");
    await page.waitForLoadState("domcontentloaded");
    await expect(page.locator(".screen")).toBeVisible({ timeout: 15_000 });
    await expect(page.locator(".content")).toBeVisible({ timeout: 15_000 });

    const hasHorizontalOverflow = await page.evaluate(() => {
      return document.documentElement.scrollWidth > document.documentElement.clientWidth;
    });
    expect(hasHorizontalOverflow).toBeFalsy();

    if (viewport.width <= 768) {
      const menuButton = page.getByRole("button", { name: "打开导航菜单" });
      await expect(menuButton).toBeVisible();
      await menuButton.click();
      await expect(page.locator(".screen.sidebar-open")).toHaveCount(1);

      const closeButton = page.getByRole("button", { name: "关闭导航菜单" });
      await closeButton.click();
      await expect(page.locator(".screen.sidebar-open")).toHaveCount(0);
    }
  });
}
