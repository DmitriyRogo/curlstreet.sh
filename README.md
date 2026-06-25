<h1 align="center">curlstreet.sh</h1>

<p align="center">
  <strong>📈 The stock market in your terminal — stocks, crypto & the economic calendar, straight from <code>curl</code>.</strong>
</p>

<p align="center">
  <img src=".github/demo.gif" alt="curlstreet.sh demo" width="700">
</p>

<p align="center">
  <a href="#-quick-start">Quick Start</a> ·
  <a href="#-usage">Usage</a> ·
  <a href="#-features">Features</a> ·
  <a href="#-development">Development</a>
</p>

---

## ⚡ Quick Start

```bash
curl curlstreet.sh/AAPL          # 📊 a single stock quote
curl curlstreet.sh/BTC,ETH       # 🪙 batch crypto in one shot
curl curlstreet.sh/TSLA?format=json   # 🤖 clean JSON for your scripts
```

No signup. No API key. No rate limits for personal use. Just `curl` and go.

> Inspired by [wttr.in](https://github.com/chubin/wttr.in) — terminals get crisp **ANSI color**, browsers get a **dark HTML view**, and `?format=json` hands you **machine-readable** output.

## 🚀 Usage

| Command | What you get |
|---|---|
| `curl curlstreet.sh/AAPL` | 📈 Single stock quote |
| `curl curlstreet.sh/BTC` | 🪙 Crypto quote |
| `curl curlstreet.sh/AAPL,MSFT,GOOG` | 🧺 Batch up to **10** symbols |
| `curl curlstreet.sh/AAPL?format=json` | 🤖 JSON output for scripts |
| Open in your **browser** | 🌙 Dark terminal theme with market overview |

## ✨ Features

- 🎨 **Smart formatting** — auto-detects your client. Terminal → ANSI, browser → HTML, scripts → JSON.
- 💹 **Stocks _and_ crypto** — equities via Finnhub, coins via CoinGecko, one clean interface.
- ⚡ **Fast** — built-in LRU caching keeps responses snappy.
- 🪶 **Zero dependencies for you** — it's just `curl`. Works anywhere a terminal does.

## 🛠 Development

```bash
go test ./...     # ✅ run the suite
go build ./...    # 🔨 build it
go vet ./...      # 🔍 lint it
```

## 📄 License

MIT — go wild.
