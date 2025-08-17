import { describe, it, expect, vi, beforeEach } from 'vitest'

describe('工具函数测试', () => {
  describe('格式化工具', () => {
    describe('formatDate', () => {
      it('应该正确格式化日期', () => {
        const date = new Date('2025-01-15T10:30:00Z')
        expect(formatDate(date)).toBe('2025-01-15')
      })

      it('应该支持自定义格式', () => {
        const date = new Date('2025-01-15T10:30:00Z')
        expect(formatDate(date, 'YYYY/MM/DD')).toBe('2025/01/15')
      })

      it('应该处理无效日期', () => {
        expect(formatDate(new Date('invalid'))).toBe('Invalid Date')
      })

      it('应该支持本地化', () => {
        const date = new Date('2025-01-15T10:30:00Z')
        expect(formatDate(date, 'default', 'zh-CN')).toContain('2025')
      })
    })

    describe('formatNumber', () => {
      it('应该格式化数字', () => {
        expect(formatNumber(1234)).toBe('1,234')
        expect(formatNumber(1234567)).toBe('1,234,567')
      })

      it('应该处理小数', () => {
        expect(formatNumber(1234.56)).toBe('1,234.56')
        expect(formatNumber(1234.567, 2)).toBe('1,234.57')
      })

      it('应该支持不同的区域设置', () => {
        expect(formatNumber(1234.56, 2, 'de-DE')).toBe('1.234,56')
      })

      it('应该处理零和负数', () => {
        expect(formatNumber(0)).toBe('0')
        expect(formatNumber(-1234)).toBe('-1,234')
      })
    })

    describe('formatFileSize', () => {
      it('应该格式化文件大小', () => {
        expect(formatFileSize(1024)).toBe('1 KB')
        expect(formatFileSize(1024 * 1024)).toBe('1 MB')
        expect(formatFileSize(1024 * 1024 * 1024)).toBe('1 GB')
      })

      it('应该处理小文件', () => {
        expect(formatFileSize(500)).toBe('500 B')
        expect(formatFileSize(0)).toBe('0 B')
      })

      it('应该支持自定义精度', () => {
        expect(formatFileSize(1536, 2)).toBe('1.50 KB')
      })
    })
  })

  describe('验证工具', () => {
    describe('isValidEmail', () => {
      it('应该验证有效邮箱', () => {
        expect(isValidEmail('test@example.com')).toBe(true)
        expect(isValidEmail('user.name+tag@domain.co.uk')).toBe(true)
      })

      it('应该拒绝无效邮箱', () => {
        expect(isValidEmail('invalid-email')).toBe(false)
        expect(isValidEmail('test@')).toBe(false)
        expect(isValidEmail('@example.com')).toBe(false)
        expect(isValidEmail('')).toBe(false)
      })
    })

    describe('isValidUrl', () => {
      it('应该验证有效URL', () => {
        expect(isValidUrl('https://example.com')).toBe(true)
        expect(isValidUrl('http://localhost:3000')).toBe(true)
        expect(isValidUrl('https://sub.domain.co.uk/path?query=1')).toBe(true)
      })

      it('应该拒绝无效URL', () => {
        expect(isValidUrl('not-a-url')).toBe(false)
        expect(isValidUrl('javascript:alert(1)')).toBe(false)
        expect(isValidUrl('')).toBe(false)
      })

      it('应该支持协议限制', () => {
        expect(isValidUrl('ftp://example.com', ['http', 'https'])).toBe(false)
        expect(isValidUrl('https://example.com', ['http', 'https'])).toBe(true)
      })
    })

    describe('isValidJSON', () => {
      it('应该验证有效JSON', () => {
        expect(isValidJSON('{"key": "value"}')).toBe(true)
        expect(isValidJSON('[1, 2, 3]')).toBe(true)
        expect(isValidJSON('"string"')).toBe(true)
      })

      it('应该拒绝无效JSON', () => {
        expect(isValidJSON('invalid json')).toBe(false)
        expect(isValidJSON('{key: value}')).toBe(false)
        expect(isValidJSON('')).toBe(false)
      })
    })
  })

  describe('字符串工具', () => {
    describe('slugify', () => {
      it('应该创建URL友好的slug', () => {
        expect(slugify('Hello World')).toBe('hello-world')
        expect(slugify('API Documentation')).toBe('api-documentation')
      })

      it('应该处理特殊字符', () => {
        expect(slugify('Hello, World!')).toBe('hello-world')
        expect(slugify('Café & Bar')).toBe('cafe-bar')
      })

      it('应该处理中文字符', () => {
        expect(slugify('你好世界')).toBe('你好世界')
        expect(slugify('API 文档')).toBe('api-文档')
      })

      it('应该移除多余的分隔符', () => {
        expect(slugify('  Hello   World  ')).toBe('hello-world')
        expect(slugify('---test---')).toBe('test')
      })
    })

    describe('truncate', () => {
      it('应该截断长文本', () => {
        expect(truncate('This is a long text', 10)).toBe('This is a...')
        expect(truncate('Short', 10)).toBe('Short')
      })

      it('应该支持自定义省略号', () => {
        expect(truncate('This is a long text', 10, ' [...]')).toBe('This is a [...]')
      })

      it('应该处理边界情况', () => {
        expect(truncate('', 10)).toBe('')
        expect(truncate('Text', 0)).toBe('...')
      })
    })

    describe('capitalizeFirst', () => {
      it('应该首字母大写', () => {
        expect(capitalizeFirst('hello')).toBe('Hello')
        expect(capitalizeFirst('HELLO')).toBe('Hello')
      })

      it('应该处理空字符串', () => {
        expect(capitalizeFirst('')).toBe('')
      })

      it('应该保持其他字符不变', () => {
        expect(capitalizeFirst('hello world')).toBe('Hello world')
      })
    })

    describe('camelCase', () => {
      it('应该转换为驼峰命名', () => {
        expect(camelCase('hello-world')).toBe('helloWorld')
        expect(camelCase('hello_world')).toBe('helloWorld')
        expect(camelCase('hello world')).toBe('helloWorld')
      })

      it('应该处理已经是驼峰的字符串', () => {
        expect(camelCase('helloWorld')).toBe('helloWorld')
      })
    })

    describe('kebabCase', () => {
      it('应该转换为短横线命名', () => {
        expect(kebabCase('HelloWorld')).toBe('hello-world')
        expect(kebabCase('hello_world')).toBe('hello-world')
        expect(kebabCase('hello world')).toBe('hello-world')
      })
    })
  })

  describe('数组工具', () => {
    describe('unique', () => {
      it('应该移除重复元素', () => {
        expect(unique([1, 2, 2, 3, 3, 3])).toEqual([1, 2, 3])
        expect(unique(['a', 'b', 'a', 'c'])).toEqual(['a', 'b', 'c'])
      })

      it('应该处理对象数组', () => {
        const items = [
          { id: 1, name: 'A' },
          { id: 2, name: 'B' },
          { id: 1, name: 'A' }
        ]
        expect(unique(items, 'id')).toEqual([
          { id: 1, name: 'A' },
          { id: 2, name: 'B' }
        ])
      })

      it('应该处理空数组', () => {
        expect(unique([])).toEqual([])
      })
    })

    describe('groupBy', () => {
      it('应该按属性分组', () => {
        const items = [
          { category: 'A', value: 1 },
          { category: 'B', value: 2 },
          { category: 'A', value: 3 }
        ]
        
        const grouped = groupBy(items, 'category')
        expect(grouped.A).toHaveLength(2)
        expect(grouped.B).toHaveLength(1)
      })

      it('应该支持函数分组', () => {
        const items = [1, 2, 3, 4, 5, 6]
        const grouped = groupBy(items, (item) => item % 2 === 0 ? 'even' : 'odd')
        
        expect(grouped.even).toEqual([2, 4, 6])
        expect(grouped.odd).toEqual([1, 3, 5])
      })
    })

    describe('sortBy', () => {
      it('应该按属性排序', () => {
        const items = [
          { name: 'C', value: 3 },
          { name: 'A', value: 1 },
          { name: 'B', value: 2 }
        ]
        
        const sorted = sortBy(items, 'name')
        expect(sorted.map(item => item.name)).toEqual(['A', 'B', 'C'])
      })

      it('应该支持降序排序', () => {
        const items = [1, 3, 2]
        const sorted = sortBy(items, undefined, 'desc')
        expect(sorted).toEqual([3, 2, 1])
      })
    })

    describe('chunk', () => {
      it('应该将数组分块', () => {
        expect(chunk([1, 2, 3, 4, 5], 2)).toEqual([[1, 2], [3, 4], [5]])
        expect(chunk([1, 2, 3, 4], 2)).toEqual([[1, 2], [3, 4]])
      })

      it('应该处理边界情况', () => {
        expect(chunk([], 2)).toEqual([])
        expect(chunk([1, 2, 3], 0)).toEqual([])
        expect(chunk([1, 2, 3], 5)).toEqual([[1, 2, 3]])
      })
    })
  })

  describe('对象工具', () => {
    describe('deepClone', () => {
      it('应该深度克隆对象', () => {
        const obj = { a: 1, b: { c: 2 } }
        const cloned = deepClone(obj)
        
        expect(cloned).toEqual(obj)
        expect(cloned).not.toBe(obj)
        expect(cloned.b).not.toBe(obj.b)
      })

      it('应该处理数组', () => {
        const arr = [1, [2, 3], { a: 4 }]
        const cloned = deepClone(arr)
        
        expect(cloned).toEqual(arr)
        expect(cloned).not.toBe(arr)
        expect(cloned[1]).not.toBe(arr[1])
        expect(cloned[2]).not.toBe(arr[2])
      })

      it('应该处理null和undefined', () => {
        expect(deepClone(null)).toBe(null)
        expect(deepClone(undefined)).toBe(undefined)
      })
    })

    describe('mergeDeep', () => {
      it('应该深度合并对象', () => {
        const obj1 = { a: 1, b: { c: 2 } }
        const obj2 = { b: { d: 3 }, e: 4 }
        
        const merged = mergeDeep(obj1, obj2)
        expect(merged).toEqual({
          a: 1,
          b: { c: 2, d: 3 },
          e: 4
        })
      })

      it('应该处理数组合并', () => {
        const obj1 = { arr: [1, 2] }
        const obj2 = { arr: [3, 4] }
        
        const merged = mergeDeep(obj1, obj2)
        expect(merged.arr).toEqual([3, 4]) // 后者覆盖前者
      })
    })

    describe('pick', () => {
      it('应该选择指定属性', () => {
        const obj = { a: 1, b: 2, c: 3 }
        expect(pick(obj, ['a', 'c'])).toEqual({ a: 1, c: 3 })
      })

      it('应该忽略不存在的属性', () => {
        const obj = { a: 1, b: 2 }
        expect(pick(obj, ['a', 'c'])).toEqual({ a: 1 })
      })
    })

    describe('omit', () => {
      it('应该排除指定属性', () => {
        const obj = { a: 1, b: 2, c: 3 }
        expect(omit(obj, ['b'])).toEqual({ a: 1, c: 3 })
      })
    })
  })

  describe('异步工具', () => {
    describe('debounce', () => {
      it('应该防抖函数调用', async () => {
        const fn = vi.fn()
        const debouncedFn = debounce(fn, 100)
        
        debouncedFn()
        debouncedFn()
        debouncedFn()
        
        expect(fn).not.toHaveBeenCalled()
        
        await new Promise(resolve => setTimeout(resolve, 150))
        expect(fn).toHaveBeenCalledTimes(1)
      })

      it('应该传递最后的参数', async () => {
        const fn = vi.fn()
        const debouncedFn = debounce(fn, 100)
        
        debouncedFn('first')
        debouncedFn('second')
        debouncedFn('third')
        
        await new Promise(resolve => setTimeout(resolve, 150))
        expect(fn).toHaveBeenCalledWith('third')
      })
    })

    describe('throttle', () => {
      it('应该限流函数调用', async () => {
        const fn = vi.fn()
        const throttledFn = throttle(fn, 100)
        
        throttledFn()
        throttledFn()
        throttledFn()
        
        expect(fn).toHaveBeenCalledTimes(1)
        
        await new Promise(resolve => setTimeout(resolve, 150))
        throttledFn()
        expect(fn).toHaveBeenCalledTimes(2)
      })
    })

    describe('delay', () => {
      it('应该延迟执行', async () => {
        const start = Date.now()
        await delay(100)
        const end = Date.now()
        
        expect(end - start).toBeGreaterThanOrEqual(90) // 允许一些误差
      })
    })

    describe('retry', () => {
      it('应该重试失败的函数', async () => {
        let attempts = 0
        const fn = vi.fn().mockImplementation(() => {
          attempts++
          if (attempts < 3) {
            throw new Error('Failed')
          }
          return 'success'
        })
        
        const result = await retry(fn, 3, 10)
        expect(result).toBe('success')
        expect(fn).toHaveBeenCalledTimes(3)
      })

      it('应该在超过重试次数后抛出错误', async () => {
        const fn = vi.fn().mockRejectedValue(new Error('Always fails'))
        
        await expect(retry(fn, 2, 10)).rejects.toThrow('Always fails')
        expect(fn).toHaveBeenCalledTimes(2)
      })
    })
  })

  describe('DOM 工具', () => {
    beforeEach(() => {
      document.body.innerHTML = ''
    })

    describe('createElement', () => {
      it('应该创建HTML元素', () => {
        const element = createElement('div', { 
          class: 'test-class',
          'data-testid': 'test'
        }, 'Hello World')
        
        expect(element.tagName).toBe('DIV')
        expect(element.className).toBe('test-class')
        expect(element.getAttribute('data-testid')).toBe('test')
        expect(element.textContent).toBe('Hello World')
      })

      it('应该支持子元素', () => {
        const child = createElement('span', {}, 'Child')
        const parent = createElement('div', {}, child)
        
        expect(parent.children).toHaveLength(1)
        expect(parent.children[0]).toBe(child)
      })
    })

    describe('findElements', () => {
      it('应该查找元素', () => {
        document.body.innerHTML = `
          <div class="container">
            <p class="text">Paragraph 1</p>
            <p class="text">Paragraph 2</p>
          </div>
        `
        
        const elements = findElements('.text')
        expect(elements).toHaveLength(2)
      })
    })

    describe('addClass', () => {
      it('应该添加CSS类', () => {
        const element = document.createElement('div')
        addClass(element, 'new-class')
        expect(element.classList.contains('new-class')).toBe(true)
      })

      it('应该支持多个类', () => {
        const element = document.createElement('div')
        addClass(element, 'class1 class2')
        expect(element.classList.contains('class1')).toBe(true)
        expect(element.classList.contains('class2')).toBe(true)
      })
    })

    describe('removeClass', () => {
      it('应该移除CSS类', () => {
        const element = document.createElement('div')
        element.className = 'old-class new-class'
        removeClass(element, 'old-class')
        expect(element.classList.contains('old-class')).toBe(false)
        expect(element.classList.contains('new-class')).toBe(true)
      })
    })
  })

  describe('性能工具', () => {
    describe('memoize', () => {
      it('应该缓存函数结果', () => {
        const expensiveFn = vi.fn((x: number) => x * 2)
        const memoizedFn = memoize(expensiveFn)
        
        expect(memoizedFn(5)).toBe(10)
        expect(memoizedFn(5)).toBe(10)
        expect(expensiveFn).toHaveBeenCalledTimes(1)
      })

      it('应该支持不同参数', () => {
        const fn = vi.fn((x: number) => x * 2)
        const memoizedFn = memoize(fn)
        
        memoizedFn(1)
        memoizedFn(2)
        memoizedFn(1)
        
        expect(fn).toHaveBeenCalledTimes(2)
      })
    })

    describe('measurePerformance', () => {
      it('应该测量函数执行时间', async () => {
        const slowFn = async () => {
          await delay(100)
          return 'result'
        }
        
        const { result, duration } = await measurePerformance(slowFn)
        expect(result).toBe('result')
        expect(duration).toBeGreaterThanOrEqual(90)
      })
    })
  })
})

// 工具函数实现
function formatDate(date: Date, format: string = 'YYYY-MM-DD', locale: string = 'en-US'): string {
  if (isNaN(date.getTime())) return 'Invalid Date'
  
  if (format === 'default') {
    return new Intl.DateTimeFormat(locale).format(date)
  }
  
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  
  return format
    .replace('YYYY', String(year))
    .replace('MM', month)
    .replace('DD', day)
}

function formatNumber(num: number, decimals?: number, locale: string = 'en-US'): string {
  return new Intl.NumberFormat(locale, {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals
  }).format(num)
}

function formatFileSize(bytes: number, decimals: number = 1): string {
  if (bytes === 0) return '0 B'
  
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return `${(bytes / Math.pow(k, i)).toFixed(decimals)} ${sizes[i]}`
}

function isValidEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return emailRegex.test(email)
}

function isValidUrl(url: string, allowedProtocols: string[] = ['http', 'https']): boolean {
  try {
    const urlObj = new URL(url)
    return allowedProtocols.includes(urlObj.protocol.slice(0, -1))
  } catch {
    return false
  }
}

function isValidJSON(str: string): boolean {
  try {
    JSON.parse(str)
    return true
  } catch {
    return false
  }
}

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-\u4e00-\u9fa5]/g, '')
    .replace(/[\s_-]+/g, '-')
    .replace(/^-+|-+$/g, '')
}

function truncate(text: string, length: number, suffix: string = '...'): string {
  if (text.length <= length) return text
  return text.substring(0, length - suffix.length) + suffix
}

function capitalizeFirst(text: string): string {
  if (!text) return text
  return text.charAt(0).toUpperCase() + text.slice(1).toLowerCase()
}

function camelCase(text: string): string {
  return text.replace(/[-_\s]+(.)/g, (_, char) => char.toUpperCase())
}

function kebabCase(text: string): string {
  return text
    .replace(/([A-Z])/g, '-$1')
    .replace(/[-_\s]+/g, '-')
    .toLowerCase()
    .replace(/^-/, '')
}

function unique<T>(array: T[], key?: keyof T): T[] {
  if (key) {
    const seen = new Set()
    return array.filter(item => {
      const value = item[key]
      if (seen.has(value)) return false
      seen.add(value)
      return true
    })
  }
  return [...new Set(array)]
}

function groupBy<T>(array: T[], key: keyof T | ((item: T) => string)): Record<string, T[]> {
  return array.reduce((groups, item) => {
    const groupKey = typeof key === 'function' ? key(item) : String(item[key])
    if (!groups[groupKey]) groups[groupKey] = []
    groups[groupKey].push(item)
    return groups
  }, {} as Record<string, T[]>)
}

function sortBy<T>(array: T[], key?: keyof T, order: 'asc' | 'desc' = 'asc'): T[] {
  const sorted = [...array].sort((a, b) => {
    const aValue = key ? a[key] : a
    const bValue = key ? b[key] : b
    
    if (aValue < bValue) return order === 'asc' ? -1 : 1
    if (aValue > bValue) return order === 'asc' ? 1 : -1
    return 0
  })
  
  return sorted
}

function chunk<T>(array: T[], size: number): T[][] {
  if (size <= 0) return []
  const chunks: T[][] = []
  for (let i = 0; i < array.length; i += size) {
    chunks.push(array.slice(i, i + size))
  }
  return chunks
}

function deepClone<T>(obj: T): T {
  if (obj === null || typeof obj !== 'object') return obj
  if (obj instanceof Date) return new Date(obj.getTime()) as any
  if (obj instanceof Array) return obj.map(item => deepClone(item)) as any
  
  const cloned = {} as T
  for (const key in obj) {
    if (obj.hasOwnProperty(key)) {
      cloned[key] = deepClone(obj[key])
    }
  }
  return cloned
}

function mergeDeep<T extends Record<string, any>>(target: T, source: Partial<T>): T {
  const result = { ...target }
  
  for (const key in source) {
    if (source.hasOwnProperty(key)) {
      const sourceValue = source[key]
      const targetValue = result[key]
      
      if (typeof sourceValue === 'object' && sourceValue !== null && !Array.isArray(sourceValue) &&
          typeof targetValue === 'object' && targetValue !== null && !Array.isArray(targetValue)) {
        result[key] = mergeDeep(targetValue, sourceValue)
      } else {
        result[key] = sourceValue as any
      }
    }
  }
  
  return result
}

function pick<T extends Record<string, any>, K extends keyof T>(obj: T, keys: K[]): Pick<T, K> {
  const result = {} as Pick<T, K>
  keys.forEach(key => {
    if (key in obj) {
      result[key] = obj[key]
    }
  })
  return result
}

function omit<T extends Record<string, any>, K extends keyof T>(obj: T, keys: K[]): Omit<T, K> {
  const result = { ...obj }
  keys.forEach(key => {
    delete result[key]
  })
  return result
}

function debounce<T extends (...args: any[]) => any>(func: T, delay: number): T {
  let timeoutId: NodeJS.Timeout
  
  return ((...args: Parameters<T>) => {
    clearTimeout(timeoutId)
    timeoutId = setTimeout(() => func(...args), delay)
  }) as T
}

function throttle<T extends (...args: any[]) => any>(func: T, delay: number): T {
  let lastCall = 0
  
  return ((...args: Parameters<T>) => {
    const now = Date.now()
    if (now - lastCall >= delay) {
      lastCall = now
      return func(...args)
    }
  }) as T
}

function delay(ms: number): Promise<void> {
  return new Promise(resolve => setTimeout(resolve, ms))
}

async function retry<T>(fn: () => Promise<T>, maxAttempts: number, delayMs: number): Promise<T> {
  let lastError: Error
  
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      return await fn()
    } catch (error) {
      lastError = error as Error
      if (attempt < maxAttempts) {
        await delay(delayMs)
      }
    }
  }
  
  throw lastError!
}

function createElement(tag: string, attributes: Record<string, string> = {}, content?: string | HTMLElement): HTMLElement {
  const element = document.createElement(tag)
  
  Object.entries(attributes).forEach(([key, value]) => {
    element.setAttribute(key, value)
  })
  
  if (typeof content === 'string') {
    element.textContent = content
  } else if (content instanceof HTMLElement) {
    element.appendChild(content)
  }
  
  return element
}

function findElements(selector: string, parent: Element | Document = document): Element[] {
  return Array.from(parent.querySelectorAll(selector))
}

function addClass(element: Element, className: string): void {
  className.split(' ').forEach(cls => {
    if (cls) element.classList.add(cls)
  })
}

function removeClass(element: Element, className: string): void {
  className.split(' ').forEach(cls => {
    if (cls) element.classList.remove(cls)
  })
}

function memoize<T extends (...args: any[]) => any>(fn: T): T {
  const cache = new Map()
  
  return ((...args: Parameters<T>) => {
    const key = JSON.stringify(args)
    if (cache.has(key)) {
      return cache.get(key)
    }
    
    const result = fn(...args)
    cache.set(key, result)
    return result
  }) as T
}

async function measurePerformance<T>(fn: () => Promise<T>): Promise<{ result: T; duration: number }> {
  const start = performance.now()
  const result = await fn()
  const end = performance.now()
  
  return {
    result,
    duration: end - start
  }
}