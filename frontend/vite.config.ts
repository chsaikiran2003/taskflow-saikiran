import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// In Docker, the frontend container proxies /api → backend:8080 via nginx.
// For local dev (npm run dev), proxy to localhost:8080.
const apiTarget = process.env.VITE_API_URL ?? 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: apiTarget,
        rewrite: (path) => path.replace(/^\/api/, ''),
        changeOrigin: true,
      },
    },
  },
})
