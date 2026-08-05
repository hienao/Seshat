import { Button } from '@appica/ui-react/button'
import { Moon, Sun } from '@appica/icons-react'
import { useEffect, useState } from 'react'

export function ThemeToggle() {
  const [dark, setDark] = useState(false)
  useEffect(() => setDark(document.documentElement.classList.contains('dark')), [])

  function toggleTheme() {
    const next = !dark
    document.documentElement.classList.toggle('dark', next)
    document.documentElement.style.colorScheme = next ? 'dark' : 'light'
    localStorage.setItem('basegoapp-theme', next ? 'dark' : 'light')
    setDark(next)
  }

  return (
    <Button variant="ghost" size="icon-md" aria-label={dark ? '切换到浅色模式' : '切换到深色模式'} onClick={toggleTheme}>
      {dark ? <Sun size={18} /> : <Moon size={18} />}
    </Button>
  )
}
