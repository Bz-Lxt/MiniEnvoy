import { test, expect } from "@playwright/test";

const base = process.env.E2E_BASE || "http://127.0.0.1:31881";

test("console renders live metrics and topology", async ({ page }) => {
  await page.goto(base);
  await expect(page.getByText("MINI ENVOY")).toBeVisible();
  await expect(page.getByText("动态代理拓扑")).toBeVisible();
  await expect(page.getByText("连接池健康仪")).toBeVisible();
  await expect(page.locator("table tbody tr").first()).toBeVisible({ timeout: 15000 });
});

test("eject confirm dialog is custom", async ({ page }) => {
  await page.goto(base);
  const btn = page.getByRole("button", { name: "摘除上游" }).first();
  await expect(btn).toBeVisible({ timeout: 15000 });
  await btn.click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await page.getByRole("button", { name: "取消" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
});
