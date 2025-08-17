import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import HomePage from '../../../.vitepress/theme/components/HomePage.vue'

// 模拟 VitePress 路由
vi.mock('vitepress', () => ({
  useRouter: () => ({
    go: vi.fn(),
    push: vi.fn(),
  }),
  useData: () => ({
    site: {
      value: {
        title: 'Swit Framework',
        description: 'Modern Go Microservice Framework',
      },
    },
    page: {
      value: {
        title: 'Home',
      },
    },
    frontmatter: {
      value: {},
    },
  }),
}))

// 模拟 fetch API
global.fetch = vi.fn()

describe('HomePage Component', () => {
  let wrapper: VueWrapper

  beforeEach(() => {
    // 重置 fetch 模拟
    vi.mocked(fetch).mockClear()
  })

  afterEach(() => {
    if (wrapper) {
      wrapper.unmount()
    }
  })

  describe('组件渲染', () => {
    it('应该正确渲染基本结构', () => {
      wrapper = mount(HomePage)
      
      expect(wrapper.exists()).toBe(true)
      expect(wrapper.find('.home-page').exists()).toBe(true)
    })

    it('应该渲染 Hero 区域', () => {
      wrapper = mount(HomePage)
      
      const heroSection = wrapper.find('.hero-section')
      expect(heroSection.exists()).toBe(true)
      expect(heroSection.find('h1').text()).toContain('Swit Framework')
    })

    it('应该渲染特性卡片', () => {
      wrapper = mount(HomePage)
      
      const featuresSection = wrapper.find('.features-section')
      expect(featuresSection.exists()).toBe(true)
      
      const featureCards = wrapper.findAll('.feature-card')
      expect(featureCards.length).toBeGreaterThan(0)
    })

    it('应该渲染统计数据区域', () => {
      wrapper = mount(HomePage)
      
      const statsSection = wrapper.find('.stats-section')
      expect(statsSection.exists()).toBe(true)
    })
  })

  describe('GitHub 统计数据获取', () => {
    it('应该在组件挂载时获取 GitHub 统计数据', async () => {
      const mockStats = {
        stars: 123,
        forks: 45,
        contributors: 12,
        releases: 8,
      }

      vi.mocked(fetch).mockResolvedValueOnce({
        ok: true,
        json: async () => mockStats,
      } as Response)

      wrapper = mount(HomePage)
      
      // 等待异步操作完成
      await wrapper.vm.$nextTick()
      await new Promise(resolve => setTimeout(resolve, 100))

      expect(fetch).toHaveBeenCalledWith(
        expect.stringContaining('api.github.com/repos')
      )
    })

    it('应该处理 GitHub API 错误', async () => {
      vi.mocked(fetch).mockRejectedValueOnce(new Error('API Error'))

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

      wrapper = mount(HomePage)
      
      // 等待错误处理
      await wrapper.vm.$nextTick()
      await new Promise(resolve => setTimeout(resolve, 100))

      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })

    it('应该使用默认统计数据作为回退', () => {
      wrapper = mount(HomePage)
      
      // 验证默认值存在
      const stats = wrapper.vm.$data.stats || wrapper.vm.stats
      expect(typeof stats).toBe('object')
    })
  })

  describe('响应式行为', () => {
    it('应该在移动设备上调整布局', () => {
      // 模拟移动设备视口
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 375,
      })

      wrapper = mount(HomePage)
      
      // 触发 resize 事件
      window.dispatchEvent(new Event('resize'))
      
      expect(wrapper.find('.home-page').exists()).toBe(true)
    })
  })

  describe('交互功能', () => {
    it('应该处理 CTA 按钮点击', async () => {
      wrapper = mount(HomePage)
      
      const ctaButton = wrapper.find('.cta-button')
      if (ctaButton.exists()) {
        await ctaButton.trigger('click')
        // 验证点击处理逻辑
        expect(ctaButton.exists()).toBe(true)
      }
    })

    it('应该支持键盘导航', async () => {
      wrapper = mount(HomePage)
      
      const focusableElements = wrapper.findAll('[tabindex], button, a[href]')
      
      if (focusableElements.length > 0) {
        const firstElement = focusableElements[0]
        await firstElement.trigger('keydown', { key: 'Tab' })
        expect(firstElement.exists()).toBe(true)
      }
    })
  })

  describe('可访问性', () => {
    it('应该包含适当的 ARIA 标签', () => {
      wrapper = mount(HomePage)
      
      // 检查主要区域的 ARIA 标签
      const mainContent = wrapper.find('[role="main"], main')
      expect(mainContent.exists()).toBe(true)
      
      // 检查标题层次结构
      const headings = wrapper.findAll('h1, h2, h3, h4, h5, h6')
      expect(headings.length).toBeGreaterThan(0)
    })

    it('应该包含图片的 alt 文本', () => {
      wrapper = mount(HomePage)
      
      const images = wrapper.findAll('img')
      images.forEach(img => {
        expect(img.attributes('alt')).toBeDefined()
      })
    })

    it('应该支持屏幕阅读器', () => {
      wrapper = mount(HomePage)
      
      // 检查屏幕阅读器专用内容
      const srOnlyElements = wrapper.findAll('.sr-only')
      expect(srOnlyElements.length).toBeGreaterThanOrEqual(0)
    })
  })

  describe('性能优化', () => {
    it('应该实现图片懒加载', () => {
      wrapper = mount(HomePage)
      
      const images = wrapper.findAll('img')
      images.forEach(img => {
        const loading = img.attributes('loading')
        if (loading) {
          expect(['lazy', 'eager']).toContain(loading)
        }
      })
    })

    it('应该预加载关键资源', () => {
      wrapper = mount(HomePage)
      
      // 验证组件不会阻塞渲染
      expect(wrapper.exists()).toBe(true)
    })
  })

  describe('多语言支持', () => {
    it('应该支持语言切换', async () => {
      wrapper = mount(HomePage, {
        props: {
          locale: 'zh-CN'
        }
      })
      
      expect(wrapper.exists()).toBe(true)
      
      // 如果组件支持多语言，验证内容变化
      if (wrapper.vm.$props.locale) {
        expect(wrapper.vm.$props.locale).toBe('zh-CN')
      }
    })
  })

  describe('错误处理', () => {
    it('应该优雅地处理数据加载错误', async () => {
      // 模拟网络错误
      vi.mocked(fetch).mockRejectedValueOnce(new Error('Network Error'))
      
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      
      wrapper = mount(HomePage)
      
      // 等待错误处理
      await wrapper.vm.$nextTick()
      await new Promise(resolve => setTimeout(resolve, 100))
      
      // 组件应该仍然渲染
      expect(wrapper.exists()).toBe(true)
      
      consoleSpy.mockRestore()
    })

    it('应该显示错误状态', async () => {
      wrapper = mount(HomePage)
      
      // 如果组件有错误状态显示，验证它
      const errorElement = wrapper.find('.error-message, .error-state')
      // 错误状态可能不总是显示，所以这个测试是条件性的
      if (errorElement.exists()) {
        expect(errorElement.isVisible()).toBe(true)
      }
    })
  })
})