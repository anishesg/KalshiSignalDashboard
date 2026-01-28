import { useEffect, useState, useMemo } from 'react'
import { 
  Vote, 
  Landmark, 
  UserCheck, 
  Scale, 
  Briefcase, 
  Building2, 
  FileText, 
  Globe, 
  TrendingUp, 
  DollarSign, 
  BarChart3,
  Gavel,
  Award,
  Users,
  MapPin,
  AlertCircle,
  Folder
} from 'lucide-react'
import { apiFetch } from '../config'
import './Explore.css'

interface Category {
  category: string
  event_tickers: string[]
  total_markets: number
  events: Record<string, {
    event_ticker: string
    markets: Array<{
      ticker: string
      title: string
      status: string
    }>
    count: number
  }>
}

interface SubcategoryData {
  fullCategory: string
  subcategory: string
  totalMarkets: number
  eventTickers: string[]
  events: Record<string, Category['events'][string]>
  states?: string[] // Extracted state abbreviations if applicable
}

interface CategoryGroup {
  parentCategory: string
  subcategories: SubcategoryData[]
  totalMarkets: number
  icon: React.ReactNode
}

// US State abbreviations
const STATE_ABBREVIATIONS = new Set([
  'AL', 'AK', 'AZ', 'AR', 'CA', 'CO', 'CT', 'DE', 'FL', 'GA',
  'HI', 'ID', 'IL', 'IN', 'IA', 'KS', 'KY', 'LA', 'ME', 'MD',
  'MA', 'MI', 'MN', 'MS', 'MO', 'MT', 'NE', 'NV', 'NH', 'NJ',
  'NM', 'NY', 'NC', 'ND', 'OH', 'OK', 'OR', 'PA', 'RI', 'SC',
  'SD', 'TN', 'TX', 'UT', 'VT', 'VA', 'WA', 'WV', 'WI', 'WY'
])

function extractStates(eventTickers: string[]): string[] {
  const states = new Set<string>()
  eventTickers.forEach(ticker => {
    // Check if ticker contains state abbreviation (e.g., "CA-", "-CA-", "CA-10")
    const upperTicker = ticker.toUpperCase()
    STATE_ABBREVIATIONS.forEach(state => {
      if (upperTicker.includes(`-${state}-`) || 
          upperTicker.includes(`-${state}`) ||
          upperTicker.startsWith(`${state}-`)) {
        states.add(state)
      }
    })
  })
  return Array.from(states).sort()
}

interface ExploreProps {
  onSelectCategory: (category: string, eventTicker?: string) => void
}

function Explore({ onSelectCategory }: ExploreProps) {
  const [categories, setCategories] = useState<Category[]>([])
  const [loading, setLoading] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')

  useEffect(() => {
    const fetchCategories = async () => {
      try {
        const response = await apiFetch('/api/v1/categories')
        if (response.ok) {
          const data = await response.json()
          setCategories(data.categories || [])
        }
      } catch (error) {
        console.error('Failed to fetch categories:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchCategories()
    const interval = setInterval(fetchCategories, 30000)
    return () => clearInterval(interval)
  }, [])

  // Parse categories into hierarchical structure
  const categoryGroups = useMemo(() => {
    const groups = new Map<string, CategoryGroup>()
    
    categories.forEach(cat => {
      // Parse category like "Elections - Senate" into parent "Elections" and subcategory "Senate"
      const parts = cat.category.split(' - ')
      const parentCategory = parts.length > 1 ? parts[0] : cat.category
      const subcategory = parts.length > 1 ? parts.slice(1).join(' - ') : 'General'
      
      if (!groups.has(parentCategory)) {
        groups.set(parentCategory, {
          parentCategory,
          subcategories: [],
          totalMarkets: 0,
          icon: getCategoryIcon(parentCategory)
        })
      }
      
      const group = groups.get(parentCategory)!
      const states = extractStates(cat.event_tickers)
      group.subcategories.push({
        fullCategory: cat.category,
        subcategory,
        totalMarkets: cat.total_markets,
        eventTickers: cat.event_tickers,
        events: cat.events,
        states: states.length > 0 ? states : undefined
      })
      group.totalMarkets += cat.total_markets
    })
    
    return Array.from(groups.values()).sort((a, b) => b.totalMarkets - a.totalMarkets)
  }, [categories])

  const filteredGroups = categoryGroups.filter(group =>
    group.parentCategory.toLowerCase().includes(searchQuery.toLowerCase()) ||
    group.subcategories.some(sub => 
      sub.subcategory.toLowerCase().includes(searchQuery.toLowerCase())
    )
  )

  function getCategoryIcon(category: string): React.ReactNode {
    const catLower = category.toLowerCase()
    
    // Elections & Politics
    if (catLower.includes('election') || catLower.includes('vote') || catLower.includes('primary')) {
      return <Vote size={20} />
    }
    if (catLower.includes('senate')) {
      return <Landmark size={20} />
    }
    if (catLower.includes('house') || catLower.includes('congress')) {
      return <Building2 size={20} />
    }
    if (catLower.includes('governor') || catLower.includes('governorship')) {
      return <UserCheck size={20} />
    }
    if (catLower.includes('president') || catLower.includes('presidential')) {
      return <Award size={20} />
    }
    if (catLower.includes('attorney general') || catLower.includes('attorney')) {
      return <Scale size={20} />
    }
    
    // Appointments
    if (catLower.includes('appointment') || catLower.includes('confirm')) {
      return <Briefcase size={20} />
    }
    if (catLower.includes('supreme court') || catLower.includes('scotus') || catLower.includes('judge')) {
      return <Scale size={20} />
    }
    if (catLower.includes('cabinet')) {
      return <Users size={20} />
    }
    
    // White House & Executive
    if (catLower.includes('white house') || catLower.includes('executive')) {
      return <Building2 size={20} />
    }
    
    // Legislation
    if (catLower.includes('legislation') || catLower.includes('bill') || catLower.includes('law')) {
      return <FileText size={20} />
    }
    
    // International
    if (catLower.includes('international') || catLower.includes('foreign')) {
      return <Globe size={20} />
    }
    if (catLower.includes('nato') || catLower.includes('alliance')) {
      return <Globe size={20} />
    }
    
    // Economics
    if (catLower.includes('economic') || catLower.includes('gdp') || catLower.includes('inflation')) {
      return <DollarSign size={20} />
    }
    if (catLower.includes('federal reserve') || catLower.includes('fed')) {
      return <TrendingUp size={20} />
    }
    if (catLower.includes('budget') || catLower.includes('trade')) {
      return <DollarSign size={20} />
    }
    
    // Polls
    if (catLower.includes('poll') || catLower.includes('approval')) {
      return <BarChart3 size={20} />
    }
    
    // Legal
    if (catLower.includes('legal') || catLower.includes('arrest') || catLower.includes('charge')) {
      return <Gavel size={20} />
    }
    if (catLower.includes('impeach')) {
      return <Gavel size={20} />
    }
    
    // Policy
    if (catLower.includes('policy') || catLower.includes('regulation')) {
      return <FileText size={20} />
    }
    if (catLower.includes('immigration') || catLower.includes('border')) {
      return <MapPin size={20} />
    }
    if (catLower.includes('healthcare') || catLower.includes('health care')) {
      return <AlertCircle size={20} />
    }
    if (catLower.includes('climate') || catLower.includes('carbon')) {
      return <Globe size={20} />
    }
    
    // Local
    if (catLower.includes('local') || catLower.includes('mayor')) {
      return <MapPin size={20} />
    }
    
    // Default
    return <Folder size={20} />
  }

  const [expandedParents, setExpandedParents] = useState<Set<string>>(new Set())
  
  const toggleParent = (parentCategory: string) => {
    setExpandedParents(prev => {
      const next = new Set(prev)
      if (next.has(parentCategory)) {
        next.delete(parentCategory)
      } else {
        next.add(parentCategory)
      }
      return next
    })
  }

  const handleSubcategoryClick = (fullCategory: string) => {
    onSelectCategory(fullCategory)
  }

  if (loading) {
    return (
      <div className="explore">
        <div className="explore-loading">
          <div className="spinner" />
          <span>Loading categories...</span>
        </div>
      </div>
    )
  }

  return (
    <div className="explore">
      <div className="explore-header">
        <div className="explore-title-section">
          <h1 className="explore-title">Explore Markets</h1>
          <p className="explore-subtitle">Discover categories and find opportunities</p>
        </div>
        <div className="explore-search">
          <input
            type="text"
            placeholder="Search categories..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="explore-search-input"
          />
        </div>
      </div>

      <div className="explore-section">
        <div className="explore-section-header">
          <h2 className="explore-section-title">Categories</h2>
          <span className="explore-section-count">{filteredGroups.length} categories</span>
        </div>
        <div className="category-groups">
          {filteredGroups.length === 0 ? (
            <div className="explore-empty">
              {searchQuery ? 'No categories match your search' : 'No categories available'}
            </div>
          ) : (
            filteredGroups.map((group) => {
              const isExpanded = expandedParents.has(group.parentCategory)
              
              return (
                <div key={group.parentCategory} className="category-group-card">
                  <div 
                    className="category-group-header"
                    onClick={() => toggleParent(group.parentCategory)}
                  >
                    <div className="category-group-header-left">
                      <div className="category-group-icon">
                        {group.icon}
                      </div>
                      <div className="category-group-info">
                        <h3 className="category-group-name">{group.parentCategory}</h3>
                        <div className="category-group-meta">
                          {group.subcategories.length} subcategor{group.subcategories.length === 1 ? 'y' : 'ies'} • {group.totalMarkets} markets
                        </div>
                      </div>
                    </div>
                    <div className="category-group-expand">
                      {isExpanded ? '−' : '+'}
                    </div>
                  </div>
                  
                  {isExpanded && (
                    <div className="category-group-subcategories">
                      {group.subcategories.map((sub) => (
                        <div key={sub.fullCategory}>
                          <div
                            className="subcategory-card"
                            onClick={() => handleSubcategoryClick(sub.fullCategory)}
                          >
                            <div className="subcategory-content">
                              <h4 className="subcategory-name">{sub.subcategory}</h4>
                              <div className="subcategory-metrics">
                                <span className="subcategory-market-count">{sub.totalMarkets} markets</span>
                                {sub.states && sub.states.length > 0 && (
                                  <span className="subcategory-states">
                                    {sub.states.length} state{sub.states.length === 1 ? '' : 's'}: {sub.states.slice(0, 5).join(', ')}
                                    {sub.states.length > 5 && ` +${sub.states.length - 5}`}
                                  </span>
                                )}
                                {!sub.states && sub.eventTickers.length > 0 && (
                                  <span className="subcategory-event-count">{sub.eventTickers.length} events</span>
                                )}
                              </div>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )
            })
          )}
        </div>
      </div>
    </div>
  )
}

export default Explore

