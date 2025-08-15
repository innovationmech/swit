import { test, expect, Page } from '@playwright/test'

test.describe('页面路由和导航测试', () => {
  test.beforeEach(async ({ page }) => {
    // 每个测试前都访问首页
    await page.goto('/')
  })

  test.describe('基础导航功能', () => {
    test('应该正确加载首页', async ({ page }) => {
      await expect(page).toHaveTitle(/Swit Framework/)
      await expect(page.locator('h1')).toContainText('Swit')
      
      // 验证关键元素存在
      await expect(page.locator('nav')).toBeVisible()
      await expect(page.locator('main')).toBeVisible()
      await expect(page.locator('footer')).toBeVisible()
    })

    test('应该能够通过导航菜单访问各个页面', async ({ page }) => {
      // 测试指南页面导航
      await page.click('nav a[href*="guide"]')
      await expect(page).toHaveURL(/\/guide/)
      await expect(page.locator('h1')).toBeVisible()

      // 测试示例页面导航
      await page.click('nav a[href*="examples"]')
      await expect(page).toHaveURL(/\/examples/)
      await expect(page.locator('h1')).toBeVisible()

      // 测试 API 文档导航
      await page.click('nav a[href*="api"]')
      await expect(page).toHaveURL(/\/api/)
      await expect(page.locator('h1')).toBeVisible()

      // 测试社区页面导航
      await page.click('nav a[href*="community"]')
      await expect(page).toHaveURL(/\/community/)
      await expect(page.locator('h1')).toBeVisible()
    })

    test('应该支持面包屑导航', async ({ page }) => {
      // 导航到深层页面
      await page.goto('/en/guide/getting-started')
      
      // 检查面包屑是否存在
      const breadcrumb = page.locator('.breadcrumb, [aria-label*="breadcrumb"], nav[aria-label*="Breadcrumb"]')
      if (await breadcrumb.count() > 0) {
        await expect(breadcrumb).toBeVisible()
        
        // 测试面包屑点击功能
        await breadcrumb.locator('a').first().click()
        await expect(page).toHaveURL(/\/guide/)
      }
    })

    test('应该支持侧边栏导航', async ({ page }) => {
      await page.goto('/en/guide/getting-started')
      
      const sidebar = page.locator('.sidebar, aside, [role="navigation"]').last()
      if (await sidebar.count() > 0) {
        await expect(sidebar).toBeVisible()
        
        // 测试侧边栏链接
        const sidebarLinks = sidebar.locator('a')
        const linkCount = await sidebarLinks.count()
        
        if (linkCount > 0) {
          // 点击第一个链接
          await sidebarLinks.first().click()
          
          // 验证页面跳转
          await page.waitForLoadState('networkidle')
          await expect(page.locator('h1')).toBeVisible()
        }
      }
    })
  })

  test.describe('多语言导航', () => {
    test('应该支持语言切换', async ({ page }) => {
      // 检查语言切换器
      const langSwitcher = page.locator('[data-testid="language-switcher"], .language-switcher, [aria-label*="language"]')
      
      if (await langSwitcher.count() > 0) {
        // 切换到中文
        await langSwitcher.click()
        const chineseOption = page.locator('text=中文, text=Chinese').first()
        if (await chineseOption.count() > 0) {
          await chineseOption.click()
          
          // 验证 URL 变化
          await expect(page).toHaveURL(/\/zh\//)
          
          // 验证页面内容变化
          await expect(page.locator('html')).toHaveAttribute('lang', /zh/)
        }
      }
    })

    test('应该在不同语言版本间保持页面结构', async ({ page }) => {
      // 英文版本
      await page.goto('/en/guide/getting-started')
      const englishTitle = await page.locator('h1').textContent()
      
      // 中文版本
      await page.goto('/zh/guide/getting-started')
      const chineseTitle = await page.locator('h1').textContent()
      
      // 验证标题存在但内容不同
      expect(englishTitle).toBeTruthy()
      expect(chineseTitle).toBeTruthy()
      expect(englishTitle).not.toBe(chineseTitle)
    })

    test('应该在语言切换时保持当前页面路径', async ({ page }) => {
      await page.goto('/en/guide/getting-started')
      
      const langSwitcher = page.locator('[data-testid="language-switcher"], .language-switcher')
      if (await langSwitcher.count() > 0) {
        await langSwitcher.click()
        
        const chineseOption = page.locator('text=中文').first()
        if (await chineseOption.count() > 0) {
          await chineseOption.click()
          await expect(page).toHaveURL(/\/zh\/guide\/getting-started/)
        }
      }
    })
  })

  test.describe('响应式导航', () => {
    test('应该在移动设备上显示汉堡菜单', async ({ page, isMobile }) => {
      if (isMobile) {
        const mobileMenu = page.locator('[data-testid="mobile-menu"], .mobile-menu, [aria-label*="menu"]')
        if (await mobileMenu.count() > 0) {
          await expect(mobileMenu).toBeVisible()
          
          // 点击汉堡菜单
          await mobileMenu.click()
          
          // 验证菜单展开
          const mobileNav = page.locator('[data-testid="mobile-nav"], .mobile-nav')
          if (await mobileNav.count() > 0) {
            await expect(mobileNav).toBeVisible()
          }
        }
      }
    })

    test('应该在平板设备上正确显示导航', async ({ page, browserName }) => {
      // 设置平板视口
      await page.setViewportSize({ width: 768, height: 1024 })
      await page.reload()
      
      // 验证导航可见性
      const nav = page.locator('nav')
      await expect(nav).toBeVisible()
      
      // 验证菜单项可点击
      const navLinks = nav.locator('a')
      const linkCount = await navLinks.count()
      
      if (linkCount > 0) {
        await navLinks.first().click()
        await page.waitForLoadState('networkidle')
        await expect(page.locator('h1')).toBeVisible()
      }
    })
  })

  test.describe('URL 和深度链接', () => {
    test('应该支持直接访问深层页面', async ({ page }) => {
      const deepUrls = [
        '/en/guide/getting-started',
        '/en/guide/installation',
        '/en/api/switserve',
        '/en/examples/simple-http',
        '/zh/guide/getting-started'
      ]

      for (const url of deepUrls) {
        await page.goto(url)
        await expect(page.locator('h1')).toBeVisible()
        await expect(page).toHaveURL(new RegExp(url.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')))
      }
    })

    test('应该处理不存在的页面', async ({ page }) => {
      const response = await page.goto('/non-existent-page', { waitUntil: 'networkidle' })
      
      // 应该显示 404 页面或重定向到有效页面
      if (response?.status() === 404) {
        await expect(page.locator('text=404, text=Not Found')).toBeVisible()
      } else {
        // 如果重定向，确保到达了有效页面
        await expect(page.locator('h1')).toBeVisible()
      }
    })

    test('应该支持锚点链接', async ({ page }) => {
      await page.goto('/en/guide/getting-started')
      
      // 查找页面内的锚点链接
      const anchorLinks = page.locator('a[href^="#"]')
      const anchorCount = await anchorLinks.count()
      
      if (anchorCount > 0) {
        const firstAnchor = anchorLinks.first()
        const href = await firstAnchor.getAttribute('href')
        
        await firstAnchor.click()
        
        // 验证 URL 包含锚点
        await expect(page).toHaveURL(new RegExp(`#${href?.substring(1)}$`))
        
        // 验证页面滚动到对应位置
        const targetElement = page.locator(href!)
        if (await targetElement.count() > 0) {
          await expect(targetElement).toBeInViewport()
        }
      }
    })

    test('应该支持查询参数', async ({ page }) => {
      await page.goto('/en/guide/getting-started?section=installation&tab=npm')
      
      // 验证页面正常加载
      await expect(page.locator('h1')).toBeVisible()
      
      // 验证 URL 保持查询参数
      await expect(page).toHaveURL(/\?section=installation&tab=npm/)
    })
  })

  test.describe('导航性能', () => {
    test('应该快速加载页面', async ({ page }) => {
      const startTime = Date.now()
      
      await page.goto('/en/guide/getting-started', { waitUntil: 'networkidle' })
      
      const loadTime = Date.now() - startTime
      
      // 页面应该在 3 秒内加载完成
      expect(loadTime).toBeLessThan(3000)
      
      // 验证关键内容已加载
      await expect(page.locator('h1')).toBeVisible()
    })

    test('应该支持预加载功能', async ({ page }) => {
      await page.goto('/')
      
      // 检查是否有预加载链接
      const preloadLinks = page.locator('link[rel="preload"], link[rel="prefetch"]')
      const preloadCount = await preloadLinks.count()
      
      if (preloadCount > 0) {
        console.log(`Found ${preloadCount} preload/prefetch links`)
        
        // 验证预加载资源类型
        for (let i = 0; i < Math.min(preloadCount, 5); i++) {
          const link = preloadLinks.nth(i)
          const rel = await link.getAttribute('rel')
          const as = await link.getAttribute('as')
          const href = await link.getAttribute('href')
          
          expect(rel).toMatch(/preload|prefetch/)
          expect(href).toBeTruthy()
        }
      }
    })
  })

  test.describe('导航状态管理', () => {
    test('应该正确高亮当前页面的导航项', async ({ page }) => {
      await page.goto('/en/guide/getting-started')
      
      // 查找活动导航项
      const activeNavItems = page.locator('nav a.active, nav a[aria-current], nav a.router-link-active')
      const activeCount = await activeNavItems.count()
      
      if (activeCount > 0) {
        const activeItem = activeNavItems.first()
        await expect(activeItem).toBeVisible()
        
        // 验证活动项的样式
        const classes = await activeItem.getAttribute('class')
        expect(classes).toMatch(/active|current|router-link-active/)
      }
    })

    test('应该支持浏览器前进后退', async ({ page }) => {
      // 导航序列
      await page.goto('/')
      await page.click('a[href*="guide"]')
      await page.click('a[href*="examples"]')
      
      // 后退
      await page.goBack()
      await expect(page).toHaveURL(/\/guide/)
      
      // 前进
      await page.goForward()
      await expect(page).toHaveURL(/\/examples/)
      
      // 再次后退
      await page.goBack()
      await page.goBack()
      await expect(page).toHaveURL(/\/$/)
    })
  })

  test.describe('错误处理', () => {
    test('应该优雅处理网络错误', async ({ page }) => {
      // 模拟离线状态
      await page.context().setOffline(true)
      
      try {
        await page.goto('/en/guide/getting-started', { timeout: 5000 })
      } catch (error) {
        // 验证错误处理
        expect(error.message).toContain('net::ERR_INTERNET_DISCONNECTED')
      }
      
      // 恢复在线状态
      await page.context().setOffline(false)
    })

    test('应该处理慢速网络', async ({ page }) => {
      // 模拟慢速网络
      await page.route('**/*', route => {
        setTimeout(() => route.continue(), 1000) // 延迟 1 秒
      })
      
      const startTime = Date.now()
      await page.goto('/en/guide/getting-started', { waitUntil: 'networkidle' })
      const loadTime = Date.now() - startTime
      
      // 验证页面最终加载成功
      await expect(page.locator('h1')).toBeVisible()
      
      // 验证加载时间符合预期（考虑延迟）
      expect(loadTime).toBeGreaterThan(1000)
    })
  })
})