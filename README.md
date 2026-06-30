# yze-cliv3

A [go/analysis](https://pkg.go.dev/golang.org/x/tools/go/analysis) analyzer in the [gomatic `yze`](https://github.com/gomatic/yze) suite. It enforces the gomatic CLI standard that command-line programs use [urfave/cli **v3**](https://github.com/urfave/cli) — reporting any import of the legacy `github.com/urfave/cli` (v1) or `github.com/urfave/cli/v2` paths.

- **Rule id:** `yze/cliv3`
- **Capability:** `convention:cliv3`

## Use

Run via the `yze` aggregator (recommended) or standalone:

```sh
go run github.com/gomatic/yze-cliv3/cmd/yze-cliv3@latest ./...
```

A finding looks like:

```
main.go:4:2: use urfave/cli/v3; the legacy urfave/cli v1/v2 import path is forbidden by the gomatic CLI standard
```
