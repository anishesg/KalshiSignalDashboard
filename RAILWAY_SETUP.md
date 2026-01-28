# Railway Deployment Setup

This guide explains how to deploy the Kalshi Signal Dashboard to Railway. Railway will host both the frontend and backend together.

## How It Works

The Go backend serves both:
- API endpoints at `/api/v1/*`
- Static frontend files from `dashboard/dist/`

Railway will build both the frontend (React) and backend (Go), then run the Go server which serves everything.

## Step 1: Create Railway Account

1. Go to [railway.app](https://railway.app) and sign up
2. Connect your GitHub account

## Step 2: Create New Project

1. Click "New Project"
2. Select "Deploy from GitHub repo"
3. Choose your `KalshiSignalDashboard` repository
4. Railway will auto-detect it's a Go project

## Step 3: Configure Build Settings

Railway should auto-detect the setup, but verify these settings:

**Build Command:**
```bash
cd dashboard && npm install && npm run build && cd .. && go build -o kalshi-signal-feed
```

Or Railway will use the `build.sh` script automatically.

**Start Command:**
```bash
./kalshi-signal-feed
```

## Step 4: Set Environment Variables

In Railway, go to your project → Variables tab and add:

### Required:
- `KALSHI__KALSHI__API_KEY_ID` - Your Kalshi API key ID
- `KALSHI__KALSHI__PRIVATE_KEY_PATH` - Path to your private key file (or use Railway's file storage)

### Optional:
- `KALSHI__ALERTING__SLACK_WEBHOOK_URL` - Slack webhook for alerts
- `KALSHI__ALERTING__DISCORD_WEBHOOK_URL` - Discord webhook for alerts
- `KALSHI__API__CORS_ORIGINS` - Comma-separated list of allowed origins (defaults to "*")

### Private Key Setup

You have two options for the private key:

**Option A: Upload as File**
1. In Railway, go to your service
2. Click "Settings" → "Files"
3. Upload your `market_signal_bot.txt` file
4. Set `KALSHI__KALSHI__PRIVATE_KEY_PATH` to the file path Railway provides

**Option B: Use Environment Variable**
1. Read your private key file: `cat market_signal_bot.txt`
2. In Railway Variables, add:
   - `KALSHI__KALSHI__PRIVATE_KEY` - Paste the entire key content
3. You'll need to update the code to read from this env var instead of file path

## Step 5: Deploy

1. Railway will automatically deploy when you push to your main branch
2. Or click "Deploy" in the Railway dashboard
3. Wait for the build to complete

## Step 6: Get Your URL

1. Once deployed, Railway will give you a URL like `https://your-app.railway.app`
2. Your app is now live at that URL
3. The frontend and API are both served from the same domain

## How It Works

- Frontend is built to `dashboard/dist/`
- Go backend serves static files from `dashboard/dist/` for all non-API routes
- API routes are handled at `/api/v1/*`
- The Go server automatically handles SPA routing (serves index.html for non-API routes)

## Troubleshooting

### Build Fails
- Check that Node.js and Go are available in Railway's build environment
- Railway should auto-detect both, but you can specify in `nixpacks.toml` if needed

### Frontend Not Loading
- Make sure `dashboard/dist/` exists after build
- Check that the Go server is serving from the correct path

### API Not Working
- Check that environment variables are set correctly
- Verify the backend logs in Railway dashboard
- Make sure CORS is configured if accessing from different domain

### Private Key Issues
- Verify the file path is correct
- Check file permissions
- Try using the environment variable method instead

## Custom Domain

1. In Railway, go to your service → Settings → Domains
2. Add your custom domain
3. Railway will provide DNS instructions
4. Update CORS origins if needed

## Monitoring

- View logs in Railway dashboard
- Check build logs if deployment fails
- Monitor resource usage in Railway dashboard

