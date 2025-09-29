import { describe, it, expect } from 'vitest'
import fs from 'fs'
import path from 'path'

describe('Deployment templates presence and docs references', () => {
  const repoRoot = path.resolve(__dirname, '../../../..')
  const templatesDir = path.join(repoRoot, 'examples/deployment-templates')
  const enDoc = path.join(repoRoot, 'en/guide/deployment-examples.md')
  const zhDoc = path.join(repoRoot, 'zh/guide/deployment-examples.md')

  it('templates directory and key files exist', () => {
    expect(fs.existsSync(templatesDir)).toBe(true)
    ;[
      'README.md',
      'swit.docker.yaml',
      'docker-compose.override.yml',
      'docker-compose.prod.override.yml',
      'swit.dev.yaml',
      'swit.prod.yaml',
      'switauth.dev.yaml',
      'switauth.prod.yaml',
    ].forEach((f) => {
      expect(fs.existsSync(path.join(templatesDir, f))).toBe(true)
    })
  })

  it('docs mention the templates directory and prod override', () => {
    const en = fs.readFileSync(enDoc, 'utf-8')
    const zh = fs.readFileSync(zhDoc, 'utf-8')

    expect(en).toMatch(/examples\/deployment-templates\//)
    expect(zh).toMatch(/examples\/deployment-templates\//)

    expect(en).toMatch(/docker-compose\.prod\.override\.yml/)
    expect(zh).toMatch(/docker-compose\.prod\.override\.yml/)
  })
})


