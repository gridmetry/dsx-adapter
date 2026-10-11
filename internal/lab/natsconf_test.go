package lab

import (
	"strings"
	"testing"
)

// TestRenderNATSConfigUsesSingleSharedAccount guards against the account
// topology bug where the publisher and bridge lived in two separate NATS
// accounts with no imports/exports: NATS subject space is isolated per
// account, so a command published in one account can never reach a
// subscriber in another without an explicit import/export, which this
// generator does not add. Both lab users must therefore share one
// account, each with distinct (publish-only / subscribe-only)
// permissions on the single command subject.
func TestRenderNATSConfigUsesSingleSharedAccount(t *testing.T) {
	conf, err := RenderNATSConfig(validNATSParams())
	if err != nil {
		t.Fatalf("RenderNATSConfig: %v", err)
	}

	if got := strings.Count(conf, "jetstream: enabled"); got != 1 {
		t.Fatalf("expected exactly 1 account-level jetstream stanza (one shared account), got %d:\n%s", got, conf)
	}
	if got := strings.Count(conf, "users: ["); got != 1 {
		t.Fatalf("expected exactly 1 users list (one shared account), got %d:\n%s", got, conf)
	}
	if strings.Contains(conf, "BRIDGE: {") {
		t.Fatalf("expected no separate BRIDGE account, got:\n%s", conf)
	}

	if !strings.Contains(conf, `user: "gridmetry"`) {
		t.Fatalf("expected publisher user in shared account, got:\n%s", conf)
	}
	if !strings.Contains(conf, `user: "bridge"`) {
		t.Fatalf("expected bridge user in shared account, got:\n%s", conf)
	}

	if got := strings.Count(conf, `publish: { allow: ["grid.v1.isv.gridmetry.loadtarget.set"]`); got != 1 {
		t.Fatalf("expected exactly 1 publish-allow permission (publisher only), got %d:\n%s", got, conf)
	}
	if got := strings.Count(conf, `subscribe: { allow: ["grid.v1.isv.gridmetry.loadtarget.set"]`); got != 1 {
		t.Fatalf("expected exactly 1 subscribe-allow permission (bridge only), got %d:\n%s", got, conf)
	}
	if got := strings.Count(conf, `subscribe: { deny: [">"] }`); got != 1 {
		t.Fatalf("expected exactly 1 subscribe-deny-all permission (publisher only), got %d:\n%s", got, conf)
	}
	if got := strings.Count(conf, `publish: { deny: [">"] }`); got != 1 {
		t.Fatalf("expected exactly 1 publish-deny-all permission (bridge only), got %d:\n%s", got, conf)
	}
}
