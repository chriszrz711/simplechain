# 决策记录

## 001：以最小 Go 程序作为初始基线

- 决定：使用 Go，module 暂名 `simplechain`，当前仅保留打印环境就绪信息的入口。
- 原因：先验证工具链，再逐步理解并手写区块链基础。
- 影响：暂不实现 Block、Blockchain、Proof of Work 或其他区块链功能。

## 002：用户目录安装与项目内环境配置

- 决定：使用 Go 1.27.1 ARM64 工具链，安装到 `~/.local/share/simplechain/go`；通过 `source ./env.sh` 在当前 shell 启用 Go 和 code CLI。
- 原因：避免管理员权限和全局 shell 配置变更。
- 影响：新终端需重新加载脚本；路径配置依赖本机安装布局。

## 003：暂不引入额外基础设施

- 决定：当前仅使用 Go 标准库；不安装 Bitcoin Core、Geth、Solidity、Hardhat、Ganache、Cosmos SDK、CometBFT、Docker、PostgreSQL 或 Redis。
- 原因：保持学习项目简单，按实际需求逐步讨论依赖。

## 004：源码与文档纳入版本控制

- 决定：提交源码、module、项目配置和学习文档，忽略本地编译产物与 macOS 杂项文件。
- 验证：提交前运行 gofmt、`go test ./...` 和 `go build`；当前无测试用例，不将命令成功视为已有行为测试覆盖。
