//go:build !darwin

package main

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/energye/systray"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// quitRequested 标记用户已从托盘主动选择「退出」。beforeClose 据此区分
// 退出来源:窗口关闭(X)应隐藏到托盘,而主动退出必须放行。没有这个标志,
// runtime.Quit 也会触发 OnBeforeClose 并被隐藏逻辑拦截,导致永远退不出。
var quitRequested atomic.Bool

var (
	trayApp     *App
	trayReady   bool
	trayMu      sync.Mutex // 保护 trayReady / trayVisible 与菜单文案更新
	trayVisible = true     // 主窗口初始可见

	// miRunning / miToggle 会被其他 goroutine 读取(如 runner 状态回调),
	// 所有访问必须持 trayMu。miStopAll / miQuit 仅在 trayOnReady 内使用,
	// 已降级为局部变量。
	miRunning *systray.MenuItem
	miToggle  *systray.MenuItem
)

// startTray 在独立 goroutine 里启动系统托盘。托盘是增强能力:
// 若初始化 panic(无托盘环境),recover 后降级,应用照常运行,
// trayReady 保持 false,OnBeforeClose 据此放行正常退出。
func startTray(a *App) {
	trayApp = a
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if a.ctx != nil {
					runtime.LogErrorf(a.ctx, "system tray failed to start: %v", r)
				} else {
					fmt.Fprintf(os.Stderr, "system tray failed to start: %v\n", r)
				}
			}
		}()
		systray.Run(trayOnReady, trayOnExit)
	}()
}

func traySupported() bool {
	return true
}

func trayOnReady() {
	systray.SetIcon(trayIcon)
	systray.SetTitle("")
	systray.SetTooltip("AT Starter")

	// 菜单项指针会被其他 goroutine 读取(miRunning/miToggle),因此对它们的
	// 写入、点击回调绑定与 trayReady 置位统一在 trayMu 下完成,以此对 updateTrayRunning
	// / refreshToggleLabelLocked 等读取方建立 happens-before 边界。
	// updateTrayRunning 自身也会加锁,故初始化调用放在解锁之后,避免重入死锁。
	trayMu.Lock()
	miRunning = systray.AddMenuItem("运行中: 0 个", "当前运行中的项目数")
	miRunning.Disable()
	systray.AddSeparator()
	miToggle = systray.AddMenuItem("隐藏窗口", "显示或隐藏主窗口")
	miStopAll := systray.AddMenuItem("停止全部项目", "停止所有运行中的项目")
	systray.AddSeparator()
	miQuit := systray.AddMenuItem("退出", "停止全部项目并退出应用")

	// 菜单点击用回调模式(energye/systray v1.0.3 的 MenuItem.Click,
	// 不是上游 getlantern 的 ClickedCh channel)。
	miToggle.Click(func() { toggleWindow() })
	miStopAll.Click(func() { trayApp.runner.StopAll() })
	miQuit.Click(func() {
		quitRequested.Store(true) // 让 beforeClose 放行,而非隐藏
		runtime.Quit(trayApp.ctx) // 触发 OnShutdown 停全部进程
	})

	// 左键单击图标 → 切换窗口(部分 Linux 桌面不触发,菜单项作兜底)
	systray.SetOnClick(func(menu systray.IMenu) { toggleWindow() })

	trayReady = true
	trayMu.Unlock()

	// 初始化运行数(updateTrayRunning 内部会加锁,必须在解锁后调用)
	updateTrayRunning(trayApp.runner.RunningCount())
}

func trayOnExit() {}

// toggleWindow 根据当前可见性显示或隐藏主窗口,并翻转标志、更新菜单文案。
func toggleWindow() {
	trayMu.Lock()
	defer trayMu.Unlock()
	if trayVisible {
		runtime.WindowHide(trayApp.ctx)
		trayVisible = false
	} else {
		runtime.WindowShow(trayApp.ctx)
		runtime.WindowUnminimise(trayApp.ctx)
		trayVisible = true
	}
	refreshToggleLabelLocked()
}

// setTrayWindowVisible 供 OnBeforeClose 调用,同步隐藏状态。
func setTrayWindowVisible(v bool) {
	trayMu.Lock()
	defer trayMu.Unlock()
	trayVisible = v
	refreshToggleLabelLocked()
}

// refreshToggleLabelLocked 更新切换菜单项文案。调用者须已持 trayMu。
func refreshToggleLabelLocked() {
	if miToggle == nil {
		return
	}
	if trayVisible {
		miToggle.SetTitle("隐藏窗口")
	} else {
		miToggle.SetTitle("显示窗口")
	}
}

// updateTrayRunning 更新运行数菜单项标题与 tooltip。
// 可能从 runner 状态回调所在 goroutine 调用,故加 trayMu 保护对 miRunning 的读取。
// systray.SetTooltip / miRunning.SetTitle 内部各自加锁,在本互斥下调用是安全的。
func updateTrayRunning(n int) {
	trayMu.Lock()
	defer trayMu.Unlock()
	if miRunning == nil {
		return
	}
	miRunning.SetTitle(fmt.Sprintf("运行中: %d 个", n))
	systray.SetTooltip(fmt.Sprintf("AT Starter — %d 个运行中", n))
}

// isTrayReady 报告托盘是否已成功初始化。
func isTrayReady() bool {
	trayMu.Lock()
	defer trayMu.Unlock()
	return trayReady
}
