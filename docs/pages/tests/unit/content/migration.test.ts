import { describe, it, expect, beforeAll, afterAll, vi } from 'vitest'
import fs from 'fs'
import path from 'path'

describe('Migration & Upgrade docs', () => {
  const root = path.resolve(__dirname, '../../..')
  const en = path.join(root, 'en/guide/migration.md')
  const zh = path.join(root, 'zh/guide/migration.md')

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

  it('EN doc should reference switctl config migrate and PlanBrokerSwitch', () => {
    const content = fs.readFileSync(en, 'utf-8')
    expect(content).toMatch(/switctl\s+config\s+migrate/i)
    expect(content).toMatch(/PlanBrokerSwitch/) // reference to messaging.PlanBrokerSwitch section exists earlier in file
  })

  it('ZH doc should reference switctl config migrate and PlanBrokerSwitch', () => {
    const content = fs.readFileSync(zh, 'utf-8')
    expect(content).toMatch(/switctl\s+config\s+migrate/i)
    expect(content).toMatch(/PlanBrokerSwitch/)
  })
})


