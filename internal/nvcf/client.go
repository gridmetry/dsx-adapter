package nvcf

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ProductionBaseURL is the only NVCF host this CLI is permitted to contact.
// There is no flag to override it.
const ProductionBaseURL = "https://api.nvcf.nvidia.com"

const (
	maxListBodyBytes   = 5 << 20 // 5MB
	maxDetailBodyBytes = 1 << 20 // 1MB
	requestTimeout     = 15 * time.Second
)

var errRedirectRefused = errors.New("redirect refused")

// RefuseRedirects is an http.Client.CheckRedirect implementation that never
// follows a redirect, so an authorization header can never be replayed to an
// unverified host. It is used by both the production client and tests.
func RefuseRedirects(req *http.Request, via []*http.Request) error {
	return errRedirectRefused
}

// NewProductionHTTPClient builds the bounded HTTP client used for live
// discovery: verified TLS, no implicit proxy environment, no redirect
// following, and a bounded request timeout.
func NewProductionHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy:           nil,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
	}
	return &http.Client{
		Transport:     transport,
		Timeout:       requestTimeout,
		CheckRedirect: RefuseRedirects,
	}
}

// Client is a bounded, read-only NVCF discovery client.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// NewClient constructs a Client. httpClient is injected so tests can supply
// an httptest-backed client without the production binary exposing any
// arbitrary-endpoint flag.
func NewClient(baseURL string, httpClient *http.Client, token string) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("base URL is required")
	}
	if httpClient == nil {
		return nil, errors.New("http client is required")
	}
	if strings.TrimSpace(token) == "" {
		return nil, errors.New("token is required")
	}
	return &Client{baseURL: baseURL, httpClient: httpClient, token: token}, nil
}

// Candidate is a sanitized, safe-to-print representation of an NVCF
// function/version. Free-text names are never included; only a heuristic
// match boolean is derived from them.
type Candidate struct {
	ID             string `json:"id"`
	VersionID      string `json:"version_id,omitempty"`
	FunctionType   string `json:"function_type,omitempty"`
	Status         string `json:"status,omitempty"`
	NameMatchesDSX bool   `json:"name_matches_dsx"`
}

// VisibilityResult is the sanitized outcome of one functions-list request.
type VisibilityResult struct {
	Visibilities        []string    `json:"visibilities"`
	HTTPStatus          int         `json:"http_status"`
	StatusClass         string      `json:"status_class"`
	FunctionCount       int         `json:"function_count"`
	StreamingCandidates []Candidate `json:"streaming_candidates,omitempty"`
	Error               string      `json:"error,omitempty"`
}

// DetailResult is the sanitized outcome of one explicit function/version
// detail request.
type DetailResult struct {
	HTTPStatus  int    `json:"http_status"`
	StatusClass string `json:"status_class"`
	Found       bool   `json:"found"`
	// Active reports only whether the API-reported function status is
	// exactly ACTIVE. It is a function-availability signal, never proof
	// that a DSX session is ready or reachable.
	Active    bool      `json:"function_active"`
	Candidate Candidate `json:"candidate,omitempty"`
	Error     string    `json:"error,omitempty"`
}

var dsxNamePattern = regexp.MustCompile(`(?i)dsx|omniverse`)

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func isValidUUID(s string) bool {
	return uuidPattern.MatchString(s)
}

// IsValidUUID reports whether s is a syntactically valid UUID. It is exposed
// so callers can validate ids before ever constructing a request.
func IsValidUUID(s string) bool {
	return isValidUUID(s)
}

// classifyStatus turns an HTTP status code into a safe, honest category
// without ever echoing a response body.
func classifyStatus(code int) string {
	switch {
	case code == http.StatusOK:
		return "ok"
	case code == http.StatusUnauthorized:
		return "auth_denied"
	case code == http.StatusForbidden:
		return "permissions_or_auth_unknown"
	case code == http.StatusTooManyRequests:
		return "rate_limited"
	case code >= 500:
		return "service_unavailable"
	case code >= 400:
		return "client_error"
	default:
		return "unexpected_status"
	}
}

// apiFunction is the subset of fields this client reads from an NVCF
// function/version JSON object. Unknown fields are ignored; any secret,
// container, environment, or account data in the real response is never
// decoded into this struct.
type apiFunction struct {
	ID           string `json:"id"`
	VersionID    string `json:"versionId"`
	Name         string `json:"name"`
	FunctionType string `json:"functionType"`
	Status       string `json:"status"`
}

// functionDetailResponse is the actual NVCF function/version detail
// envelope: the object is wrapped under a "function" key, not flat.
type functionDetailResponse struct {
	Function *apiFunction `json:"function"`
}

// validFunctionTypes and validFunctionStatuses are the known, exact-case API
// enum values. Any other value is sanitized to empty rather than echoed, so
// an attacker-controlled response body can never inject arbitrary text into
// a serialized report.
var validFunctionTypes = map[string]bool{
	"DEFAULT":   true,
	"STREAMING": true,
	"LLM":       true,
}

var validFunctionStatuses = map[string]bool{
	"ACTIVE":    true,
	"DEPLOYING": true,
	"DEGRADING": true,
	"DEGRADED":  true,
	"ERROR":     true,
	"DELETED":   true,
	"INACTIVE":  true,
}

// toCandidate converts a decoded API function into a safe-to-print
// Candidate. ok is false when id/versionId are not both syntactically valid
// UUIDs, the minimum bar for any candidate to be usable at all. Enum fields
// that do not match a known API value are sanitized to empty rather than
// echoed raw.
func (f apiFunction) toCandidate() (Candidate, bool) {
	if !isValidUUID(f.ID) || !isValidUUID(f.VersionID) {
		return Candidate{}, false
	}
	functionType := f.FunctionType
	if !validFunctionTypes[functionType] {
		functionType = ""
	}
	status := f.Status
	if !validFunctionStatuses[status] {
		status = ""
	}
	return Candidate{
		ID:             f.ID,
		VersionID:      f.VersionID,
		FunctionType:   functionType,
		Status:         status,
		NameMatchesDSX: dsxNamePattern.MatchString(f.Name),
	}, true
}

// doGet issues a single bounded GET request. It never follows redirects (the
// injected client enforces that), never logs the Authorization header, and
// bounds the response body size. Returned errors are generic and never wrap
// the underlying request/response.
func (c *Client) doGet(ctx context.Context, path string, query url.Values, maxBody int64) (int, []byte, error) {
	u := c.baseURL + path
	if encoded := query.Encode(); encoded != "" {
		u += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, nil, errors.New("request construction failed")
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, nil, errors.New("request failed")
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxBody+1)
	body, err := io.ReadAll(limited)
	if err != nil {
		return resp.StatusCode, nil, errors.New("response read failed")
	}
	if int64(len(body)) > maxBody {
		return resp.StatusCode, nil, errors.New("response body exceeded bound")
	}
	return resp.StatusCode, body, nil
}

// ListFunctions requests the functions list for the given visibility values
// as explicit repeated query parameters.
func (c *Client) ListFunctions(ctx context.Context, visibilities []string) VisibilityResult {
	result := VisibilityResult{Visibilities: visibilities}

	query := url.Values{}
	for _, v := range visibilities {
		query.Add("visibility", v)
	}

	status, body, err := c.doGet(ctx, "/v2/nvcf/functions", query, maxListBodyBytes)
	result.HTTPStatus = status
	result.StatusClass = classifyStatus(status)
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if status != http.StatusOK {
		return result
	}

	var raw map[string]json.RawMessage
	if jsonErr := json.Unmarshal(body, &raw); jsonErr != nil {
		result.Error = "response body had unexpected structure"
		return result
	}

	functionsRaw, present := raw["functions"]
	if !present {
		result.Error = "functions field missing from response"
		return result
	}
	if string(functionsRaw) == "null" {
		result.Error = "functions field was null"
		return result
	}

	var functions []apiFunction
	if jsonErr := json.Unmarshal(functionsRaw, &functions); jsonErr != nil {
		result.Error = "functions field had unexpected structure"
		return result
	}

	result.FunctionCount = len(functions)
	for _, f := range functions {
		if f.FunctionType != "STREAMING" {
			continue
		}
		if candidate, ok := f.toCandidate(); ok {
			result.StreamingCandidates = append(result.StreamingCandidates, candidate)
		}
	}
	return result
}

// GetFunctionVersion requests a single function/version detail. Both ids
// must already be validated UUIDs; the server is never contacted otherwise.
func (c *Client) GetFunctionVersion(ctx context.Context, functionID, versionID string) DetailResult {
	if !isValidUUID(functionID) || !isValidUUID(versionID) {
		return DetailResult{Error: "function-id and version-id must be valid UUIDs"}
	}

	path := fmt.Sprintf("/v2/nvcf/functions/%s/versions/%s", functionID, versionID)
	query := url.Values{}
	query.Set("includeSecrets", "false")

	status, body, err := c.doGet(ctx, path, query, maxDetailBodyBytes)
	result := DetailResult{HTTPStatus: status, StatusClass: classifyStatus(status)}
	if err != nil {
		result.Error = err.Error()
		return result
	}
	if status != http.StatusOK {
		return result
	}

	var envelope functionDetailResponse
	if jsonErr := json.Unmarshal(body, &envelope); jsonErr != nil {
		result.Error = "response body had unexpected structure"
		return result
	}
	if envelope.Function == nil {
		result.Error = "function field missing from response"
		return result
	}

	f := *envelope.Function
	if !strings.EqualFold(f.ID, functionID) || !strings.EqualFold(f.VersionID, versionID) {
		result.Error = "returned function identifiers did not match the requested ids"
		return result
	}

	candidate, ok := f.toCandidate()
	if !ok {
		result.Error = "function field had invalid or incomplete identifiers"
		return result
	}

	result.Found = true
	result.Candidate = candidate
	result.Active = candidate.Status == "ACTIVE"
	return result
}
