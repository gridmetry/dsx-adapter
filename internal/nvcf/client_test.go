package nvcf

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const fakeToken = "fake-test-bearer-token-not-real"

func newTestClient(t *testing.T, handler http.HandlerFunc) (*Client, *httptest.Server) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	httpClient := server.Client()
	httpClient.CheckRedirect = RefuseRedirects
	httpClient.Timeout = 5 * time.Second
	client, err := NewClient(server.URL, httpClient, fakeToken)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	return client, server
}

func TestListFunctions_AuthHeaderAndVisibilityQuery(t *testing.T) {
	var gotAuth, gotMethod, gotPath, gotQuery string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"functions":[]}`)
	})

	res := client.ListFunctions(context.Background(), []string{"private", "authorized"})

	if gotMethod != http.MethodGet {
		t.Fatalf("expected GET, got %s", gotMethod)
	}
	if gotAuth != "Bearer "+fakeToken {
		t.Fatalf("expected bearer auth header, got %q", gotAuth)
	}
	if gotPath != "/v2/nvcf/functions" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if !strings.Contains(gotQuery, "visibility=private") || !strings.Contains(gotQuery, "visibility=authorized") {
		t.Fatalf("expected explicit visibility query params, got %q", gotQuery)
	}
	if res.HTTPStatus != 200 || res.StatusClass != "ok" {
		t.Fatalf("expected ok result, got %+v", res)
	}
	if res.FunctionCount != 0 {
		t.Fatalf("expected zero functions, got %d", res.FunctionCount)
	}
}

func TestListFunctions_ZeroFunctionsNotTreatedAsBadCredential(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"functions":[]}`)
	})
	res := client.ListFunctions(context.Background(), []string{"private", "authorized"})
	if res.StatusClass != "ok" {
		t.Fatalf("zero functions must remain status class ok, got %q", res.StatusClass)
	}
	if res.Error != "" {
		t.Fatalf("zero functions must not be represented as an error, got %q", res.Error)
	}
}

func TestListFunctions_PublicWithStreamingCandidates(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"functions":[
			{"id":"11111111-2222-3333-4444-555555555555","versionId":"66666666-7777-8888-9999-aaaaaaaaaaaa","name":"example-unrelated-streaming-fn","functionType":"STREAMING","status":"ACTIVE"},
			{"id":"22222222-3333-4444-5555-666666666666","versionId":"77777777-8888-9999-aaaa-bbbbbbbbbbbb","name":"demo-dsx-omniverse-candidate","functionType":"STREAMING","status":"DEGRADING"},
			{"id":"33333333-4444-5555-6666-777777777777","versionId":"88888888-9999-aaaa-bbbb-cccccccccccc","name":"unrelated-batch-fn","functionType":"BATCH","status":"ACTIVE"}
		]}`)
	})
	res := client.ListFunctions(context.Background(), []string{"public"})
	if res.FunctionCount != 3 {
		t.Fatalf("expected total count 3, got %d", res.FunctionCount)
	}
	if len(res.StreamingCandidates) != 2 {
		t.Fatalf("expected 2 streaming candidates (BATCH excluded), got %d: %+v", len(res.StreamingCandidates), res.StreamingCandidates)
	}
	matched := 0
	for _, c := range res.StreamingCandidates {
		if c.NameMatchesDSX {
			matched++
		}
	}
	if matched != 1 {
		t.Fatalf("expected exactly 1 candidate to match DSX/Omniverse heuristic, got %d", matched)
	}
}

func TestListFunctions_NamesNeverLeakedInOutput(t *testing.T) {
	secretLikeName := "fake-bearer-token-abc123-leak-test"
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"functions":[{"id":"11111111-2222-3333-4444-555555555555","versionId":"66666666-7777-8888-9999-aaaaaaaaaaaa","name":%q,"functionType":"STREAMING","status":"ACTIVE"}]}`, secretLikeName)
	})
	res := client.ListFunctions(context.Background(), []string{"public"})
	out, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(out), secretLikeName) {
		t.Fatalf("response name leaked into marshaled output: %s", out)
	}
	if strings.Contains(string(out), fakeToken) {
		t.Fatalf("bearer token leaked into marshaled output: %s", out)
	}
}

func TestListFunctions_FunctionsNull(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"functions":null}`)
	})
	res := client.ListFunctions(context.Background(), []string{"public"})
	if res.FunctionCount != 0 {
		t.Fatalf("expected 0 count for null functions, got %d", res.FunctionCount)
	}
	if res.Error == "" {
		t.Fatal("expected an error for a null functions field, not a silent success")
	}
}

func TestListFunctions_FunctionsMissing(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{}`)
	})
	res := client.ListFunctions(context.Background(), []string{"public"})
	if res.FunctionCount != 0 {
		t.Fatalf("expected 0 count for missing functions field, got %d", res.FunctionCount)
	}
	if res.Error == "" {
		t.Fatal("expected an error for a missing functions field, not a silent success")
	}
}

func TestListFunctions_EmptyFunctionsIsLegitimateSuccess(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"functions":[]}`)
	})
	res := client.ListFunctions(context.Background(), []string{"public"})
	if res.Error != "" {
		t.Fatalf("expected no error for a legitimate empty functions list, got %q", res.Error)
	}
	if res.StatusClass != "ok" {
		t.Fatalf("expected status class ok for empty functions list, got %q", res.StatusClass)
	}
}

func TestListFunctions_InvalidUUIDCandidateExcluded(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"functions":[{"id":"not-a-uuid","versionId":"also-not-a-uuid","name":"dsx-fake","functionType":"STREAMING","status":"ACTIVE"}]}`)
	})
	res := client.ListFunctions(context.Background(), []string{"public"})
	if res.FunctionCount != 1 {
		t.Fatalf("expected total count 1, got %d", res.FunctionCount)
	}
	if len(res.StreamingCandidates) != 0 {
		t.Fatalf("expected candidate with invalid ids to be excluded, got %+v", res.StreamingCandidates)
	}
}

func TestListFunctions_MaliciousStatusSanitizedNotEchoed(t *testing.T) {
	malicious := "fake-bearer-token-leak-zzz"
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"functions":[{"id":"11111111-2222-3333-4444-555555555555","versionId":"66666666-7777-8888-9999-aaaaaaaaaaaa","name":"dsx-demo","functionType":"STREAMING","status":%q}]}`, malicious)
	})
	res := client.ListFunctions(context.Background(), []string{"public"})
	if len(res.StreamingCandidates) != 1 {
		t.Fatalf("expected 1 candidate with sanitized status, got %d", len(res.StreamingCandidates))
	}
	if res.StreamingCandidates[0].Status != "" {
		t.Fatalf("expected unknown status sanitized to empty, got %q", res.StreamingCandidates[0].Status)
	}
	out, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(out), malicious) {
		t.Fatalf("malicious status value leaked into output: %s", out)
	}
}

func TestListFunctions_MalformedJSON(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{not-json`)
	})
	res := client.ListFunctions(context.Background(), []string{"public"})
	if res.Error == "" {
		t.Fatal("expected an error for malformed JSON body")
	}
	if strings.Contains(res.Error, "not-json") {
		t.Fatalf("error must not echo raw response body: %q", res.Error)
	}
}

func TestListFunctions_UnexpectedSchema(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `["totally", "unexpected", "array"]`)
	})
	res := client.ListFunctions(context.Background(), []string{"public"})
	if res.Error == "" {
		t.Fatal("expected an error for unexpected top-level schema")
	}
}

func TestListFunctions_BodyTooLarge(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"functions":[`)
		for i := 0; i < 2_000_000; i++ {
			fmt.Fprint(w, `{"id":"pad"},`)
		}
		fmt.Fprint(w, `{"id":"pad"}]}`)
	})
	res := client.ListFunctions(context.Background(), []string{"public"})
	if res.Error == "" {
		t.Fatal("expected an error for oversized response body")
	}
}

func TestListFunctions_MaliciousErrorBodyNotLeaked(t *testing.T) {
	fakeCred := "fake-leaked-credential-zzz-999"
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, `{"error":"internal failure, token=%s"}`, fakeCred)
	})
	res := client.ListFunctions(context.Background(), []string{"public"})
	out, _ := json.Marshal(res)
	if strings.Contains(string(out), fakeCred) {
		t.Fatalf("error body leaked fake credential: %s", out)
	}
	if res.StatusClass != "service_unavailable" {
		t.Fatalf("expected service_unavailable classification, got %q", res.StatusClass)
	}
}

func TestListFunctions_StatusClassification(t *testing.T) {
	cases := []struct {
		code  int
		class string
	}{
		{http.StatusUnauthorized, "auth_denied"},
		{http.StatusForbidden, "permissions_or_auth_unknown"},
		{http.StatusTooManyRequests, "rate_limited"},
		{http.StatusInternalServerError, "service_unavailable"},
		{http.StatusBadGateway, "service_unavailable"},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("status_%d", tc.code), func(t *testing.T) {
			client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
				fmt.Fprint(w, `{"error":"denied"}`)
			})
			res := client.ListFunctions(context.Background(), []string{"public"})
			if res.HTTPStatus != tc.code {
				t.Fatalf("expected http status %d, got %d", tc.code, res.HTTPStatus)
			}
			if res.StatusClass != tc.class {
				t.Fatalf("expected class %q, got %q", tc.class, res.StatusClass)
			}
		})
	}
}

func TestListFunctions_NoRedirectFollowed(t *testing.T) {
	hits := 0
	var redirectHandler http.HandlerFunc
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectHandler(w, r)
	}))
	defer server.Close()
	redirectHandler = func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Redirect(w, r, "https://evil.example.test/steal?token=leak", http.StatusFound)
	}
	httpClient := server.Client()
	httpClient.CheckRedirect = RefuseRedirects
	httpClient.Timeout = 5 * time.Second
	client, err := NewClient(server.URL, httpClient, fakeToken)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	res := client.ListFunctions(context.Background(), []string{"public"})
	if hits != 1 {
		t.Fatalf("expected exactly 1 request (no redirect follow), got %d", hits)
	}
	if res.Error == "" {
		t.Fatal("expected an error result when server attempts a redirect")
	}
}

func TestListFunctions_OnlyGETMethodUsed(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method used: %s", r.Method)
		}
		fmt.Fprint(w, `{"functions":[]}`)
	})
	client.ListFunctions(context.Background(), []string{"private", "authorized"})
	client.ListFunctions(context.Background(), []string{"public"})
}

func TestListFunctions_ContextTimeout(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		fmt.Fprint(w, `{"functions":[]}`)
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	res := client.ListFunctions(ctx, []string{"public"})
	if res.Error == "" {
		t.Fatal("expected timeout error")
	}
}

func TestGetFunctionVersion_WrappedValidDetail(t *testing.T) {
	var gotPath, gotQuery string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		fmt.Fprint(w, `{"function":{"id":"11111111-2222-3333-4444-555555555555","versionId":"66666666-7777-8888-9999-aaaaaaaaaaaa","name":"demo-dsx-candidate","functionType":"STREAMING","status":"ACTIVE"}}`)
	})
	res := client.GetFunctionVersion(context.Background(), "11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa")
	if res.HTTPStatus != 200 || !res.Found {
		t.Fatalf("expected found detail result, got %+v", res)
	}
	if !res.Active {
		t.Fatal("expected ACTIVE status to be reported function_active")
	}
	wantPath := "/v2/nvcf/functions/11111111-2222-3333-4444-555555555555/versions/66666666-7777-8888-9999-aaaaaaaaaaaa"
	if gotPath != wantPath {
		t.Fatalf("unexpected detail path: %s", gotPath)
	}
	if !strings.Contains(gotQuery, "includeSecrets=false") {
		t.Fatalf("expected includeSecrets=false query param, got %q", gotQuery)
	}
}

func TestGetFunctionVersion_FlatLegacyBodyNoLongerAccepted(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":"11111111-2222-3333-4444-555555555555","versionId":"66666666-7777-8888-9999-aaaaaaaaaaaa","name":"demo-dsx-candidate","functionType":"STREAMING","status":"ACTIVE"}`)
	})
	res := client.GetFunctionVersion(context.Background(), "11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa")
	if res.Found {
		t.Fatal("a flat (unwrapped) body must not be treated as found")
	}
	if res.Error == "" {
		t.Fatal("expected an error for a response missing the function wrapper")
	}
}

func TestGetFunctionVersion_MissingFunctionField(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{}`)
	})
	res := client.GetFunctionVersion(context.Background(), "11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa")
	if res.Found {
		t.Fatal("expected Found=false when function field is missing")
	}
	if res.Error == "" {
		t.Fatal("expected an error when function field is missing")
	}
}

func TestGetFunctionVersion_NullFunctionField(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"function":null}`)
	})
	res := client.GetFunctionVersion(context.Background(), "11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa")
	if res.Found {
		t.Fatal("expected Found=false when function field is null")
	}
	if res.Error == "" {
		t.Fatal("expected an error when function field is null")
	}
}

func TestGetFunctionVersion_EmptyFunctionObject(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"function":{}}`)
	})
	res := client.GetFunctionVersion(context.Background(), "11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa")
	if res.Found {
		t.Fatal("expected Found=false for an empty function object")
	}
	if res.Error == "" {
		t.Fatal("expected an error for an empty function object")
	}
}

func TestGetFunctionVersion_MismatchedReturnedIDs(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"function":{"id":"99999999-2222-3333-4444-555555555555","versionId":"66666666-7777-8888-9999-aaaaaaaaaaaa","name":"demo-fn","functionType":"STREAMING","status":"ACTIVE"}}`)
	})
	res := client.GetFunctionVersion(context.Background(), "11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa")
	if res.Found {
		t.Fatal("expected Found=false when returned id does not match the requested id")
	}
	if res.Error == "" {
		t.Fatal("expected an error when returned identifiers mismatch the request")
	}
}

func TestGetFunctionVersion_DegradingNotActive(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"function":{"id":"11111111-2222-3333-4444-555555555555","versionId":"66666666-7777-8888-9999-aaaaaaaaaaaa","name":"demo-fn","functionType":"STREAMING","status":"DEGRADING"}}`)
	})
	res := client.GetFunctionVersion(context.Background(), "11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa")
	if res.Active {
		t.Fatal("DEGRADING status must not be reported function_active")
	}
	if !res.Found {
		t.Fatal("expected Found=true for a valid wrapped DEGRADING function")
	}
}

func TestGetFunctionVersion_NullFunctionType(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"function":{"id":"11111111-2222-3333-4444-555555555555","versionId":"66666666-7777-8888-9999-aaaaaaaaaaaa","name":"demo-fn","functionType":null,"status":"ACTIVE"}}`)
	})
	res := client.GetFunctionVersion(context.Background(), "11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa")
	if res.Candidate.FunctionType != "" {
		t.Fatalf("expected empty function type for null value, got %q", res.Candidate.FunctionType)
	}
}

func TestGetFunctionVersion_LLMFunctionTypeAndDegradedStatusAccepted(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"function":{"id":"11111111-2222-3333-4444-555555555555","versionId":"66666666-7777-8888-9999-aaaaaaaaaaaa","name":"demo-fn","functionType":"LLM","status":"DEGRADED"}}`)
	})
	res := client.GetFunctionVersion(context.Background(), "11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa")
	if res.Candidate.FunctionType != "LLM" {
		t.Fatalf("expected functionType LLM to be accepted, got %q", res.Candidate.FunctionType)
	}
	if res.Candidate.Status != "DEGRADED" {
		t.Fatalf("expected status DEGRADED to be accepted, got %q", res.Candidate.Status)
	}
}

func TestGetFunctionVersion_MaliciousEnumFieldsNotEchoed(t *testing.T) {
	malicious := "fake-bearer-token-xyz; DROP TABLE"
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"function":{"id":"11111111-2222-3333-4444-555555555555","versionId":"66666666-7777-8888-9999-aaaaaaaaaaaa","name":"demo-fn","functionType":%q,"status":%q}}`, malicious, malicious)
	})
	res := client.GetFunctionVersion(context.Background(), "11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa")
	if !res.Found {
		t.Fatalf("expected Found=true with sanitized unknown enum values, got %+v", res)
	}
	out, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(out), malicious) {
		t.Fatalf("unknown enum value leaked raw into output: %s", out)
	}
	if res.Candidate.FunctionType != "" || res.Candidate.Status != "" {
		t.Fatalf("expected unknown enum values sanitized to empty, got %+v", res.Candidate)
	}
}

func TestGetFunctionVersion_InvalidUUIDNeverCallsServer(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server must not be contacted for invalid UUID input")
	})
	res := client.GetFunctionVersion(context.Background(), "not-a-uuid", "also-not-a-uuid")
	if res.Error == "" {
		t.Fatal("expected validation error for invalid UUIDs")
	}
}

func TestGetFunctionVersion_404NotFound(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"error":"not found"}`)
	})
	res := client.GetFunctionVersion(context.Background(), "11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa")
	if res.Found {
		t.Fatal("expected Found=false for 404")
	}
	if res.HTTPStatus != 404 {
		t.Fatalf("expected 404 status, got %d", res.HTTPStatus)
	}
}

func TestIsValidUUID(t *testing.T) {
	if !isValidUUID("11111111-2222-3333-4444-555555555555") {
		t.Fatal("expected valid UUID to pass")
	}
	for _, bad := range []string{"", "not-a-uuid", "11111111222233334444555555555555", "11111111-2222-3333-4444-55555555555g"} {
		if isValidUUID(bad) {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}

func TestNewClient_RequiresToken(t *testing.T) {
	if _, err := NewClient("https://example.test", &http.Client{}, ""); err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestNewProductionHTTPClient_NoProxyNoRedirect(t *testing.T) {
	c := NewProductionHTTPClient()
	transport, ok := c.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("expected *http.Transport, got %T", c.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("expected implicit proxy environment to be disabled")
	}
	if c.CheckRedirect == nil {
		t.Fatal("expected redirect refusal to be configured")
	}
	if c.Timeout <= 0 {
		t.Fatal("expected a bounded timeout")
	}
}
