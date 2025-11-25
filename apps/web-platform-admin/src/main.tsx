import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App'
import { ThemeProvider } from '@/providers/theme'
import { I18nProvider } from '@/providers/i18n'
import { QueryProvider } from '@/providers/query'
import { AuthProvider } from '@/providers/auth'

const rootElement = document.getElementById('root')

if (!rootElement) {
  throw new Error('Root container element not found')
}

createRoot(rootElement).render(
  <StrictMode>
    <ThemeProvider>
      <BrowserRouter>
        <I18nProvider>
          <QueryProvider>
            <AuthProvider>
              <App />
            </AuthProvider>
          </QueryProvider>
        </I18nProvider>
      </BrowserRouter>
    </ThemeProvider>
  </StrictMode>,
)
