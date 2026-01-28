import React from 'react'
import ReactDOM from 'react-dom/client'
import App from './App'
import { WatchlistProvider } from './contexts/WatchlistContext'
import './index.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <WatchlistProvider>
      <App />
    </WatchlistProvider>
  </React.StrictMode>,
)

