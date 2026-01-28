#!/bin/bash
set -e

echo "Building frontend..."
if command -v npm &> /dev/null; then
  cd dashboard
  npm install
  npm run build
  cd ..
  echo "Frontend build complete!"
else
  echo "Warning: npm not found, skipping frontend build"
  echo "Frontend should be built separately or via nixpacks.toml"
fi

echo "Building backend..."
go build -o kalshi-signal-feed

echo "Build complete!"
