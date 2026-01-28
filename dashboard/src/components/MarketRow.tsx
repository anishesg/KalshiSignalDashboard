import { Eye } from 'lucide-react'
import { useWatchlist } from '../contexts/WatchlistContext'
import './MarketRow.css'

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

interface MarketRowProps {
  opportunity: Opportunity
  isSelected: boolean
  onClick: () => void
}

function MarketRow({ opportunity, isSelected, onClick }: MarketRowProps) {
  const { addToWatchlist, removeFromWatchlist, isInWatchlist } = useWatchlist()
  const inWatchlist = isInWatchlist(opportunity.market_ticker)

  const formatPrice = (price: number) => {
    return price.toFixed(1) + '%'
  }

  const getLiquidityColor = (score: number) => {
    if (score > 0.7) return 'var(--semantic-positive)'
    if (score > 0.4) return 'var(--semantic-warning)'
    return 'var(--semantic-negative)'
  }

  const truncateTitle = (title: string, maxLength: number = 80) => {
    if (title.length <= maxLength) return title
    return title.substring(0, maxLength) + '...'
  }

  return (
    <div
      className={`market-row ${isSelected ? 'selected' : ''}`}
      onClick={onClick}
    >
      <div className="market-row-main">
        <div className="market-row-info">
          <div className="market-row-ticker">{opportunity.market_ticker}</div>
          <div className="market-row-title">{truncateTitle(opportunity.title)}</div>
        </div>

        <div className="market-row-metrics">
          <div className="market-row-metric">
            <div className="metric-label">Price</div>
            <div className="metric-value">{formatPrice(opportunity.mid_price)}</div>
            <div className="metric-subvalue">
              {formatPrice(opportunity.best_bid)} / {formatPrice(opportunity.best_ask)}
            </div>
          </div>

          <div className="market-row-metric">
            <div className="metric-label">Spread</div>
            <div className={`metric-value ${opportunity.spread_percent < 1 ? 'tight' : ''}`}>
              {opportunity.spread_percent.toFixed(2)}%
            </div>
          </div>

          <div className="market-row-metric">
            <div className="metric-label">Liquidity</div>
            <div className="market-row-liquidity">
              <div className="liquidity-bar">
                <div
                  className="liquidity-fill"
                  style={{
                    width: `${opportunity.liquidity_score * 100}%`,
                    backgroundColor: getLiquidityColor(opportunity.liquidity_score),
                  }}
                />
              </div>
              <div className="liquidity-value">
                {(opportunity.liquidity_score * 100).toFixed(0)}%
              </div>
            </div>
          </div>

          <div className="market-row-metric">
            <div className="metric-label">Activity</div>
            <div className="metric-value">{opportunity.recent_trades}</div>
          </div>
        </div>
      </div>

      <div className="market-row-actions">
        <button
          className={`market-row-action-btn ${inWatchlist ? 'active' : ''}`}
          onClick={(e) => {
            e.stopPropagation()
            if (inWatchlist) {
              removeFromWatchlist(opportunity.market_ticker)
            } else {
              addToWatchlist(opportunity.market_ticker)
            }
          }}
          title={inWatchlist ? 'Remove from watchlist' : 'Add to watchlist'}
        >
          <Eye size={16} fill={inWatchlist ? 'currentColor' : 'none'} />
        </button>
      </div>
    </div>
  )
}

export default MarketRow

