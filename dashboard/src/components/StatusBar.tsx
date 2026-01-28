import { useEffect, useState } from 'react'
import { apiFetch } from '../config'
import './StatusBar.css'

function StatusBar() {
  const [uptime, setUptime] = useState(0)
  const [marketsCount, setMarketsCount] = useState(0)
  const [alertsCount, setAlertsCount] = useState(0)

  useEffect(() => {
    const startTime = Date.now()
    const interval = setInterval(() => {
      setUptime(Math.floor((Date.now() - startTime) / 1000))
    }, 1000)
    return () => clearInterval(interval)
  }, [])

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const [healthRes, alertsRes] = await Promise.all([
          apiFetch('/api/v1/health'),
          apiFetch('/api/v1/alerts?limit=1')
        ])
        
        if (healthRes.ok) {
          const health = await healthRes.json()
          setMarketsCount(health.markets || 0)
        }
        
        if (alertsRes.ok) {
          const alerts = await alertsRes.json()
          setAlertsCount(alerts.count || 0)
        }
      } catch (error) {
        console.error('Failed to fetch stats:', error)
      }
    }

    fetchStats()
    const interval = setInterval(fetchStats, 10000)
    return () => clearInterval(interval)
  }, [])

  const formatUptime = (seconds: number): string => {
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60
    return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }

  return (
    <div className="status-bar">
      <div className="status-item">
        <span className="status-label">Markets</span>
        <span className="status-value">{marketsCount}</span>
      </div>
      <div className="status-divider" />
      <div className="status-item">
        <span className="status-label">Alerts</span>
        <span className="status-value">{alertsCount}</span>
      </div>
      <div className="status-divider" />
      <div className="status-item">
        <span className="status-label">Uptime</span>
        <span className="status-value">{formatUptime(uptime)}</span>
      </div>
    </div>
  )
}

export default StatusBar
