# Deployment Guide

This guide explains how to deploy the Kalshi Signal Dashboard to Railway or Render. Both platforms can host the entire application (frontend + backend) in a single service.

## Architecture

The application is deployed as a single service:
- **Go backend** serves the API and static frontend files
- **Frontend** is built during deployment and served by the Go server
- Everything runs on one port

## Option 1: Railway

### Step 1: Create Railway Project

1. Go to [railway.app](https://railway.app) and sign up/login
2. Click "New Project"
3. Select "Deploy from GitHub repo"
4. Connect your GitHub account and select the `KalshiSignalDashboard` repository

### Step 2: Configure Build

Railway will auto-detect Go, but we need to set the build command:

1. In your Railway project, go to Settings
2. Under "Build", set:
   - **Build Command**: `./build.sh`
   - **Start Command**: `./kalshi-signal-feed`

Or add a `Procfile`:
```
web: ./kalshi-signal-feed
```

### Step 3: Set Environment Variables

In Railway, go to Variables and add:

- `KALSHI__KALSHI__API_KEY_ID` - Your Kalshi API key ID
- `KALSHI__KALSHI__PRIVATE_KEY_PATH` - Path to private key (or use Railway's file storage)
- `KALSHI__ALERTING__SLACK_WEBHOOK_URL` (optional)
- `KALSHI__ALERTING__DISCORD_WEBHOOK_URL` (optional)

For the private key, you can:
- Upload it as a file in Railway and reference the path
- Or set the key content directly as an environment variable and write it to a file in the build script

### Step 4: Deploy

Railway will automatically deploy when you push to your main branch. The first deployment will:
1. Install Node.js dependencies
2. Build the React frontend
3. Build the Go backend
4. Start the server

Your app will be available at `https://your-app-name.up.railway.app`

## Option 2: Render

### Step 1: Create Render Service

1. Go to [render.com](https://render.com) and sign up/login
2. Click "New +" → "Web Service"
3. Connect your GitHub repository
4. Select the `KalshiSignalDashboard` repository

### Step 2: Configure Service

Render will use the `render.yaml` file, but you can also configure manually:

- **Environment**: Go
- **Build Command**: `./build.sh`
- **Start Command**: `./kalshi-signal-feed`

### Step 3: Set Environment Variables

In Render dashboard, go to Environment and add:

- `KALSHI__KALSHI__API_KEY_ID`
- `KALSHI__KALSHI__PRIVATE_KEY_PATH`
- `KALSHI__ALERTING__SLACK_WEBHOOK_URL` (optional)
- `KALSHI__ALERTING__DISCORD_WEBHOOK_URL` (optional)

### Step 4: Deploy

Click "Create Web Service". Render will:
1. Run the build script
2. Deploy your service
3. Give you a URL like `https://kalshi-signal-dashboard.onrender.com`

## Private Key Setup

Both platforms need access to your Kalshi private key. Options:

### Option A: Upload as Secret File

1. Upload your `market_signal_bot.txt` file to the platform's file storage
2. Set `KALSHI__KALSHI__PRIVATE_KEY_PATH` to the file path

### Option B: Environment Variable

1. Base64 encode your private key: `cat market_signal_bot.txt | base64`
2. Set environment variable: `KALSHI_PRIVATE_KEY_B64` with the encoded value
3. Update `build.sh` to decode and write the file:

```bash
if [ -n "$KALSHI_PRIVATE_KEY_B64" ]; then
  echo "$KALSHI_PRIVATE_KEY_B64" | base64 -d > market_signal_bot.txt
fi
```

## Troubleshooting

### Build Fails

- Check that Node.js is available (Railway/Render should auto-detect)
- Verify `build.sh` is executable: `chmod +x build.sh`
- Check build logs for specific errors

### Frontend Not Loading

- Verify `dashboard/dist` directory exists after build
- Check that the Go server is serving static files correctly
- Look at server logs for routing issues

### API Not Working

- Verify environment variables are set correctly
- Check that the private key file is accessible
- Review CORS settings if making requests from different domains

### Port Issues

- Railway and Render set the `PORT` environment variable automatically
- The Go server reads this and binds to `0.0.0.0:$PORT`
- No manual port configuration needed

## Updating

Both platforms support automatic deployments:
- Push to your main branch
- Platform detects changes
- Automatically rebuilds and redeploys

You can also trigger manual deployments from the platform dashboard.

