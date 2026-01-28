import { useEffect, useState } from 'react'
import { ArrowLeft, List, Table, Grid, Star, Filter, SortAsc } from 'lucide-react'
import './CategoryFeed.css'
import MarketRow from './MarketRow'
import MarketTable from './MarketTable'
import DetailsPanel from './DetailsPanel'

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

type ViewMode = 'list' | 'table' | 'cards'

interface CategoryFeedProps {
  category: string
  eventTicker?: string
  onBack: () => void
  onSelectMarket: (ticker: string) => void
  selectedMarket: string | null
}

function CategoryFeed({ category, eventTicker, onBack, onSelectMarket, selectedMarket }: CategoryFeedProps) {
  const [opportunities, setOpportunities] = useState<Opportunity[]>([])
  const [loading, setLoading] = useState(true)
  const [viewMode, setViewMode] = useState<ViewMode>('list')
  const [sortBy, setSortBy] = useState<'liquidity' | 'spread' | 'activity'>('liquidity')
  const [searchQuery, setSearchQuery] = useState('')
  const [isPinned, setIsPinned] = useState(false)

  useEffect(() => {
    const fetchOpportunities = async () => {
      try {
        const response = await fetch('/api/v1/scanner/opportunities')
        if (response.ok) {
          const data = await response.json()
          let opps = data.opportunities || []
          
          // Filter by search query
          if (searchQuery) {
            opps = opps.filter((o: Opportunity) =>
              o.market_ticker.toLowerCase().includes(searchQuery.toLowerCase()) ||
              o.title.toLowerCase().includes(searchQuery.toLowerCase())
            )
          }
          
          // Sort
          opps.sort((a: Opportunity, b: Opportunity) => {
            switch (sortBy) {
              case 'liquidity':
                return b.liquidity_score - a.liquidity_score
              case 'spread':
                return a.spread_percent - b.spread_percent
              case 'activity':
                return b.recent_trades - a.recent_trades
              default:
                return 0
            }
          })
          
          setOpportunities(opps)
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
  }, [sortBy, searchQuery])

  const handleMarketClick = (ticker: string) => {
    onSelectMarket(ticker)
  }

  return (
    <div className="category-feed">
      <div className="category-feed-header">
        <div className="category-feed-header-top">
          <button className="category-feed-back" onClick={onBack}>
            <ArrowLeft size={18} />
            <span>Back</span>
          </button>
          <div className="category-feed-title-section">
            <div className="category-feed-title-row">
              <h1 className="category-feed-title">{category}</h1>
              <button
                className={`category-feed-pin ${isPinned ? 'pinned' : ''}`}
                onClick={() => setIsPinned(!isPinned)}
                title={isPinned ? 'Unpin category' : 'Pin category'}
              >
                <Star size={16} fill={isPinned ? 'currentColor' : 'none'} />
              </button>
            </div>
            <p className="category-feed-subtitle">Real-time market opportunities and liquidity</p>
          </div>
        </div>

        <div className="category-feed-controls">
          <div className="category-feed-search">
            <input
              type="text"
              placeholder="Search markets..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="category-feed-search-input"
            />
          </div>

          <div className="category-feed-controls-right">
            <select
              className="category-feed-sort"
              value={sortBy}
              onChange={(e) => setSortBy(e.target.value as any)}
            >
              <option value="liquidity">Sort by Liquidity</option>
              <option value="spread">Sort by Spread</option>
              <option value="activity">Sort by Activity</option>
            </select>

            <div className="category-feed-view-toggle">
              <button
                className={`view-toggle-btn ${viewMode === 'list' ? 'active' : ''}`}
                onClick={() => setViewMode('list')}
                title="List view"
              >
                <List size={16} />
              </button>
              <button
                className={`view-toggle-btn ${viewMode === 'table' ? 'active' : ''}`}
                onClick={() => setViewMode('table')}
                title="Table view"
              >
                <Table size={16} />
              </button>
            </div>
          </div>
        </div>
      </div>

      <div className="category-feed-content">
        <div className={`category-feed-main ${selectedMarket ? 'with-details' : ''}`}>
          {loading ? (
            <div className="category-feed-loading">
              <div className="spinner" />
              <span>Loading markets...</span>
            </div>
          ) : viewMode === 'table' ? (
            <MarketTable
              opportunities={opportunities}
              selectedMarket={selectedMarket}
              onSelectMarket={handleMarketClick}
            />
          ) : (
            <div className="category-feed-list">
              {opportunities.length === 0 ? (
                <div className="category-feed-empty">No markets found</div>
              ) : (
                opportunities.map((opp) => (
                  <MarketRow
                    key={opp.market_ticker}
                    opportunity={opp}
                    isSelected={selectedMarket === opp.market_ticker}
                    onClick={() => handleMarketClick(opp.market_ticker)}
                  />
                ))
              )}
            </div>
          )}
        </div>

        {selectedMarket && (
          <div className="category-feed-details">
            <DetailsPanel
              marketTicker={selectedMarket}
              onClose={() => onSelectMarket('')}
            />
          </div>
        )}
      </div>
    </div>
  )
}

export default CategoryFeed

