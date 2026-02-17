package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ============================================================================
// INTERZOID MCP TOOL REGISTRY
// ============================================================================
//
// IMPORTANT: Parameter names here MUST match the actual API query parameter
// names as documented in the Interzoid API request formats. The genericHandler
// passes these names directly as query parameters in the HTTP GET request.
//
// x402 Mode: When INTERZOID_API_KEY is not set, the "license" parameter is
// omitted from requests, triggering the x402 payment flow. The calling
// agent/client handles payment negotiation.
//
// Pricing (x402, USDC on Base):
//   Standard APIs:  $0.0125 per call  (12500 atomic units)
//   Premium APIs:   $0.3125 per call  (312500 atomic units)
//
// Total: 29 APIs (gettechstack excluded for now)
// ============================================================================

// getAPIKey extracts the API key using the following priority:
//   1. Authorization header from the incoming MCP request (remote HTTP transport)
//   2. INTERZOID_API_KEY environment variable (local stdio transport)
//   3. Empty string — triggers x402 payment flow
func getAPIKey(request mcp.CallToolRequest) string {
	// Check for Authorization: Bearer <key> header from the connecting client
	if auth := request.Header.Get("Authorization"); auth != "" {
		// Strip "Bearer " prefix if present
		if len(auth) > 7 && (auth[:7] == "Bearer " || auth[:7] == "bearer ") {
			return auth[7:]
		}
		return auth
	}
	// Fall back to environment variable
	return os.Getenv("INTERZOID_API_KEY")
}

// getArguments safely extracts the arguments map from the request,
// handling different mcp-go versions where Arguments may be
// map[string]interface{} or any.
func getArguments(request mcp.CallToolRequest) map[string]interface{} {
	if request.Params.Arguments == nil {
		return make(map[string]interface{})
	}
	switch args := request.Params.Arguments.(type) {
	case map[string]interface{}:
		return args
	default:
		return make(map[string]interface{})
	}
}

// genericHandler creates a tool handler that calls the Interzoid API.
// paramMap maps MCP tool parameter names to API query parameter names.
// This allows the tool to use descriptive param names while sending the
// correct query param names to the API.
func genericHandler(endpoint string, requiredParams []paramMapping, optionalParams []paramMapping) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		apiKey := getAPIKey(request)
		args := getArguments(request)

		params := make(map[string]string)

		// Required params
		for _, p := range requiredParams {
			raw, ok := args[p.toolName]
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("Missing required parameter: %s", p.toolName)), nil
			}
			val, ok := raw.(string)
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("Parameter %s must be a string", p.toolName)), nil
			}
			params[p.apiName] = val
		}

		// Optional params
		for _, p := range optionalParams {
			raw, ok := args[p.toolName]
			if ok {
				if s, ok := raw.(string); ok && s != "" {
					params[p.apiName] = s
				}
			}
		}

		result, err := callInterzoidAPI(apiKey, endpoint, params)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonBytes)), nil
	}
}

// paramMapping maps a tool-facing parameter name to the actual API query parameter name.
// When they're the same, use same() helper. When different, use mapped().
type paramMapping struct {
	toolName string // name shown to the LLM / MCP client
	apiName  string // actual query parameter name sent to the API
}

func same(name string) paramMapping {
	return paramMapping{toolName: name, apiName: name}
}

func mapped(toolName, apiName string) paramMapping {
	return paramMapping{toolName: toolName, apiName: apiName}
}

// registerAllTools registers every Interzoid API as an MCP tool.
func registerAllTools(s *server.MCPServer) {

	// =====================================================================
	// DATA MATCHING — Similarity Key Generation & Scoring
	// =====================================================================

	// /getcompanymatchadvanced?company=[name]&algorithm=[algo]
	s.AddTool(
		mcp.NewTool("interzoid_company_match_advanced",
			mcp.WithDescription("Generate an advanced AI-powered similarity key for company/organization name matching. Names like 'IBM', 'International Business Machines', 'IBM Corp' produce the same key for deduplication and record linkage. Cost: $0.0125 USDC via x402."),
			mcp.WithString("company", mcp.Required(), mcp.Description("Company or organization name")),
			mcp.WithString("algorithm", mcp.Description("Algorithm variant (optional, e.g. 'ai-deep')")),
		),
		genericHandler("/getcompanymatchadvanced",
			[]paramMapping{same("company")},
			[]paramMapping{same("algorithm")},
		),
	)

	// /getfullnamematch?fullname=[name]
	s.AddTool(
		mcp.NewTool("interzoid_fullname_match",
			mcp.WithDescription("Generate an AI-powered similarity key for individual/person name matching. Handles variations like 'Bob Smith', 'Robert Smith', 'Smith, Robert J.' producing the same key. Cost: $0.0125 USDC via x402."),
			mcp.WithString("fullname", mcp.Required(), mcp.Description("Full individual name")),
		),
		genericHandler("/getfullnamematch",
			[]paramMapping{same("fullname")},
			nil,
		),
	)

	// /getaddressmatchadvanced?address=[addr]&algorithm=[algo]
	s.AddTool(
		mcp.NewTool("interzoid_address_match_advanced",
			mcp.WithDescription("Generate an advanced AI-powered similarity key for US street address matching. Handles unit numbers, directionals, and abbreviations. Cost: $0.0125 USDC via x402."),
			mcp.WithString("address", mcp.Required(), mcp.Description("Street address")),
			mcp.WithString("algorithm", mcp.Description("Algorithm variant (optional)")),
		),
		genericHandler("/getaddressmatchadvanced",
			[]paramMapping{same("address")},
			[]paramMapping{same("algorithm")},
		),
	)

	// /getglobaladdressmatch?address=[addr] (uses same endpoint path but different matching)
	s.AddTool(
		mcp.NewTool("interzoid_global_address_match",
			mcp.WithDescription("Generate an AI-powered similarity key for global/international address matching. Handles international address formats and variations across countries. Cost: $0.0125 USDC via x402."),
			mcp.WithString("address", mcp.Required(), mcp.Description("Full international address string")),
		),
		genericHandler("/getglobaladdressmatch",
			[]paramMapping{same("address")},
			nil,
		),
	)

	// /getproductmatch?product=[name]&algorithm=[algo]
	s.AddTool(
		mcp.NewTool("interzoid_product_match",
			mcp.WithDescription("Generate an AI-powered similarity key for product name matching. Handles variations in product names, model numbers, and descriptions. Cost: $0.0125 USDC via x402."),
			mcp.WithString("product", mcp.Required(), mcp.Description("Product name, description, or model")),
			mcp.WithString("algorithm", mcp.Description("Algorithm variant (optional)")),
		),
		genericHandler("/getproductmatch",
			[]paramMapping{same("product")},
			[]paramMapping{same("algorithm")},
		),
	)

	// /getorgmatchscore?org1=[name1]&org2=[name2]
	s.AddTool(
		mcp.NewTool("interzoid_org_match_score",
			mcp.WithDescription("Compare two organization/company names and receive a match score from 0-100 indicating similarity. Useful for determining if two company names refer to the same entity. Cost: $0.0125 USDC via x402."),
			mcp.WithString("org1", mcp.Required(), mcp.Description("First organization name")),
			mcp.WithString("org2", mcp.Required(), mcp.Description("Second organization name to compare")),
		),
		genericHandler("/getorgmatchscore",
			[]paramMapping{same("org1"), same("org2")},
			nil,
		),
	)

	// /getfullnamematchscore?fullname1=[name1]&fullname2=[name2]
	s.AddTool(
		mcp.NewTool("interzoid_fullname_match_score",
			mcp.WithDescription("Compare two individual/person names and receive a match score from 0-100 indicating similarity. Handles name order, nicknames, and abbreviations. Cost: $0.0125 USDC via x402."),
			mcp.WithString("fullname1", mcp.Required(), mcp.Description("First full name")),
			mcp.WithString("fullname2", mcp.Required(), mcp.Description("Second full name to compare")),
		),
		genericHandler("/getfullnamematchscore",
			[]paramMapping{same("fullname1"), same("fullname2")},
			nil,
		),
	)

	// =====================================================================
	// DATA ENRICHMENT — Premium AI-Powered ($0.3125/call)
	// =====================================================================

	// /getbusinessinfo?lookup=[company]
	s.AddTool(
		mcp.NewTool("interzoid_business_info",
			mcp.WithDescription("Retrieve comprehensive AI-powered business intelligence for a company including industry, revenue, employee counts, and executive info. Premium API. Cost: $0.3125 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Company name, website, or email")),
		),
		genericHandler("/getbusinessinfo",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getparentcompanyinfo?lookup=[company name or domain]
	s.AddTool(
		mcp.NewTool("interzoid_parent_company_info",
			mcp.WithDescription("Retrieve parent company information for a given company or subsidiary. Identifies corporate ownership hierarchies and holding company relationships. Premium API. Cost: $0.3125 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Company name or domain to find parent company for")),
		),
		genericHandler("/getparentcompanyinfo",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getexecutiveprofile?lookup=[company and title]
	s.AddTool(
		mcp.NewTool("interzoid_executive_profile",
			mcp.WithDescription("Retrieve executive profile information for a company including leadership details, roles, and professional background. Premium API. Cost: $0.3125 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Company name and job title (e.g. 'Coinbase CEO')")),
		),
		genericHandler("/getexecutiveprofile",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getrecentnews?topic=[topic]
	s.AddTool(
		mcp.NewTool("interzoid_recent_news",
			mcp.WithDescription("Retrieve recent news and developments for a company or topic. AI-powered aggregation from multiple real-time sources. Premium API. Cost: $0.3125 USDC via x402."),
			mcp.WithString("topic", mcp.Required(), mcp.Description("Company name or topic to get news for")),
		),
		genericHandler("/getrecentnews",
			[]paramMapping{same("topic")},
			nil,
		),
	)

	// /emailtrustscore?lookup=[email address]
	s.AddTool(
		mcp.NewTool("interzoid_email_trust_score",
			mcp.WithDescription("Get an email trust score (0-99) and AI-generated risk analysis. Validates deliverability, identifies disposable addresses, and assesses legitimacy. Premium API. Cost: $0.3125 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Email address to score and validate")),
		),
		genericHandler("/emailtrustscore",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getipprofile?lookup=[ip]
	s.AddTool(
		mcp.NewTool("interzoid_ip_profile",
			mcp.WithDescription("Get comprehensive profile for an IP address including geolocation, ISP, organization, CIDR block, and reputation assessment. Premium API. Cost: $0.3125 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("IPv4 or IPv6 address to profile")),
		),
		genericHandler("/getipprofile",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getphoneprofile?lookup=[phone]
	s.AddTool(
		mcp.NewTool("interzoid_phone_profile",
			mcp.WithDescription("Get profile for a phone number including carrier, line type, geographic location, validation status, and risk assessment. Premium API. Cost: $0.3125 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Phone number to profile")),
		),
		genericHandler("/getphoneprofile",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getcompanyverification?lookup=[company]
	s.AddTool(
		mcp.NewTool("interzoid_company_verification",
			mcp.WithDescription("Verify whether a company exists and get a verification score (0-99) with AI-generated reasoning about legitimacy and credibility. Premium API. Cost: $0.3125 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Company or organization name to verify")),
		),
		genericHandler("/getcompanyverification",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getstockinfo?lookup=[ticker]
	s.AddTool(
		mcp.NewTool("interzoid_stock_info",
			mcp.WithDescription("Get AI-powered stock analysis for a ticker symbol including price, market cap, P/E ratio, EPS, and analyst assessment. Premium API. Cost: $0.3125 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Stock ticker symbol or company name (e.g. 'AAPL', 'COIN')")),
		),
		genericHandler("/getstockinfo",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// =====================================================================
	// DATA STANDARDIZATION ($0.0125/call)
	// =====================================================================

	// /getorgstandard?org=[name]
	s.AddTool(
		mcp.NewTool("interzoid_org_standard",
			mcp.WithDescription("Standardize an organization name to its canonical form. Normalizes abbreviations, suffixes, and formatting (e.g. 'b.o.a.' -> 'Bank of America'). Cost: $0.0125 USDC via x402."),
			mcp.WithString("org", mcp.Required(), mcp.Description("Organization name to standardize")),
		),
		genericHandler("/getorgstandard",
			[]paramMapping{same("org")},
			nil,
		),
	)

	// /getcountrystandard?country=[name]&algorithm=[algo]
	s.AddTool(
		mcp.NewTool("interzoid_country_standard",
			mcp.WithDescription("Standardize a country name to a consistent canonical form. Handles variations like 'Great Britain', 'UK', 'United Kingdom'. Cost: $0.0125 USDC via x402."),
			mcp.WithString("country", mcp.Required(), mcp.Description("Country name to standardize")),
			mcp.WithString("algorithm", mcp.Description("Algorithm variant (optional)")),
		),
		genericHandler("/getcountrystandard",
			[]paramMapping{same("country")},
			[]paramMapping{same("algorithm")},
		),
	)

	// /getcountryinfo?country=[name]&algorithm=ai-medium
	s.AddTool(
		mcp.NewTool("interzoid_country_info",
			mcp.WithDescription("Standardize a country name and return comprehensive info: ISO codes (2/3-letter, 3-digit), currency details, internet code, and calling code. Cost: $0.0125 USDC via x402."),
			mcp.WithString("country", mcp.Required(), mcp.Description("Country name in any language or format")),
			mcp.WithString("algorithm", mcp.Description("Algorithm variant (optional, defaults to 'ai-medium')")),
		),
		genericHandler("/getcountryinfo",
			[]paramMapping{same("country")},
			[]paramMapping{same("algorithm")},
		),
	)

	// /getstateabbreviation?state=[name]&algorithm=[algo]
	s.AddTool(
		mcp.NewTool("interzoid_state_abbreviation",
			mcp.WithDescription("Standardize US state/province names to full name plus abbreviation. Handles 'Calif', 'CA', 'Cal' -> 'California' / 'CA'. Cost: $0.0125 USDC via x402."),
			mcp.WithString("state", mcp.Required(), mcp.Description("State or province name/abbreviation")),
			mcp.WithString("algorithm", mcp.Description("Algorithm variant (optional)")),
		),
		genericHandler("/getstateabbreviation",
			[]paramMapping{same("state")},
			[]paramMapping{same("algorithm")},
		),
	)

	// /getcitystandard?city=[name]&algorithm=[algo]
	s.AddTool(
		mcp.NewTool("interzoid_city_standard",
			mcp.WithDescription("Standardize city name data to a consistent canonical form. Handles abbreviations, alternate spellings, and local variations. Cost: $0.0125 USDC via x402."),
			mcp.WithString("city", mcp.Required(), mcp.Description("City name to standardize")),
			mcp.WithString("algorithm", mcp.Description("Algorithm variant (optional)")),
		),
		genericHandler("/getcitystandard",
			[]paramMapping{same("city")},
			[]paramMapping{same("algorithm")},
		),
	)

	// =====================================================================
	// DATA ENHANCEMENT ($0.0125/call)
	// =====================================================================

	// /getentitytype?data=[text]
	s.AddTool(
		mcp.NewTool("interzoid_entity_type",
			mcp.WithDescription("Determine the entity type of a data value - whether it represents a person, company/organization, location, or other entity type. Cost: $0.0125 USDC via x402."),
			mcp.WithString("data", mcp.Required(), mcp.Description("Text data value to classify")),
		),
		genericHandler("/getentitytype",
			[]paramMapping{same("data")},
			nil,
		),
	)

	// /getgender?name=[first name]
	s.AddTool(
		mcp.NewTool("interzoid_gender",
			mcp.WithDescription("Determine the likely gender associated with an individual name. Supports international names. Cost: $0.0125 USDC via x402."),
			mcp.WithString("name", mcp.Required(), mcp.Description("First name to determine gender for")),
		),
		genericHandler("/getgender",
			[]paramMapping{same("name")},
			nil,
		),
	)

	// /getnameorigin?name=[full name]
	s.AddTool(
		mcp.NewTool("interzoid_name_origin",
			mcp.WithDescription("Determine the likely cultural or geographic origin of an individual name. Useful for demographic analysis and internationalization. Cost: $0.0125 USDC via x402."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Full name to determine origin for")),
		),
		genericHandler("/getnameorigin",
			[]paramMapping{same("name")},
			nil,
		),
	)

	// /identifylanguage?text=[text]
	s.AddTool(
		mcp.NewTool("interzoid_identify_language",
			mcp.WithDescription("Identify the language of a given text string. Supports detection of numerous world languages. Cost: $0.0125 USDC via x402."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Text snippet to identify the language of")),
		),
		genericHandler("/identifylanguage",
			[]paramMapping{same("text")},
			nil,
		),
	)

	// /translatetoenglish?text=[text]
	s.AddTool(
		mcp.NewTool("interzoid_translate_to_english",
			mcp.WithDescription("Detect the language of input text and translate it to English. AI-powered translation supporting numerous world languages. Cost: $0.0125 USDC via x402."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Text in any language to translate to English")),
		),
		genericHandler("/translatetoenglish",
			[]paramMapping{same("text")},
			nil,
		),
	)

	// /translatetoany?text=[text]&to=[target language]
	s.AddTool(
		mcp.NewTool("interzoid_translate_to_any",
			mcp.WithDescription("Detect the language of input text and translate it to any specified target language. Cost: $0.0125 USDC via x402."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Text to translate")),
			mcp.WithString("to", mcp.Required(), mcp.Description("Target language name (e.g. 'Japanese', 'French', 'Spanish')")),
		),
		genericHandler("/translatetoany",
			[]paramMapping{same("text"), same("to")},
			nil,
		),
	)

	// /addressparse?address=[full address]
	s.AddTool(
		mcp.NewTool("interzoid_address_parse",
			mcp.WithDescription("Parse a full address string into component parts: street number, street name, unit, city, state, zip code. Cost: $0.0125 USDC via x402."),
			mcp.WithString("address", mcp.Required(), mcp.Description("Full address string to parse")),
		),
		genericHandler("/addressparse",
			[]paramMapping{same("address")},
			nil,
		),
	)

	// =====================================================================
	// UTILITY ($0.0125/call)
	// =====================================================================

	// /getzipcodeinfo?zip=[zipcode]
	s.AddTool(
		mcp.NewTool("interzoid_zipcode_info",
			mcp.WithDescription("Get detailed info for a US ZIP code: city, state, county, timezone, area codes, latitude/longitude. Cost: $0.0125 USDC via x402."),
			mcp.WithString("zip", mcp.Required(), mcp.Description("US ZIP code (5-digit)")),
		),
		genericHandler("/getzipcodeinfo",
			[]paramMapping{same("zip")},
			nil,
		),
	)

	// /getrates?from=[currency]&to=[currency]
	s.AddTool(
		mcp.NewTool("interzoid_currency_rate",
			mcp.WithDescription("Get live currency exchange rates between two currencies. Returns current mid-market rates. Cost: $0.0125 USDC via x402."),
			mcp.WithString("from", mcp.Required(), mcp.Description("Source currency code (e.g. USD, EUR, GBP)")),
			mcp.WithString("to", mcp.Required(), mcp.Description("Target currency code (e.g. JPY, GBP, EUR)")),
		),
		genericHandler("/getrates",
			[]paramMapping{same("from"), same("to")},
			nil,
		),
	)

	// /getglobalweather?location=[city name]
	s.AddTool(
		mcp.NewTool("interzoid_global_weather",
			mcp.WithDescription("Get current weather for any city worldwide including temperature (F/C), conditions, and wind speed. Cost: $0.0125 USDC via x402."),
			mcp.WithString("location", mcp.Required(), mcp.Description("City name (e.g. 'London', 'Tokyo', 'San Francisco')")),
		),
		genericHandler("/getglobalweather",
			[]paramMapping{same("location")},
			nil,
		),
	)
}
