# Testing Checklist

This document lists all features to test before deploying to Railway.

## Backend API Endpoints

### Core Endpoints
- [ ] `GET /api/v1/health` - Health check with market count
- [ ] `GET /api/v1/categories` - List all categories
- [ ] `GET /api/v1/markets` - List all markets
- [ ] `GET /api/v1/markets/{ticker}` - Get market details
- [ ] `GET /api/v1/markets/{ticker}/orderbook` - Get orderbook
- [ ] `GET /api/v1/markets/{ticker}/debug` - Get market debug info

### Scanner Endpoints
- [ ] `GET /api/v1/scanner/opportunities` - Get trading opportunities
- [ ] `GET /api/v1/scanner/noarb` - Get no-arb violations

### Signals & Alerts
- [ ] `GET /api/v1/signals` - Get recent signals
- [ ] `GET /api/v1/stream/signals` - Stream signals (SSE)
- [ ] `GET /api/v1/alerts` - Get alerts

## Frontend Features

### Navigation
- [ ] Sidebar navigation works (Explore, Markets, Watchlist, Alerts)
- [ ] Active state highlights correct nav item
- [ ] Markets nav is disabled when no category selected

### Explore Page
- [ ] Categories load and display
- [ ] Category cards show correct information
- [ ] Search filters categories
- [ ] Clicking category navigates to CategoryFeed
- [ ] Pinned categories section (if implemented)

### Category Feed Page
- [ ] Markets/opportunities load
- [ ] List view displays correctly
- [ ] Table view displays correctly
- [ ] View mode toggle works
- [ ] Sort by (liquidity, spread, activity) works
- [ ] Search filters markets
- [ ] Back button returns to Explore
- [ ] Clicking market opens details panel

### Market Details Panel
- [ ] Market information displays
- [ ] Orderbook displays with bids/asks
- [ ] Yes/No labels show correctly
- [ ] Price formatting is correct
- [ ] Watchlist add/remove works
- [ ] Close button works
- [ ] Data refreshes every second

### Watchlist
- [ ] Empty state shows when no items
- [ ] Markets in watchlist display
- [ ] Remove from watchlist works
- [ ] Clicking market opens details

### Alerts Panel
- [ ] Alerts load and display
- [ ] Filter (all/actionable) works
- [ ] Alert information is correct
- [ ] Clicking alert selects market

### Status Bar
- [ ] Uptime displays correctly
- [ ] Market count updates
- [ ] Alert count updates
- [ ] Connection status indicator

### Connection Error
- [ ] Shows when backend is down
- [ ] Instructions are clear
- [ ] Auto-retries connection

## Integration Tests

### User Flow 1: Browse Markets
1. Open app → Explore page loads
2. Click a category → CategoryFeed loads
3. View markets in list/table
4. Click a market → Details panel opens
5. View orderbook and market info
6. Add to watchlist
7. Close details panel

### User Flow 2: Watchlist Management
1. Add markets to watchlist from details panel
2. Navigate to Watchlist page
3. View watchlist markets
4. Remove market from watchlist
5. Verify removal

### User Flow 3: Alerts
1. Navigate to Alerts page
2. View all alerts
3. Filter to actionable alerts
4. Click alert to view market

### User Flow 4: Search & Filter
1. On Explore page, search for category
2. On CategoryFeed, search for market
3. Change sort order
4. Switch view modes

## Error Handling

- [ ] Backend down shows connection error
- [ ] API errors are handled gracefully
- [ ] Missing data shows appropriate messages
- [ ] Network errors don't crash app

## Performance

- [ ] Page loads quickly
- [ ] Data updates smoothly
- [ ] No console errors
- [ ] No memory leaks (check intervals cleanup)

## Build & Deploy

- [ ] Frontend builds without errors
- [ ] TypeScript compiles successfully
- [ ] No linting errors
- [ ] Build output in `dashboard/dist/`
- [ ] Go backend compiles
- [ ] Static files served correctly

