---
layout: home
hero:
  name: AT Starter
  text: 本地项目快速启动器
  tagline: 一处托管启动/停止多个项目,查看实时日志。读取本地目录代码,自动识别项目类型,也可通过 CLI/MCP 交给 AI 操作。
  actions:
    - theme: brand
      text: 下载最新版
      link: https://github.com/attson/atstarter/releases/latest
    - theme: alt
      text: 使用文档
      link: /guide/
    - theme: alt
      text: AI / CLI
      link: /guide/ai-cli
---

<script setup>
import HomeDemo from './.vitepress/theme/components/HomeDemo.vue'
</script>

<div class="tech-home">

<HomeDemo />

<section class="badge-strip reveal">
  <div class="badge-wrap">
    <div class="badge"><span class="badge-k">6+</span><span class="badge-v">项目类型识别</span></div>
    <div class="badge"><span class="badge-k">3</span><span class="badge-v">平台 · macOS / Linux / Windows</span></div>
    <div class="badge"><span class="badge-k">Ed25519</span><span class="badge-v">签名自更新</span></div>
    <div class="badge"><span class="badge-k">MCP</span><span class="badge-v">AI / CLI 本地控制</span></div>
    <div class="badge"><span class="badge-k">login shell</span><span class="badge-v">完整 PATH 启动</span></div>
    <div class="badge"><span class="badge-k">MIT</span><span class="badge-v">开源免费</span></div>
  </div>
</section>

<section class="home-section">

<div class="glow glow-a"></div>

## 功能截图

<p class="section-sub reveal">看看 AT Starter 实际长什么样。</p>

<div class="shot reveal">
  <div class="shot-text">
    <h3>一处托管,实时日志</h3>
    <p>左侧项目树集中管理所有项目和启动分组,选中即看实时 stdout / stderr,顶部状态栏汇总运行 / 退出数。</p>
  </div>
  <div class="win-frame">
    <div class="win-bar"><span class="dot dot-r"></span><span class="dot dot-y"></span><span class="dot dot-g"></span></div>
    <img src="/shot-overview.png" alt="AT Starter 主界面:项目树 + 实时日志" loading="lazy" />
  </div>
</div>

<div class="shot reveal">
  <div class="shot-text">
    <h3>项目文件浏览</h3>
    <p>内置懒加载文件树 + CodeMirror 代码编辑,支持语法高亮、保存、目录操作和多类型预览。</p>
  </div>
  <div class="win-frame">
    <div class="win-bar"><span class="dot dot-r"></span><span class="dot dot-y"></span><span class="dot dot-g"></span></div>
    <img src="/shot-files.png" alt="AT Starter 文件浏览器:文件树 + 代码预览" loading="lazy" />
  </div>
</div>

<div class="shot reveal">
  <div class="shot-text">
    <h3>启动分组,一键启停</h3>
    <p>把「前端 + 后端」编成一组,一次性全部启动或停止,不用逐个点。</p>
  </div>
  <div class="win-frame">
    <div class="win-bar"><span class="dot dot-r"></span><span class="dot dot-y"></span><span class="dot dot-g"></span></div>
    <img src="/shot-group.png" alt="AT Starter 启动分组详情:一键 Start / Stop 整组" loading="lazy" />
  </div>
</div>

<div class="shot reveal">
  <div class="shot-text">
    <h3>明暗主题</h3>
    <p>内置深色主题,日志、项目树和文件编辑器都会跟随主题切换。</p>
  </div>
  <div class="win-frame">
    <div class="win-bar"><span class="dot dot-r"></span><span class="dot dot-y"></span><span class="dot dot-g"></span></div>
    <img src="/shot-dark.png" alt="AT Starter 暗色主题下的主界面与日志" loading="lazy" />
  </div>
</div>

</section>

<section class="home-section">

<div class="glow glow-b"></div>

## 快速上手 3 步

<div class="steps">
  <div class="step-card reveal">
    <div class="step-num">1</div>
    <h3>扫描工作区</h3>
    <p>指定工作区根目录(支持 ~),扫描直接子目录与常见 worktree 目录,自动识别项目类型并批量加入。</p>
  </div>
  <div class="step-card reveal">
    <div class="step-num">2</div>
    <h3>选中项目启动</h3>
    <p>在项目树里选中目标项目,点 Start 即以登录 shell 启动,拿到完整 PATH。</p>
  </div>
  <div class="step-card reveal">
    <div class="step-num">3</div>
    <h3>查看日志 / 文件</h3>
    <p>实时查看 stdout / stderr,或切到「文件」tab 浏览、预览、编辑项目文件。</p>
  </div>
</div>

</section>

<section class="home-section">

## 适用场景

<div class="scenarios">
  <div class="scenario-card reveal">
    <p class="pain">多项目来回切终端、记不住每个项目的启动命令</p>
    <p class="solve">一处保存每个项目的多套命令,一键启动。</p>
  </div>
  <div class="scenario-card reveal">
    <p class="pain">从桌面 / IDE 启动子进程报 pnpm / go command not found</p>
    <p class="solve">经登录交互式 shell 启动,拿到完整 PATH,修复 nvm / pnpm / go 找不到。</p>
  </div>
  <div class="scenario-card reveal">
    <p class="pain">前后端要一起起,逐个开终端太麻烦</p>
    <p class="solve">编成启动分组,一键全启 / 全停。</p>
  </div>
  <div class="scenario-card reveal">
    <p class="pain">compose 项目 + 独立容器混在一起管不过来</p>
    <p class="solve">内置 Docker / compose 面板,项目树里整体或单 service 启停。</p>
  </div>
</div>

</section>

<section class="home-section">

## AI / CLI 控制

<div class="scenarios">
  <div class="scenario-card reveal">
    <p class="pain">想让 AI 扫描项目、建分组、启动服务并读日志</p>
    <p class="solve">通过 <code>atstarter mcp</code> 暴露本地工具,AI 调用的仍是桌面 App 的运行态。</p>
  </div>
  <div class="scenario-card reveal">
    <p class="pain">需要在脚本或终端里复用页面上的启动能力</p>
    <p class="solve"><code>atstarter cli</code> 输出固定 JSON,支持 scan、project、group、Docker、compose 和 logs。</p>
  </div>
  <div class="scenario-card reveal">
    <p class="pain">桌面 App 还没启动,AI 无法操作</p>
    <p class="solve"><code>atstarter cli app start --wait</code> 可启动桌面进程并等待控制服务就绪。</p>
  </div>
</div>

<p class="download-note">完整命令见 <a href="/guide/ai-cli">AI / CLI 控制文档</a>。</p>

</section>

<section class="home-section">

<div class="glow glow-c"></div>

## 下载

<div class="downloads">
  <a class="download-card reveal" href="https://github.com/attson/atstarter/releases/latest" target="_blank" rel="noreferrer">
    <div class="os">macOS</div>
    <div class="os-sub">.dmg / .zip · Intel / Apple Silicon</div>
  </a>
  <a class="download-card reveal" href="https://github.com/attson/atstarter/releases/latest" target="_blank" rel="noreferrer">
    <div class="os">Linux</div>
    <div class="os-sub">.deb / .tar.gz · amd64 / arm64</div>
  </a>
  <a class="download-card reveal" href="https://github.com/attson/atstarter/releases/latest" target="_blank" rel="noreferrer">
    <div class="os">Windows</div>
    <div class="os-sub">NSIS .exe / .zip · amd64</div>
  </a>
</div>

<p class="download-note">AT Starter 支持 macOS / Linux / Windows,均前往 <a href="https://github.com/attson/atstarter/releases/latest" target="_blank" rel="noreferrer">GitHub Releases</a> 下载最新版。</p>

</section>
</div>
