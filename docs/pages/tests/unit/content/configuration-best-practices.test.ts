import { describe, it, expect, beforeAll, afterAll, vi } from 'vitest'
import fs from 'fs'
import path from 'path'

describe('Configuration Best Practices docs', () => {
  const root = path.resolve(__dirname, '../../..')
  const en = path.join(root, 'en/guide/configuration-best-practices.md')
  const zh = path.join(root, 'zh/guide/configuration-best-practices.md')

  beforeAll(() => {
    vi.useFakeTimers()
  })

  afterAll(() => {
    vi.useRealTimers()
  })

  it('should exist in both EN and ZH', () => {
    expect(fs.existsSync(en)).toBe(true)
    expect(fs.existsSync(zh)).toBe(true)
  })

  it('should contain key sections', () => {
    const enContent = fs.readFileSync(en, 'utf-8')
    const zhContent = fs.readFileSync(zh, 'utf-8')

    // Check EN
    expect(enContent).toMatch(/Configuration Best Practices/i)
    expect(enContent).toMatch(/Environment Variable Mapping/i)
    expect(enContent).toMatch(/Validation and Testing/i)
    expect(enContent).toMatch(/Security Best Practices/i)

    // Check ZH
    expect(zhContent).toMatch(/配置最佳实践/)
    expect(zhContent).toMatch(/环境变量映射/)
    expect(zhContent).toMatch(/校验与测试/)
    expect(zhContent).toMatch(/安全最佳实践/)
  })
})


