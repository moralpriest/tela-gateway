# Aliases & the indexer

An **alias** maps a short app name (or dURL) to a TELA INDEX contract SCID, so
`derobeats.tela.example.com` or `/durl/derobeats` resolves without the viewer
knowing the 64-hex SCID.

## Resolution priority

For a name like `derobeats` (or `derobeats.tela`):

1. **`TELA_ALIASES` env** — `name=scid,name2=scid2`. Highest priority, no chain
   scan needed. Best way to pin known apps.
2. **`aliases.json`** bundled with the deploy, optionally refreshed from
   `ALIASES_S3_URI` (`s3://bucket/aliases.json`) on cold start.
3. **Built-in aliases** compiled into the binary (`derobeats`, `explorer`).

Any app is always reachable without an alias via `/scid/<64-hex>/`.

## Adding apps the easy way

For a handful of apps, skip the indexer entirely:

```bash
TELA_ALIASES="derobeats=b1e1cba5...,explorer=<scid>,myapp=<scid>"
```

Set it as an env var on whatever platform you deployed to.

## The indexer (optional, for discovery at scale)

`cmd/indexer` scans the DERO chain with Gnomon, finds every TELA INDEX
contract, and writes `durl → scid` (plus the short alias) into `aliases.json`.

Run locally:

```bash
go run ./cmd/indexer
```

Deploy as a weekly AWS Lambda that uploads to S3 (feeds the gateway via
`ALIASES_S3_URI`):

```bash
bash scripts/deploy-indexer-lambda.sh
# writes s3://cypher-punks-tela-aliases/aliases.json on a weekly EventBridge rule
```

### Caveats

- A full-chain Gnomon scan is time-consuming — mainnet is millions of blocks.
- Run the indexer on a **persistent host** that keeps its `datashards/` BoltDB
  between runs, so it resumes from the last indexed height instead of rescanning.
- For anything beyond the seed list, `TELA_ALIASES` is the fastest path and needs
  no scan.
