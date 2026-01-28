import { createContext, useContext, useState, useEffect, ReactNode } from 'react'

interface WatchlistContextType {
  watchlist: string[]
  addToWatchlist: (ticker: string) => void
  removeFromWatchlist: (ticker: string) => void
  isInWatchlist: (ticker: string) => boolean
}

const WatchlistContext = createContext<WatchlistContextType | undefined>(undefined)

export function WatchlistProvider({ children }: { children: ReactNode }) {
  const [watchlist, setWatchlist] = useState<string[]>([])

  // Load watchlist from localStorage on mount
  useEffect(() => {
    const saved = localStorage.getItem('watchlist')
    if (saved) {
      try {
        setWatchlist(JSON.parse(saved))
      } catch (e) {
        console.error('Failed to load watchlist:', e)
      }
    }
  }, [])

  // Save watchlist to localStorage whenever it changes
  useEffect(() => {
    localStorage.setItem('watchlist', JSON.stringify(watchlist))
  }, [watchlist])

  const addToWatchlist = (ticker: string) => {
    setWatchlist(prev => {
      if (prev.includes(ticker)) return prev
      return [...prev, ticker]
    })
  }

  const removeFromWatchlist = (ticker: string) => {
    setWatchlist(prev => prev.filter(t => t !== ticker))
  }

  const isInWatchlist = (ticker: string) => {
    return watchlist.includes(ticker)
  }

  return (
    <WatchlistContext.Provider value={{
      watchlist,
      addToWatchlist,
      removeFromWatchlist,
      isInWatchlist,
    }}>
      {children}
    </WatchlistContext.Provider>
  )
}

export function useWatchlist() {
  const context = useContext(WatchlistContext)
  if (context === undefined) {
    throw new Error('useWatchlist must be used within a WatchlistProvider')
  }
  return context
}

