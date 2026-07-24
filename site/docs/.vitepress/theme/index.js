import DefaultTheme from 'vitepress/theme'
import { useRoute } from 'vitepress'
import { watch, nextTick, onMounted } from 'vue'
import './custom.css'

// 滚动入场:给进入视口的 .reveal 元素加 .in-view,触发 CSS 过渡。
// SSR 安全:所有 DOM/IO 逻辑仅在 onMounted(仅浏览器执行)内运行。
function useRevealOnScroll() {
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
    const route = useRoute()
    watch(
      () => route.path,
      () => nextTick(setup)
    )
  })
}

export default {
  extends: DefaultTheme,
  setup() {
    useRevealOnScroll()
  },
}
