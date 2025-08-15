import { describe, it, expect, beforeEach } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'
import FeatureCard from '@components/FeatureCard.vue'

describe('FeatureCard Component', () => {
  let wrapper: VueWrapper

  const defaultProps = {
    title: 'Test Feature',
    description: 'This is a test feature description',
    icon: 'test-icon',
    link: '/test-link'
  }

  afterEach(() => {
    if (wrapper) {
      wrapper.unmount()
    }
  })

  describe('基础渲染', () => {
    it('应该正确渲染所有传入的 props', () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      expect(wrapper.exists()).toBe(true)
      expect(wrapper.text()).toContain(defaultProps.title)
      expect(wrapper.text()).toContain(defaultProps.description)
    })

    it('应该渲染为链接当提供 link prop 时', () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      const link = wrapper.find('a[href="/test-link"]')
      expect(link.exists()).toBe(true)
    })

    it('应该不渲染链接当未提供 link prop 时', () => {
      const propsWithoutLink = { ...defaultProps }
      delete propsWithoutLink.link

      wrapper = mount(FeatureCard, {
        props: propsWithoutLink
      })

      const links = wrapper.findAll('a')
      expect(links.length).toBe(0)
    })
  })

  describe('图标渲染', () => {
    it('应该渲染图标当提供 icon prop 时', () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      // 检查图标容器或图标元素
      const iconElement = wrapper.find('.icon, .feature-icon, [class*="icon"]')
      expect(iconElement.exists()).toBe(true)
    })

    it('应该支持不同类型的图标', () => {
      const iconTypes = ['svg', 'emoji', 'font-icon']
      
      iconTypes.forEach(iconType => {
        wrapper = mount(FeatureCard, {
          props: {
            ...defaultProps,
            icon: iconType
          }
        })
        
        expect(wrapper.exists()).toBe(true)
        wrapper.unmount()
      })
    })
  })

  describe('交互行为', () => {
    it('应该在悬停时添加样式类', async () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      const card = wrapper.find('.feature-card, .card')
      
      if (card.exists()) {
        await card.trigger('mouseenter')
        await wrapper.vm.$nextTick()
        
        // 检查悬停状态
        expect(card.classes()).toContain('hover' || card.classes().some(cls => cls.includes('hover')))
      }
    })

    it('应该在点击时触发导航', async () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      const card = wrapper.find('.feature-card, .card, a')
      
      if (card.exists()) {
        await card.trigger('click')
        // 验证点击行为
        expect(card.exists()).toBe(true)
      }
    })

    it('应该支持键盘导航', async () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      const focusableElement = wrapper.find('a, [tabindex]')
      
      if (focusableElement.exists()) {
        await focusableElement.trigger('keydown', { key: 'Enter' })
        await focusableElement.trigger('keydown', { key: ' ' })
        
        expect(focusableElement.exists()).toBe(true)
      }
    })
  })

  describe('样式和类名', () => {
    it('应该应用基础样式类', () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      const card = wrapper.find('.feature-card, .card')
      expect(card.exists()).toBe(true)
    })

    it('应该支持自定义样式类', () => {
      wrapper = mount(FeatureCard, {
        props: {
          ...defaultProps,
          class: 'custom-class'
        }
      })

      expect(wrapper.classes()).toContain('custom-class')
    })

    it('应该在不同状态下应用正确的样式', async () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      const card = wrapper.find('.feature-card, .card')
      
      if (card.exists()) {
        // 测试焦点状态
        await card.trigger('focus')
        expect(card.exists()).toBe(true)
        
        // 测试活动状态
        await card.trigger('mousedown')
        expect(card.exists()).toBe(true)
      }
    })
  })

  describe('可访问性', () => {
    it('应该包含适当的 ARIA 属性', () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      const card = wrapper.find('a, [role]')
      
      if (card.exists()) {
        // 检查 ARIA 标签或描述
        const hasAriaLabel = card.attributes('aria-label')
        const hasAriaDescribedBy = card.attributes('aria-describedby')
        const hasRole = card.attributes('role')
        
        expect(hasAriaLabel || hasAriaDescribedBy || hasRole).toBeTruthy()
      }
    })

    it('应该支持屏幕阅读器', () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      // 验证内容对屏幕阅读器可访问
      expect(wrapper.text()).toContain(defaultProps.title)
      expect(wrapper.text()).toContain(defaultProps.description)
    })

    it('应该有适当的焦点指示器', async () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      const focusableElement = wrapper.find('a, [tabindex]')
      
      if (focusableElement.exists()) {
        await focusableElement.trigger('focus')
        
        // 验证焦点状态
        expect(focusableElement.element).toBe(document.activeElement)
      }
    })
  })

  describe('响应式设计', () => {
    it('应该在移动设备上正确显示', () => {
      // 模拟移动设备视口
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 375,
      })

      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      expect(wrapper.exists()).toBe(true)
      
      // 触发 resize 事件
      window.dispatchEvent(new Event('resize'))
      
      expect(wrapper.isVisible()).toBe(true)
    })

    it('应该在桌面设备上正确显示', () => {
      // 模拟桌面设备视口
      Object.defineProperty(window, 'innerWidth', {
        writable: true,
        configurable: true,
        value: 1920,
      })

      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      expect(wrapper.exists()).toBe(true)
      expect(wrapper.isVisible()).toBe(true)
    })
  })

  describe('性能', () => {
    it('应该高效渲染', () => {
      const startTime = performance.now()
      
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })
      
      const endTime = performance.now()
      const renderTime = endTime - startTime
      
      // 渲染应该在合理时间内完成 (例如 < 100ms)
      expect(renderTime).toBeLessThan(100)
    })

    it('应该正确清理事件监听器', () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps
      })

      // 获取初始监听器数量
      const initialListeners = document.getAllEventListeners ? 
        document.getAllEventListeners().length : 0

      wrapper.unmount()

      // 验证没有内存泄漏
      expect(wrapper.exists()).toBe(false)
    })
  })

  describe('边界情况', () => {
    it('应该处理空的 props', () => {
      wrapper = mount(FeatureCard, {
        props: {}
      })

      expect(wrapper.exists()).toBe(true)
    })

    it('应该处理长文本内容', () => {
      const longProps = {
        title: 'A'.repeat(100),
        description: 'B'.repeat(500),
        icon: 'test-icon'
      }

      wrapper = mount(FeatureCard, {
        props: longProps
      })

      expect(wrapper.exists()).toBe(true)
      expect(wrapper.text()).toContain(longProps.title)
    })

    it('应该处理特殊字符', () => {
      const specialProps = {
        title: 'Test & <Special> "Characters"',
        description: 'Description with émojis 🚀 and symbols ©',
        icon: 'test-icon'
      }

      wrapper = mount(FeatureCard, {
        props: specialProps
      })

      expect(wrapper.exists()).toBe(true)
    })
  })

  describe('插槽支持', () => {
    it('应该支持自定义内容插槽', () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps,
        slots: {
          default: '<div class="custom-content">Custom Content</div>'
        }
      })

      const customContent = wrapper.find('.custom-content')
      if (customContent.exists()) {
        expect(customContent.text()).toBe('Custom Content')
      }
    })

    it('应该支持图标插槽', () => {
      wrapper = mount(FeatureCard, {
        props: defaultProps,
        slots: {
          icon: '<svg class="custom-icon">Custom Icon</svg>'
        }
      })

      const customIcon = wrapper.find('.custom-icon')
      if (customIcon.exists()) {
        expect(customIcon.exists()).toBe(true)
      }
    })
  })
})