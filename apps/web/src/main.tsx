import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router'
import { Provider as UrqlProvider } from 'urql'

import { App } from './App'
import { urqlClient } from './lib/urql'
import './index.css'

const rootElement = document.getElementById('root')
if (!rootElement) {
  throw new Error('Root element #root was not found')
}

createRoot(rootElement).render(
  <StrictMode>
    <UrqlProvider value={urqlClient}>
      <BrowserRouter>
        <App />
      </BrowserRouter>
    </UrqlProvider>
  </StrictMode>,
)
