import { beforeAll, afterAll, beforeEach, afterEach, vi } from 'vitest'

// 全局测试设置

// 模拟浏览器 API
beforeAll(() => {
  // 模拟 localStorage
  Object.defineProperty(window, 'localStorage', {
    value: {
      getItem: vi.fn(),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    },
    writable: true,
  })

  // 模拟 sessionStorage
  Object.defineProperty(window, 'sessionStorage', {
    value: {
      getItem: vi.fn(),
      setItem: vi.fn(),
      removeItem: vi.fn(),
      clear: vi.fn(),
    },
    writable: true,
  })

  // 模拟 matchMedia
  Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: vi.fn().mockImplementation(query => ({
      matches: false,
      media: query,
      onchange: null,
      addListener: vi.fn(), // deprecated
      removeListener: vi.fn(), // deprecated
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  })

  // 模拟 IntersectionObserver
  global.IntersectionObserver = vi.fn().mockImplementation((callback) => ({
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
    root: null,
    rootMargin: '',
    thresholds: [],
  }))

  // 模拟 ResizeObserver
  global.ResizeObserver = vi.fn().mockImplementation((callback) => ({
    observe: vi.fn(),
    unobserve: vi.fn(),
    disconnect: vi.fn(),
  }))

  // 模拟 MutationObserver
  global.MutationObserver = vi.fn().mockImplementation((callback) => ({
    observe: vi.fn(),
    disconnect: vi.fn(),
    takeRecords: vi.fn(),
  }))

  // 模拟 scrollTo
  Object.defineProperty(window, 'scrollTo', {
    value: vi.fn(),
    writable: true,
  })

  // 模拟 scrollIntoView
  Element.prototype.scrollIntoView = vi.fn()

  // 模拟 fetch
  global.fetch = vi.fn()

  // 模拟 console 方法以减少测试输出噪音
  global.console = {
    ...console,
    log: vi.fn(),
    debug: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
  }

  // 模拟 window.getComputedStyle
  Object.defineProperty(window, 'getComputedStyle', {
    value: vi.fn().mockImplementation(() => ({
      getPropertyValue: vi.fn(),
      setProperty: vi.fn(),
      removeProperty: vi.fn(),
    })),
    writable: true,
  })
})

// 每个测试前的清理
beforeEach(() => {
  // 清除所有模拟的调用历史
  vi.clearAllMocks()
  
  // 重置 localStorage
  const localStorageMock = window.localStorage as any
  localStorageMock.getItem.mockClear()
  localStorageMock.setItem.mockClear()
  localStorageMock.removeItem.mockClear()
  localStorageMock.clear.mockClear()

  // 重置 sessionStorage
  const sessionStorageMock = window.sessionStorage as any
  sessionStorageMock.getItem.mockClear()
  sessionStorageMock.setItem.mockClear()
  sessionStorageMock.removeItem.mockClear()
  sessionStorageMock.clear.mockClear()

  // 重置 DOM
  document.body.innerHTML = ''
  document.head.innerHTML = ''
  
  // 重置 document.title
  document.title = 'Test Document'
  
  // 重置任何可能影响测试的全局状态
  if (typeof window !== 'undefined') {
    // 重置 location
    Object.defineProperty(window, 'location', {
      value: {
        href: 'http://localhost:3000/',
        origin: 'http://localhost:3000',
        protocol: 'http:',
        host: 'localhost:3000',
        hostname: 'localhost',
        port: '3000',
        pathname: '/',
        search: '',
        hash: '',
        assign: vi.fn(),
        replace: vi.fn(),
        reload: vi.fn(),
      },
      writable: true,
    })
  }
})

// 每个测试后的清理
afterEach(() => {
  // 清理任何活动的定时器
  vi.clearAllTimers()
  
  // 清理任何未完成的异步操作
  vi.runAllTimers()
})

// 全局清理
afterAll(() => {
  // 恢复所有模拟
  vi.restoreAllMocks()
  
  // 清理全局设置
  vi.clearAllMocks()
})

// 测试工具函数
export const createMockElement = (tagName: string = 'div', attributes: Record<string, string> = {}) => {
  const element = document.createElement(tagName)
  Object.entries(attributes).forEach(([key, value]) => {
    element.setAttribute(key, value)
  })
  return element
}

export const createMockEvent = (type: string, properties: Record<string, any> = {}) => {
  const event = new Event(type, { bubbles: true, cancelable: true })
  Object.assign(event, properties)
  return event
}

export const waitFor = (callback: () => boolean | Promise<boolean>, timeout: number = 5000): Promise<void> => {
  return new Promise((resolve, reject) => {
    const startTime = Date.now()
    
    const check = async () => {
      try {
        const result = await callback()
        if (result) {
          resolve()
          return
        }
      } catch (error) {
        // 继续等待
      }
      
      if (Date.now() - startTime >= timeout) {
        reject(new Error(`Timeout after ${timeout}ms`))
        return
      }
      
      setTimeout(check, 100)
    }
    
    check()
  })
}

export const mockConsole = () => {
  const originalConsole = { ...console }
  
  return {
    mock: () => {
      global.console = {
        ...console,
        log: vi.fn(),
        debug: vi.fn(),
        info: vi.fn(),
        warn: vi.fn(),
        error: vi.fn(),
      }
    },
    restore: () => {
      global.console = originalConsole
    },
    getLogs: () => (global.console.log as any).mock.calls,
    getErrors: () => (global.console.error as any).mock.calls,
    getWarnings: () => (global.console.warn as any).mock.calls,
  }
}