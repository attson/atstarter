import DefaultTheme from 'vitepress/theme'
import { useRoute } from 'vitepress'
import { watch, nextTick, onMounted, onUnmounted } from 'vue'
import './custom.css'

// 首页整页深色化:仅在「首页」给 <body> 挂 `home-dark` class,用于给 .VPHome
// 之外的导航栏(.VPNav)上色;离开首页(进 /guide/)时立即移除,文档页导航栏
// 恢复默认。hero / features / 自定义 section 本身在 .VPHome 后代,由 CSS 直接
// 覆盖,不依赖这个 class。
// 判定首页用 DOM 探测 .VPHome 是否存在(VitePress 仅在 home layout 渲染它),
// 避免 route.path 因 base 前缀(/atstarter/)导致的字符串比较不稳。DOM 操作
// 延后到 onMounted(浏览器)执行,保持 SSR 安全;route 仅用于触发路由切换重扫。
function useHomeDarkBodyClass(route) {
  if (typeof window === 'undefined') return

  const sync = () => {
    const isHome = !!document.querySelector('.VPHome')
    document.body.classList.toggle('home-dark', isHome)
  }

  onMounted(() => {
    nextTick(sync)
    watch(() => route.path, () => nextTick(sync))
  })

  onUnmounted(() => {
    if (typeof document !== 'undefined') document.body.classList.remove('home-dark')
  })
}

// 滚动入场:给进入视口的 .reveal 元素加 .in-view,触发 CSS 过渡。
// SSR 安全:所有 DOM/IO 逻辑仅在 onMounted(仅浏览器执行)内运行。
function useRevealOnScroll(route) {
  if (typeof window === 'undefined') return

  const setup = () => {
    const els = document.querySelectorAll('.tech-home .reveal:not(.in-view)')
    if (!els.length) return

    // 无 IntersectionObserver 时直接全部显示,避免内容永久隐藏。
    if (!('IntersectionObserver' in window)) {
      els.forEach((el) => el.classList.add('in-view'))
      return
    }

    const io = new IntersectionObserver(
      (entries, obs) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            entry.target.classList.add('in-view')
            obs.unobserve(entry.target)
          }
        })
      },
      { rootMargin: '0px 0px -8% 0px', threshold: 0.12 }
    )

    els.forEach((el) => io.observe(el))
  }

  onMounted(() => {
    nextTick(setup)
    // 路由切换(SPA 内跳回首页)时重新扫描。
    watch(
      () => route.path,
      () => nextTick(setup)
    )
  })
}

export default {
  extends: DefaultTheme,
  setup() {
    // useRoute() 在 setup 同步调用,再传入各 composable。
    const route = useRoute()
    useRevealOnScroll(route)
    useHomeDarkBodyClass(route)
  },
}
