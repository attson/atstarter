//go:build darwin

package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

// platformInstall keeps the cross-platform installer signature. The PKG owns
// the fixed destination, so the current target and executable paths are not
// needed on macOS.
func platformInstall(asset, target, execPath string) error {
	return installDarwin(asset, target)
}

// installDarwin mounts the release DMG and runs the PKG inside it. The PKG is
// responsible for replacing /Applications/AT Starter.app and installing the
// /usr/local/bin/atstarter symlink as one privileged operation.
func installDarwin(dmgPath, _ string) error {
	return installMacOSPackageDMG(dmgPath, macOSPackageInstallOps{
		attach:   hdiutilAttach,
		detach:   hdiutilDetach,
		install:  installDarwinPackage,
		relaunch: startDarwinRelaunchAfterExit,
	})
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

func hdiutilDetach(mountPoint string) error {
	if mountPoint == "" {
		return nil
	}
	return runCommand("hdiutil", "detach", mountPoint, "-quiet", "-force")
}

func installDarwinPackage(pkgPath string) error {
	return runCommand("osascript", macOSPackageInstallerArgs(pkgPath)...)
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

func startDarwinRelaunchAfterExit(appPath string) error {
	cmd := exec.Command("/bin/sh", "-c", darwinRelaunchAfterExitScript(os.Getpid(), appPath))
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
}

func darwinRelaunchAfterExitScript(pid int, appPath string) string {
	return fmt.Sprintf(`i=0
while kill -0 %d 2>/dev/null && [ "$i" -lt 200 ]; do
  i=$((i + 1))
  sleep 0.1
done
/usr/bin/open -n %s
`, pid, darwinShellQuote(appPath))
}

func darwinShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
