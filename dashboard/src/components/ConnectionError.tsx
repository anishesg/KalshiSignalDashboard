import { AlertTriangle } from 'lucide-react'
import './ConnectionError.css'

function ConnectionError() {
  return (
    <div className="connection-error">
      <div className="error-content">
        <AlertTriangle size={48} className="error-icon" />
        <h2 className="error-title">Backend Not Connected</h2>
        <p className="error-message">
          The Go backend API server is not running.
        </p>
        <div className="error-instructions">
          <p>To start the backend:</p>
          <code className="error-code">
            cd /Users/anish/Desktop/kalshi_api_bot<br />
            export KALSHI__KALSHI__API_KEY_ID="your-api-key-id"<br />
            go run main.go
          </code>
          <p className="error-note">
            Make sure the backend is running on http://localhost:8080
          </p>
        </div>
      </div>
    </div>
  )
}

export default ConnectionError

