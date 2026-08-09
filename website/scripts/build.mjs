import { execFileSync } from 'node:child_process'
import { cpSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const websiteRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const repositoryRoot = resolve(websiteRoot, '..')
const backendRoot = join(repositoryRoot, 'backend')
const sourceRoot = join(websiteRoot, 'src')
const outputRoot = join(websiteRoot, 'dist')

function readJSON(path) {
  return JSON.parse(readFileSync(path, 'utf8'))
}

function flattenKeys(value, prefix = '') {
  return Object.entries(value).flatMap(([key, child]) => {
    const next = prefix ? `${prefix}.${key}` : key
    return child && typeof child === 'object' && !Array.isArray(child) ? flattenKeys(child, next) : [next]
  }).sort()
}

function getTranslation(resources, key) {
  const value = key.split('.').reduce((current, part) => current?.[part], resources)
  if (typeof value !== 'string') throw new Error(`Missing English translation: ${key}`)
  return value
}

function renderTemplate(name, resources, data) {
  const template = readFileSync(join(sourceRoot, name), 'utf8')
  const rendered = template.replace(/\{\{([a-zA-Z0-9_.-]+)\}\}/g, (_match, key) => getTranslation(resources.en, key))
  if (/\{\{[^}]+\}\}/.test(rendered)) throw new Error(`Unresolved template token in ${name}`)
  return rendered.replace('</head>', `<script>window.__SESHAT_I18N__=${safeJSON(resources)};window.__SESHAT_DATA__=${safeJSON(data)};</script>\n</head>`)
}

function safeJSON(value) {
  return JSON.stringify(value).replaceAll('<', '\\u003c')
}

function versionFromFile(name) {
  const version = readFileSync(join(repositoryRoot, name), 'utf8').trim()
  if (!/^v\d+\.\d+\.\d+$/.test(version)) throw new Error(`${name} must contain a semantic version`)
  return version
}

function buildFeed(channel, version) {
  const path = join(outputRoot, 'updates', 'v1', `${channel}.json`)
  execFileSync('go', ['run', './cmd/release-notes', 'feed', channel, version, path], {
    cwd: backendRoot,
    stdio: 'inherit',
  })
  const feed = readJSON(path)
  for (const release of feed.releases) {
    const tag = channel === 'beta' ? `beta-${release.version}` : release.version
    release.image_tag = tag
    release.release_url ||= `https://github.com/hienao/Seshat/releases/tag/${encodeURIComponent(tag)}`
  }
  writeFileSync(path, `${JSON.stringify(feed, null, 2)}\n`)
  return feed
}

export function build() {
  if (dirname(outputRoot) !== websiteRoot || outputRoot !== join(websiteRoot, 'dist')) {
    throw new Error(`Refusing to replace unexpected output directory: ${outputRoot}`)
  }
  rmSync(outputRoot, { recursive: true, force: true })
  mkdirSync(join(outputRoot, 'updates', 'v1'), { recursive: true })

  const resources = {
    en: readJSON(join(sourceRoot, 'locales', 'en.json')),
    'zh-CN': readJSON(join(sourceRoot, 'locales', 'zh-CN.json')),
  }
  const englishKeys = flattenKeys(resources.en)
  const chineseKeys = flattenKeys(resources['zh-CN'])
  if (JSON.stringify(englishKeys) !== JSON.stringify(chineseKeys)) {
    throw new Error('English and Chinese website locale keys do not match')
  }

  const data = {
    feeds: {
      beta: buildFeed('beta', versionFromFile('VERSION_BETA')),
      release: buildFeed('release', versionFromFile('VERSION_RELEASE')),
    },
  }
  writeFileSync(join(outputRoot, 'index.html'), renderTemplate('index.html', resources, data))
  mkdirSync(join(outputRoot, 'releases'), { recursive: true })
  writeFileSync(join(outputRoot, 'releases', 'index.html'), renderTemplate('releases.html', resources, data))
  cpSync(join(sourceRoot, 'assets'), join(outputRoot, 'assets'), { recursive: true })
  cpSync(join(sourceRoot, '_headers'), join(outputRoot, '_headers'))
  cpSync(join(sourceRoot, 'robots.txt'), join(outputRoot, 'robots.txt'))
  console.log(`Built Seshat website at ${outputRoot}`)
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) build()
