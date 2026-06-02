# Cozy Canvas - Frontend

This directory contains the frontend part of the **Cozy Canvas** monorepo, which is an ultra-modern and responsive interactive infinite canvas built with HTML5, CSS3, and D3.js.

## Local Run (Vite)

1. Install dependencies:
   ```bash
   npm install
   ```

2. Start the dev server:
   ```bash
   npm run dev
   ```

   The frontend will automatically proxy API requests to the backend at `http://localhost:8080` for all paths starting with `/api`.

## Deploy to Vercel 🚀

The interactive canvas is designed to be easily hosted as a fully static site with dynamic requests to the Go API (database server).

### Step 1: Prepare for Import to Vercel

You can import the entire monorepo directly from your GitHub. During the import process, you will need to specify the **Root Directory** for your frontend.

### Step 2: Project Settings in Vercel Dashboard

When creating a new project on Vercel, specify the following parameters:

1. **Framework Preset**: Select **Vite** or **Other / Vanilla JS**.
2. **Root Directory**: Specify `website` (this is the frontend folder in the monorepo).
3. **Build Command**: `npm run build`
4. **Output Directory**: `dist`

### Step 3: Configure Proxying on Vercel (vercel.json)

To have `/api/...` requests automatically directed to the remote Go backend, create a `vercel.json` file in this directory (`website/`) with the following content:

```json
{
  "rewrites": [
    {
      "source": "/api/(.*)",
      "destination": "https://your-go-backend-api.com/api/$1"
    }
  ]
}
```

Replace `https://your-go-backend-api.com` with the actual URL of your deployed Go backend.
After this, any calls from your Canvas to `/api/...` will be transparently forwarded to your backend without CORS restrictions!
