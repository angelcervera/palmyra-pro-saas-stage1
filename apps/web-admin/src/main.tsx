import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App'
import { ThemeProvider } from '@/providers/theme'
import { I18nProvider } from '@/providers/i18n'
import { QueryProvider } from '@/providers/query'
import { AuthProvider } from '@/providers/auth'

createRoot(document.getElementById('root')!).render(
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
