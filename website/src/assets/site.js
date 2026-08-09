export function chooseLanguage(manualLanguage, browserLanguages = []) {
  if (manualLanguage === 'en' || manualLanguage === 'zh-CN') return manualLanguage
  return browserLanguages.some((language) => String(language).toLowerCase().startsWith('zh')) ? 'zh-CN' : 'en'
}

if (typeof document !== 'undefined') (() => {
  const storageKey = 'seshat-site-language'
  const resources = window.__SESHAT_I18N__ || {}
  const data = window.__SESHAT_DATA__ || { feeds: {} }
  let activeFilter = 'all'

  function storedLanguage() {
    try {
      const value = localStorage.getItem(storageKey)
      return value === 'en' || value === 'zh-CN' ? value : null
    } catch {
      return null
    }
  }

  function translation(language, key) {
    return key.split('.').reduce((current, part) => current?.[part], resources[language]) ?? key
  }

  function applyLanguage(language, persist = false) {
    if (!resources[language]) language = 'en'
    document.documentElement.lang = language
    document.querySelectorAll('[data-i18n]').forEach((element) => {
      element.textContent = translation(language, element.dataset.i18n)
    })
    document.querySelectorAll('[data-i18n-aria]').forEach((element) => {
      element.setAttribute('aria-label', translation(language, element.dataset.i18nAria))
    })
    document.querySelectorAll('[data-i18n-content]').forEach((element) => {
      element.setAttribute('content', translation(language, element.dataset.i18nContent))
    })
    document.querySelectorAll('[data-language]').forEach((button) => {
      const selected = button.dataset.language === language
      button.classList.toggle('selected', selected)
      button.setAttribute('aria-pressed', String(selected))
    })
    if (persist) {
      try { localStorage.setItem(storageKey, language) } catch { /* Storage can be disabled. */ }
    }
    if (document.body.dataset.page === 'releases') renderReleases(language, activeFilter)
  }

  function escapeHTML(value) {
    const element = document.createElement('span')
    element.textContent = String(value ?? '')
    return element.innerHTML
  }

  function releaseCard(release, language, latestVersion) {
    const localized = language === 'zh-CN' ? 'zh-CN' : 'en'
    const summary = release.summary?.[localized] || ''
    const changes = (release.changes || []).map((change) => `
      <li><span class="change-type type-${escapeHTML(change.type)}">${escapeHTML(translation(language, `releases.types.${change.type}`))}</span><span>${escapeHTML(change.text?.[localized])}</span></li>`).join('')
    const notes = release.upgrade_notes?.[localized] || []
    const upgrade = notes.length ? `<aside class="upgrade-notes"><strong>${escapeHTML(translation(language, 'releases.upgradeNotes'))}</strong><ul>${notes.map((note) => `<li>${escapeHTML(note)}</li>`).join('')}</ul></aside>` : ''
    return `<article class="release-card" id="${escapeHTML(release.channel)}-${escapeHTML(release.version)}">
      <header><div><span class="release-channel channel-${escapeHTML(release.channel)}">${escapeHTML(release.channel === 'beta' ? 'Beta' : 'Release')}</span>${release.version === latestVersion ? `<span class="latest-pill">${escapeHTML(translation(language, 'releases.latest'))}</span>` : ''}</div><time>${escapeHTML(release.version)}</time></header>
      <h2>${escapeHTML(summary)}</h2>
      <ul class="change-list">${changes}</ul>
      ${upgrade}
      <a class="release-link" href="${escapeHTML(release.release_url)}" target="_blank" rel="noreferrer">${escapeHTML(translation(language, 'releases.viewOnGithub'))} <span aria-hidden="true">↗</span></a>
    </article>`
  }

  function renderReleases(language, filter) {
    const container = document.querySelector('#release-list')
    if (!container) return
    const channels = filter === 'all' ? ['beta', 'release'] : [filter]
    const releases = channels.flatMap((channel) => (data.feeds[channel]?.releases || []).map((release) => ({ ...release, channel })))
      .sort((left, right) => right.version.localeCompare(left.version, undefined, { numeric: true }))
    container.innerHTML = releases.length
      ? releases.map((release) => releaseCard(release, language, data.feeds[release.channel]?.latest_version)).join('')
      : `<p class="empty-state">${escapeHTML(translation(language, 'releases.empty'))}</p>`
  }

  document.querySelectorAll('[data-language]').forEach((button) => {
    button.addEventListener('click', () => applyLanguage(button.dataset.language, true))
  })
  document.querySelectorAll('[data-release-filter]').forEach((button) => {
    button.addEventListener('click', () => {
      activeFilter = button.dataset.releaseFilter
      document.querySelectorAll('[data-release-filter]').forEach((candidate) => {
        const selected = candidate === button
        candidate.classList.toggle('selected', selected)
        candidate.setAttribute('aria-selected', String(selected))
      })
      renderReleases(document.documentElement.lang, activeFilter)
    })
  })

  for (const channel of ['beta', 'release']) {
    document.querySelectorAll(`[data-version="${channel}"]`).forEach((element) => { element.textContent = data.feeds[channel]?.latest_version || '—' })
  }
  document.querySelectorAll('[data-latest-tag]').forEach((element) => {
    const version = data.feeds.beta?.latest_version
    element.textContent = version ? `beta-${version}` : 'beta'
  })

  const initialLanguage = chooseLanguage(storedLanguage(), navigator.languages || [navigator.language || 'en'])
  applyLanguage(initialLanguage)
  document.querySelector('[data-release-filter="all"]')?.classList.add('selected')
  document.querySelector('[data-release-filter="all"]')?.setAttribute('aria-selected', 'true')
})()
