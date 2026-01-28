import { useEffect, useState } from 'react'
import { TrendingDown, TrendingUp, Zap, DollarSign, CheckCircle, Bell, AlertCircle } from 'lucide-react'
import { apiFetch } from '../config'
import './AlertsPanel.css'

interface Alert {
  id: string
  type: string
  market_ticker: string
  title: string
  timestamp: string
  reason: string
  suggestion: string
  action: string
  confidence: number
  hit_rate: number
  sample_size: number
  estimated_edge?: number
  estimated_slippage?: number
  can_execute: boolean
}

interface AlertsPanelProps {
  selectedMarket: string | null
  onSelectMarket: (ticker: string) => void
}

function AlertsPanel({ selectedMarket, onSelectMarket }: AlertsPanelProps) {
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [loading, setLoading] = useState(true)
  const [filter, setFilter] = useState<'all' | 'actionable'>('all')

  useEffect(() => {
    const fetchAlerts = async () => {
      try {
        const response = await apiFetch('/api/v1/alerts?limit=50')
        if (response.ok) {
          const data = await response.json()
          let alertsList = data.alerts || []
          
          if (filter === 'actionable') {
            alertsList = alertsList.filter((a: Alert) => a.can_execute && a.confidence > 0.6)
          }
          
          // Sort by confidence and timestamp
          alertsList.sort((a: Alert, b: Alert) => {
            if (b.confidence !== a.confidence) {
              return b.confidence - a.confidence
            }
            return new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime()
          })
          
          setAlerts(alertsList)
        }
      } catch (error) {
        console.error('Failed to fetch alerts:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchAlerts()
    const interval = setInterval(fetchAlerts, 2000)
    return () => clearInterval(interval)
  }, [filter])

  const getAlertIcon = (type: string) => {
    const iconProps = { size: 16, className: 'alert-type-icon' }
    switch (type) {
      case 'spread_tightened': return <TrendingDown {...iconProps} />
      case 'depth_increased': return <TrendingUp {...iconProps} />
      case 'imbalance_pressure': return <Zap {...iconProps} />
      case 'no_arb_violation': return <DollarSign {...iconProps} />
      case 'execution_ready': return <CheckCircle {...iconProps} />
      default: return <Bell {...iconProps} />
    }
  }

  const getAlertColor = (type: string) => {
    switch (type) {
      case 'no_arb_violation': return 'var(--color-arb)'
      case 'imbalance_pressure': return 'var(--color-pressure)'
      case 'execution_ready': return 'var(--color-good)'
      default: return 'var(--color-info)'
    }
  }

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const seconds = Math.floor(diff / 1000)
    
    if (seconds < 60) return `${seconds}s ago`
    const minutes = Math.floor(seconds / 60)
    if (minutes < 60) return `${minutes}m ago`
    const hours = Math.floor(minutes / 60)
    return `${hours}h ago`
  }

  if (loading) {
    return (
      <div className="alerts-panel">
        <div className="alerts-loading">Loading alerts...</div>
      </div>
    )
  }

  return (
    <div className="alerts-panel">
      <div className="alerts-header">
        <h1 className="alerts-title">Trading Alerts</h1>
        <p className="alerts-subtitle">Actionable opportunities with validated confidence</p>
        
        <div className="alerts-filters">
          <button
            className={`filter-btn ${filter === 'all' ? 'active' : ''}`}
            onClick={() => setFilter('all')}
          >
            All Alerts
          </button>
          <button
            className={`filter-btn ${filter === 'actionable' ? 'active' : ''}`}
            onClick={() => setFilter('actionable')}
          >
            Actionable Only
          </button>
        </div>
      </div>

      <div className="alerts-list">
        {alerts.length === 0 ? (
          <div className="alerts-empty">
            <AlertCircle size={48} className="empty-icon" />
            <div className="empty-text">No alerts yet</div>
            <div className="empty-subtext">Alerts will appear here when opportunities are detected</div>
          </div>
        ) : (
          alerts.map((alert) => (
            <div
              key={alert.id}
              className={`alert-card ${selectedMarket === alert.market_ticker ? 'selected' : ''}`}
              onClick={() => onSelectMarket(alert.market_ticker)}
            >
              <div className="alert-header">
                <div className="alert-icon" style={{ color: getAlertColor(alert.type) }}>
                  {getAlertIcon(alert.type)}
                </div>
                <div className="alert-info">
                  <div className="alert-market">{alert.market_ticker}</div>
                  <div className="alert-time">{formatTime(alert.timestamp)}</div>
                </div>
                <div className="alert-confidence">
                  <div className="confidence-value">{(alert.confidence * 100).toFixed(0)}%</div>
                  <div className="confidence-label">Confidence</div>
                </div>
              </div>

              <div className="alert-title">{alert.title}</div>

              <div className="alert-body">
                <div className="alert-reason">
                  <strong>Why:</strong> {alert.reason}
                </div>
                <div className="alert-suggestion">
                  <strong>Action:</strong> {alert.suggestion}
                </div>
              </div>

              <div className="alert-footer">
                <div className="alert-stats">
                  {alert.hit_rate > 0 && (
                    <span className="stat-badge">
                      Hit Rate: {(alert.hit_rate * 100).toFixed(0)}% ({alert.sample_size} samples)
                    </span>
                  )}
                  {alert.estimated_edge && (
                    <span className="stat-badge edge">
                      Edge: {alert.estimated_edge.toFixed(2)}¢
                    </span>
                  )}
                </div>
                <div className={`alert-action ${alert.action}`}>
                  {alert.action.toUpperCase()}
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

export default AlertsPanel

