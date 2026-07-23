// Package detector 根据项目目录内的文件特征识别项目类型并给出建议启动命令。
// 纯函数式:只读文件系统,无副作用,给定目录输出恒定。
package detector

import "fmt"

// Result 是一次识别的结果。Command 为空表示未能识别(Type == "unknown")。
type Result struct {
	Type    string // 如 "go" / "node-pnpm" / "unknown"
	Command string // 建议的单行启动命令,供 UI 回显;可为空
}

// Detect 按优先级从上到下匹配规则,命中即返回。
func Detect(dir string) Result {
	return detect(dir, false)
}

// DetectOptions 返回可供用户切换的识别结果。compose 仍作为主结果,但会追加忽略
// compose 文件后的普通项目识别结果,用于目录里同时存在 compose 与源码入口的场景。
func DetectOptions(dir string) []Result {
	primary := Detect(dir)
	options := []Result{primary}
	if primary.Type == "compose" {
		fallback := detect(dir, true)
		if fallback.Type != "unknown" && fallback.Type != primary.Type {
			options = append(options, fallback)
		}
	}
	return options
}

func detect(dir string, ignoreCompose bool) Result {
	hasPkg := exists(dir, "package.json")

	switch {
	case !ignoreCompose && firstExisting(dir, "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml") != "":
		return Result{"compose", ""}

	case hasPkg && exists(dir, "pnpm-lock.yaml"):
		return Result{"node-pnpm", "pnpm run " + pickNodeScript(dir)}
	case hasPkg && exists(dir, "yarn.lock"):
		return Result{"node-yarn", "yarn " + pickNodeScript(dir)}
	case hasPkg && exists(dir, "bun.lockb"):
		return Result{"node-bun", "bun run " + pickNodeScript(dir)}
	case hasPkg:
		return Result{"node-npm", "npm run " + pickNodeScript(dir)}

	case exists(dir, "go.mod") && exists(dir, "main.go"):
		return Result{"go", "go run main.go"}
	case exists(dir, "go.mod"):
		if name := firstCmdMain(dir); name != "" {
			return Result{"go", fmt.Sprintf("go run ./cmd/%s", name)}
		}
		return Result{"go", "go run ."}

	case exists(dir, "Cargo.toml"):
		return Result{"rust", "cargo run"}

	case exists(dir, "manage.py"):
		return Result{"python-django", "python manage.py runserver"}

	case exists(dir, "pyproject.toml") && exists(dir, "poetry.lock"):
		if f := firstExisting(dir, "main.py", "app.py"); f != "" {
			return Result{"python-poetry", "poetry run python " + f}
		}
		return Result{"python-poetry", "poetry run python main.py"}

	case firstExisting(dir, "main.py", "app.py") != "":
		return Result{"python", "python " + firstExisting(dir, "main.py", "app.py")}
	case exists(dir, "requirements.txt"):
		return Result{"python", "python main.py"}
	}

	return Result{"unknown", ""}
}
