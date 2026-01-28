import { useEffect, useState } from 'react'
import { Activity, Clock } from 'lucide-react'
import './Header.css'

interface HeaderProps {
  connectionStatus: 'connected' | 'disconnected' | 'connecting'
}

function Header({ connectionStatus }: HeaderProps) {
  const [time, setTime] = useState(new Date())

  useEffect(() => {
    const interval = setInterval(() => setTime(new Date()), 1000)
    return () => clearInterval(interval)
  }, [])

  const getStatusColor = () => {
    switch (connectionStatus) {
      case 'connected':
        return 'var(--accent-success)'
      case 'disconnected':
        return 'var(--accent-danger)'
      default:
        return 'var(--accent-warning)'
    }
  }

  const getStatusText = () => {
    switch (connectionStatus) {
      case 'connected':
        return 'Connected'
      case 'disconnected':
        return 'Disconnected'
      default:
        return 'Connecting...'
    }
  }

  return (
    <header className="header">
      <div className="header-content">
        <div className="header-left">
          <div className="header-brand">
            <Activity size={20} className="header-logo" />
            <div>
              <h1 className="header-title">Kalshi Signal Feed</h1>
              <div className="header-subtitle">Real-Time Market Analysis</div>
            </div>
          </div>
        </div>
        <div className="header-right">
          <div className="header-status">
            <div
              className="status-indicator"
              style={{ backgroundColor: getStatusColor() }}
            />
            <span className="status-text">{getStatusText()}</span>
          </div>
          <div className="header-time">
            <Clock size={14} />
            <span>{time.toLocaleTimeString('en-US', {
              hour12: false,
              hour: '2-digit',
              minute: '2-digit',
              second: '2-digit',
            })}</span>
          </div>
        </div>
      </div>
    </header>
  )
}

export default Header
