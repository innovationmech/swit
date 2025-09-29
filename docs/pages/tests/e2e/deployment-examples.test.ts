import { test, expect } from '@playwright/test'

test.describe('Deployment Examples Pages', () => {
  test('English page is accessible and has key sections', async ({ page }) => {
    await page.goto('/en/guide/deployment-examples')
    await expect(page.getByRole('heading', { level: 1, name: /Deployment/i })).toBeVisible()
    await expect(page.getByRole('heading', { level: 2, name: /Docker Examples/ })).toBeVisible()
    await expect(page.getByRole('heading', { level: 2, name: /Kubernetes Examples/ })).toBeVisible()
  })

  test('Chinese page is accessible and has key sections', async ({ page }) => {
    await page.goto('/zh/guide/deployment-examples')
    await expect(page.getByRole('heading', { level: 1, name: /部署场景配置示例/ })).toBeVisible()
    await expect(page.getByRole('heading', { level: 2, name: /Docker 示例/ })).toBeVisible()
    await expect(page.getByRole('heading', { level: 2, name: /Kubernetes 示例/ })).toBeVisible()
  })
})


