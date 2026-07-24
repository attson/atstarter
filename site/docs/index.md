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
