// Command dsx-access performs bounded, read-only NVCF discovery requests.
//
// It never starts a hosted session, invokes a function, or proves that a
// specific DSX deployment is reachable. It only reports whether the NVCF
// discovery API accepted the supplied credential and, heuristically,
// whether any discovered candidate's name suggests a DSX/Omniverse
// deployment. See the README for the full safety contract.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"

	"gridmetry/dsx-adapter/internal/nvcf"
)

// discoveryClient is satisfied by *nvcf.Client. It exists so CLI
// orchestration can be tested with a fake, without any network access.
type discoveryClient interface {
	ListFunctions(ctx context.Context, visibilities []string) nvcf.VisibilityResult
	GetFunctionVersion(ctx context.Context, functionID, versionID string) nvcf.DetailResult
}

// validateIDFlags enforces that --function-id and --version-id are supplied
// together, and that each is a syntactically valid UUID.
func validateIDFlags(functionID, versionID string) error {
	if functionID == "" && versionID == "" {
		return nil
	}
	if functionID == "" || versionID == "" {
		return errors.New("--function-id and --version-id must be supplied together")
	}
	if !isValidUUIDFormat(functionID) {
		return errors.New("--function-id must be a valid UUID")
	}
	if !isValidUUIDFormat(versionID) {
		return errors.New("--version-id must be a valid UUID")
	}
	return nil
}

// runDiscovery performs the two visibility requests (private+authorized,
// then public) and assembles the final report.
func runDiscovery(ctx context.Context, client discoveryClient, credentialUsed bool) nvcf.Report {
	results := []nvcf.VisibilityResult{
		client.ListFunctions(ctx, []string{"private", "authorized"}),
		client.ListFunctions(ctx, []string{"public"}),
	}
	return nvcf.BuildDiscoveryReport(credentialUsed, results)
}

// runDetail performs the single explicit function/version detail request.
func runDetail(ctx context.Context, client discoveryClient, functionID, versionID string, credentialUsed bool) nvcf.Report {
	detail := client.GetFunctionVersion(ctx, functionID, versionID)
	return nvcf.BuildDetailReport(credentialUsed, detail)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout))
}

func run(args []string, stdout *os.File) int {
	fs := flag.NewFlagSet("dsx-access", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	keyFile := fs.String("key-file", "", "path to a local file containing the NVCF bearer token (required)")
	functionID := fs.String("function-id", "", "optional: function UUID, requires --version-id")
	versionID := fs.String("version-id", "", "optional: version UUID, requires --function-id")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: dsx-access --key-file <path> [--function-id <uuid> --version-id <uuid>]\n\n")
		fmt.Fprintf(os.Stderr, "Performs bounded read-only NVCF discovery requests. See README.md.\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *keyFile == "" {
		fmt.Fprintln(os.Stderr, "--key-file is required")
		fs.Usage()
		return 2
	}
	if err := validateIDFlags(*functionID, *versionID); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		fs.Usage()
		return 2
	}

	token, err := nvcf.LoadBearerToken(*keyFile)
	if err != nil {
		report := nvcf.CredentialFailureReport(err.Error())
		printReport(stdout, report)
		return report.ExitCode()
	}

	httpClient := nvcf.NewProductionHTTPClient()
	client, err := nvcf.NewClient(nvcf.ProductionBaseURL, httpClient, token)
	if err != nil {
		report := nvcf.CredentialFailureReport("client construction failed")
		printReport(stdout, report)
		return report.ExitCode()
	}

	ctx := context.Background()
	var report nvcf.Report
	if *functionID != "" {
		report = runDetail(ctx, client, *functionID, *versionID, true)
	} else {
		report = runDiscovery(ctx, client, true)
	}

	printReport(stdout, report)
	return report.ExitCode()
}

func printReport(stdout *os.File, report nvcf.Report) {
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fmt.Fprintln(os.Stderr, "failed to encode report")
	}
}

func isValidUUIDFormat(s string) bool {
	// Delegates to the same pattern the client enforces before any request,
	// kept local so flag validation never needs a network-capable client.
	return nvcf.IsValidUUID(s)
}
