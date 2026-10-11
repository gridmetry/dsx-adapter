// Command dsx-lab bundles the Phase 2 lab utilities as subcommands so the
// Compose image stays a single minimal Go binary: init (generate isolated
// local secrets/config), serve (the MQTT-to-SSE bridge), check (a
// readiness probe for Compose healthchecks), and scenario (a self-contained
// publish-and-observe demonstration). See README.md for the full contract.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var code int
	switch os.Args[1] {
	case "init":
		code = runInit(os.Args[2:])
	case "serve":
		code = runServe(os.Args[2:])
	case "check":
		code = runCheck(os.Args[2:])
	case "scenario":
		code = runScenario(os.Args[2:])
	case "-h", "--help", "help":
		usage()
		code = 0
	default:
		fmt.Fprintf(os.Stderr, "dsx-lab: unknown subcommand %q\n", os.Args[1])
		usage()
		code = 2
	}
	os.Exit(code)
}

func usage() {
	fmt.Fprintln(os.Stderr, `Usage: dsx-lab <subcommand> [flags]

Subcommands:
  init      Generate isolated local secrets/config into the shared secrets volume.
  serve     Run the MQTT-to-SSE bridge service.
  check     Bounded readiness probe (used as a Compose healthcheck).
  scenario  Subscribe, publish one signed command, and verify receipt.`)
}
