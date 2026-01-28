import { useEffect, useState } from 'react'
import { X, Eye } from 'lucide-react'
import { useWatchlist } from '../contexts/WatchlistContext'
import { apiFetch } from '../config'
import './MarketDetail.css'

interface MarketDetailProps {
  marketTicker: string
  onClose: () => void
}

interface Market {
  ticker: string
  title: string
  category: string
  status: string
  event_ticker: string
  yes_sub_title?: string
  no_sub_title?: string
}

interface Orderbook {
  market_ticker: string
  bids: Array<{ price: number; quantity: number }>
  asks: Array<{ price: number; quantity: number }>
  last_update: string
}

function MarketDetail({ marketTicker, onClose }: MarketDetailProps) {
  const { addToWatchlist, removeFromWatchlist, isInWatchlist } = useWatchlist()
  const [orderbook, setOrderbook] = useState<Orderbook | null>(null)
  const [market, setMarket] = useState<Market | null>(null)
  const [loading, setLoading] = useState(true)
  const inWatchlist = isInWatchlist(marketTicker)

  useEffect(() => {
    const fetchData = async () => {
      try {
        // Fetch market details and orderbook in parallel
        const [marketRes, orderbookRes] = await Promise.all([
          apiFetch(`/api/v1/markets/${marketTicker}`),
          apiFetch(`/api/v1/markets/${marketTicker}/orderbook`)
        ])

        if (marketRes.ok) {
          const marketData = await marketRes.json()
          setMarket(marketData.market || marketData) // Handle both response formats
        }

        if (orderbookRes.ok) {
          const orderbookData = await orderbookRes.json()
          setOrderbook(orderbookData.orderbook || orderbookData) // Handle both response formats
        }
      } catch (error) {
        console.error('Failed to fetch market data:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
    const interval = setInterval(fetchData, 1000)
    return () => clearInterval(interval)
  }, [marketTicker])

  const formatPrice = (cents: number) => {
    return (cents / 100).toFixed(1) + '%'
  }

  if (loading) {
    return (
      <div className="market-detail">
        <div className="detail-loading">Loading...</div>
      </div>
    )
  }

  if (!orderbook) {
    return (
      <div className="market-detail">
        <button className="detail-close" onClick={onClose}>
          <X size={18} />
        </button>
        <div className="detail-empty">No orderbook data</div>
      </div>
    )
  }

  const bids = orderbook.bids || []
  const asks = orderbook.asks || []
  const bestBid = bids[0]?.price || 0
  const bestAsk = asks[0]?.price || 0
  const midPrice = (bestBid + bestAsk) / 200.0
  const spread = bestAsk - bestBid

  // For binary markets: bids are YES, asks are NO
  // Use yes_sub_title and no_sub_title from API if available, otherwise fallback to "Yes"/"No"
  const yesLabel = market?.yes_sub_title || 'Yes'
  const noLabel = market?.no_sub_title || 'No'

  return (
    <div className="market-detail">
      <div className="detail-header">
        <div className="detail-header-content">
          <div className="detail-title-section">
            <h2 className="detail-title">{marketTicker}</h2>
            {market?.title && (
              <p className="detail-subtitle">{market.title}</p>
            )}
          </div>
          <button className="detail-close" onClick={onClose} title="Close">
            <X size={18} />
          </button>
        </div>
      </div>

      <div className="detail-content">
        <div className="detail-section">
          <h3 className="detail-section-title">Overview</h3>
          <div className="detail-metrics-grid">
            <div className="detail-metric-card">
              <div className="detail-metric-label">Mid Price</div>
              <div className="detail-metric-value">{midPrice.toFixed(1)}%</div>
            </div>
            <div className="detail-metric-card">
              <div className="detail-metric-label">Spread</div>
              <div className="detail-metric-value">{(spread / 100).toFixed(2)}%</div>
            </div>
            <div className="detail-metric-card">
              <div className="detail-metric-label">Best Bid</div>
              <div className="detail-metric-value">{formatPrice(bestBid)}</div>
            </div>
            <div className="detail-metric-card">
              <div className="detail-metric-label">Best Ask</div>
              <div className="detail-metric-value">{formatPrice(bestAsk)}</div>
            </div>
          </div>
        </div>

        <div className="detail-section">
          <h3 className="detail-section-title">Orderbook</h3>
          <div className="detail-orderbook">
            <div className="orderbook-side">
              <div className="orderbook-header">{yesLabel}</div>
              <div className="orderbook-levels">
                {bids.slice(0, 10).map((level, idx) => (
                  <div key={idx} className="orderbook-level bid">
                    <span className="level-price">{formatPrice(level.price)}</span>
                    <span className="level-size">{level.quantity.toLocaleString()}</span>
                  </div>
                ))}
              </div>
            </div>
            
            <div className="orderbook-divider" />
            
            <div className="orderbook-side">
              <div className="orderbook-header">{noLabel}</div>
              <div className="orderbook-levels">
                {asks.slice(0, 10).map((level, idx) => (
                  <div key={idx} className="orderbook-level ask">
                    <span className="level-price">{formatPrice(level.price)}</span>
                    <span className="level-size">{level.quantity.toLocaleString()}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>

        <div className="detail-section">
          <h3 className="detail-section-title">Actions</h3>
          <div className="detail-actions">
            <button
              className={`detail-action-btn ${inWatchlist ? 'active' : ''}`}
              onClick={() => {
                if (inWatchlist) {
                  removeFromWatchlist(marketTicker)
                } else {
                  addToWatchlist(marketTicker)
                }
              }}
            >
              <Eye size={16} fill={inWatchlist ? 'currentColor' : 'none'} />
              <span>{inWatchlist ? 'Remove from Watchlist' : 'Add to Watchlist'}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

export default MarketDetail

