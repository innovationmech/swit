import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, VueWrapper } from '@vue/test-utils'

// 模拟 VitePress 的多语言功能
const mockRouter = {
  route: {
    path: '/en/guide/getting-started',
    data: {
      title: 'Getting Started',
      description: 'Quick start guide'
    }
  },
  go: vi.fn(),
  push: vi.fn()
}

const mockData = {
  site: {
    value: {
      locales: {
        root: {
          label: 'English',
          lang: 'en-US'
        },
        zh: {
          label: '中文',
          lang: 'zh-CN'
        }
      }
    }
  },
  page: {
    value: {
      title: 'Getting Started'
    }
  },
  localePath: {
    value: '/en/'
  }
}

vi.mock('vitepress', () => ({
  useRouter: () => mockRouter,
  useData: () => mockData,
  useRoute: () => mockRouter.route
}))

describe('多语言切换功能', () => {
  let wrapper: VueWrapper

  beforeEach(() => {
    vi.clearAllMocks()
    
    // 重置 location mock
    Object.defineProperty(window, 'location', {
      value: {
        href: 'http://localhost:3000/en/guide/getting-started',
        pathname: '/en/guide/getting-started',
        origin: 'http://localhost:3000',
        replace: vi.fn(),
        assign: vi.fn()
      },
      writable: true
    })
  })

  afterEach(() => {
    if (wrapper) {
      wrapper.unmount()
    }
  })

  describe('语言检测', () => {
    it('应该正确检测当前语言', () => {
      const currentLocale = getCurrentLocale()
      expect(currentLocale).toBe('en')
    })

    it('应该正确检测中文页面的语言', () => {
      window.location.pathname = '/zh/guide/getting-started'
      const currentLocale = getCurrentLocale()
      expect(currentLocale).toBe('zh')
    })

    it('应该为根路径返回默认语言', () => {
      window.location.pathname = '/guide/getting-started'
      const currentLocale = getCurrentLocale()
      expect(currentLocale).toBe('en')
    })
  })

  describe('路径转换', () => {
    it('应该正确将英文路径转换为中文路径', () => {
      const englishPath = '/en/guide/getting-started'
      const chinesePath = convertPathToLocale(englishPath, 'zh')
      expect(chinesePath).toBe('/zh/guide/getting-started')
    })

    it('应该正确将中文路径转换为英文路径', () => {
      const chinesePath = '/zh/guide/getting-started'
      const englishPath = convertPathToLocale(chinesePath, 'en')
      expect(englishPath).toBe('/en/guide/getting-started')
    })

    it('应该正确处理根路径', () => {
      const rootPath = '/guide/getting-started'
      const chinesePath = convertPathToLocale(rootPath, 'zh')
      expect(chinesePath).toBe('/zh/guide/getting-started')
    })

    it('应该正确处理首页路径', () => {
      const homePath = '/'
      const chinesePath = convertPathToLocale(homePath, 'zh')
      expect(chinesePath).toBe('/zh/')
    })
  })

  describe('语言切换组件', () => {
    const LanguageSwitcher = {
      template: `
        <div class="language-switcher">
          <button 
            v-for="locale in availableLocales" 
            :key="locale.code"
            :class="{ active: locale.code === currentLocale }"
            @click="switchToLocale(locale.code)"
            :aria-label="'Switch to ' + locale.name"
          >
            {{ locale.name }}
          </button>
        </div>
      `,
      data() {
        return {
          currentLocale: 'en',
          availableLocales: [
            { code: 'en', name: 'English' },
            { code: 'zh', name: '中文' }
          ]
        }
      },
      methods: {
        switchToLocale(locale) {
          this.currentLocale = locale
          const newPath = convertPathToLocale(window.location.pathname, locale)
          window.location.assign(newPath)
        }
      }
    }

    it('应该渲染语言切换按钮', () => {
      wrapper = mount(LanguageSwitcher)
      
      const buttons = wrapper.findAll('button')
      expect(buttons).toHaveLength(2)
      expect(buttons[0].text()).toBe('English')
      expect(buttons[1].text()).toBe('中文')
    })

    it('应该标记当前语言为活动状态', () => {
      wrapper = mount(LanguageSwitcher)
      
      const englishButton = wrapper.find('button:first-child')
      expect(englishButton.classes()).toContain('active')
    })

    it('应该在点击时切换语言', async () => {
      wrapper = mount(LanguageSwitcher)
      
      const chineseButton = wrapper.find('button:last-child')
      await chineseButton.trigger('click')
      
      expect(window.location.assign).toHaveBeenCalledWith('/zh/guide/getting-started')
    })

    it('应该包含适当的 ARIA 标签', () => {
      wrapper = mount(LanguageSwitcher)
      
      const buttons = wrapper.findAll('button')
      buttons.forEach(button => {
        expect(button.attributes('aria-label')).toBeDefined()
      })
    })
  })

  describe('内容本地化', () => {
    const LocalizedContent = {
      template: `
        <div class="localized-content">
          <h1>{{ t('title') }}</h1>
          <p>{{ t('description') }}</p>
          <button>{{ t('getStarted') }}</button>
        </div>
      `,
      data() {
        return {
          locale: 'en',
          translations: {
            en: {
              title: 'Swit Framework',
              description: 'Modern Go Microservice Framework',
              getStarted: 'Get Started'
            },
            zh: {
              title: 'Swit 框架',
              description: '现代 Go 微服务框架',
              getStarted: '开始使用'
            }
          }
        }
      },
      methods: {
        t(key) {
          return this.translations[this.locale]?.[key] || key
        }
      }
    }

    it('应该显示英文内容', () => {
      wrapper = mount(LocalizedContent, {
        data() {
          return { locale: 'en' }
        }
      })
      
      expect(wrapper.find('h1').text()).toBe('Swit Framework')
      expect(wrapper.find('p').text()).toBe('Modern Go Microservice Framework')
      expect(wrapper.find('button').text()).toBe('Get Started')
    })

    it('应该显示中文内容', () => {
      wrapper = mount(LocalizedContent, {
        data() {
          return { locale: 'zh' }
        }
      })
      
      expect(wrapper.find('h1').text()).toBe('Swit 框架')
      expect(wrapper.find('p').text()).toBe('现代 Go 微服务框架')
      expect(wrapper.find('button').text()).toBe('开始使用')
    })

    it('应该为缺失的翻译提供回退', () => {
      wrapper = mount(LocalizedContent, {
        data() {
          return { locale: 'fr' } // 不存在的语言
        }
      })
      
      expect(wrapper.find('h1').text()).toBe('title')
      expect(wrapper.find('p').text()).toBe('description')
      expect(wrapper.find('button').text()).toBe('getStarted')
    })
  })

  describe('URL 处理', () => {
    it('应该保持查询参数和哈希', () => {
      const pathWithQuery = '/en/guide/getting-started?section=installation#setup'
      const chinesePath = convertPathToLocale(pathWithQuery, 'zh')
      expect(chinesePath).toBe('/zh/guide/getting-started?section=installation#setup')
    })

    it('应该处理复杂的路径结构', () => {
      const complexPath = '/en/api/v1/endpoints/authentication'
      const chinesePath = convertPathToLocale(complexPath, 'zh')
      expect(chinesePath).toBe('/zh/api/v1/endpoints/authentication')
    })
  })

  describe('浏览器语言检测', () => {
    it('应该检测浏览器首选语言', () => {
      Object.defineProperty(navigator, 'language', {
        value: 'zh-CN',
        configurable: true
      })
      
      const preferredLocale = getPreferredLocale(['en', 'zh'])
      expect(preferredLocale).toBe('zh')
    })

    it('应该回退到支持的语言', () => {
      Object.defineProperty(navigator, 'language', {
        value: 'fr-FR',
        configurable: true
      })
      
      const preferredLocale = getPreferredLocale(['en', 'zh'])
      expect(preferredLocale).toBe('en') // 默认语言
    })

    it('应该处理语言代码的变体', () => {
      Object.defineProperty(navigator, 'language', {
        value: 'en-GB',
        configurable: true
      })
      
      const preferredLocale = getPreferredLocale(['en', 'zh'])
      expect(preferredLocale).toBe('en')
    })
  })

  describe('SEO 和元数据', () => {
    it('应该更新页面标题', () => {
      updatePageTitle('Getting Started', 'zh')
      expect(document.title).toContain('快速开始')
    })

    it('应该更新 meta 描述', () => {
      updateMetaDescription('Quick start guide for Swit Framework', 'zh')
      const metaDescription = document.querySelector('meta[name="description"]')
      expect(metaDescription?.getAttribute('content')).toContain('Swit 框架')
    })

    it('应该更新 lang 属性', () => {
      updatePageLanguage('zh')
      expect(document.documentElement.getAttribute('lang')).toBe('zh-CN')
    })

    it('应该更新 hreflang 标签', () => {
      updateHreflangTags('/zh/guide/getting-started')
      
      const hreflangLinks = document.querySelectorAll('link[hreflang]')
      expect(hreflangLinks.length).toBeGreaterThan(0)
    })
  })

  describe('错误处理', () => {
    it('应该处理不存在的语言代码', () => {
      const invalidPath = convertPathToLocale('/en/guide/getting-started', 'invalid')
      expect(invalidPath).toBe('/en/guide/getting-started') // 保持原路径
    })

    it('应该处理格式错误的路径', () => {
      const malformedPath = convertPathToLocale('not-a-valid-path', 'zh')
      expect(malformedPath).toBe('not-a-valid-path') // 保持原路径
    })
  })

  describe('性能', () => {
    it('应该缓存翻译数据', () => {
      const cache = new Map()
      
      // 第一次加载
      const translations1 = getTranslations('en', cache)
      expect(cache.has('en')).toBe(true)
      
      // 第二次加载应该使用缓存
      const translations2 = getTranslations('en', cache)
      expect(translations1).toBe(translations2)
    })

    it('应该高效处理大量翻译', () => {
      const startTime = performance.now()
      
      // 模拟大量翻译操作
      for (let i = 0; i < 1000; i++) {
        convertPathToLocale('/en/guide/getting-started', 'zh')
      }
      
      const endTime = performance.now()
      const totalTime = endTime - startTime
      
      expect(totalTime).toBeLessThan(100) // 应该在100ms内完成
    })
  })
})

// 辅助函数实现
function getCurrentLocale(): string {
  const pathname = window.location.pathname
  const localeMatch = pathname.match(/^\/([a-z]{2})\//);
  return localeMatch ? localeMatch[1] : 'en'
}

function convertPathToLocale(path: string, targetLocale: string): string {
  if (!path || !targetLocale) return path
  
  try {
    const url = new URL(path, 'http://localhost')
    let pathname = url.pathname
    
    // 移除现有的语言前缀
    pathname = pathname.replace(/^\/[a-z]{2}\//, '/')
    
    // 添加新的语言前缀（除非是默认语言）
    if (targetLocale !== 'en') {
      pathname = `/${targetLocale}${pathname}`
    }
    
    return pathname + url.search + url.hash
  } catch {
    return path
  }
}

function getPreferredLocale(availableLocales: string[]): string {
  const browserLang = navigator.language || 'en'
  const langCode = browserLang.split('-')[0]
  
  return availableLocales.includes(langCode) ? langCode : availableLocales[0]
}

function updatePageTitle(title: string, locale: string): void {
  const titleTranslations: Record<string, Record<string, string>> = {
    'Getting Started': {
      zh: '快速开始'
    }
  }
  
  const translatedTitle = titleTranslations[title]?.[locale] || title
  document.title = `${translatedTitle} | Swit Framework`
}

function updateMetaDescription(description: string, locale: string): void {
  const descriptionTranslations: Record<string, Record<string, string>> = {
    'Quick start guide for Swit Framework': {
      zh: 'Swit 框架快速开始指南'
    }
  }
  
  const translatedDescription = descriptionTranslations[description]?.[locale] || description
  
  let metaDescription = document.querySelector('meta[name="description"]')
  if (!metaDescription) {
    metaDescription = document.createElement('meta')
    metaDescription.setAttribute('name', 'description')
    document.head.appendChild(metaDescription)
  }
  
  metaDescription.setAttribute('content', translatedDescription)
}

function updatePageLanguage(locale: string): void {
  const langMap: Record<string, string> = {
    en: 'en-US',
    zh: 'zh-CN'
  }
  
  document.documentElement.setAttribute('lang', langMap[locale] || 'en-US')
}

function updateHreflangTags(currentPath: string): void {
  // 移除现有的 hreflang 标签
  const existingLinks = document.querySelectorAll('link[hreflang]')
  existingLinks.forEach(link => link.remove())
  
  // 添加新的 hreflang 标签
  const locales = ['en', 'zh']
  
  locales.forEach(locale => {
    const link = document.createElement('link')
    link.rel = 'alternate'
    link.hreflang = locale
    link.href = convertPathToLocale(currentPath, locale)
    document.head.appendChild(link)
  })
}

function getTranslations(locale: string, cache?: Map<string, any>): Record<string, string> {
  if (cache?.has(locale)) {
    return cache.get(locale)
  }
  
  const translations = {
    en: {
      title: 'Swit Framework',
      description: 'Modern Go Microservice Framework'
    },
    zh: {
      title: 'Swit 框架',
      description: '现代 Go 微服务框架'
    }
  }
  
  const result = translations[locale as keyof typeof translations] || translations.en
  
  if (cache) {
    cache.set(locale, result)
  }
  
  return result
}