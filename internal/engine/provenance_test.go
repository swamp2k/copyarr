package engine

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/swamp2k/copyarr/internal/config"
	"github.com/swamp2k/copyarr/internal/db"
)

func testEngine(t *testing.T, rules ...config.Rule) (*Engine, *db.DB) {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "copyarr.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = d.Close() })

	for i := range rules {
		if err := config.NormalizeRule(&rules[i], i); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Config{DataDir: t.TempDir(), Rules: rules}
	return New(cfg, d, "test", "test"), d
}

func configRule(id string) config.Rule {
	return config.Rule{
		ID:          id,
		Name:        id,
		Enabled:     true,
		Source:      config.Endpoint{Remote: "seedbox", Path: "/downloads"},
		Destination: config.Endpoint{Path: "/mnt/media"},
	}
}

func viewByID(t *testing.T, e *Engine, id string) RuleView {
	t.Helper()
	for _, v := range e.Rules() {
		if v.ID == id {
			return v
		}
	}
	t.Fatalf("rule %q not found", id)
	return RuleView{}
}

func TestRulesReportProvenance(t *testing.T) {
	e, _ := testEngine(t, configRule("from-config"))

	uiJob := configRule("from-ui")
	uiJob.Source = config.Endpoint{Path: "/incoming"}
	if err := e.SaveJobDefinition(uiJob); err != nil {
		t.Fatal(err)
	}

	cfgView := viewByID(t, e, "from-config")
	if cfgView.Origin != OriginConfig {
		t.Errorf("config job origin = %q, want %q", cfgView.Origin, OriginConfig)
	}
	if cfgView.HasOverride {
		t.Error("an unedited config job must not report an override")
	}

	uiView := viewByID(t, e, "from-ui")
	if uiView.Origin != OriginUI {
		t.Errorf("ui job origin = %q, want %q", uiView.Origin, OriginUI)
	}
	if uiView.HasOverride {
		t.Error("a UI job is not an override of anything")
	}
}

func TestEditingAConfigJobIsReportedAsAnOverride(t *testing.T) {
	e, _ := testEngine(t, configRule("from-config"))

	edited := configRule("from-config")
	edited.Destination = config.Endpoint{Path: "/somewhere/else"}
	if err := e.SaveJobDefinition(edited); err != nil {
		t.Fatal(err)
	}

	view := viewByID(t, e, "from-config")
	if view.Origin != OriginConfig {
		t.Errorf("origin = %q, want %q", view.Origin, OriginConfig)
	}
	if !view.HasOverride {
		t.Error("editing a config job must be reported as an override, not hidden")
	}
	if view.Destination.Path != "/somewhere/else" {
		t.Errorf("the edit did not take effect: %q", view.Destination.Path)
	}
}

func TestConfigJobCannotBeDeleted(t *testing.T) {
	e, _ := testEngine(t, configRule("from-config"))
	if err := e.DeleteJobDefinition("from-config"); err == nil {
		t.Fatal("expected deleting a config.json job to be refused")
	}
	if _, ok := e.rule("from-config"); !ok {
		t.Error("the refused delete removed the rule anyway")
	}
}

func TestResetJobDefinitionRestoresConfig(t *testing.T) {
	original := configRule("from-config")
	original.Destination = config.Endpoint{Path: "/mnt/media"}
	e, d := testEngine(t, original)

	edited := configRule("from-config")
	edited.Destination = config.Endpoint{Path: "/tmp/override"}
	retries := 9
	edited.RetryCount = &retries
	if err := e.SaveJobDefinition(edited); err != nil {
		t.Fatal(err)
	}
	if got := viewByID(t, e, "from-config").Destination.Path; got != "/tmp/override" {
		t.Fatalf("setup failed, destination = %q", got)
	}

	if err := e.ResetJobDefinition("from-config"); err != nil {
		t.Fatalf("reset: %v", err)
	}

	view := viewByID(t, e, "from-config")
	if view.Destination.Path != "/mnt/media" {
		t.Errorf("destination = %q, want the config.json value", view.Destination.Path)
	}
	if view.HasOverride {
		t.Error("the override should be gone after a reset")
	}
	if p, ok := e.RetryPolicy("from-config"); !ok || p.RetryCount != 3 {
		t.Errorf("retry policy = %+v, want the config default of 3", p)
	}
	// The stored override and its retry policy must be gone from the database,
	// or the next start would resurrect them.
	if _, ok, err := d.Meta("jobdef:from-config"); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Error("stored job definition survived the reset")
	}
	if _, ok, err := d.Meta("rule:from-config:retry_policy"); err != nil {
		t.Fatal(err)
	} else if ok {
		t.Error("stored retry policy survived the reset")
	}
}

func TestResetJobDefinitionRejectsUIJobs(t *testing.T) {
	e, _ := testEngine(t)
	if err := e.SaveJobDefinition(configRule("from-ui")); err != nil {
		t.Fatal(err)
	}
	if err := e.ResetJobDefinition("from-ui"); err == nil {
		t.Fatal("expected resetting a UI-created job to be refused")
	}
	if _, ok := e.rule("from-ui"); !ok {
		t.Error("the refused reset removed the rule anyway")
	}
}

func TestOverrideSurvivesRestartAndResetsCleanly(t *testing.T) {
	base := configRule("from-config")
	d, err := db.Open(filepath.Join(t.TempDir(), "copyarr.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = d.Close() }()
	if err := config.NormalizeRule(&base, 0); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{DataDir: t.TempDir(), Rules: []config.Rule{base}}

	first := New(cfg, d, "test", "test")
	edited := configRule("from-config")
	edited.Destination = config.Endpoint{Path: "/tmp/override"}
	if err := first.SaveJobDefinition(edited); err != nil {
		t.Fatal(err)
	}

	// A fresh engine over the same database is what a restart looks like.
	restarted := New(cfg, d, "test", "test")
	view := viewByID(t, restarted, "from-config")
	if !view.HasOverride {
		t.Error("the override was not recognised after a restart")
	}
	if view.Destination.Path != "/tmp/override" {
		t.Errorf("destination = %q after restart", view.Destination.Path)
	}

	if err := restarted.ResetJobDefinition("from-config"); err != nil {
		t.Fatal(err)
	}
	if viewByID(t, restarted, "from-config").Destination.Path != "/mnt/media" {
		t.Error("reset after a restart did not restore config.json")
	}
}

func TestRemoteUsageFindsEveryReference(t *testing.T) {
	src := configRule("reads-from-seedbox")
	src.Source = config.Endpoint{Remote: "seedbox", Path: "/a"}
	src.Destination = config.Endpoint{Path: "/local"}

	dst := configRule("writes-to-seedbox")
	dst.Source = config.Endpoint{Path: "/local"}
	dst.Destination = config.Endpoint{Remote: "seedbox", Path: "/b"}

	both := configRule("both-ends")
	both.Source = config.Endpoint{Remote: "seedbox", Path: "/a"}
	both.Destination = config.Endpoint{Remote: "seedbox", Path: "/b"}

	disabled := configRule("disabled-but-still-a-reference")
	disabled.Enabled = false
	disabled.Source = config.Endpoint{Remote: "seedbox", Path: "/c"}
	disabled.Destination = config.Endpoint{Path: "/local"}

	unrelated := configRule("uses-another-remote")
	unrelated.Source = config.Endpoint{Remote: "nas", Path: "/x"}
	unrelated.Destination = config.Endpoint{Path: "/local"}

	e, _ := testEngine(t, src, dst, both, disabled, unrelated)

	usage := e.RemoteUsage("seedbox")
	if len(usage) != 4 {
		t.Fatalf("found %d references, want 4: %+v", len(usage), usage)
	}
	roles := map[string]string{}
	for _, u := range usage {
		roles[u.JobID] = u.Role
	}
	want := map[string]string{
		"reads-from-seedbox":             "source",
		"writes-to-seedbox":              "destination",
		"both-ends":                      "source and destination",
		"disabled-but-still-a-reference": "source",
	}
	for id, role := range want {
		if roles[id] != role {
			t.Errorf("%s role = %q, want %q", id, roles[id], role)
		}
	}
	if _, found := roles["uses-another-remote"]; found {
		t.Error("a job pointing at a different remote was reported as a reference")
	}
}

func TestRemoteUsageIgnoresLocalEndpoints(t *testing.T) {
	local := configRule("all-local")
	local.Source = config.Endpoint{Path: "/in"}
	local.Destination = config.Endpoint{Path: "/out"}
	e, _ := testEngine(t, local)

	// The empty remote name means the local filesystem; it must never match,
	// or deleting any remote would look like it breaks every local job.
	if usage := e.RemoteUsage(""); len(usage) != 0 {
		t.Errorf("empty remote name matched %d jobs", len(usage))
	}
	if usage := e.RemoteUsage("seedbox"); len(usage) != 0 {
		t.Errorf("unused remote reported %d references", len(usage))
	}
}

func TestRemoteUsageSeesJobsCreatedInTheUI(t *testing.T) {
	e, _ := testEngine(t)
	job := configRule("ui-job")
	job.Source = config.Endpoint{Remote: "wasabi", Path: "/backups"}
	if err := e.SaveJobDefinition(job); err != nil {
		t.Fatal(err)
	}
	usage := e.RemoteUsage("wasabi")
	if len(usage) != 1 || usage[0].JobID != "ui-job" {
		t.Fatalf("usage = %+v, want the UI-created job", usage)
	}
}

// A RuleView embeds config.Rule, so any JSON name it declares itself wins over
// the embedded one and silently removes that field from the API. This caught a
// real regression where the provenance field was called "source" and erased the
// transfer's source endpoint.
func TestRuleViewJSONDoesNotShadowEmbeddedRuleFields(t *testing.T) {
	rule := configRule("job")
	rule.Source = config.Endpoint{Remote: "seedbox", Path: "/downloads"}
	rule.Destination = config.Endpoint{Remote: "nas", Path: "/media"}
	e, _ := testEngine(t, rule)

	raw, err := json.Marshal(e.Rules())
	if err != nil {
		t.Fatal(err)
	}
	var decoded []map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(decoded))
	}
	got := decoded[0]

	for _, field := range []string{"source", "destination"} {
		endpoint, ok := got[field].(map[string]any)
		if !ok {
			t.Fatalf("%q is %T, want the endpoint object; a view field is shadowing it", field, got[field])
		}
		if endpoint["remote"] == "" || endpoint["path"] == "" {
			t.Errorf("%q lost its contents: %v", field, endpoint)
		}
	}
	if got["source"].(map[string]any)["remote"] != "seedbox" {
		t.Errorf("source remote = %v", got["source"])
	}
	if got["origin"] != OriginConfig {
		t.Errorf("origin = %v, want %q", got["origin"], OriginConfig)
	}
}
