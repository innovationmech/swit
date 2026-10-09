const assert = require('node:assert/strict')
const { test } = require('node:test')
const path = require('node:path')
const postcss = require('postcss')
const config = require('../../postcss.config')

function processStyles(css) {
  const plugins = Object.entries(config.plugins).map(([name, options]) => require(name)(options))
  return postcss(plugins).process(css, { from: path.join(__dirname, 'fixture.css') })
}

test('custom theme CSS keeps vendor prefixes without adding a global reset', async () => {
  const { css } = await processStyles('.site-card { color: var(--vp-c-brand-1); user-select: none; }')
  assert.match(css, /color: var\(--vp-c-brand-1\)/)
  assert.match(css, /-webkit-user-select: none/)
  assert.doesNotMatch(css, /box-sizing|@layer|margin: 0/)
})

test('Tailwind 4 imports compile with the existing Swit theme and class-based dark mode', async () => {
  const { css } = await processStyles(`
    @import "tailwindcss";
    @config "../../tailwind.config.js";
    @source inline("bg-swit-primary dark:bg-swit-dark animate-fade-in font-sans");
  `)
  assert.match(css, /\.bg-swit-primary\s*\{[^}]*background-color:\s*#3490dc/)
  assert.match(css, /font-family:[^;}]*Inter/)
  assert.match(css, /@keyframes fadeIn/)
  assert.match(css, /\.dark/)
  assert.doesNotMatch(css, /prefers-color-scheme|@import "tailwindcss"|@source|@config/)
})
