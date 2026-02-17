# Interzoid MCP Server

An MCP (Model Context Protocol) server that exposes [Interzoid](https://interzoid.com)'s AI-powered data quality, matching, enrichment, and standardization APIs to AI agents and LLM applications.

## What This Does

This MCP server makes 29 Interzoid APIs discoverable and callable by any MCP-compatible client including Claude Desktop, Claude Code, Cursor, Windsurf, and other AI tools. AI agents can discover the available data quality tools and invoke them as needed during conversations and workflows.

### Available API Categories (30 APIs)

| Category | Tools | Price (USDC) |
|---|---|---|
| **Data Matching** — Similarity key generation & scoring | Company match, org match score, name match/score, address match, global address, product match | $0.0125/call |
| **Data Enrichment** — AI-powered intelligence (Premium) | Business info, parent company, executives, news, email trust, stock, verification, tech stack, IP/phone profiles | $0.3125/call |
| **Data Standardization** — Canonical form normalization | Org, country, country info, city, state abbreviation | $0.0125/call |
| **Data Enhancement** — Classification & analysis | Entity type, gender, name origin, language ID, translation (to English & any), address parsing | $0.0125/call |
| **Utility** — Weather, currency, ZIP lookup | Global weather, exchange rates, ZIP code info | $0.0125/call |

All APIs are x402-enabled for native crypto micropayments on Base (USDC).

## Quick Start

### Prerequisites

- Go 1.22+

### Build

```bash
git clone https://github.com/interzoid/interzoid-mcp-server.git
cd interzoid-mcp-server
go mod tidy
go build -o interzoid-mcp-server .
```

### Authentication

The MCP server supports three authentication methods (in priority order):

1. **Authorization header** (remote HTTP transport) — The connecting client sends `Authorization: Bearer <api-key>` when connecting. The MCP server forwards it to the Interzoid API via the `x-api-key` header. This is the standard method for remote/hosted deployments.

2. **Environment variable** (local stdio transport) — Set `INTERZOID_API_KEY` in your MCP client config. This is the standard method for local installations.

3. **x402 micropayments** — When no API key is provided, requests trigger the x402 payment flow. The Interzoid API returns a `402 Payment Required` response with payment requirements, and the calling agent/client handles payment negotiation using USDC on Base.

### Run in x402 Mode (default — for crypto micropayments)

```bash
./interzoid-mcp-server
```

When no API key is provided via environment variable or Authorization header, requests trigger the x402 payment flow.

### Run with API Key (local, direct access)

```bash
export INTERZOID_API_KEY="your-api-key-here"
./interzoid-mcp-server
```

### Run with HTTP Transport (for remote/hosted access)

```bash
./interzoid-mcp-server -transport http -port 8080
```

Remote clients authenticate by passing `Authorization: Bearer <api-key>` in their connection headers.

The MCP endpoint will be available at `http://localhost:8080/mcp`.

## Client Configuration

### Claude Desktop

Add to `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "interzoid": {
      "command": "/path/to/interzoid-mcp-server"
    }
  }
}
```

Or with an API key for direct access (bypasses x402):

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

### Cursor / Windsurf / Other MCP Clients (remote HTTP)

Point the client to the StreamableHTTP endpoint:

```
http://your-server:8080/mcp
```

### Using via npx (once published to npm)

If you wrap the binary and publish to npm:

```json
{
  "mcpServers": {
    "interzoid": {
      "command": "npx",
      "args": ["-y", "@interzoid/mcp-server"],
      "env": {
        "INTERZOID_API_KEY": "your-api-key-here"
      }
    }
  }
}
```

## Example Interactions

Once configured, AI agents can use the tools naturally:

> **User:** "Check if 'IBM' and 'International Business Machines' are the same company"
> **Agent uses:** `interzoid_company_match_score` with company1="IBM", company2="International Business Machines"
> **Result:** Score: 98

> **User:** "What can you tell me about Anthropic's business?"
> **Agent uses:** `interzoid_business_info` with lookup="Anthropic"
> **Result:** Headquarters, revenue, employees, industry, description

> **User:** "Translate 'Bonjour le monde' to English"
> **Agent uses:** `interzoid_translate_to_english` with text="Bonjour le monde"
> **Result:** Translation: "Hello world"

## x402 Payment Integration

All Interzoid APIs support the [x402 protocol](https://x402.org) for native crypto micropayments. When accessed through the x402 payment flow:

- **Standard APIs:** 12,500 atomic USDC ($0.0125) per call
- **Premium APIs:** 312,500 atomic USDC ($0.3125) per call
- **Network:** Base (EIP-155:8453)
- **Asset:** USDC (0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913)

The `.well-known/x402.json` manifest at `https://api.interzoid.com/.well-known/x402.json` provides full machine-readable discovery for x402 clients.

## Publishing & Discovery

To make this server discoverable in the AI ecosystem:

1. **GitHub** — Publish the repo publicly
2. **mcp.so** — Submit at [mcp.so](https://mcp.so) for MCP directory listing
3. **mcpservers.org** — Submit at [mcpservers.org](https://mcpservers.org)
4. **Coinbase Bazaar** — Already listed via x402 payment integration
5. **npm** — Optional: wrap the binary for `npx` distribution
6. **Smithery** — Submit at [smithery.ai](https://smithery.ai) for discovery

## Project Structure

```
interzoid-mcp-server/
├── main.go        # Entry point, transport selection (stdio/HTTP)
├── tools.go       # MCP tool registration for all ~30 APIs
├── client.go      # HTTP client for calling api.interzoid.com
├── go.mod         # Go module definition
└── README.md      # This file
```

## Adding New APIs

To add a new Interzoid API as an MCP tool:

1. Add a new `s.AddTool(...)` block in `tools.go` within the appropriate category
2. Use `genericHandler("/your-endpoint", []string{"param1", "param2"})` as the handler
3. Write a clear description explaining WHEN and WHY an agent should use this tool
4. Rebuild: `go build -o interzoid-mcp-server .`

## License

MIT
