#!/bin/bash

echo "🚀 Deploying DGI Service to Railway..."

# Check if Railway CLI is installed
if ! command -v railway &> /dev/null; then
    echo "❌ Railway CLI not found. Installing..."
    npm install -g @railway/cli
fi

# Login to Railway
echo "🔐 Logging in to Railway..."
railway login

# Link to project (if not already linked)
echo "🔗 Linking to Railway project..."
railway link

# Deploy
echo "📦 Deploying to Railway..."
railway up

echo "✅ Deployment completed!"
echo "🌐 Your service should be available at: https://your-app-name.railway.app"
echo ""
echo "📋 Next steps:"
echo "1. Configure environment variables in Railway Dashboard"
echo "2. Update SERVER_BASE_URL to your Railway URL"
echo "3. Test the health endpoint: https://your-app-name.railway.app/health"
