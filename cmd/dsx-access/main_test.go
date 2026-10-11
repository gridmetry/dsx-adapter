package main

import (
	"context"
	"testing"

	"gridmetry/dsx-adapter/internal/nvcf"
)

func TestValidateIDFlags_BothEmpty(t *testing.T) {
	if err := validateIDFlags("", ""); err != nil {
		t.Fatalf("expected no error when both ids are omitted, got %v", err)
	}
}

func TestValidateIDFlags_OnlyOneProvided(t *testing.T) {
	if err := validateIDFlags("11111111-2222-3333-4444-555555555555", ""); err == nil {
		t.Fatal("expected error when only function-id is provided")
	}
	if err := validateIDFlags("", "11111111-2222-3333-4444-555555555555"); err == nil {
		t.Fatal("expected error when only version-id is provided")
	}
}

func TestValidateIDFlags_InvalidUUID(t *testing.T) {
	if err := validateIDFlags("not-a-uuid", "11111111-2222-3333-4444-555555555555"); err == nil {
		t.Fatal("expected error for invalid function-id UUID")
	}
	if err := validateIDFlags("11111111-2222-3333-4444-555555555555", "not-a-uuid"); err == nil {
		t.Fatal("expected error for invalid version-id UUID")
	}
}

func TestValidateIDFlags_BothValid(t *testing.T) {
	err := validateIDFlags("11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("expected no error for two valid UUIDs, got %v", err)
	}
}

type fakeDiscoveryClient struct {
	listCalls   [][]string
	listResults map[string]nvcf.VisibilityResult
	detailCall  *[2]string
	detailResult nvcf.DetailResult
}

func (f *fakeDiscoveryClient) ListFunctions(ctx context.Context, visibilities []string) nvcf.VisibilityResult {
	f.listCalls = append(f.listCalls, visibilities)
	key := joinKey(visibilities)
	return f.listResults[key]
}

func (f *fakeDiscoveryClient) GetFunctionVersion(ctx context.Context, functionID, versionID string) nvcf.DetailResult {
	f.detailCall = &[2]string{functionID, versionID}
	return f.detailResult
}

func joinKey(vs []string) string {
	out := ""
	for i, v := range vs {
		if i > 0 {
			out += ","
		}
		out += v
	}
	return out
}

func TestRunDiscovery_CallsPrivateAuthorizedAndPublicSeparately(t *testing.T) {
	fake := &fakeDiscoveryClient{
		listResults: map[string]nvcf.VisibilityResult{
			"private,authorized": {Visibilities: []string{"private", "authorized"}, HTTPStatus: 200, StatusClass: "ok"},
			"public":             {Visibilities: []string{"public"}, HTTPStatus: 200, StatusClass: "ok"},
		},
	}
	report := runDiscovery(context.Background(), fake, true)
	if len(fake.listCalls) != 2 {
		t.Fatalf("expected 2 separate list calls, got %d: %+v", len(fake.listCalls), fake.listCalls)
	}
	if len(report.Discovery) != 2 {
		t.Fatalf("expected 2 discovery results in report, got %d", len(report.Discovery))
	}
}

func TestRunDetail_CallsGetFunctionVersionOnce(t *testing.T) {
	fake := &fakeDiscoveryClient{
		detailResult: nvcf.DetailResult{HTTPStatus: 200, StatusClass: "ok", Found: true},
	}
	report := runDetail(context.Background(), fake, "11111111-2222-3333-4444-555555555555", "66666666-7777-8888-9999-aaaaaaaaaaaa", true)
	if fake.detailCall == nil {
		t.Fatal("expected GetFunctionVersion to be called")
	}
	if report.Detail == nil || !report.Detail.Found {
		t.Fatalf("expected resolved detail report, got %+v", report)
	}
	if report.ExitCode() != 0 {
		t.Fatalf("expected exit code 0, got %d", report.ExitCode())
	}
}
