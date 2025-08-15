module.exports = {
  root: true,
  env: {
    browser: true,
    es2021: true,
    node: true
  },
  extends: [
    'eslint:recommended',
    'plugin:vue/vue3-recommended'
  ],
  parserOptions: {
    ecmaVersion: 'latest',
    sourceType: 'module'
  },
  plugins: [
    'vue'
  ],
  rules: {
    // 关闭一些在开发阶段过于严格的规则
    'no-unused-vars': 'warn',
    'vue/multi-word-component-names': 'off',
    'vue/no-v-html': 'warn', // 安全考虑，但允许在某些情况下使用
    
    // 安全相关规则
    'no-eval': 'error',
    'no-implied-eval': 'error',
    'no-new-func': 'error',
    'no-script-url': 'error',
    
    // 代码质量规则
    'prefer-const': 'error',
    'no-var': 'error',
    'eqeqeq': 'error'
  },
  ignorePatterns: [
    'node_modules/',
    'dist/',
    '.vitepress/cache/',
    '.security-report.*',
    'scripts/security-scanner.js', // 安全扫描器有特殊的代码模式
    'tests/**/*.ts' // 忽略TypeScript测试文件，避免解析错误
  ]
}