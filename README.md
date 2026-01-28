# Kalshi Signal Dashboard

A real-time dashboard for monitoring Kalshi prediction markets. The system ingests market data from the Kalshi API, processes trading signals, and displays them in a modern web interface.

## What It Does

The backend connects to Kalshi's API and websocket to pull market data, orderbook information, and trades. It processes this data to generate trading signals like price drift, orderbook imbalance, and volume surges. The frontend dashboard lets you browse markets by category, view orderbooks, set up alerts, and track your watchlist.

## Architecture

The system has two main parts:

- **Backend (Go)**: Handles API authentication, data ingestion, signal processing, and serves a REST API
- **Frontend (React/TypeScript)**: A dashboard for exploring markets, viewing details, and managing alerts

## Setup

### Backend

You need Go 1.21 or later. Set your Kalshi API credentials as environment variables:

```
export KALSHI__KALSHI__API_KEY_ID="your-api-key-id"
export KALSHI__KALSHI__PRIVATE_KEY_PATH="path/to/your/private/key.txt"
```

Optionally set alerting webhooks:

```
export KALSHI__ALERTING__SLACK_WEBHOOK_URL="your-slack-webhook"
export KALSHI__ALERTING__DISCORD_WEBHOOK_URL="your-discord-webhook"
```

Then run:

```
go run main.go
```

The API server starts on port 8080 by default. You can change this in `config/default.toml`.

### Frontend

Navigate to the dashboard directory and install dependencies:

```
cd dashboard
npm install
```

Start the development server:

```
npm run dev
```

The dashboard will be available at http://localhost:3000 (or whatever port Vite assigns).

For production, build the frontend:

```
npm run build
```

The built files will be in `dashboard/dist/`.

## Configuration

Edit `config/default.toml` to adjust settings like:
- API endpoints
- WebSocket reconnect delays
- Signal computation intervals
- Alert cooldown periods
- CORS origins

Local overrides can go in `config/local.toml` (this file is gitignored).

## Features

- Real-time market data ingestion via REST and WebSocket
- Signal detection for price movements, orderbook imbalances, and volume spikes
- Category-based market browsing
- Orderbook visualization with Yes/No labels
- Alert system with Slack and Discord integration
- Watchlist functionality
- Market detail views with historical data

## API Endpoints

The backend exposes these endpoints:

- `GET /api/v1/health` - Health check
- `GET /api/v1/markets` - List all markets
- `GET /api/v1/markets/{ticker}` - Get market details
- `GET /api/v1/markets/{ticker}/orderbook` - Get orderbook
- `GET /api/v1/categories` - List categories
- `GET /api/v1/signals` - Get recent signals
- `GET /api/v1/alerts` - Get alerts
- `GET /api/v1/stream/signals` - Stream signals via Server-Sent Events

## License

This project is private and not licensed for public use.

