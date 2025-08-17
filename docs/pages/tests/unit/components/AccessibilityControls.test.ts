import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import AccessibilityControls from '../../../.vitepress/theme/components/AccessibilityControls.vue'

describe('AccessibilityControls Component', () => {
  let wrapper: VueWrapper

  beforeEach(() => {
    // 清理 localStorage
    localStorage.clear()
    
    // 重置 document 类
    document.documentElement.className = ''
    
    // 模拟 matchMedia
    Object.defineProperty(window, 'matchMedia', {
      writable: true,
      value: vi.fn().mockImplementation(query => ({
        matches: false,
        media: query,
        onchange: null,
        addListener: vi.fn(),
        removeListener: vi.fn(),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
        dispatchEvent: vi.fn(),
      })),
    })
  })

  afterEach(() => {
    if (wrapper) {
      wrapper.unmount()
    }
    localStorage.clear()
    document.documentElement.className = ''
  })

  describe('组件初始化', () => {
    it('应该正确渲染', () => {
      wrapper = mount(AccessibilityControls)
      
      expect(wrapper.exists()).toBe(true)
      expect(wrapper.find('.accessibility-controls').exists()).toBe(true)
    })

    it('应该渲染跳转到主内容链接', () => {
      wrapper = mount(AccessibilityControls)
      
      const skipLink = wrapper.find('.skip-to-content')
      expect(skipLink.exists()).toBe(true)
      expect(skipLink.attributes('href')).toBe('#main-content')
    })

    it('应该渲染字体大小控制按钮', () => {
      wrapper = mount(AccessibilityControls)
      
      const fontControls = wrapper.find('.font-size-controls')
      expect(fontControls.exists()).toBe(true)
      
      const fontButtons = wrapper.findAll('.font-size-btn')
      expect(fontButtons.length).toBe(4) // small, normal, large, xlarge
    })

    it('应该渲染对比度切换按钮', () => {
      wrapper = mount(AccessibilityControls)
      
      const contrastToggle = wrapper.find('.contrast-toggle')
      expect(contrastToggle.exists()).toBe(true)
    })
  })

  describe('字体大小控制', () => {
    it('应该设置默认字体大小为 normal', () => {
      wrapper = mount(AccessibilityControls)
      
      const normalButton = wrapper.find('.font-size-btn:nth-child(2)')
      expect(normalButton.classes()).toContain('active')
    })

    it('应该在点击时改变字体大小', async () => {
      wrapper = mount(AccessibilityControls)
      
      const largeButton = wrapper.find('.font-size-btn:nth-child(3)')
      await largeButton.trigger('click')
      
      expect(document.documentElement.classList.contains('font-large')).toBe(true)
      expect(localStorage.getItem('swit-font-size')).toBe('large')
    })

    it('应该更新活动状态', async () => {
      wrapper = mount(AccessibilityControls)
      
      const smallButton = wrapper.find('.font-size-btn:nth-child(1)')
      await smallButton.trigger('click')
      
      await wrapper.vm.$nextTick()
      
      expect(smallButton.classes()).toContain('active')
    })

    it('应该从 localStorage 恢复字体大小设置', () => {
      localStorage.setItem('swit-font-size', 'xlarge')
      
      wrapper = mount(AccessibilityControls)
      
      expect(document.documentElement.classList.contains('font-xlarge')).toBe(true)
    })
  })

  describe('高对比度控制', () => {
    it('应该默认关闭高对比度模式', () => {
      wrapper = mount(AccessibilityControls)
      
      const contrastToggle = wrapper.find('.contrast-toggle')
      expect(contrastToggle.attributes('aria-pressed')).toBe('false')
    })

    it('应该在点击时切换高对比度模式', async () => {
      wrapper = mount(AccessibilityControls)
      
      const contrastToggle = wrapper.find('.contrast-toggle')
      await contrastToggle.trigger('click')
      
      expect(document.documentElement.classList.contains('high-contrast')).toBe(true)
      expect(localStorage.getItem('swit-high-contrast')).toBe('true')
    })

    it('应该从 localStorage 恢复对比度设置', () => {
      localStorage.setItem('swit-high-contrast', 'true')
      
      wrapper = mount(AccessibilityControls)
      
      expect(document.documentElement.classList.contains('high-contrast')).toBe(true)
    })

    it('应该检测系统高对比度偏好', () => {
      // 模拟系统高对比度偏好
      vi.mocked(window.matchMedia).mockImplementation(query => {
        if (query === '(prefers-contrast: high)') {
          return {
            matches: true,
            media: query,
            onchange: null,
            addListener: vi.fn(),
            removeListener: vi.fn(),
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
            dispatchEvent: vi.fn(),
          }
        }
        return {
          matches: false,
          media: query,
          onchange: null,
          addListener: vi.fn(),
          removeListener: vi.fn(),
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
          dispatchEvent: vi.fn(),
        }
      })

      wrapper = mount(AccessibilityControls)
      
      expect(document.documentElement.classList.contains('high-contrast')).toBe(true)
    })
  })

  describe('动画减少控制', () => {
    it('应该检测系统动画减少偏好', () => {
      vi.mocked(window.matchMedia).mockImplementation(query => {
        if (query === '(prefers-reduced-motion: reduce)') {
          return {
            matches: true,
            media: query,
            onchange: null,
            addListener: vi.fn(),
            removeListener: vi.fn(),
            addEventListener: vi.fn(),
            removeEventListener: vi.fn(),
            dispatchEvent: vi.fn(),
          }
        }
        return {
          matches: false,
          media: query,
          onchange: null,
          addListener: vi.fn(),
          removeListener: vi.fn(),
          addEventListener: vi.fn(),
          removeEventListener: vi.fn(),
          dispatchEvent: vi.fn(),
        }
      })

      wrapper = mount(AccessibilityControls)
      
      expect(document.documentElement.classList.contains('reduce-motion')).toBe(true)
    })
  })

  describe('跳转到主内容功能', () => {
    it('应该在点击时跳转到主内容', async () => {
      // 创建主内容元素
      const mainContent = document.createElement('main')
      mainContent.id = 'main-content'
      mainContent.tabIndex = -1
      document.body.appendChild(mainContent)

      const focusSpy = vi.spyOn(mainContent, 'focus')
      const scrollSpy = vi.spyOn(mainContent, 'scrollIntoView').mockImplementation(() => {})

      wrapper = mount(AccessibilityControls)
      
      const skipLink = wrapper.find('.skip-to-content')
      await skipLink.trigger('click')
      
      expect(focusSpy).toHaveBeenCalled()
      expect(scrollSpy).toHaveBeenCalledWith({ behavior: 'smooth' })
      
      document.body.removeChild(mainContent)
    })
  })

  describe('设置面板', () => {
    it('应该在点击设置按钮时显示面板', async () => {
      wrapper = mount(AccessibilityControls)
      
      const settingsToggle = wrapper.find('.settings-toggle')
      await settingsToggle.trigger('click')
      
      await wrapper.vm.$nextTick()
      
      const panel = wrapper.find('.accessibility-panel')
      expect(panel.exists()).toBe(true)
    })

    it('应该在点击关闭按钮时隐藏面板', async () => {
      wrapper = mount(AccessibilityControls)
      
      // 打开面板
      const settingsToggle = wrapper.find('.settings-toggle')
      await settingsToggle.trigger('click')
      
      await wrapper.vm.$nextTick()
      
      // 关闭面板
      const closeButton = wrapper.find('.close-btn')
      await closeButton.trigger('click')
      
      await wrapper.vm.$nextTick()
      
      const panel = wrapper.find('.accessibility-panel')
      expect(panel.exists()).toBe(false)
    })

    it('应该在设置面板中显示当前设置', async () => {
      wrapper = mount(AccessibilityControls)
      
      // 设置一些值
      await wrapper.vm.setFontSize('large')
      await wrapper.vm.toggleContrast()
      
      // 打开面板
      const settingsToggle = wrapper.find('.settings-toggle')
      await settingsToggle.trigger('click')
      
      await wrapper.vm.$nextTick()
      
      const largeRadio = wrapper.find('input[type="radio"][value="large"]')
      const contrastCheckbox = wrapper.find('input[type="checkbox"]')
      
      expect(largeRadio.element.checked).toBe(true)
      expect(contrastCheckbox.element.checked).toBe(true)
    })
  })

  describe('键盘快捷键', () => {
    it('应该响应 Escape 键关闭设置面板', async () => {
      wrapper = mount(AccessibilityControls)
      
      // 打开面板
      await wrapper.vm.$data.showSettingsPanel || (wrapper.vm.showSettingsPanel = true)
      await wrapper.vm.$nextTick()
      
      // 按 Escape 键
      await document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
      await wrapper.vm.$nextTick()
      
      const panel = wrapper.find('.accessibility-panel')
      expect(panel.exists()).toBe(false)
    })

    it('应该响应 Alt+A 切换设置面板', async () => {
      wrapper = mount(AccessibilityControls)
      
      const event = new KeyboardEvent('keydown', { 
        key: 'a', 
        altKey: true,
        preventDefault: vi.fn()
      })
      
      await document.dispatchEvent(event)
      await wrapper.vm.$nextTick()
      
      expect(event.preventDefault).toHaveBeenCalled()
    })

    it('应该响应 Alt+数字键设置字体大小', async () => {
      wrapper = mount(AccessibilityControls)
      
      const event = new KeyboardEvent('keydown', { 
        key: '3', 
        altKey: true,
        preventDefault: vi.fn()
      })
      
      await document.dispatchEvent(event)
      await wrapper.vm.$nextTick()
      
      expect(document.documentElement.classList.contains('font-large')).toBe(true)
      expect(event.preventDefault).toHaveBeenCalled()
    })

    it('应该响应 Alt+C 切换对比度', async () => {
      wrapper = mount(AccessibilityControls)
      
      const event = new KeyboardEvent('keydown', { 
        key: 'c', 
        altKey: true,
        preventDefault: vi.fn()
      })
      
      await document.dispatchEvent(event)
      await wrapper.vm.$nextTick()
      
      expect(document.documentElement.classList.contains('high-contrast')).toBe(true)
      expect(event.preventDefault).toHaveBeenCalled()
    })
  })

  describe('屏幕阅读器通知', () => {
    it('应该在设置更改时通知屏幕阅读器', async () => {
      wrapper = mount(AccessibilityControls)
      
      // 模拟屏幕阅读器通知函数
      const announceSpy = vi.spyOn(wrapper.vm, 'announceToScreenReader')
      
      const largeButton = wrapper.find('.font-size-btn:nth-child(3)')
      await largeButton.trigger('click')
      
      expect(announceSpy).toHaveBeenCalledWith(expect.stringContaining('Font size changed'))
    })
  })

  describe('ARIA 属性', () => {
    it('应该设置正确的 ARIA 属性', () => {
      wrapper = mount(AccessibilityControls)
      
      const fontControls = wrapper.find('.font-size-controls')
      expect(fontControls.attributes('role')).toBe('region')
      expect(fontControls.attributes('aria-label')).toBeDefined()
      
      const contrastToggle = wrapper.find('.contrast-toggle')
      expect(contrastToggle.attributes('aria-pressed')).toBeDefined()
      
      const fontButtons = wrapper.findAll('.font-size-btn')
      fontButtons.forEach(button => {
        expect(button.attributes('aria-label')).toBeDefined()
        expect(button.attributes('aria-pressed')).toBeDefined()
      })
    })
  })

  describe('响应式设计', () => {
    it('应该在移动设备上调整控件位置', () => {
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 375,
      })

      wrapper = mount(AccessibilityControls)
      
      window.dispatchEvent(new Event('resize'))
      
      expect(wrapper.exists()).toBe(true)
    })
  })

  describe('性能', () => {
    it('应该在组件卸载时清理事件监听器', () => {
      const removeEventListenerSpy = vi.spyOn(document, 'removeEventListener')
      
      wrapper = mount(AccessibilityControls)
      wrapper.unmount()
      
      expect(removeEventListenerSpy).toHaveBeenCalledWith('keydown', expect.any(Function))
    })

    it('应该高效处理频繁的设置更改', async () => {
      wrapper = mount(AccessibilityControls)
      
      const startTime = performance.now()
      
      // 快速连续更改设置
      for (let i = 0; i < 10; i++) {
        await wrapper.vm.setFontSize(i % 2 === 0 ? 'large' : 'small')
        await wrapper.vm.toggleContrast()
      }
      
      const endTime = performance.now()
      const totalTime = endTime - startTime
      
      // 操作应该在合理时间内完成
      expect(totalTime).toBeLessThan(1000)
    })
  })
})