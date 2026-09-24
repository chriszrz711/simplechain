# Simplechain v1

An educational Go PoW public-chain prototype, **not production cryptocurrency software**.

## Supported features and architecture

- Wallets provide addresses and ECDSA signatures, with encrypted wallet persistence APIs.
- Transactions use UTXOs, signatures, ownership and amount validation, and double-spend prevention.
- Blockchain maintains an incremental `UTXOSet` for runtime payments and balances. `ValidateChain` audits history independently; `BuildUTXOSet` rebuilds the cache. Blockchain persistence APIs save/load validated history.
- Mempool admits valid transactions spending confirmed UTXOs only, rejects conflicts, selects deterministically, and revalidates after chain changes.
- Mining prepends a fixed Coinbase reward and uses the existing safe block insertion and PoW path, then cleans the mempool.
- Node supports static peers, transaction/block propagation, and manual same-chain synchronization. Local mining remains committed when broadcasting fails.

## Test

From the project directory, using the existing local environment setup:

```sh
source ./env.sh
go test -count=1 ./...
```

The multi-node integration test covers signing, local admission, transaction propagation, mining, block propagation, balances, UTXO consistency, cleanup, and a third node catching up:

```sh
go test -run '^TestSimplechainV1MultiNodeLifecycle$' -count=1 -v
```

## Start HTTP nodes

In separate terminals:

```sh
source ./env.sh
go run . -port 8001 -miner '<existing-wallet-address-A>' -peers http://localhost:8002
```

```sh
source ./env.sh
go run . -port 8002 -miner '<existing-wallet-address-B>' -peers http://localhost:8001
```

Replace the address placeholders. The CLI does not create/manage wallets. Port defaults to 8001; `-miner` is required; `-peers` is optional and accepts comma-separated URLs. Use `go run . -h` for flags. The listener binds all interfaces on the requested port; stop it with Ctrl-C.

Each process starts a **fresh in-memory chain and mempool**. Restarting loses runtime state; this CLI does not automatically load/save chain files. Existing persistence APIs remain available to Go callers.

The server uses HTTP timeouts and serializes requests because v1's core state is not concurrency-safe. Direct concurrent Go API access is unsupported. There is no automatic mining or synchronization at startup; Go callers invoke `SubmitTransaction`, `MineAndBroadcastBlock`, and `SyncFromPeer` explicitly. Static peers configure outbound propagation when these methods are called.

## HTTP endpoints

| Method | Endpoint | Behavior |
| --- | --- | --- |
| GET | `/status` | JSON `height` and `mempool_size` |
| POST | `/transaction` | Validate and admit a signed Transaction JSON object; no automatic mining/rebroadcast |
| POST | `/block` | Safely accept an already-mined next Block JSON object and clean the mempool |
| GET | `/blocks?from_height=N` | Return blocks from the requested height in chain order |

```sh
curl http://localhost:8001/status
curl 'http://localhost:8002/blocks?from_height=0'
```

Transaction and Block requests use Go's existing JSON serialization (`[]byte` fields are base64). Admission responses include `accepted` and either a hex ID/hash or an error. There are no HTTP wallet, mining, or sync-control endpoints.

## Intentional limitations

- One canonical chain; synchronization only extends the same chain. No forks, reorgs, longest-chain/cumulative-work selection, or background sync.
- Fixed PoW difficulty and fixed `CoinbaseReward` (50); no transaction fees or fee priority.
- No peer discovery, gossip rebroadcast, unconfirmed mempool dependencies, or background networking jobs.
- No databases, Merkle trees/proofs, tokens, smart contracts, or wallet management CLI.
- No authentication/TLS layer is configured by this runtime.
