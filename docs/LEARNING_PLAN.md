# SimpleChain 学习计划

## 目标与当前进度

使用 Go 从零理解并逐步编写一条简单的公链。本项目用于学习，当前仅完成开发环境与文档基线，不具备区块链或生产网络能力。

当前程序只打印 `SimpleChain development environment ready.`，尚无测试用例或第三方依赖。

## 后续学习顺序

1. 理解 Block、Hash 与 Blockchain 数据结构，先建立内存中的最小模型。
2. 学习确定性编码、哈希计算、前序哈希连接与链有效性验证，并添加针对行为的测试。
3. 理解 Proof of Work 的基本原理与难度，再决定教学实现方式。
4. 逐步讨论持久化、交易、签名与节点通信；每阶段单独明确范围。

以上均为后续计划，本次基线不实现这些功能。

## 开发与验证

在项目目录运行：

```sh
source ./env.sh
gofmt -w main.go
go test ./...
go build
./simplechain
```

新增 Go 文件后，对变更的 Go 文件运行 gofmt。每个阶段保持可编译、可运行，记录重要设计决定。
