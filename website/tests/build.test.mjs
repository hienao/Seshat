import assert from 'node:assert/strict'
import { existsSync, readFileSync } from 'node:fs'
import { join, resolve } from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import { build } from '../scripts/build.mjs'
import { chooseLanguage } from '../src/assets/site.js'

const websiteRoot = resolve(fileURLToPath(new URL('..', import.meta.url)))
const outputRoot = join(websiteRoot, 'dist')

test('manual language preference overrides browser language', () => {
  assert.equal(chooseLanguage('en', ['zh-CN']), 'en')
  assert.equal(chooseLanguage('zh-CN', ['en-US']), 'zh-CN')
  assert.equal(chooseLanguage(null, ['zh-Hans-CN', 'en']), 'zh-CN')
  assert.equal(chooseLanguage(null, ['fr-FR']), 'en')
})

test('build produces isolated valid feeds and static pages', () => {
  build()
  const beta = JSON.parse(readFileSync(join(outputRoot, 'updates', 'v1', 'beta.json'), 'utf8'))
  const release = JSON.parse(readFileSync(join(outputRoot, 'updates', 'v1', 'release.json'), 'utf8'))

  assert.equal(beta.channel, 'beta')
  assert.equal(release.channel, 'release')
  assert.ok(beta.releases.every((item) => item.channel === 'beta' && item.image_tag.startsWith('beta-v')))
  assert.ok(release.releases.every((item) => item.channel === 'release' && item.image_tag.startsWith('v')))
  assert.ok(beta.releases.some((item) => item.version === beta.latest_version))
  assert.ok(release.releases.some((item) => item.version === release.latest_version))

  for (const path of ['index.html', 'releases/index.html', 'assets/styles.css', 'assets/site.js', '_headers', 'robots.txt']) {
    assert.ok(existsSync(join(outputRoot, path)), `missing build artifact: ${path}`)
  }
  const home = readFileSync(join(outputRoot, 'index.html'), 'utf8')
  const releases = readFileSync(join(outputRoot, 'releases', 'index.html'), 'utf8')
  assert.doesNotMatch(home, /\{\{[^}]+\}\}/)
  assert.doesNotMatch(releases, /\{\{[^}]+\}\}/)
  assert.match(home, /window\.__SESHAT_I18N__/)
  assert.match(releases, /window\.__SESHAT_DATA__/)
})
