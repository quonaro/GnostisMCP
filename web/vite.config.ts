import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [svelte(), tailwindcss()],
  server: {
    port: 7879,
    proxy: {
      '/api': 'http://localhost:7878',
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})
