import { defineConfig } from 'vitepress'
import { sharedConfig } from './shared.mjs'
import { enConfig } from './en.mjs'
import { zhConfig } from './zh.mjs'

export default defineConfig({
  ...sharedConfig,
  
  // Multi-language configuration
  locales: {
    root: {
      label: 'English',
      ...enConfig
    },
    zh: {
      label: '中文',
      ...zhConfig
    }
  }
})