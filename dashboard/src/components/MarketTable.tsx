import { Eye } from 'lucide-react'
import { useWatchlist } from '../contexts/WatchlistContext'
import './MarketTable.css'

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

interface MarketTableProps {
  opportunities: Opportunity[]
  selectedMarket: string | null
  onSelectMarket: (ticker: string) => void
}

function MarketTable({ opportunities, selectedMarket, onSelectMarket }: MarketTableProps) {
  const { addToWatchlist, removeFromWatchlist, isInWatchlist } = useWatchlist()

  const formatPrice = (price: number) => {
    return price.toFixed(1) + '%'
  }

  const getLiquidityColor = (score: number) => {
    if (score > 0.7) return 'var(--semantic-positive)'
    if (score > 0.4) return 'var(--semantic-warning)'
    return 'var(--semantic-negative)'
  }

  const truncateTitle = (title: string, maxLength: number = 60) => {
    if (title.length <= maxLength) return title
    return title.substring(0, maxLength) + '...'
  }

  return (
    <div className="market-table-container">
      <table className="market-table">
        <thead className="market-table-header">
          <tr>
            <th className="col-market">Market</th>
            <th className="col-price">Price</th>
            <th className="col-spread">Spread</th>
            <th className="col-liquidity">Liquidity</th>
            <th className="col-activity">Activity</th>
            <th className="col-action">Action</th>
          </tr>
        </thead>
        <tbody className="market-table-body">
          {opportunities.length === 0 ? (
            <tr>
              <td colSpan={6} className="table-empty-cell">
                No markets found
              </td>
            </tr>
          ) : (
            opportunities.map((opp, index) => (
              <tr
                key={opp.market_ticker}
                className={`market-table-row ${selectedMarket === opp.market_ticker ? 'selected' : ''} ${index % 2 === 0 ? 'even' : 'odd'}`}
                onClick={() => onSelectMarket(opp.market_ticker)}
              >
                <td className="col-market">
                  <div className="market-cell-ticker">{opp.market_ticker}</div>
                  <div className="market-cell-title">{truncateTitle(opp.title)}</div>
                </td>
                <td className="col-price">
                  <div className="price-cell-primary">{formatPrice(opp.mid_price)}</div>
                  <div className="price-cell-secondary">
                    {formatPrice(opp.best_bid)} / {formatPrice(opp.best_ask)}
                  </div>
                </td>
                <td className="col-spread">
                  <div className={`spread-cell-value ${opp.spread_percent < 1 ? 'tight' : ''}`}>
                    {opp.spread_percent.toFixed(2)}%
                  </div>
                </td>
                <td className="col-liquidity">
                  <div className="liquidity-cell">
                    <div className="liquidity-bar">
                      <div
                        className="liquidity-fill"
                        style={{
                          width: `${opp.liquidity_score * 100}%`,
                          backgroundColor: getLiquidityColor(opp.liquidity_score),
                        }}
                      />
                    </div>
                    <div className="liquidity-value">
                      {(opp.liquidity_score * 100).toFixed(0)}%
                    </div>
                  </div>
                </td>
                <td className="col-activity">
                  <div className="activity-cell-value">{opp.recent_trades}</div>
                </td>
                <td className="col-action">
                  <button
                    className={`action-watch-btn ${isInWatchlist(opp.market_ticker) ? 'active' : ''}`}
                    onClick={(e) => {
                      e.stopPropagation()
                      if (isInWatchlist(opp.market_ticker)) {
                        removeFromWatchlist(opp.market_ticker)
                      } else {
                        addToWatchlist(opp.market_ticker)
                      }
                    }}
                    title={isInWatchlist(opp.market_ticker) ? 'Remove from watchlist' : 'Add to watchlist'}
                  >
                    <Eye size={14} fill={isInWatchlist(opp.market_ticker) ? 'currentColor' : 'none'} />
                  </button>
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}

export default MarketTable

