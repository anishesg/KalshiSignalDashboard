import { useState, useEffect } from 'react'
import { Compass, BarChart3, Bell, Star } from 'lucide-react'
import Header from './components/Header'
import Explore from './components/Explore'
import CategoryFeed from './components/CategoryFeed'
import AlertsPanel from './components/AlertsPanel'
import Watchlist from './components/Watchlist'
import StatusBar from './components/StatusBar'
import ConnectionError from './components/ConnectionError'
import './App.css'

type ViewType = 'explore' | 'markets' | 'watchlist' | 'alerts'

function App() {
  const [selectedMarket, setSelectedMarket] = useState<string | null>(null)
  const [connectionStatus, setConnectionStatus] = useState<'connected' | 'disconnected' | 'connecting'>('connecting')
  const [showError, setShowError] = useState(false)
  const [activeView, setActiveView] = useState<ViewType>('explore')
  const [selectedCategory, setSelectedCategory] = useState<string | null>(null)
  const [selectedEvent, setSelectedEvent] = useState<string | undefined>()
  const [pinnedCategories, setPinnedCategories] = useState<string[]>([])

  useEffect(() => {
    const checkHealth = async () => {
      try {
        const response = await fetch('/api/v1/health')
        if (response.ok) {
          setConnectionStatus('connected')
          setShowError(false)
        } else {
          setConnectionStatus('disconnected')
          setShowError(true)
        }
      } catch (error) {
        setConnectionStatus('disconnected')
        setShowError(true)
      }
    }

    checkHealth()
    const interval = setInterval(checkHealth, 5000)
    return () => clearInterval(interval)
  }, [])

  const handleCategorySelect = (category: string, eventTicker?: string) => {
    setSelectedCategory(category)
    setSelectedEvent(eventTicker)
    setActiveView('markets')
  }

  const handleBackToExplore = () => {
    setSelectedCategory(null)
    setSelectedEvent(undefined)
    setSelectedMarket(null)
    setActiveView('explore')
  }

  if (showError && connectionStatus === 'disconnected') {
    return (
      <div className="app">
        <Header connectionStatus={connectionStatus} />
        <ConnectionError />
        <StatusBar />
      </div>
    )
  }

  return (
    <div className="app">
      <Header connectionStatus={connectionStatus} />
      
      <div className="app-main">
        <div className="app-sidebar">
          <div className="sidebar-nav">
            <button
              className={`nav-item ${activeView === 'explore' ? 'active' : ''}`}
              onClick={() => {
                setActiveView('explore')
                setSelectedCategory(null)
                setSelectedMarket(null)
              }}
            >
              <Compass size={18} />
              <span>Explore</span>
            </button>
            <button
              className={`nav-item ${activeView === 'markets' && selectedCategory ? 'active' : ''}`}
              onClick={() => {
                if (selectedCategory) {
                  setActiveView('markets')
                }
              }}
              disabled={!selectedCategory}
            >
              <BarChart3 size={18} />
              <span>Markets</span>
            </button>
            <button
              className={`nav-item ${activeView === 'watchlist' ? 'active' : ''}`}
              onClick={() => {
                setActiveView('watchlist')
                setSelectedMarket(null)
              }}
            >
              <Star size={18} />
              <span>Watchlist</span>
            </button>
            <button
              className={`nav-item ${activeView === 'alerts' ? 'active' : ''}`}
              onClick={() => {
                setActiveView('alerts')
                setSelectedMarket(null)
              }}
            >
              <Bell size={18} />
              <span>Alerts</span>
            </button>
          </div>
        </div>

        <div className="app-content">
          {activeView === 'explore' ? (
            <Explore
              onSelectCategory={handleCategorySelect}
            />
          ) : activeView === 'markets' && selectedCategory ? (
            <CategoryFeed
              category={selectedCategory}
              eventTicker={selectedEvent}
              onBack={handleBackToExplore}
              onSelectMarket={setSelectedMarket}
              selectedMarket={selectedMarket}
            />
          ) : activeView === 'alerts' ? (
            <AlertsPanel
              selectedMarket={selectedMarket}
              onSelectMarket={setSelectedMarket}
            />
          ) : activeView === 'watchlist' ? (
            <Watchlist />
          ) : (
            <Explore
              onSelectCategory={handleCategorySelect}
              pinnedCategories={pinnedCategories}
            />
          )}
        </div>
      </div>

      <StatusBar />
    </div>
  )
}

export default App
