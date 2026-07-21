import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  // 用冷门端口,避开业务项目普遍占用的 5173(如 team-manage/dev.sh)。
  // 否则被启动项目的 vite 抢占 5173 后,其清理脚本的 kill_port 5173 会连带
  // 杀掉 atstarter 自己的 dev server,导致 wails 窗口关闭。
  // strictPort:端口被占直接失败,不静默 +1 漂移(漂移会让 wails 连不上)。
  server: {
    port: 9245,
    strictPort: true,
  },
})
