# Interzoid MCP Server

An MCP (Model Context Protocol) server that exposes [Interzoid](https://interzoid.com)'s AI-powered data quality, matching, enrichment, and standardization APIs to AI agents and LLM applications.

## What This Does

This MCP server makes 29 Interzoid APIs discoverable and callable by any MCP-compatible client including Claude Desktop, Claude Code, Cursor, Windsurf, and other AI tools. AI agents can discover the available data quality tools and invoke them as needed during conversations and workflows.

### Available APIs (29 Tools)

| Category | Tools | Price (USDC) |
|---|---|---|
| **Data Matching** — Similarity key generation & scoring | Company match, org match score, name match/score, address match, global address, product match | $0.0125/call |
| **Data Enrichment** — AI-powered intelligence (Premium) | Business info, parent company, executives, news, email trust, stock, verification, IP/phone profiles | $0.3125/call |
| **Data Standardization** — Canonical form normalization | Org, country, country info, city, state abbreviation | $0.0125/call |
| **Data Enhancement** — Classification & analysis | Entity type, gender, name origin, language ID, translation (to English & any), address parsing | $0.0125/call |
| **Utility** — Weather, currency, ZIP lookup | Global weather, exchange rates, ZIP code info | $0.0125/call |

## Getting Started

### Option 1: Use the Hosted Remote Server (no installation required)

Connect any MCP client that supports remote HTTP servers to:

```
https://mcp.interzoid.com/mcp
```

Pass your API key via the `Authorization` header:

```
Authorization: Bearer your-api-key-here
```

Get a free API key at [interzoid.com](https://www.interzoid.com/signup) to get started.

### Option 2: Download a Prebuilt Binary (no Go required)

Download the binary for your platform from the [GitHub Releases](https://github.com/interzoid/interzoid-mcp-server/releases) page:

- **Windows:** `interzoid-mcp-server-windows-amd64.exe`
- **macOS (Apple Silicon):** `interzoid-mcp-server-macos-arm64`
- **Linux:** `interzoid-mcp-server-linux-amd64`

Then configure your MCP client to run it (see [Client Configuration](#client-configuration) below).

### Option 3: Build from Source

Requires Go 1.21+:

```bash
git clone https://github.com/interzoid/interzoid-mcp-server.git
cd interzoid-mcp-server
go mod tidy
go build -o interzoid-mcp-server .
```

## Authentication

There are three ways to authenticate, depending on how you're using the server:

### 1. API Key via Environment Variable (local installations)

Set `INTERZOID_API_KEY` in your MCP client config. This is the standard method when running the binary locally.

```bash
export INTERZOID_API_KEY="your-api-key-here"
./interzoid-mcp-server
```

### 2. API Key via Authorization Header (remote/hosted server)

When connecting to the hosted server at `https://mcp.interzoid.com/mcp` or any remote deployment, pass your API key in the `Authorization` header:

```
Authorization: Bearer your-api-key-here
```

The MCP server forwards this to the Interzoid API via the `x-api-key` header.

### 3. x402 Crypto Micropayments (no API key needed)

When no API key is provided by either method above, requests trigger the [x402 payment protocol](https://x402.org). The Interzoid API returns a `402 Payment Required` response with payment requirements, and the calling agent/client handles payment negotiation using USDC on Base. No signup or API key is needed — just a compatible wallet.

### Where to Get an API Key

Sign up for a free API key at [interzoid.com/signup](https://www.interzoid.com/signup). Keys work with both the local binary (via environment variable) and the remote server (via Authorization header).

## Client Configuration

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "interzoid": {
      "command": "/path/to/interzoid-mcp-server",
      "env": {
        "INTERZOID_API_KEY": "your-api-key-here"
      }
    }
  }
}
```

### Claude Code

```bash
claude mcp add interzoid -- /path/to/interzoid-mcp-server
```

Or add to `.claude/settings.json`:

```json
{
  "mcpServers": {
    "interzoid": {
      "command": "/path/to/interzoid-mcp-server",
      "env": {
        "INTERZOID_API_KEY": "your-api-key-here"
      }
    }
  }
}
```

### Cursor / Windsurf / Other MCP Clients (remote)

Point the client to the hosted server:

```
https://mcp.interzoid.com/mcp
```

Set the Authorization header to `Bearer your-api-key-here`.

## Example Interactions

Once configured, AI agents can use the tools naturally:

> **User:** "Check if 'IBM' and 'International Business Machines' are the same company"
> **Agent uses:** `interzoid_org_match_score` with org1="IBM", org2="International Business Machines"
> **Result:** Score: 98

> **User:** "What can you tell me about Anthropic's business?"
> **Agent uses:** `interzoid_business_info` with lookup="Anthropic"
> **Result:** Headquarters, revenue, employees, industry, description

> **User:** "Translate 'Bonjour le monde' to English"
> **Agent uses:** `interzoid_translate_to_english` with text="Bonjour le monde"
> **Result:** Translation: "Hello world"

## Self-Hosting the Remote Server

To host your own remote instance:

```bash
./interzoid-mcp-server -transport http -port 8080
```

The MCP endpoint will be available at `http://localhost:8080/mcp`. Place behind Nginx or a load balancer with HTTPS for production use. Ensure `proxy_buffering off` is set in your Nginx config to support SSE streaming.

## x402 Payment Integration

All Interzoid APIs support the [x402 protocol](https://x402.org) for native crypto micropayments. When accessed without an API key:

- **Standard APIs:** 12,500 atomic USDC ($0.0125) per call
- **Premium APIs:** 312,500 atomic USDC ($0.3125) per call
- **Network:** Base (EIP-155:8453)
- **Asset:** USDC (0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913)

The `.well-known/x402.json` manifest at `https://api.interzoid.com/.well-known/x402.json` provides full machine-readable discovery for x402 clients.

## Project Structure

```
interzoid-mcp-server/
├── main.go        # Entry point, transport selection (stdio/HTTP)
├── tools.go       # MCP tool registration for all 29 APIs
├── client.go      # HTTP client for calling api.interzoid.com
├── go.mod         # Go module definition
└── README.md      # This file
```

## License

MIT
