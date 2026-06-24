# curlstreet.sh

Market data for your terminal — stocks, crypto, and economic calendar, straight from `curl`.

```
curl curlstreet.sh/AAPL
curl curlstreet.sh/BTC,ETH
curl curlstreet.sh/TSLA?format=json
```

Inspired by [wttr.in](https://github.com/chubin/wttr.in). Terminals get ANSI color, browsers get a dark HTML view, and `?format=json` gives you clean machine-readable output. No signup, no rate limits for personal use.

## Usage

| | |
|---|---|
| `curl curlstreet.sh/AAPL` | Single stock quote |
| `curl curlstreet.sh/BTC` | Crypto quote |
| `curl curlstreet.sh/AAPL,MSFT,GOOG` | Batch up to 10 symbols |
| `curl curlstreet.sh/AAPL?format=json` | JSON output |
| Open in browser | Dark terminal theme with market overview |

## Development

```bash
go test ./...
go build ./...
go vet ./...
```

## License

MIT
