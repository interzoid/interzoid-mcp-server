# Interzoid MCP Server

An MCP (Model Context Protocol) server that exposes [Interzoid](https://interzoid.com)'s AI-powered data quality, matching, enrichment, and standardization APIs to AI agents and LLM applications.

## What This Does

This MCP server makes 58 Interzoid APIs discoverable and callable by any MCP-compatible client including Claude Desktop, Claude Code, Cursor, OpenAI Codex, Gemini CLI, VS Code, Windsurf, and other AI tools. AI agents can discover the available data quality and enrichment tools and invoke them as needed during conversations and workflows.

### Available APIs (58 Tools)

| Category | Tools | Price (x402 USDC) |
|---|---|---|
| **Data Matching** (10) | Similarity keys and match scores for company names, individual names, US and global addresses, products, plus combined keys (company+address, name+address, company+name) for high-precision deduplication | $0.01/call |
| **Company Intelligence** (13) | Business info, parent company, executives, recent news, verification, industry codes (NAICS/SIC), competitor analysis, buying signals, private company deal intel, ESG profile, government contracts, facilities profile, tech stack | $0.25/call |
| **Contact & Identity** (3) | Email trust score, IP address profile, phone number profile | $0.25/call |
| **Financial & Market** (2) | Stock analysis by ticker, municipal issuer profile | $0.25/call |
| **Property & Location** (3) | Commercial building profile, property transaction history, nearest coffee shops | $0.25/call |
| **Tax & Regulatory** (4) | US sales & use tax rates, IRS per diem rates, European VAT rates, customs duty rates by HS code | $0.25-$0.35/call |
| **Research & Lookup** (2) | University and college info, product recall information | $0.25/call |
| **X / Twitter** (3) | Find X handle for any entity, profile snapshot by handle, last three posts by handle | $0.05/call |
| **Custom & Identity Resolution** (2) | AI custom self-defined data enrichment (you define the output fields), official legal organization name resolution for KYB and compliance | $0.25/call |
| **Data Standardization** (5) | Organization, country, country info, city, state abbreviation | $0.01/call |
| **Data Enhancement** (7) | Entity type, gender, name origin, language ID, translation (to English & any language), address parsing | $0.01/call |
| **Utility** (4) | Global weather, currency exchange rates, ZIP code info, global page latency measurement | $0.01/call |

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

Get a free API key at [interzoid.com](https://www.interzoid.com/register-api-account) to get started.

### Option 2: Download a Prebuilt Binary (no Go required)

Download the binary for your platform from the [GitHub Releases](https://github.com/interzoid/interzoid-mcp-server/releases) page:

- **Windows:** `interzoid-mcp-server-windows-amd64.exe`
- **macOS (Apple Silicon):** `interzoid-mcp-server-macos-arm64`
- **Linux:** `interzoid-mcp-server-linux-amd64`

Then configure your MCP client to run it (see [Client Configuration](#client-configuration) below).

### Option 3: Build from Source

Requires Go 1.23+:

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

### 3. x402 USDC Micropayments (no API key needed)

When no API key is provided by either method above, requests trigger the [x402 payment protocol](https://x402.org). The Interzoid API returns a `402 Payment Required` response with payment requirements, and the calling agent/client handles payment negotiation using USDC on Base. No signup or API key is needed, just a compatible wallet.

### Where to Get an API Key

Sign up for a free API key at [interzoid.com/register-api-account](https://www.interzoid.com/register-api-account). Keys work with both the local binary (via environment variable) and the remote server (via Authorization header).

## Client Configuration

### Cursor

**One-click install:** Use the install button on the [Interzoid MCP Server page](https://www.interzoid.com/mcp-server#cursor), or copy this deep link into your browser:

```
cursor://anysphere.cursor-deeplink/mcp/install?name=Interzoid&config=eyJ1cmwiOiJodHRwczovL21jcC5pbnRlcnpvaWQuY29tL21jcCJ9
```

Or manually add to `.cursor/mcp.json` (project) or `~/.cursor/mcp.json` (global):

```json
{
  "mcpServers": {
    "interzoid": {
      "url": "https://mcp.interzoid.com/mcp",
      "headers": {
        "Authorization": "Bearer your-api-key-here"
      }
    }
  }
}
```

The `headers` block is optional. Omit it to use x402 USDC micropayments instead of an API key.

### Claude Code

Run in your terminal:

```bash
claude mcp add --transport http interzoid https://mcp.interzoid.com/mcp --header "Authorization: Bearer your-api-key-here"
```

The `--header` flag is optional. Omit it to use x402 USDC micropayments instead of an API key. Then use `/mcp` in Claude Code to verify the connection.

### OpenAI Codex

Run in your terminal (works for Codex CLI, the IDE extension, and the Codex app, which share the same configuration):

```bash
codex mcp add interzoid --url https://mcp.interzoid.com/mcp --bearer-token-env-var INTERZOID_API_KEY
```

Then set the `INTERZOID_API_KEY` environment variable to your API key. Or edit `~/.codex/config.toml` directly:

```toml
[mcp_servers.interzoid]
url = "https://mcp.interzoid.com/mcp"
bearer_token_env_var = "INTERZOID_API_KEY"
```

Run `/mcp` inside Codex to verify the server is connected.

### Gemini CLI

Run in your terminal:

```bash
gemini mcp add -t http interzoid https://mcp.interzoid.com/mcp
```

Or manually add to `~/.gemini/settings.json` (global) or `.gemini/settings.json` (project):

```json
{
  "mcpServers": {
    "interzoid": {
      "httpUrl": "https://mcp.interzoid.com/mcp",
      "headers": {
        "Authorization": "Bearer your-api-key-here"
      }
    }
  }
}
```

Run `/mcp` inside Gemini CLI to verify the connection.

### VS Code

Open the Command Palette (`Ctrl+Shift+P` / `⌘+Shift+P`), select **MCP: Open User Configuration**, and add:

```json
{
  "servers": {
    "interzoid": {
      "url": "https://mcp.interzoid.com/mcp",
      "type": "http"
    }
  }
}
```

### Windsurf

Add to `~/.codeium/windsurf/mcp_config.json`:

```json
{
  "mcpServers": {
    "interzoid": {
      "serverUrl": "https://mcp.interzoid.com/mcp"
    }
  }
}
```

### Claude Desktop

**Remote server (Pro/Max/Team/Enterprise):** Go to **Settings > Connectors > Add Custom Connector** and enter `https://mcp.interzoid.com/mcp`.

**Local binary:** Add to `claude_desktop_config.json` (macOS: `~/Library/Application Support/Claude/`, Windows: `%APPDATA%\Claude\`):

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

### Other MCP Clients

Point any MCP-compatible client to the hosted remote server:

```
https://mcp.interzoid.com/mcp
```

The server supports Streamable HTTP transport. Pass your API key via the `Authorization: Bearer your-api-key` header, or use x402 USDC micropayments with no API key needed.

## Example Interactions

Once configured, AI agents can chain the tools into complete workflows:

> **User:** "Here are 40 customer records. Find the duplicates even where names are spelled differently."
> **Agent uses:** `interzoid_company_match_advanced` on each record, then groups rows sharing a similarity key
> **Result:** A duplicate report with 'IBM', 'I.B.M.', and 'International Business Machines' correctly clustered

> **User:** "Build me a briefing on Databricks before my call tomorrow."
> **Agent uses:** `interzoid_business_info`, `interzoid_buying_signals`, `interzoid_competitor_analysis`, `interzoid_private_company_deal_intel`
> **Result:** Revenue, funding history, competitors, and current buying signals in one briefing

> **User:** "Is 'Mitsubishi UFJ' a legitimate entity? Get me the official legal name and registry details."
> **Agent uses:** `interzoid_official_name`, `interzoid_company_verification`
> **Result:** Official legal name, registry identifier, incorporation country, verification score, and an authoritative source URL

> **User:** "What's the combined sales tax rate at our new Beverly Hills store? And the duty rate for importing laptops into Germany?"
> **Agent uses:** `interzoid_sales_use_tax_rates`, `interzoid_customs_duty_rates` (HS code 8471.30)
> **Result:** Rate breakdowns by jurisdiction, plus MFN duty rates and import VAT

> **User:** "For each city on this list, get me life expectancy, flu vaccination coverage, and diabetes rate."
> **Agent uses:** `interzoid_custom_data` with topic "healthcare data by city" and self-defined output fields
> **Result:** Structured JSON with exactly the fields you asked for, for any subject domain

## Self-Hosting the Remote Server

To host your own remote instance:

```bash
./interzoid-mcp-server -transport http -port 8080
```

The MCP endpoint will be available at `http://localhost:8080/mcp`. Place behind Nginx or a load balancer with HTTPS for production use. Ensure `proxy_buffering off` is set in your Nginx config to support SSE streaming.

## x402 Payment Integration

Interzoid APIs support the [x402 protocol](https://x402.org) for native USDC micropayments. When accessed without an API key:

- **Standard APIs:** 10,000 atomic USDC ($0.01) per call
- **X/Twitter APIs:** 50,000 atomic USDC ($0.05) per call
- **Premium APIs:** 250,000 atomic USDC ($0.25) per call
- **Customs Duty API:** 350,000 atomic USDC ($0.35) per call
- **Network:** Base (EIP-155:8453)
- **Asset:** USDC (0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913)

The `.well-known/x402.json` manifest at `https://api.interzoid.com/.well-known/x402.json` provides full machine-readable discovery for x402 clients.

## Project Structure

```
interzoid-mcp-server/
├── main.go        # Entry point, transport selection (stdio/HTTP)
├── tools.go       # MCP tool registration for all 58 APIs
├── client.go      # HTTP client for calling api.interzoid.com
├── go.mod         # Go module definition
└── README.md      # This file
```

## License

MIT