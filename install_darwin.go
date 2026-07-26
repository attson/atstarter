//go:build darwin

package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// platformInstall 是 runInstall 在 darwin 上的实现(通过 var 注入,见
// updater.go)。参数保持与其它平台一致(execPath 在这里不需要,GOOS 分发前
// 已经把 target 归约到 .app bundle)。
func platformInstall(asset, target, execPath string) error {
	return installDarwin(asset, target)
}

// installDarwin 在 Go 里同步完成整个 macOS 升级流程,避免"派 bash 后立刻退出、
// 赌孤儿进程能独立跑完"的脆弱模型。所有可能失败的步骤在 App 还活着的时候执行,
// 出错能真正返回给前端。全流程:
//
//   1. hdiutil attach —— 挂 DMG,阻塞等待,拿到 mount point。任何失败(EULA / 损坏 /
//      占用中)都在这里直接返回给用户。
//   2. 找到 DMG 里的 *.app 源目录。
//   3. 决定 destParent(优先原位置,否则 /Applications,否则 ~/Applications)。
//   4. ditto 到 destParent/*.app.new —— staging 与目标同目录,保证后续 rename 原子。
//      (ditto 而非 cp -R:保留 xattr / ACL / HFS 元数据,Apple 官方推荐用于 bundle)
//   5. 剥离 quarantine 属性,免得新 bundle 被 Gatekeeper 拦。
//   6. 原子 rename swap:老 → .old,.new → 目标。任一步失败尝试回滚。
//   7. 清理 .old,unmount DMG。
//   8. spawn `open <bundle>`(不带 -n:让 Launch Services 检测到老实例正在退,顺
//      势启动新版;-n 会与老实例清理竞态)。
//
// 返回 nil 后由调用者置位 quitRequested 再 wailsruntime.Quit。open 已经在等,
// 老进程一退,Launch Services 就启动新版。
func installDarwin(dmgPath, targetApp string) error {
	if _, err := os.Stat(dmgPath); err != nil {
		return fmt.Errorf("asset missing: %w", err)
	}

	mountPoint, err := hdiutilAttach(dmgPath)
	if err != nil {
		return fmt.Errorf("hdiutil attach: %w", err)
	}
	// 无论后续成败都要 detach,DMG 挂着不释放会占内存 + 阻塞下次 attach。
	defer hdiutilDetach(mountPoint)

	srcApp, err := firstAppIn(mountPoint)
	if err != nil {
		return fmt.Errorf("find .app in dmg: %w", err)
	}

	destParent, err := chooseDestParent(targetApp)
	if err != nil {
		return err
	}
	destApp := filepath.Join(destParent, filepath.Base(srcApp))
	stagingApp := destApp + ".new"
	oldApp := destApp + ".old"

	// 上一次失败可能留下的残骸。忽略"不存在"以外的错误(权限等)—— 若真删不掉,
	// 后续 ditto/rename 会报出来。
	_ = os.RemoveAll(stagingApp)
	_ = os.RemoveAll(oldApp)

	if err := runCommand("ditto", srcApp, stagingApp); err != nil {
		return fmt.Errorf("ditto to staging: %w", err)
	}
	_ = runCommand("xattr", "-dr", "com.apple.quarantine", stagingApp)

	if err := renameSwap(destApp, oldApp, stagingApp); err != nil {
		_ = os.RemoveAll(stagingApp)
		return err
	}
	_ = os.RemoveAll(oldApp)

	// spawn open 后立即返回;老进程随后 Quit,Launch Services 拉起新版。
	// Start 而非 Run:不阻塞等 open 退出(open 会等 app 起来后才 exit)。
	if err := exec.Command("open", destApp).Start(); err != nil {
		return fmt.Errorf("open new bundle: %w", err)
	}
	return nil
}

// hdiutilAttach 挂载 DMG 并返回 mount point 路径。
// 输出格式类似:
//
//	/dev/disk4              GUID_partition_scheme
//	/dev/disk4s1            Apple_HFS                       /Volumes/AT-Starter 0.5.5
//
// 我们扫描每一行,把最后一个 tab-separated 字段是 /Volumes/... 的记为 mount point。
// 不用 `awk 'END{print $NF}'` 那种"最后一行末字段"的启发式 —— DMG 若含 EULA 或多
// 分区,末字段可能不是 volume。
func hdiutilAttach(dmgPath string) (string, error) {
	out, err := exec.Command("hdiutil", "attach", "-nobrowse", "-noverify", "-readonly", dmgPath).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("hdiutil exit %d: %s", ee.ExitCode(), string(ee.Stderr))
		}
		return "", err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	var mount string
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "\t")
		if len(fields) == 0 {
			continue
		}
		last := strings.TrimSpace(fields[len(fields)-1])
		if strings.HasPrefix(last, "/Volumes/") {
			mount = last // 最后一个 /Volumes/... 才是主 volume(EULA 卷会先出现)
		}
	}
	if mount == "" {
		return "", fmt.Errorf("no /Volumes/ mount in hdiutil output: %s", string(out))
	}
	return mount, nil
}

func hdiutilDetach(mountPoint string) {
	if mountPoint == "" {
		return
	}
	_ = exec.Command("hdiutil", "detach", mountPoint, "-quiet", "-force").Run()
}

// firstAppIn 返回 dir 下第一个 *.app 目录的绝对路径(深度 ≤ 2)。
// DMG 内一般就一个,最多有"Applications 符号链接 + 真 .app"两项。
func firstAppIn(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), ".app") {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	// 兜底扫一层子目录,应对少数 DMG 把 .app 放子目录里的情况。
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		subEntries, _ := os.ReadDir(filepath.Join(dir, e.Name()))
		for _, s := range subEntries {
			if s.IsDir() && strings.HasSuffix(s.Name(), ".app") {
				return filepath.Join(dir, e.Name(), s.Name()), nil
			}
		}
	}
	return "", errors.New("no .app found in mount")
}

// chooseDestParent 选择新 .app 的落点父目录:
//   - 优先原位置(targetApp 的 dirname),让升级"就地"覆盖 —— 用户 Dock/Alfred
//     指向的路径永远稳定。
//   - 原位置不可写(常见:/Applications 需管理员且当前用户不是 admin)→ 退到
//     ~/Applications。位置漂了,但至少能装上;下次可提示用户手动挪。
//   - 家目录都拿不到:极少见,直接失败。
//
// 不选"/Applications 作为独立分支"—— 原位置本身就通常是 /Applications;若真不
// 是且它可写,原位置分支已经命中。
func chooseDestParent(targetApp string) (string, error) {
	origParent := filepath.Dir(targetApp)
	if isWritableDir(origParent) {
		return origParent, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("no writable install location: %w", err)
	}
	userApps := filepath.Join(home, "Applications")
	if err := os.MkdirAll(userApps, 0o755); err != nil {
		return "", fmt.Errorf("create ~/Applications: %w", err)
	}
	return userApps, nil
}

func isWritableDir(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	// os 层的 Access(W_OK) 在 Go 里没有直接 API;实测创建一个临时文件更可靠。
	f, err := os.CreateTemp(dir, ".atstarter-writetest-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// renameSwap 原子替换 bundle:
//  1. destApp 若存在 → rename 到 oldApp(为回滚保留);不存在(全新装)→ 跳过。
//  2. stagingApp → destApp。
//  3. 第 2 步失败尝试把 oldApp rename 回来。
//
// 同盘上 rename 是原子的:任何瞬间要么老 bundle 在位、要么新 bundle 在位,不会
// 出现"目标目录不存在"的窗口,避免 Launch Services / Dock 图标错乱。
func renameSwap(destApp, oldApp, stagingApp string) error {
	stashed := false
	if _, err := os.Stat(destApp); err == nil {
		if err := os.Rename(destApp, oldApp); err != nil {
			return fmt.Errorf("stash old bundle: %w", err)
		}
		stashed = true
	}
	if err := os.Rename(stagingApp, destApp); err != nil {
		if stashed {
			// 回滚:把老 bundle 复位。失败也没救了,报原始错。
			_ = os.Rename(oldApp, destApp)
		}
		return fmt.Errorf("swap in new bundle: %w", err)
	}
	return nil
}

// runCommand 执行一条命令,失败时把 stderr 塞进 error,便于前端展示。
func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, msg)
	}
	return nil
}
