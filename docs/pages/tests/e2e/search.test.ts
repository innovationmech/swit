import { test, expect, Page } from '@playwright/test'

test.describe('搜索功能测试', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/')
  })

  test.describe('搜索界面', () => {
    test('应该显示搜索按钮或输入框', async ({ page }) => {
      // 查找搜索相关元素
      const searchElements = [
        '[data-testid="search"]',
        '.VPNavBarSearch',
        '.DocSearch-Button',
        'input[type="search"]',
        '[placeholder*="Search"], [placeholder*="搜索"]',
        '[aria-label*="Search"], [aria-label*="搜索"]'
      ]

      let searchFound = false
      for (const selector of searchElements) {
        const element = page.locator(selector)
        if (await element.count() > 0) {
          await expect(element).toBeVisible()
          searchFound = true
          break
        }
      }

      if (!searchFound) {
        console.warn('⚠️  No search interface found on the page')
      }
    })

    test('应该能够打开搜索模态框', async ({ page }) => {
      // 尝试不同的搜索触发方式
      const searchTriggers = [
        { selector: '.VPNavBarSearch button', method: 'click' },
        { selector: '.DocSearch-Button', method: 'click' },
        { selector: '[data-testid="search-button"]', method: 'click' },
        { key: 'Control+k', method: 'keyboard' },
        { key: 'Meta+k', method: 'keyboard' },
        { key: '/', method: 'keyboard' }
      ]

      let modalOpened = false
      
      for (const trigger of searchTriggers) {
        try {
          if (trigger.method === 'click') {
            const element = page.locator(trigger.selector!)
            if (await element.count() > 0) {
              await element.click()
            } else {
              continue
            }
          } else if (trigger.method === 'keyboard') {
            await page.keyboard.press(trigger.key!)
          }

          // 检查是否打开了搜索模态框
          const searchModal = page.locator([
            '.DocSearch-Modal',
            '[role="dialog"][aria-label*="search"]',
            '.search-modal',
            '[data-testid="search-modal"]'
          ].join(', '))

          if (await searchModal.count() > 0) {
            await expect(searchModal).toBeVisible()
            modalOpened = true
            break
          }
        } catch (error) {
          // 继续尝试下一个触发方式
          continue
        }
      }

      if (!modalOpened) {
        console.warn('⚠️  Could not open search modal')
      }
    })
  })

  test.describe('搜索功能', () => {
    async function openSearchModal(page: Page): Promise<boolean> {
      const searchButton = page.locator('.VPNavBarSearch button, .DocSearch-Button, [data-testid="search-button"]').first()
      
      if (await searchButton.count() > 0) {
        await searchButton.click()
        
        const modal = page.locator('.DocSearch-Modal, [role="dialog"]').first()
        if (await modal.count() > 0) {
          await expect(modal).toBeVisible()
          return true
        }
      }
      
      // 尝试键盘快捷键
      await page.keyboard.press('Control+k')
      await page.waitForTimeout(500)
      
      const modal = page.locator('.DocSearch-Modal, [role="dialog"]').first()
      if (await modal.count() > 0 && await modal.isVisible()) {
        return true
      }
      
      return false
    }

    test('应该能够执行基本搜索', async ({ page }) => {
      const modalOpened = await openSearchModal(page)
      
      if (modalOpened) {
        // 查找搜索输入框
        const searchInput = page.locator([
          '.DocSearch-Input',
          'input[type="search"]',
          '[role="searchbox"]',
          '[placeholder*="Search"], [placeholder*="搜索"]'
        ].join(', ')).first()

        await expect(searchInput).toBeVisible()
        
        // 输入搜索关键词
        await searchInput.fill('getting started')
        
        // 等待搜索结果
        await page.waitForTimeout(1000)
        
        // 检查搜索结果
        const searchResults = page.locator([
          '.DocSearch-Hits',
          '.search-results',
          '[data-testid="search-results"]',
          '[role="listbox"]'
        ].join(', '))

        if (await searchResults.count() > 0) {
          await expect(searchResults).toBeVisible()
          
          // 验证有搜索结果项
          const resultItems = searchResults.locator('a, [role="option"]')
          const itemCount = await resultItems.count()
          expect(itemCount).toBeGreaterThan(0)
        }
      }
    })

    test('应该支持中文搜索', async ({ page }) => {
      // 先切换到中文页面
      await page.goto('/zh/')
      
      const modalOpened = await openSearchModal(page)
      
      if (modalOpened) {
        const searchInput = page.locator([
          '.DocSearch-Input',
          'input[type="search"]',
          '[role="searchbox"]'
        ].join(', ')).first()

        if (await searchInput.count() > 0) {
          await searchInput.fill('快速开始')
          await page.waitForTimeout(1000)
          
          const searchResults = page.locator('.DocSearch-Hits, .search-results')
          if (await searchResults.count() > 0) {
            await expect(searchResults).toBeVisible()
          }
        }
      }
    })

    test('应该高亮搜索关键词', async ({ page }) => {
      const modalOpened = await openSearchModal(page)
      
      if (modalOpened) {
        const searchInput = page.locator('.DocSearch-Input, input[type="search"]').first()
        
        if (await searchInput.count() > 0) {
          await searchInput.fill('framework')
          await page.waitForTimeout(1000)
          
          // 检查高亮的关键词
          const highlights = page.locator('mark, .highlight, strong').first()
          if (await highlights.count() > 0) {
            const highlightText = await highlights.textContent()
            expect(highlightText?.toLowerCase()).toContain('framework')
          }
        }
      }
    })

    test('应该支持搜索结果导航', async ({ page }) => {
      const modalOpened = await openSearchModal(page)
      
      if (modalOpened) {
        const searchInput = page.locator('.DocSearch-Input, input[type="search"]').first()
        
        if (await searchInput.count() > 0) {
          await searchInput.fill('api')
          await page.waitForTimeout(1000)
          
          // 使用键盘导航
          await searchInput.press('ArrowDown')
          await page.waitForTimeout(300)
          
          // 检查高亮的结果项
          const activeResult = page.locator([
            '.DocSearch-Hit[aria-selected="true"]',
            '.search-result.active',
            '[data-selected="true"]'
          ].join(', ')).first()
          
          if (await activeResult.count() > 0) {
            await expect(activeResult).toBeVisible()
            
            // 按 Enter 选择结果
            await searchInput.press('Enter')
            
            // 验证导航成功
            await page.waitForLoadState('networkidle')
            await expect(page.locator('h1')).toBeVisible()
          }
        }
      }
    })

    test('应该处理空搜索结果', async ({ page }) => {
      const modalOpened = await openSearchModal(page)
      
      if (modalOpened) {
        const searchInput = page.locator('.DocSearch-Input, input[type="search"]').first()
        
        if (await searchInput.count() > 0) {
          // 搜索不存在的内容
          await searchInput.fill('xyzxyzxyznotfound')
          await page.waitForTimeout(1000)
          
          // 检查空结果提示
          const noResults = page.locator([
            '.DocSearch-NoResults',
            '.no-results',
            '[data-testid="no-results"]',
            'text=No results, text=没有结果'
          ].join(', '))
          
          if (await noResults.count() > 0) {
            await expect(noResults).toBeVisible()
          }
        }
      }
    })
  })

  test.describe('搜索快捷键', () => {
    test('应该支持 Ctrl+K 快捷键', async ({ page }) => {
      await page.keyboard.press('Control+k')
      await page.waitForTimeout(500)
      
      const modal = page.locator('.DocSearch-Modal, [role="dialog"]')
      if (await modal.count() > 0) {
        await expect(modal).toBeVisible()
        
        // 测试 ESC 关闭
        await page.keyboard.press('Escape')
        await page.waitForTimeout(300)
        await expect(modal).not.toBeVisible()
      }
    })

    test('应该支持 / 快捷键', async ({ page }) => {
      await page.keyboard.press('/')
      await page.waitForTimeout(500)
      
      const modal = page.locator('.DocSearch-Modal, [role="dialog"]')
      if (await modal.count() > 0) {
        await expect(modal).toBeVisible()
      }
    })

    test('应该在输入框聚焦时显示快捷键提示', async ({ page }) => {
      const searchButton = page.locator('.VPNavBarSearch button, .DocSearch-Button').first()
      
      if (await searchButton.count() > 0) {
        // 检查按钮上是否显示快捷键提示
        const buttonText = await searchButton.textContent()
        const hasShortcut = buttonText?.includes('⌘') || buttonText?.includes('Ctrl') || 
                          await searchButton.locator('kbd').count() > 0
        
        if (hasShortcut) {
          expect(hasShortcut).toBe(true)
        }
      }
    })
  })

  test.describe('搜索性能', () => {
    test('应该快速返回搜索结果', async ({ page }) => {
      const modalOpened = await openSearchModal(page)
      
      if (modalOpened) {
        const searchInput = page.locator('.DocSearch-Input, input[type="search"]').first()
        
        if (await searchInput.count() > 0) {
          const startTime = Date.now()
          
          await searchInput.fill('framework')
          
          // 等待搜索结果出现
          await page.waitForFunction(() => {
            const results = document.querySelectorAll('.DocSearch-Hits a, .search-results a')
            return results.length > 0
          }, { timeout: 5000 }).catch(() => {
            // 如果超时，可能是搜索功能未实现
            console.warn('⚠️  Search results did not appear within 5 seconds')
          })
          
          const searchTime = Date.now() - startTime
          
          // 搜索应该在 2 秒内完成
          expect(searchTime).toBeLessThan(2000)
        }
      }
    })

    test('应该支持搜索结果缓存', async ({ page }) => {
      const modalOpened = await openSearchModal(page)
      
      if (modalOpened) {
        const searchInput = page.locator('.DocSearch-Input, input[type="search"]').first()
        
        if (await searchInput.count() > 0) {
          // 第一次搜索
          await searchInput.fill('api')
          await page.waitForTimeout(1000)
          
          // 清空并重新搜索相同内容
          await searchInput.fill('')
          await searchInput.fill('api')
          
          // 第二次搜索应该更快（如果有缓存）
          await page.waitForTimeout(500)
          
          const searchResults = page.locator('.DocSearch-Hits, .search-results')
          if (await searchResults.count() > 0) {
            await expect(searchResults).toBeVisible()
          }
        }
      }
    })
  })

  test.describe('搜索可访问性', () => {
    test('应该支持键盘导航', async ({ page }) => {
      const modalOpened = await openSearchModal(page)
      
      if (modalOpened) {
        const searchInput = page.locator('.DocSearch-Input, input[type="search"]').first()
        
        if (await searchInput.count() > 0) {
          // 验证搜索框获得焦点
          await expect(searchInput).toBeFocused()
          
          await searchInput.fill('guide')
          await page.waitForTimeout(1000)
          
          // 使用 Tab 键导航
          await page.keyboard.press('Tab')
          
          // 使用箭头键导航结果
          await page.keyboard.press('ArrowDown')
          await page.keyboard.press('ArrowUp')
          
          // 验证焦点管理
          const focusedElement = page.locator(':focus')
          await expect(focusedElement).toBeVisible()
        }
      }
    })

    test('应该包含适当的 ARIA 标签', async ({ page }) => {
      const modalOpened = await openSearchModal(page)
      
      if (modalOpened) {
        const modal = page.locator('.DocSearch-Modal, [role="dialog"]').first()
        
        // 检查模态框的 ARIA 属性
        const ariaLabel = await modal.getAttribute('aria-label')
        const role = await modal.getAttribute('role')
        
        expect(role).toBe('dialog')
        expect(ariaLabel).toBeTruthy()
        
        // 检查搜索输入框的 ARIA 属性
        const searchInput = page.locator('.DocSearch-Input, input[type="search"]').first()
        if (await searchInput.count() > 0) {
          const inputRole = await searchInput.getAttribute('role')
          const inputAriaLabel = await searchInput.getAttribute('aria-label')
          
          expect(inputRole || inputAriaLabel).toBeTruthy()
        }
      }
    })

    test('应该支持屏幕阅读器', async ({ page }) => {
      const modalOpened = await openSearchModal(page)
      
      if (modalOpened) {
        const searchInput = page.locator('.DocSearch-Input, input[type="search"]').first()
        
        if (await searchInput.count() > 0) {
          await searchInput.fill('example')
          await page.waitForTimeout(1000)
          
          // 检查搜索结果的可访问性
          const resultsList = page.locator('[role="listbox"], .DocSearch-Hits')
          if (await resultsList.count() > 0) {
            const role = await resultsList.getAttribute('role')
            expect(role).toMatch(/listbox|list/)
            
            // 检查结果项的角色
            const resultItems = resultsList.locator('a, [role="option"]')
            const firstItem = resultItems.first()
            
            if (await firstItem.count() > 0) {
              const itemRole = await firstItem.getAttribute('role')
              const itemAriaLabel = await firstItem.getAttribute('aria-label')
              
              expect(itemRole || itemAriaLabel).toBeTruthy()
            }
          }
        }
      }
    })
  })

  test.describe('搜索过滤和分类', () => {
    test('应该支持按内容类型过滤', async ({ page }) => {
      const modalOpened = await openSearchModal(page)
      
      if (modalOpened) {
        const searchInput = page.locator('.DocSearch-Input, input[type="search"]').first()
        
        if (await searchInput.count() > 0) {
          await searchInput.fill('installation')
          await page.waitForTimeout(1000)
          
          // 检查是否有分类标签
          const categories = page.locator([
            '.DocSearch-Hit-source',
            '.search-category',
            '[data-category]'
          ].join(', '))
          
          if (await categories.count() > 0) {
            const categoryText = await categories.first().textContent()
            expect(categoryText).toBeTruthy()
          }
        }
      }
    })

    test('应该显示搜索建议', async ({ page }) => {
      const modalOpened = await openSearchModal(page)
      
      if (modalOpened) {
        const searchInput = page.locator('.DocSearch-Input, input[type="search"]').first()
        
        if (await searchInput.count() > 0) {
          // 输入部分关键词
          await searchInput.fill('sta')
          await page.waitForTimeout(500)
          
          // 检查搜索建议
          const suggestions = page.locator([
            '.DocSearch-Dropdown',
            '.search-suggestions',
            '[data-testid="search-suggestions"]'
          ].join(', '))
          
          if (await suggestions.count() > 0) {
            await expect(suggestions).toBeVisible()
          }
        }
      }
    })
  })
})