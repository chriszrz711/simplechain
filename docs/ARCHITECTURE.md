# 当前架构

## 基线范围

当前只有一个 Go 可执行程序，module 为 `simplechain`，入口为根目录的 `main.go`。程序调用标准库 `fmt.Println` 输出环境就绪信息，没有区块链数据结构、网络服务、数据库或第三方依赖。

## 文件职责

| 路径 | 职责 |
| --- | --- |
| `go.mod` | module 名称与 Go 版本声明 |
| `main.go` | 最小程序入口 |
| `env.sh` | 为当前 shell 添加用户目录 Go 和 VS Code CLI 路径 |
| `.vscode/settings.json` | 为本项目指定 Go SDK 位置 |
| `.gitignore` | 忽略本地可执行文件和常见 macOS 文件 |
| `docs/LEARNING_PLAN.md` | 学习目标与阶段顺序 |
| `docs/ARCHITECTURE.md` | 当前实际结构 |
| `docs/DECISIONS.md` | 已采用的设计决定 |

## 本地环境

当前基线使用 Go 1.27.1、darwin/arm64。Go 安装在 `~/.local/share/simplechain/go`，不随仓库提交。`env.sh` 和 VS Code 设置针对本机安装位置；其他机器需要自行安装 Go 并调整对应路径。

`go build` 在根目录生成 `simplechain`，该文件不纳入版本控制。当前没有测试用例，因此 `go test ./...` 的预期结果包含 `[no test files]`。

后续架构随学习阶段演进，暂不预设包拆分、共识、存储或网络方案。
