package nvcf

import "net/http"

// Report is the safe, machine-readable output of the dsx-access CLI. It
// always distinguishes discovery/API success from any confirmed DSX session,
// which this read-only CLI never establishes.
type Report struct {
	CredentialUsed bool               `json:"credential_used"`
	Discovery      []VisibilityResult `json:"discovery,omitempty"`
	Detail         *DetailResult      `json:"detail,omitempty"`
	// CandidatesDiscovered is true only when discovery found at least one
	// STREAMING candidate whose name heuristically matches dsx/omniverse.
	// This is a heuristic signal, never proof of identity.
	CandidatesDiscovered bool `json:"candidates_discovered"`
	// TargetMetadataVerified is true only for an explicit detail request
	// whose returned function/version ids matched the requested ids. It
	// means metadata was read, not that a DSX session exists.
	TargetMetadataVerified bool `json:"target_metadata_verified"`
	// FunctionActive mirrors the explicit detail candidate's exact ACTIVE
	// status. ACTIVE does not imply a session is ready or reachable.
	FunctionActive bool `json:"function_active"`
	// DSXIdentityVerified is always false. This CLI never proves a DSX
	// deployment's identity.
	DSXIdentityVerified bool `json:"dsx_identity_verified"`
	// SessionAccessVerified is always false. This CLI never starts or
	// confirms a hosted session.
	SessionAccessVerified bool   `json:"session_access_verified"`
	Error                 string `json:"error,omitempty"`
	blocked               bool
}

// ExitCode returns the CLI exit status implied by this report:
//
//	0 - explicit metadata was verified against the request, or discovery
//	    found a name-matching streaming candidate (heuristic only)
//	1 - an execution error occurred (bad credential, request failure, or
//	    any non-200 status, including 404)
//	2 - requests completed cleanly (200) but no target was resolved
func (r Report) ExitCode() int {
	if r.blocked {
		return 1
	}
	if r.TargetMetadataVerified || r.CandidatesDiscovered {
		return 0
	}
	return 2
}

// CredentialFailureReport builds the report emitted when the bearer token
// could not be loaded. The message must already be a safe, generic string.
func CredentialFailureReport(message string) Report {
	return Report{
		CredentialUsed:        false,
		SessionAccessVerified: false,
		Error:                 message,
		blocked:               true,
	}
}

// BuildDiscoveryReport assembles a Report from one or more visibility list
// results.
func BuildDiscoveryReport(credentialUsed bool, results []VisibilityResult) Report {
	report := Report{
		CredentialUsed:        credentialUsed,
		Discovery:             results,
		SessionAccessVerified: false,
	}

	blocked := false
	matched := false
	for _, res := range results {
		if res.Error != "" || res.StatusClass != "ok" {
			blocked = true
		}
		for _, c := range res.StreamingCandidates {
			if c.NameMatchesDSX {
				matched = true
			}
		}
	}

	report.blocked = blocked
	report.CandidatesDiscovered = !blocked && matched
	return report
}

// BuildDetailReport assembles a Report from one explicit function/version
// detail request. Any non-200 status, including 404, is treated as an
// execution error rather than a clean unresolved result, because the caller
// supplied explicit ids and a non-200 response cannot be distinguished from
// a request/identifier problem.
func BuildDetailReport(credentialUsed bool, detail DetailResult) Report {
	report := Report{
		CredentialUsed:        credentialUsed,
		Detail:                &detail,
		SessionAccessVerified: false,
	}

	if detail.Error != "" || detail.HTTPStatus != http.StatusOK {
		report.blocked = true
		return report
	}

	report.TargetMetadataVerified = detail.Found
	report.FunctionActive = detail.Active
	return report
}
