import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

// Demo development config
export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0', // 监听所有网卡，允许局域网 IP 访问
    port: 3000,
    open: true,
    proxy: {
      '/captcha': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  }
});

