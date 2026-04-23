package ocr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"backend/internal/config"

	genai "google.golang.org/genai"
)

type Provider interface {
	Extract(ctx context.Context, artifactPath string, contentType string) (ExtractedDocument, error)
	SuggestLineResolution(ctx context.Context, input LineResolutionInput) (OCRLineAssistSuggestion, error)
	Name() string
}

type LineResolutionInput struct {
	SupplierName string
	Line         OCRResultLine
	Candidates   []OCRItemCandidate
	Categories   []string
}

type MockProvider struct {
	name string
}

func NewMockProvider(name string) *MockProvider {
	if name == "" {
		name = "mock"
	}
	return &MockProvider{name: name}
}

func (p *MockProvider) Extract(_ context.Context, artifactPath string, _ string) (ExtractedDocument, error) {
	base := strings.TrimSuffix(filepath.Base(artifactPath), filepath.Ext(artifactPath))
	return ExtractedDocument{
		SupplierName:    "MISUMI",
		SupplierID:      "sup-misumi",
		QuotationNumber: strings.ToUpper(base),
		IssueDate:       "2026-04-22",
		RawPayload:      `{"provider":"mock","confidence":"draft"}`,
		Lines: []OCRResultLine{
			{
				ID:               "draft-line-1",
				ItemID:           "item-er2",
				ManufacturerName: "Omron",
				ItemNumber:       "ER2-P4",
				ItemDescription:  "Control relay pack of 4",
				Quantity:         12,
				LeadTimeDays:     14,
			},
			{
				ID:               "draft-line-2",
				ItemID:           "item-mk44",
				ManufacturerName: "Phoenix Contact",
				ItemNumber:       "MK44-BX",
				ItemDescription:  "Terminal block bulk box",
				Quantity:         8,
				LeadTimeDays:     10,
			},
		},
	}, nil
}

func (p *MockProvider) SuggestLineResolution(_ context.Context, input LineResolutionInput) (OCRLineAssistSuggestion, error) {
	suggestion := OCRLineAssistSuggestion{
		LineID:                   input.Line.ID,
		SuggestedCanonicalNumber: input.Line.ItemNumber,
		SuggestedManufacturer:    input.Line.ManufacturerName,
		SuggestedCategoryKey:     "misc",
		SuggestedAliasNumber:     input.Line.ItemNumber,
		Confidence:               0.4,
		Rationale:                "mock suggestion generated from OCR line fields",
		Candidates:               input.Candidates,
	}
	if len(input.Candidates) > 0 {
		suggestion.MatchedItemID = input.Candidates[0].ItemID
		suggestion.Confidence = input.Candidates[0].Score
		suggestion.Rationale = input.Candidates[0].MatchReason
	}
	return suggestion, nil
}

func (p *MockProvider) Name() string {
	return p.name
}

type VertexAIProvider struct {
	projectID string
	location  string
	model     string
}

func NewVertexAIProvider(cfg config.OCRConfig) (*VertexAIProvider, error) {
	if cfg.GoogleCloudProject == "" {
		return nil, fmt.Errorf("GOOGLE_CLOUD_PROJECT is required for vertex_ai OCR provider")
	}
	if cfg.VertexAILocation == "" {
		return nil, fmt.Errorf("VERTEX_AI_LOCATION is required for vertex_ai OCR provider")
	}
	if cfg.GeminiModel == "" {
		return nil, fmt.Errorf("GEMINI_MODEL is required for vertex_ai OCR provider")
	}
	return &VertexAIProvider{
		projectID: cfg.GoogleCloudProject,
		location:  resolveVertexLocation(cfg.GeminiModel, cfg.VertexAILocation),
		model:     cfg.GeminiModel,
	}, nil
}

func (p *VertexAIProvider) Name() string {
	return "vertex_ai"
}

func (p *VertexAIProvider) Extract(ctx context.Context, artifactPath string, contentType string) (ExtractedDocument, error) {
	data, err := os.ReadFile(artifactPath)
	if err != nil {
		return ExtractedDocument{}, fmt.Errorf("read artifact: %w", err)
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:     p.projectID,
		Location:    p.location,
		Backend:     genai.BackendVertexAI,
		HTTPOptions: genai.HTTPOptions{APIVersion: "v1"},
	})
	if err != nil {
		return ExtractedDocument{}, fmt.Errorf("create vertex ai client: %w", err)
	}

	prompt := `You are an OCR extraction service for procurement quotations.
Return only valid JSON with this exact shape:
{
  "supplier_name": "string or empty",
  "supplier_id": "string or empty",
  "quotation_number": "string",
  "issue_date": "YYYY-MM-DD or empty",
  "raw_payload": {"provider":"vertex_ai","notes":"short summary","supplier_name":"string or empty"},
  "lines": [
    {
      "manufacturer_name": "string",
      "item_number": "string",
      "item_description": "string",
      "quantity": 1,
      "lead_time_days": 0
    }
  ]
}

Rules:
- Do not wrap the JSON in markdown.
- Keep unknown strings empty.
- quantity must be a positive integer.
- lead_time_days must be a non-negative integer.
- Extract line items from the document.`

	contents := []*genai.Content{
		{
			Role: genai.RoleUser,
			Parts: []*genai.Part{
				{Text: prompt},
				{InlineData: &genai.Blob{
					MIMEType: contentType,
					Data:     data,
				}},
			},
		},
	}

	resp, err := client.Models.GenerateContent(ctx, p.model, contents, nil)
	if err != nil {
		return ExtractedDocument{}, fmt.Errorf("generate content with vertex ai: %w", err)
	}

	parsed, err := parseVertexAIResponse(resp.Text())
	if err != nil {
		return ExtractedDocument{}, err
	}

	return ExtractedDocument{
		SupplierName:    parsed.SupplierName,
		SupplierID:      parsed.SupplierID,
		QuotationNumber: parsed.QuotationNumber,
		IssueDate:       parsed.IssueDate,
		RawPayload:      parsed.RawPayload,
		Lines:           parsed.Lines,
	}, nil
}

func (p *VertexAIProvider) SuggestLineResolution(ctx context.Context, input LineResolutionInput) (OCRLineAssistSuggestion, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		Project:     p.projectID,
		Location:    p.location,
		Backend:     genai.BackendVertexAI,
		HTTPOptions: genai.HTTPOptions{APIVersion: "v1"},
	})
	if err != nil {
		return OCRLineAssistSuggestion{}, fmt.Errorf("create vertex ai client: %w", err)
	}

	candidateJSON, _ := json.Marshal(input.Candidates)
	categoryJSON, _ := json.Marshal(input.Categories)
	lineJSON, _ := json.Marshal(input.Line)
	prompt := fmt.Sprintf(`You help normalize OCR procurement lines to an internal item master.
Return only valid JSON with this exact shape:
{
  "matched_item_id": "string or empty",
  "suggested_canonical_number": "string",
  "suggested_manufacturer": "string",
  "suggested_category_key": "string",
  "suggested_alias_number": "string",
  "confidence": 0.0,
  "rationale": "short explanation"
}

Rules:
- Prefer matched_item_id only if one candidate is clearly the same item.
- If no candidate is strong enough, leave matched_item_id empty and propose a new canonical number and alias.
- suggested_category_key must be chosen from the provided category keys when possible.
- confidence must be between 0 and 1.
- Do not wrap the JSON in markdown.

Supplier: %s
OCR line:
%s
Existing candidates:
%s
Available category keys:
%s`, input.SupplierName, string(lineJSON), string(candidateJSON), string(categoryJSON))

	resp, err := client.Models.GenerateContent(ctx, p.model, []*genai.Content{
		{
			Role:  genai.RoleUser,
			Parts: []*genai.Part{{Text: prompt}},
		},
	}, nil)
	if err != nil {
		return OCRLineAssistSuggestion{}, fmt.Errorf("generate assist suggestion with vertex ai: %w", err)
	}

	type assistPayload struct {
		MatchedItemID            string  `json:"matched_item_id"`
		SuggestedCanonicalNumber string  `json:"suggested_canonical_number"`
		SuggestedManufacturer    string  `json:"suggested_manufacturer"`
		SuggestedCategoryKey     string  `json:"suggested_category_key"`
		SuggestedAliasNumber     string  `json:"suggested_alias_number"`
		Confidence               float64 `json:"confidence"`
		Rationale                string  `json:"rationale"`
	}

	var payload assistPayload
	if err := json.Unmarshal([]byte(extractJSONBlock(resp.Text())), &payload); err != nil {
		return OCRLineAssistSuggestion{}, fmt.Errorf("parse assist suggestion json: %w", err)
	}
	if payload.Confidence < 0 {
		payload.Confidence = 0
	}
	if payload.Confidence > 1 {
		payload.Confidence = 1
	}

	return OCRLineAssistSuggestion{
		LineID:                   input.Line.ID,
		MatchedItemID:            payload.MatchedItemID,
		SuggestedCanonicalNumber: payload.SuggestedCanonicalNumber,
		SuggestedManufacturer:    payload.SuggestedManufacturer,
		SuggestedCategoryKey:     payload.SuggestedCategoryKey,
		SuggestedAliasNumber:     payload.SuggestedAliasNumber,
		Confidence:               payload.Confidence,
		Rationale:                payload.Rationale,
		Candidates:               input.Candidates,
	}, nil
}

type vertexAIResponse struct {
	SupplierName    string          `json:"supplier_name"`
	SupplierID      string          `json:"supplier_id"`
	QuotationNumber string          `json:"quotation_number"`
	IssueDate       string          `json:"issue_date"`
	RawPayload      json.RawMessage `json:"raw_payload"`
	Lines           []vertexAILine  `json:"lines"`
}

type vertexAILine struct {
	ManufacturerName string `json:"manufacturer_name"`
	ItemNumber       string `json:"item_number"`
	ItemDescription  string `json:"item_description"`
	Quantity         int    `json:"quantity"`
	LeadTimeDays     int    `json:"lead_time_days"`
}

func parseVertexAIResponse(text string) (ExtractedDocument, error) {
	candidate := extractJSONBlock(text)
	if candidate == "" {
		return ExtractedDocument{}, fmt.Errorf("vertex ai returned an empty OCR response")
	}

	var payload vertexAIResponse
	if err := json.Unmarshal([]byte(candidate), &payload); err != nil {
		return ExtractedDocument{}, fmt.Errorf("parse vertex ai OCR json: %w", err)
	}

	lines := make([]OCRResultLine, 0, len(payload.Lines))
	for _, line := range payload.Lines {
		if line.Quantity <= 0 {
			continue
		}
		lines = append(lines, OCRResultLine{
			ManufacturerName: line.ManufacturerName,
			ItemNumber:       line.ItemNumber,
			ItemDescription:  line.ItemDescription,
			Quantity:         line.Quantity,
			LeadTimeDays:     max(line.LeadTimeDays, 0),
		})
	}
	if len(lines) == 0 {
		return ExtractedDocument{}, fmt.Errorf("vertex ai OCR returned no valid lines")
	}

	rawPayload := payload.RawPayload
	if len(rawPayload) == 0 {
		rawPayload = json.RawMessage(`{"provider":"vertex_ai"}`)
	}

	return ExtractedDocument{
		SupplierName:    payload.SupplierName,
		SupplierID:      payload.SupplierID,
		QuotationNumber: payload.QuotationNumber,
		IssueDate:       payload.IssueDate,
		RawPayload:      string(rawPayload),
		Lines:           lines,
	}, nil
}

func extractJSONBlock(text string) string {
	candidate := strings.TrimSpace(text)
	if strings.HasPrefix(candidate, "```") {
		re := regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*\\})\\s*```")
		matches := re.FindStringSubmatch(candidate)
		if len(matches) == 2 {
			candidate = matches[1]
		}
	}
	return candidate
}

func max(value, floor int) int {
	if value < floor {
		return floor
	}
	return value
}

func resolveVertexLocation(model, location string) string {
	if strings.HasPrefix(model, "gemini-3-") {
		return "global"
	}
	return location
}
