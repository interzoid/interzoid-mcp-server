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
// names as documented in the Interzoid API request formats and the x402
// manifest at https://api.interzoid.com/.well-known/x402.json. The
// genericHandler passes these names directly as query parameters in the
// HTTP GET request.
//
// x402 Mode: When no API key is available (no Authorization header and no
// INTERZOID_API_KEY environment variable), requests are sent without
// authentication, triggering the x402 payment flow. The calling agent/client
// handles payment negotiation.
//
// Pricing (x402, USDC on Base, per the live manifest):
//   Standard APIs:      $0.01 per call   (10000 atomic units)
//   X/Twitter APIs:     $0.05 per call   (50000 atomic units)
//   Premium APIs:       $0.25 per call   (250000 atomic units)
//   Customs Duty API:   $0.35 per call   (350000 atomic units)
//
// Total: 58 tools
// ============================================================================

// getAPIKey extracts the API key using the following priority:
//  1. Authorization header from the incoming MCP request (remote HTTP transport)
//  2. INTERZOID_API_KEY environment variable (local stdio transport)
//  3. Empty string - triggers x402 payment flow
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
	// DATA MATCHING - Similarity Key Generation & Scoring ($0.01/call)
	// =====================================================================

	// /getcompanymatchadvanced?company=[name]&algorithm=[algo]
	s.AddTool(
		mcp.NewTool("interzoid_company_match_advanced",
			mcp.WithDescription("Generate an advanced AI-powered similarity key for company/organization name matching. Names like 'IBM', 'International Business Machines', 'IBM Corp' produce the same key for deduplication and record linkage. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Generate an AI-powered similarity key for individual/person name matching. Handles variations like 'Bob Smith', 'Robert Smith', 'Smith, Robert J.' producing the same key. Cost: $0.01 USDC via x402."),
			mcp.WithString("fullname", mcp.Required(), mcp.Description("Full individual name")),
		),
		genericHandler("/getfullnamematch",
			[]paramMapping{same("fullname")},
			nil,
		),
	)

	// /getaddressmatchadvanced?address=[addr]&zip=[zip]&algorithm=[algo]
	s.AddTool(
		mcp.NewTool("interzoid_address_match_advanced",
			mcp.WithDescription("Generate an advanced AI-powered similarity key for US street address matching. Handles unit numbers, directionals, and abbreviations. Cost: $0.01 USDC via x402."),
			mcp.WithString("address", mcp.Required(), mcp.Description("Street address")),
			mcp.WithString("zip", mcp.Description("Zip code (optional but recommended for precision)")),
			mcp.WithString("algorithm", mcp.Description("Algorithm variant (optional)")),
		),
		genericHandler("/getaddressmatchadvanced",
			[]paramMapping{same("address")},
			[]paramMapping{same("zip"), same("algorithm")},
		),
	)

	// /getglobaladdressmatch?address=[addr]
	s.AddTool(
		mcp.NewTool("interzoid_global_address_match",
			mcp.WithDescription("Generate an AI-powered similarity key for global/international address matching. Handles international address formats and variations across countries. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Generate an AI-powered similarity key for product name matching. Handles variations in product names, model numbers, and descriptions. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Compare two organization/company names and receive a match score from 0-100 indicating similarity. Useful for determining if two company names refer to the same entity. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Compare two individual/person names and receive a match score from 0-100 indicating similarity. Handles name order, nicknames, and abbreviations. Cost: $0.01 USDC via x402."),
			mcp.WithString("fullname1", mcp.Required(), mcp.Description("First full name")),
			mcp.WithString("fullname2", mcp.Required(), mcp.Description("Second full name to compare")),
		),
		genericHandler("/getfullnamematchscore",
			[]paramMapping{same("fullname1"), same("fullname2")},
			nil,
		),
	)

	// /getaddressandfullnamematch?address=[addr]&fullname=[name]
	s.AddTool(
		mcp.NewTool("interzoid_address_and_fullname_match",
			mcp.WithDescription("Generate a single combined AI-powered similarity key from a full name plus address. Records with the same combined key share both a similar address and a similar individual name. Useful for high-precision household/contact-level deduplication. Cost: $0.01 USDC via x402."),
			mcp.WithString("address", mcp.Required(), mcp.Description("Full street address (US or international)")),
			mcp.WithString("fullname", mcp.Required(), mcp.Description("Full individual name")),
		),
		genericHandler("/getaddressandfullnamematch",
			[]paramMapping{same("address"), same("fullname")},
			nil,
		),
	)

	// /getcompanyandaddressmatch?company=[name]&address=[addr]
	s.AddTool(
		mcp.NewTool("interzoid_company_and_address_match",
			mcp.WithDescription("Generate a single combined AI-powered similarity key from a company/organization name plus a street address. Records with the same combined key likely share both a similar company name and address. Ideal for business-location deduplication across vendor, supplier, and account datasets. Cost: $0.01 USDC via x402."),
			mcp.WithString("company", mcp.Required(), mcp.Description("Company or organization name")),
			mcp.WithString("address", mcp.Required(), mcp.Description("Full street address (US or international)")),
		),
		genericHandler("/getcompanyandaddressmatch",
			[]paramMapping{same("company"), same("address")},
			nil,
		),
	)

	// /getcompanyandfullnamematch?company=[name]&fullname=[name]
	s.AddTool(
		mcp.NewTool("interzoid_company_and_fullname_match",
			mcp.WithDescription("Generate a single combined AI-powered similarity key from a company/organization name plus an individual's full name. Records with the same combined key likely share both a similar company and a similar person. Ideal for contact-level deduplication within CRM, marketing, and sales datasets. Cost: $0.01 USDC via x402."),
			mcp.WithString("company", mcp.Required(), mcp.Description("Company or organization name")),
			mcp.WithString("fullname", mcp.Required(), mcp.Description("Full individual name")),
		),
		genericHandler("/getcompanyandfullnamematch",
			[]paramMapping{same("company"), same("fullname")},
			nil,
		),
	)

	// =====================================================================
	// DATA ENRICHMENT - Company Intelligence (Premium, $0.25/call)
	// =====================================================================

	// /getbusinessinfo?lookup=[company]
	s.AddTool(
		mcp.NewTool("interzoid_business_info",
			mcp.WithDescription("Retrieve comprehensive AI-powered business intelligence for a company including industry, revenue, employee counts, and executive info. Premium API. Cost: $0.25 USDC via x402."),
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
			mcp.WithDescription("Retrieve parent company information for a given company or subsidiary. Identifies corporate ownership hierarchies and holding company relationships. Premium API. Cost: $0.25 USDC via x402."),
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
			mcp.WithDescription("Retrieve executive profile information for a company including leadership details, roles, and professional background. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Company name and job title (e.g. 'Coinbase CEO')")),
		),
		genericHandler("/getexecutiveprofile",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getrecentnews?lookup=[topic]
	s.AddTool(
		mcp.NewTool("interzoid_recent_news",
			mcp.WithDescription("Retrieve recent news and developments for a company or topic. AI-powered aggregation from multiple real-time sources. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Company name or topic to get news for")),
		),
		genericHandler("/getrecentnews",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getcompanyverification?lookup=[company]
	s.AddTool(
		mcp.NewTool("interzoid_company_verification",
			mcp.WithDescription("Verify whether a company exists and get a verification score (0-99) with AI-generated reasoning about legitimacy and credibility. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Company or organization name to verify")),
		),
		genericHandler("/getcompanyverification",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getindustrycodes?lookup=[company]
	s.AddTool(
		mcp.NewTool("interzoid_industry_classification",
			mcp.WithDescription("Identify and categorize a company's primary industry classification with standardized NAICS and SIC codes, sector and subsector details, business activity description, and a confidence score. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Company name to classify")),
		),
		genericHandler("/getindustrycodes",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getcompetitoranalysis?lookup=[company]
	s.AddTool(
		mcp.NewTool("interzoid_competitor_analysis",
			mcp.WithDescription("Identify and analyze a company's primary competitors with detailed comparison information including market positioning, products, and competitive advantages. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Company name to analyze competitors for")),
		),
		genericHandler("/getcompetitoranalysis",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getbuyingsignals?lookup=[company]
	s.AddTool(
		mcp.NewTool("interzoid_buying_signals",
			mcp.WithDescription("Identify recent buying signals for a target company including funding events, leadership changes, expansion announcements, hiring trends, and other intent signals useful for sales targeting. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Target company name to surface buying signals for")),
		),
		genericHandler("/getbuyingsignals",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getprivatecompanydealintel?lookup=[company]
	s.AddTool(
		mcp.NewTool("interzoid_private_company_deal_intel",
			mcp.WithDescription("Retrieve private company deal intelligence including funding rounds, valuations, investors, M&A activity, and ownership changes. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Private company name")),
		),
		genericHandler("/getprivatecompanydealintel",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getesgprofile?lookup=[company]
	s.AddTool(
		mcp.NewTool("interzoid_esg_profile",
			mcp.WithDescription("Retrieve an ESG (Environmental, Social, Governance) and sustainability profile for a company including key initiatives, ratings, controversies, and reporting frameworks. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Company name to retrieve ESG profile for")),
		),
		genericHandler("/getesgprofile",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getgovcontracts?lookup=[company]
	s.AddTool(
		mcp.NewTool("interzoid_gov_contracts",
			mcp.WithDescription("Retrieve US government contract information for a company including active contracts, agencies, contract values, and award history. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Company name to retrieve government contract data for")),
		),
		genericHandler("/getgovcontracts",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getfacilitiesprofile?lookup=[company]
	s.AddTool(
		mcp.NewTool("interzoid_facilities_profile",
			mcp.WithDescription("Retrieve a profile of a company's corporate facilities including headquarters, regional offices, manufacturing sites, data centers, and other key locations. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Company name to retrieve facilities profile for")),
		),
		genericHandler("/getfacilitiesprofile",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /gettechstack?lookup=[domain]
	s.AddTool(
		mcp.NewTool("interzoid_tech_stack",
			mcp.WithDescription("Identify the complete technology stack used by a company or website from a domain lookup, including CMS, frontend frameworks, backend technologies, web server, hosting provider, CDN, analytics, security technologies, and e-commerce platform. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Domain name or company website to identify the tech stack for")),
		),
		genericHandler("/gettechstack",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getofficialname?lookup=[organization]
	s.AddTool(
		mcp.NewTool("interzoid_official_name",
			mcp.WithDescription("Resolve any organization name (common name, brand, abbreviation, ticker, local-language name, or near-match with typos) to its official English legal name, commercial name, local-language name, legal form, country of incorporation, registry identifier, status, and an authoritative documentation URL. Designed for KYB, supplier verification, sanctions screening, and dataset normalization. Premium API."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Organization name in any form (common name, brand, abbreviation, ticker, or local-language name)")),
		),
		genericHandler("/getofficialname",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getcustom?topic=[desc]&lookup=[value]&output=[cols]&model=[model]
	s.AddTool(
		mcp.NewTool("interzoid_custom_data",
			mcp.WithDescription("AI Custom, Self-Defined Data Enrichment. Define your own data API: describe a topic, provide a lookup value, and specify the output fields you want. Returns real-world data as a JSON object with your self-defined fields. Example: topic='detailed information about companies', lookup='IBM', output='[\"headquarters\",\"ceo\",\"website\"]'. Premium API, cost varies by model."),
			mcp.WithString("topic", mcp.Required(), mcp.Description("Description of the data domain (e.g. 'detailed information about companies', 'healthcare data by city')")),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("The lookup value to retrieve data for (e.g. 'IBM', 'Las Vegas', '85250')")),
			mcp.WithString("output", mcp.Required(), mcp.Description("JSON array of desired output field names (e.g. '[\"headquarters\",\"ceo\",\"website\"]')")),
			mcp.WithString("model", mcp.Description("AI model to use (optional: 'default', 'model-a', 'model-a-premium', 'model-x')")),
		),
		genericHandler("/getcustom",
			[]paramMapping{same("topic"), same("lookup"), same("output")},
			[]paramMapping{same("model")},
		),
	)

	// =====================================================================
	// DATA ENRICHMENT - Contact & Identity (Premium, $0.25/call)
	// =====================================================================

	// /emailtrustscore?lookup=[email address]
	s.AddTool(
		mcp.NewTool("interzoid_email_trust_score",
			mcp.WithDescription("Get an email trust score (0-99) and AI-generated risk analysis. Validates deliverability, identifies disposable addresses, and assesses legitimacy. Premium API. Cost: $0.25 USDC via x402."),
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
			mcp.WithDescription("Get comprehensive profile for an IP address including geolocation, ISP, organization, CIDR block, and reputation assessment. Premium API. Cost: $0.25 USDC via x402."),
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
			mcp.WithDescription("Get profile for a phone number including carrier, line type, geographic location, validation status, and risk assessment. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Phone number to profile")),
		),
		genericHandler("/getphoneprofile",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// =====================================================================
	// DATA ENRICHMENT - Financial & Market (Premium, $0.25/call)
	// =====================================================================

	// /getstockinfo?lookup=[ticker]
	s.AddTool(
		mcp.NewTool("interzoid_stock_info",
			mcp.WithDescription("Get AI-powered stock analysis for a ticker symbol including price, market cap, P/E ratio, EPS, and analyst assessment. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Stock ticker symbol or company name (e.g. 'AAPL', 'COIN')")),
		),
		genericHandler("/getstockinfo",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getmuniissuerprofile?lookup=[issuer]
	s.AddTool(
		mcp.NewTool("interzoid_muni_issuer_profile",
			mcp.WithDescription("Retrieve a research profile on any US municipal securities issuer (state, city, county, school district, transit/water/sewer authority, hospital district, public university, etc.) including issuer type, credit standing, typical security types, outstanding debt, notable events, pension/OPEB exposure, revenue sources, and EMMA reference link. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Municipal issuer name (state, city, county, district, authority, university, etc.)")),
		),
		genericHandler("/getmuniissuerprofile",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// =====================================================================
	// DATA ENRICHMENT - Property & Location (Premium, $0.25/call)
	// =====================================================================

	// /getbuildingprofile?address=[address]
	s.AddTool(
		mcp.NewTool("interzoid_building_profile",
			mcp.WithDescription("Retrieve a commercial building profile for any US property address including building class, year built, total rentable square footage, floor count, primary tenants, property management, ownership entity, recent renovations, parking ratio, and transit access. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("address", mcp.Required(), mcp.Description("US property street address")),
		),
		genericHandler("/getbuildingprofile",
			[]paramMapping{same("address")},
			nil,
		),
	)

	// /getpropertyhistory?address=[address]
	s.AddTool(
		mcp.NewTool("interzoid_property_history",
			mcp.WithDescription("Retrieve property transaction history for any US address including last sale date and price, prior transactions over up to ten years, price-per-square-foot trajectory, ownership chain, recorded deed types, mortgage history, and current owner of record. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("address", mcp.Required(), mcp.Description("US property street address")),
		),
		genericHandler("/getpropertyhistory",
			[]paramMapping{same("address")},
			nil,
		),
	)

	// /getnearestcoffeeshops?lookup=[address]
	s.AddTool(
		mcp.NewTool("interzoid_nearest_coffee_shops",
			mcp.WithDescription("Find nearest coffee shops to a given address or location including names, addresses, distances, and basic details. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Reference address or location to find coffee shops near")),
		),
		genericHandler("/getnearestcoffeeshops",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// =====================================================================
	// DATA ENRICHMENT - Tax & Regulatory (Premium)
	// =====================================================================

	// /getsalesusetaxrates?lookup=[address/zip/city/state]
	s.AddTool(
		mcp.NewTool("interzoid_sales_use_tax_rates",
			mcp.WithDescription("Retrieve current US sales and use tax rates for any address, ZIP code, city, or state - fully decomposed into state, county, city, and special district components plus the combined effective rate. Submit a full street address for the most accurate rate assignment. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Full street address, ZIP code, city, or state")),
		),
		genericHandler("/getsalesusetaxrates",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getirsperdiemrates?lookup=[address/zip/city/county/state]
	s.AddTool(
		mcp.NewTool("interzoid_irs_per_diem_rates",
			mcp.WithDescription("Retrieve current IRS high-low method per diem rates for any US location to support accountable-plan travel reimbursement (Revenue Procedure 2019-48). Classifies each location as High-Cost or Low-Cost with separate lodging and meals-and-incidentals (M&IE) components. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Full street address, ZIP code, city, county, or state (CONUS)")),
		),
		genericHandler("/getirsperdiemrates",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /geteuvatrates?lookup=[country]
	s.AddTool(
		mcp.NewTool("interzoid_eu_vat_rates",
			mcp.WithDescription("Retrieve current VAT rates for any European country (EU-27, EEA, UK, Switzerland, non-EU Europe) with category-driven breakdown of standard, reduced, super-reduced, zero-rated, and exempt categories plus registration thresholds and territorial exceptions. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Country name, ISO code, city, or address")),
		),
		genericHandler("/geteuvatrates",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getcustomsdutyrates?lookup=[country]&hscode=[hs code]
	s.AddTool(
		mcp.NewTool("interzoid_customs_duty_rates",
			mcp.WithDescription("Retrieve import customs duty rates for any HS code entering any country - with MFN applied rates, preferential FTA rates, import VAT/GST, additional duties (anti-dumping, safeguard, excise), de minimis thresholds, and non-tariff restrictions. Premium API. Cost: $0.35 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Destination country name or ISO code")),
			mcp.WithString("hscode", mcp.Required(), mcp.Description("Harmonized System (HS) code, 6-digit minimum (up to 10-digit national extensions)")),
		),
		genericHandler("/getcustomsdutyrates",
			[]paramMapping{same("lookup"), same("hscode")},
			nil,
		),
	)

	// =====================================================================
	// DATA ENRICHMENT - Research & Lookup (Premium, $0.25/call)
	// =====================================================================

	// /getuniversityinfo?lookup=[university]
	s.AddTool(
		mcp.NewTool("interzoid_university_info",
			mcp.WithDescription("Retrieve detailed information about a university or college including location, type, enrollment, accreditation, notable programs, and key statistics. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("University or college name")),
		),
		genericHandler("/getuniversityinfo",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getproductrecallinfo?lookup=[product]
	s.AddTool(
		mcp.NewTool("interzoid_product_recall_info",
			mcp.WithDescription("Retrieve product recall information for a given product, brand, or category including recall date, hazard description, remedy, and regulatory authority. Premium API. Cost: $0.25 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Product name, brand, or category to check for recalls")),
		),
		genericHandler("/getproductrecallinfo",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// =====================================================================
	// DATA ENRICHMENT - X/Twitter ($0.05/call)
	// =====================================================================

	// /getuserxhandle?lookup=[entity]
	s.AddTool(
		mcp.NewTool("interzoid_x_handle",
			mcp.WithDescription("Retrieve the X (formerly Twitter) handle for a given person, organization, sports team, government entity, or descriptive phrase using AI-powered lookup. Returns the handle with a confidence score. Cost: $0.05 USDC via x402."),
			mcp.WithString("lookup", mcp.Required(), mcp.Description("Person, organization, team, title, or descriptive phrase to find an X handle for")),
		),
		genericHandler("/getuserxhandle",
			[]paramMapping{same("lookup")},
			nil,
		),
	)

	// /getxuserprofile?handle=[handle]
	s.AddTool(
		mcp.NewTool("interzoid_x_profile",
			mcp.WithDescription("Retrieve a comprehensive profile snapshot for an X (Twitter) handle including display name, bio description, follower and following counts, total post count, website URL, location, and account creation date. Cost: $0.05 USDC via x402."),
			mcp.WithString("handle", mcp.Required(), mcp.Description("X handle (with or without @ prefix)")),
		),
		genericHandler("/getxuserprofile",
			[]paramMapping{same("handle")},
			nil,
		),
	)

	// /getxlast3?handle=[handle]
	s.AddTool(
		mcp.NewTool("interzoid_x_last_three_posts",
			mcp.WithDescription("Retrieve the three most recent posts (tweets) from a given X (Twitter) handle, including the full text content and date of each post. Cost: $0.05 USDC via x402."),
			mcp.WithString("handle", mcp.Required(), mcp.Description("X handle (with or without @ prefix)")),
		),
		genericHandler("/getxlast3",
			[]paramMapping{same("handle")},
			nil,
		),
	)

	// =====================================================================
	// DATA STANDARDIZATION ($0.01/call)
	// =====================================================================

	// /getorgstandard?org=[name]
	s.AddTool(
		mcp.NewTool("interzoid_org_standard",
			mcp.WithDescription("Standardize an organization name to its canonical form. Normalizes abbreviations, suffixes, and formatting (e.g. 'b.o.a.' -> 'Bank of America'). Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Standardize a country name to a consistent canonical form. Handles variations like 'Great Britain', 'UK', 'United Kingdom'. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Standardize a country name and return comprehensive info: ISO codes (2/3-letter, 3-digit), currency details, internet code, and calling code. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Standardize US state/province names to full name plus abbreviation. Handles 'Calif', 'CA', 'Cal' -> 'California' / 'CA'. Cost: $0.01 USDC via x402."),
			mcp.WithString("state", mcp.Required(), mcp.Description("State or province name/abbreviation")),
			mcp.WithString("algorithm", mcp.Description("Algorithm variant (optional)")),
		),
		genericHandler("/getstateabbreviation",
			[]paramMapping{same("state")},
			[]paramMapping{same("algorithm")},
		),
	)

	// /getcitystandard?city=[name]&state=[state]&algorithm=[algo]
	s.AddTool(
		mcp.NewTool("interzoid_city_standard",
			mcp.WithDescription("Standardize city name data to a consistent canonical form. Handles abbreviations, alternate spellings, and local variations. Cost: $0.01 USDC via x402."),
			mcp.WithString("city", mcp.Required(), mcp.Description("City name to standardize")),
			mcp.WithString("state", mcp.Description("State or province (optional but recommended)")),
			mcp.WithString("algorithm", mcp.Description("Algorithm variant (optional)")),
		),
		genericHandler("/getcitystandard",
			[]paramMapping{same("city")},
			[]paramMapping{same("state"), same("algorithm")},
		),
	)

	// =====================================================================
	// DATA ENHANCEMENT ($0.01/call)
	// =====================================================================

	// /getentitytype?text=[text]
	s.AddTool(
		mcp.NewTool("interzoid_entity_type",
			mcp.WithDescription("Determine the entity type of a data value - whether it represents a person, company/organization, location, or other entity type. Cost: $0.01 USDC via x402."),
			mcp.WithString("text", mcp.Required(), mcp.Description("Text data value to classify")),
		),
		genericHandler("/getentitytype",
			[]paramMapping{same("text")},
			nil,
		),
	)

	// /getgender?name=[first name]
	s.AddTool(
		mcp.NewTool("interzoid_gender",
			mcp.WithDescription("Determine the likely gender associated with an individual name. Supports international names. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Determine the likely cultural or geographic origin of an individual name. Useful for demographic analysis and internationalization. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Identify the language of a given text string. Supports detection of numerous world languages. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Detect the language of input text and translate it to English. AI-powered translation supporting numerous world languages. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Detect the language of input text and translate it to any specified target language. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Parse a full address string into component parts: street number, street name, unit, city, state, zip code. Cost: $0.01 USDC via x402."),
			mcp.WithString("address", mcp.Required(), mcp.Description("Full address string to parse")),
		),
		genericHandler("/addressparse",
			[]paramMapping{same("address")},
			nil,
		),
	)

	// =====================================================================
	// UTILITY ($0.01/call)
	// =====================================================================

	// /getzipcodeinfo?zip=[zipcode]
	s.AddTool(
		mcp.NewTool("interzoid_zipcode_info",
			mcp.WithDescription("Get detailed info for a US ZIP code: city, state, county, timezone, area codes, latitude/longitude. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Get live currency exchange rates between two currencies. Returns current mid-market rates. Cost: $0.01 USDC via x402."),
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
			mcp.WithDescription("Get current weather for any city worldwide including temperature (F/C), conditions, and wind speed. Cost: $0.01 USDC via x402."),
			mcp.WithString("location", mcp.Required(), mcp.Description("City name (e.g. 'London', 'Tokyo', 'San Francisco')")),
		),
		genericHandler("/getglobalweather",
			[]paramMapping{same("location")},
			nil,
		),
	)

	// /globallatency?url=[url]&origin=[location]
	s.AddTool(
		mcp.NewTool("interzoid_global_latency",
			mcp.WithDescription("Measure real-world page load performance (response time) for any URL from a chosen global location. Returns total load time in seconds, HTTP response status, a phase-by-phase timing breakdown (DNS, TCP connect, TLS handshake, time-to-first-byte, content transfer), and a content preview. Useful for global latency monitoring, CDN validation, and uptime/performance checks. Cost: $0.01 USDC via x402."),
			mcp.WithString("url", mcp.Required(), mcp.Description("The full URL to measure (https:// is assumed if no scheme is given)")),
			mcp.WithString("origin", mcp.Description("Measurement location (optional: 'California', 'London', 'Tokyo', 'Frankfurt', 'Singapore', 'Sydney', 'Virginia', 'Sao Paulo'; defaults to California)")),
		),
		genericHandler("/globallatency",
			[]paramMapping{same("url")},
			[]paramMapping{same("origin")},
		),
	)
}
