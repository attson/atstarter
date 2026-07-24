---
layout: home
hero:
  name: AT Starter
  text: 本地项目快速启动器
  tagline: 一处托管启动/停止多个项目,查看实时日志。读取本地目录代码,自动识别项目类型并建议启动命令。
  actions:
    - theme: brand
      text: 下载最新版
      link: https://github.com/attson/atstarter/releases/latest
    - theme: alt
      text: 使用文档
      link: /guide/
features:
  - title: 批量扫描工作区
    details: 指定工作区根目录(支持 ~),扫描直接子目录,识别项目类型并勾选批量加入。
  - title: 自动识别 + 手动兜底
    details: 内置规则识别 node / Go / Rust / Python / docker compose,建议启动命令,可随时手动修改。
  - title: 多套命令 / 启动分组
    details: 每个项目保存多条命令(default / debug / …),把多个项目编成一组一键批量启停。
  - title: 进程托管 + 实时日志
    details: App 内启动/停止子进程,实时展示 stdout/stderr,进程退出追加退出码标记。
  - title: Docker / compose 管理
    details: compose 项目融入项目树,支持整体与单 service 启停;独立容器面板管理宿主机容器。
  - title: 登录 shell 启动
    details: 子进程经登录交互式 shell 启动,拿到完整 PATH,修复 pnpm / nvm / go command not found。
  - title: 系统托盘 + 自更新
    details: 关闭窗口隐藏到托盘;轮询 GitHub Release,Ed25519 签名 + SHA256 校验后自安装,内置下载加速镜像。
  - title: 明暗主题
    details: 内置浅色 / 深色主题切换。
---

<section class="home-section">

## 功能截图

<p class="section-sub">看看 AT Starter 实际长什么样。</p>

<div class="shot">
  <div class="shot-text">
    <h3>一处托管,实时日志</h3>
    <p>左侧项目树集中管理所有项目,选中即看实时 stdout / stderr,顶部状态栏汇总运行 / 退出数。</p>
  </div>
  <a class="shot-frame" href="/shot-overview.png" target="_blank" rel="noreferrer">
    <img src="/shot-overview.png" alt="AT Starter 主界面:项目树 + 实时日志" loading="lazy" />
  </a>
</div>

<div class="shot">
  <div class="shot-text">
    <h3>项目文件浏览</h3>
    <p>内置文件树 + 只读代码预览,快速看一眼 main.go 或配置文件,不用切到编辑器。</p>
  </div>
  <a class="shot-frame" href="/shot-files.png" target="_blank" rel="noreferrer">
    <img src="/shot-files.png" alt="AT Starter 文件浏览器:文件树 + 代码预览" loading="lazy" />
  </a>
</div>

<div class="shot">
  <div class="shot-text">
    <h3>启动分组,一键启停</h3>
    <p>把「前端 + 后端」编成一组,一次性全部启动或停止,不用逐个点。</p>
  </div>
  <a class="shot-frame" href="/shot-group.png" target="_blank" rel="noreferrer">
    <img src="/shot-group.png" alt="AT Starter 启动分组详情:一键 Start / Stop 整组" loading="lazy" />
  </a>
</div>

<div class="shot">
  <div class="shot-text">
    <h3>明暗主题</h3>
    <p>内置深色主题,夜间盯日志也护眼。</p>
  </div>
  <a class="shot-frame" href="/shot-dark.png" target="_blank" rel="noreferrer">
    <img src="/shot-dark.png" alt="AT Starter 暗色主题下的主界面与日志" loading="lazy" />
  </a>
</div>

</section>

<section class="home-section">

## 快速上手 3 步

<div class="steps">
  <div class="step-card">
    <div class="step-num">1</div>
    <h3>扫描工作区</h3>
    <p>指定工作区根目录(支持 ~),扫描直接子目录,自动识别项目类型并勾选批量加入。</p>
  </div>
  <div class="step-card">
    <div class="step-num">2</div>
    <h3>选中项目启动</h3>
    <p>在项目树里选中目标项目,点 Start 即以登录 shell 启动,拿到完整 PATH。</p>
  </div>
  <div class="step-card">
    <div class="step-num">3</div>
    <h3>查看日志 / 文件</h3>
    <p>实时查看 stdout / stderr,或切到「文件」tab 浏览项目文件与代码预览。</p>
  </div>
</div>

</section>

<section class="home-section">

## 适用场景

<div class="scenarios">
  <div class="scenario-card">
    <p class="pain">多项目来回切终端、记不住每个项目的启动命令</p>
    <p class="solve">一处保存每个项目的多套命令,一键启动。</p>
  </div>
  <div class="scenario-card">
    <p class="pain">从桌面 / IDE 启动子进程报 pnpm / go command not found</p>
    <p class="solve">经登录交互式 shell 启动,拿到完整 PATH,修复 nvm / pnpm / go 找不到。</p>
  </div>
  <div class="scenario-card">
    <p class="pain">前后端要一起起,逐个开终端太麻烦</p>
    <p class="solve">编成启动分组,一键全启 / 全停。</p>
  </div>
  <div class="scenario-card">
    <p class="pain">compose 项目 + 独立容器混在一起管不过来</p>
    <p class="solve">内置 Docker / compose 面板,项目树里整体或单 service 启停。</p>
  </div>
</div>

</section>

<section class="home-section">

## 下载

<div class="downloads">
  <a class="download-card" href="https://github.com/attson/atstarter/releases/latest" target="_blank" rel="noreferrer">
    <div class="os">macOS</div>
    <div class="os-sub">.dmg · Intel / Apple Silicon</div>
  </a>
  <a class="download-card" href="https://github.com/attson/atstarter/releases/latest" target="_blank" rel="noreferrer">
    <div class="os">Linux</div>
    <div class="os-sub">AppImage / 二进制</div>
  </a>
  <a class="download-card" href="https://github.com/attson/atstarter/releases/latest" target="_blank" rel="noreferrer">
    <div class="os">Windows</div>
    <div class="os-sub">.exe 安装包</div>
  </a>
</div>

<p class="download-note">AT Starter 支持 macOS / Linux / Windows,均前往 <a href="https://github.com/attson/atstarter/releases/latest" target="_blank" rel="noreferrer">GitHub Releases</a> 下载最新版。</p>

</section>
