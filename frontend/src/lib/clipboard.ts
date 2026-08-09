export async function copyTextToClipboard(value: string) {
  if (navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value)
      return
    } catch {
      // In insecure contexts and embedded browsers the Clipboard API may exist
      // but reject writes. Fall back to a selected textarea in that case.
    }
  }

  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.readOnly = true
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  textarea.style.pointerEvents = 'none'
  document.body.appendChild(textarea)
  textarea.focus()
  textarea.select()
  textarea.setSelectionRange(0, value.length)
  let copied = false
  try {
    copied = typeof document.execCommand === 'function' && document.execCommand('copy')
  } finally {
    textarea.remove()
  }
  if (!copied) throw new Error('复制失败')
}
