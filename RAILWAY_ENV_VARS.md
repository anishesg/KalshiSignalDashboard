# Railway Environment Variables Setup

## Required Environment Variables

You **MUST** set these in Railway for the app to work:

### 1. `KALSHI__KALSHI__API_KEY_ID`
- **What it is**: Your Kalshi API key ID
- **Example**: `f035131b-5ccd-48a7-9b15-590786456566`
- **Where to find**: Kalshi API dashboard
- **Required**: Yes - app won't connect to Kalshi without this

### 2. `KALSHI__KALSHI__PRIVATE_KEY_PATH`
- **What it is**: Path to your private key file
- **Default**: `market_signal_bot.txt` (if file is in root)
- **Required**: Yes - needed for API authentication

## Optional Environment Variables

These are optional but useful:

### 3. `KALSHI__ALERTING__SLACK_WEBHOOK_URL`
- **What it is**: Slack webhook URL for sending alerts
- **Required**: No
- **Format**: `https://hooks.slack.com/services/YOUR/WEBHOOK/URL`

### 4. `KALSHI__ALERTING__DISCORD_WEBHOOK_URL`
- **What it is**: Discord webhook URL for sending alerts
- **Required**: No
- **Format**: `https://discord.com/api/webhooks/YOUR/WEBHOOK/URL`

### 5. `KALSHI__API__CORS_ORIGINS`
- **What it is**: Comma-separated list of allowed origins
- **Default**: `*` (allows all origins)
- **Required**: No
- **Example**: `https://your-app.railway.app,https://yourdomain.com`

## How to Set Environment Variables in Railway

1. Go to your Railway project dashboard
2. Click on your service (`kalshi-signal-dashboard`)
3. Go to the **Variables** tab
4. Click **"New Variable"**
5. Add each variable:
   - **Name**: `KALSHI__KALSHI__API_KEY_ID`
   - **Value**: Your actual API key ID
6. Repeat for all required variables

## Private Key Setup Options

You have two ways to handle the private key:

### Option A: Upload as File (Recommended)

1. In Railway, go to your service
2. Click **Settings** → **Files** (or **Volumes**)
3. Upload your `market_signal_bot.txt` file
4. Note the path Railway gives you (e.g., `/app/market_signal_bot.txt`)
5. Set environment variable:
   - **Name**: `KALSHI__KALSHI__PRIVATE_KEY_PATH`
   - **Value**: `/app/market_signal_bot.txt` (or whatever path Railway provides)

### Option B: Use Railway Secrets

1. Copy the contents of your `market_signal_bot.txt` file
2. In Railway Variables, create a new variable:
   - **Name**: `KALSHI__KALSHI__PRIVATE_KEY` (note: different name)
   - **Value**: Paste the entire private key content
3. **Note**: This requires code changes to read from env var instead of file

**Option A is recommended** as it's simpler and doesn't require code changes.

## Automatic Variables

Railway automatically sets:
- `PORT` - The port your app should listen on (Railway sets this automatically)
- The Go server reads this and binds to `0.0.0.0:$PORT`

You don't need to set `PORT` manually.

## Quick Setup Checklist

- [ ] Set `KALSHI__KALSHI__API_KEY_ID` with your API key
- [ ] Upload `market_signal_bot.txt` file to Railway
- [ ] Set `KALSHI__KALSHI__PRIVATE_KEY_PATH` to the file path
- [ ] (Optional) Set `KALSHI__ALERTING__SLACK_WEBHOOK_URL` if using Slack
- [ ] (Optional) Set `KALSHI__ALERTING__DISCORD_WEBHOOK_URL` if using Discord
- [ ] (Optional) Set `KALSHI__API__CORS_ORIGINS` if you need specific CORS settings

## Verification

After setting variables and deploying:
1. Check Railway logs - should see "Configuration loaded"
2. Check Railway logs - should see "Ingestion layer initialized"
3. Visit your Railway URL - should see the dashboard
4. Check `/api/v1/health` endpoint - should return healthy status

If you see errors about missing API key or private key, double-check your environment variables are set correctly.

