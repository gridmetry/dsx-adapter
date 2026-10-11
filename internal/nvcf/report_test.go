package nvcf

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildDiscoveryReport_MatchFoundResolved(t *testing.T) {
	results := []VisibilityResult{
		{Visibilities: []string{"private", "authorized"}, HTTPStatus: 200, StatusClass: "ok", FunctionCount: 0},
		{Visibilities: []string{"public"}, HTTPStatus: 200, StatusClass: "ok", FunctionCount: 226, StreamingCandidates: []Candidate{
			{ID: "22222222-3333-4444-5555-666666666666", NameMatchesDSX: true},
		}},
	}
	report := BuildDiscoveryReport(true, results)
	if report.SessionAccessVerified {
		t.Fatal("session_access_verified must always be false")
	}
	if report.DSXIdentityVerified {
		t.Fatal("dsx_identity_verified must always be false")
	}
	if !report.CandidatesDiscovered {
		t.Fatal("expected candidates_discovered true for a name-matching streaming candidate")
	}
	if report.TargetMetadataVerified {
		t.Fatal("target_metadata_verified must be false for a heuristic discovery-only run")
	}
	code := report.ExitCode()
	if code != 0 {
		t.Fatalf("expected exit code 0 for resolved match, got %d", code)
	}
}

func TestBuildDiscoveryReport_NoMatchUnresolved(t *testing.T) {
	results := []VisibilityResult{
		{Visibilities: []string{"private", "authorized"}, HTTPStatus: 200, StatusClass: "ok", FunctionCount: 0},
		{Visibilities: []string{"public"}, HTTPStatus: 200, StatusClass: "ok", FunctionCount: 226, StreamingCandidates: []Candidate{
			{ID: "11111111-2222-3333-4444-555555555555", NameMatchesDSX: false},
		}},
	}
	report := BuildDiscoveryReport(true, results)
	if report.CandidatesDiscovered {
		t.Fatal("expected candidates_discovered false when no candidate name-matches")
	}
	code := report.ExitCode()
	if code != 2 {
		t.Fatalf("expected exit code 2 (target unresolved) for clean run with no match, got %d", code)
	}
}

func TestBuildDiscoveryReport_ErrorStatusIsExecutionError(t *testing.T) {
	results := []VisibilityResult{
		{Visibilities: []string{"private", "authorized"}, HTTPStatus: 401, StatusClass: "auth_denied"},
		{Visibilities: []string{"public"}, HTTPStatus: 200, StatusClass: "ok", FunctionCount: 5},
	}
	report := BuildDiscoveryReport(true, results)
	code := report.ExitCode()
	if code != 1 {
		t.Fatalf("expected exit code 1 for auth_denied result, got %d", code)
	}
}

func TestBuildDiscoveryReport_ZeroFunctionsNotBadCredential(t *testing.T) {
	results := []VisibilityResult{
		{Visibilities: []string{"private", "authorized"}, HTTPStatus: 200, StatusClass: "ok", FunctionCount: 0},
		{Visibilities: []string{"public"}, HTTPStatus: 200, StatusClass: "ok", FunctionCount: 0},
	}
	report := BuildDiscoveryReport(true, results)
	if report.ExitCode() != 2 {
		t.Fatalf("expected unresolved (2), not error, got %d", report.ExitCode())
	}
}

func TestBuildDetailReport_ResolvedWhenFound(t *testing.T) {
	detail := DetailResult{HTTPStatus: 200, StatusClass: "ok", Found: true, Active: false, Candidate: Candidate{ID: "x"}}
	report := BuildDetailReport(true, detail)
	if report.ExitCode() != 0 {
		t.Fatalf("expected exit code 0 for explicit resolved target, got %d", report.ExitCode())
	}
	if !report.TargetMetadataVerified {
		t.Fatal("expected target_metadata_verified true for an explicit found detail")
	}
	if report.FunctionActive {
		t.Fatal("expected function_active false when candidate status is not ACTIVE")
	}
	if report.SessionAccessVerified {
		t.Fatal("session_access_verified must always be false")
	}
	if report.DSXIdentityVerified {
		t.Fatal("dsx_identity_verified must always be false")
	}
}

func TestBuildDetailReport_ActiveFunctionReportsFunctionActive(t *testing.T) {
	detail := DetailResult{HTTPStatus: 200, StatusClass: "ok", Found: true, Active: true, Candidate: Candidate{ID: "x"}}
	report := BuildDetailReport(true, detail)
	if !report.FunctionActive {
		t.Fatal("expected function_active true when candidate status is ACTIVE")
	}
}

func TestBuildDetailReport_NotFoundUnresolved(t *testing.T) {
	detail := DetailResult{HTTPStatus: 200, StatusClass: "ok", Found: false}
	report := BuildDetailReport(true, detail)
	if report.ExitCode() != 2 {
		t.Fatalf("expected exit code 2 for a clean 200 response with no matching target, got %d", report.ExitCode())
	}
}

func TestBuildDetailReport_404IsExecutionError(t *testing.T) {
	detail := DetailResult{HTTPStatus: 404, StatusClass: "client_error", Found: false}
	report := BuildDetailReport(true, detail)
	if report.ExitCode() != 1 {
		t.Fatalf("expected exit code 1 for a 404 response, got %d", report.ExitCode())
	}
}

func TestBuildDetailReport_ServerErrorIsExecutionError(t *testing.T) {
	detail := DetailResult{HTTPStatus: 500, StatusClass: "service_unavailable", Found: false}
	report := BuildDetailReport(true, detail)
	if report.ExitCode() != 1 {
		t.Fatalf("expected exit code 1 for service error, got %d", report.ExitCode())
	}
}

func TestReport_JSONNeverLeaksSecrets(t *testing.T) {
	fakeCred := "fake-leak-test-credential-xyz"
	results := []VisibilityResult{
		{Visibilities: []string{"public"}, HTTPStatus: 200, StatusClass: "ok", FunctionCount: 1, Error: ""},
	}
	report := BuildDiscoveryReport(true, results)
	out, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if strings.Contains(string(out), fakeCred) {
		t.Fatalf("report leaked fake credential: %s", out)
	}
	if !strings.Contains(string(out), `"session_access_verified":false`) {
		t.Fatalf("expected session_access_verified:false in output, got %s", out)
	}
}

func TestCredentialFailureReport_NeverVerifiesSession(t *testing.T) {
	report := CredentialFailureReport("credential file not accessible")
	if report.SessionAccessVerified {
		t.Fatal("session_access_verified must be false on credential failure")
	}
	if report.ExitCode() != 1 {
		t.Fatalf("expected exit code 1 for credential failure, got %d", report.ExitCode())
	}
	if report.Error == "" {
		t.Fatal("expected error populated on credential failure report")
	}
}
