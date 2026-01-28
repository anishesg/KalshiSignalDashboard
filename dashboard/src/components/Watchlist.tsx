import { useEffect, useState } from 'react'
import { Eye, X } from 'lucide-react'
import { useWatchlist } from '../contexts/WatchlistContext'
import MarketRow from './MarketRow'
import DetailsPanel from './DetailsPanel'
import './Watchlist.css'

interface Opportunity {
  market_ticker: string
  title: string
  status: string
  best_bid: number
  best_ask: number
  mid_price: number
  spread: number
  spread_percent: number
  liquidity_score: number
  recent_trades: number
  imbalance: number
  estimated_slippage_100: number
  can_execute_100: boolean
}

function Watchlist() {
  const { watchlist, removeFromWatchlist } = useWatchlist()
  const [opportunities, setOpportunities] = useState<Record<string, Opportunity>>({})
  const [loading, setLoading] = useState(true)
  const [selectedMarket, setSelectedMarket] = useState<string | null>(null)

  useEffect(() => {
    const fetchOpportunities = async () => {
      if (watchlist.length === 0) {
        setOpportunities({})
        setLoading(false)
        return
      }

      try {
        const response = await fetch('/api/v1/scanner/opportunities')
        if (response.ok) {
          const data = await response.json()
          const opps = data.opportunities || []
          
          // Filter to only watchlist markets and create a map
          const watchlistMap: Record<string, Opportunity> = {}
          opps.forEach((opp: Opportunity) => {
            if (watchlist.includes(opp.market_ticker)) {
              watchlistMap[opp.market_ticker] = opp
            }
          })
          
          setOpportunities(watchlistMap)
        }
      } catch (error) {
        console.error('Failed to fetch opportunities:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchOpportunities()
    const interval = setInterval(fetchOpportunities, 3000)
    return () => clearInterval(interval)
  }, [watchlist])

  const watchlistOpportunities = watchlist
    .map(ticker => opportunities[ticker])
    .filter(opp => opp !== undefined)
    .sort((a, b) => a.market_ticker.localeCompare(b.market_ticker))

  if (loading) {
    return (
      <div className="watchlist">
        <div className="watchlist-loading">
          <div className="spinner" />
          <span>Loading watchlist...</span>
        </div>
      </div>
    )
  }

  return (
    <div className="watchlist">
      <div className="watchlist-header">
        <div className="watchlist-header-content">
          <h1 className="watchlist-title">Watchlist</h1>
          <p className="watchlist-subtitle">
            {watchlist.length === 0 
              ? 'No markets in watchlist' 
              : `${watchlist.length} market${watchlist.length === 1 ? '' : 's'} tracked`}
          </p>
        </div>
      </div>

      <div className="watchlist-content">
        <div className={`watchlist-main ${selectedMarket ? 'with-details' : ''}`}>
          {watchlist.length === 0 ? (
            <div className="watchlist-empty">
              <Eye size={48} className="empty-icon" />
              <div className="empty-text">Your watchlist is empty</div>
              <div className="empty-subtext">
                Click the eye icon on any market to add it to your watchlist
              </div>
            </div>
          ) : watchlistOpportunities.length === 0 ? (
            <div className="watchlist-empty">
              <div className="empty-text">Loading watchlist markets...</div>
            </div>
          ) : (
            <div className="watchlist-list">
              {watchlistOpportunities.map((opp) => (
                <div key={opp.market_ticker} className="watchlist-item-wrapper">
                  <MarketRow
                    opportunity={opp}
                    isSelected={selectedMarket === opp.market_ticker}
                    onClick={() => setSelectedMarket(opp.market_ticker)}
                  />
                  <button
                    className="watchlist-remove-btn"
                    onClick={() => removeFromWatchlist(opp.market_ticker)}
                    title="Remove from watchlist"
                  >
                    <X size={14} />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        {selectedMarket && (
          <div className="watchlist-details">
            <DetailsPanel
              marketTicker={selectedMarket}
              onClose={() => setSelectedMarket(null)}
            />
          </div>
        )}
      </div>
    </div>
  )
}

export default Watchlist

