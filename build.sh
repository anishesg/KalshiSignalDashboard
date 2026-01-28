#!/bin/bash
set -e

echo "Building frontend..."
cd dashboard
npm install
npm run build
cd ..

echo "Building backend..."
go build -o kalshi-signal-feed

echo "Build complete!"
