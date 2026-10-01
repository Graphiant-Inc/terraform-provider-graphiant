package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

func TestFlattenPlaybookFilesKeepsPriorOrder(t *testing.T) {
	skipUnitTestInGitHubActions(t)
	ctx := context.Background()
	file := func(moduleKey, filename, content string) sdk.SdkAutomationModuleFile {
		return sdk.SdkAutomationModuleFile{ModuleKey: &moduleKey, Filename: &filename, Content: &content}
	}
	prior, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: playbookModuleFileAttrTypes}, []playbookModuleFileModel{
		{ModuleKey: types.StringValue("ntp"), Filename: types.StringValue("ntp.yaml"), Content: types.StringValue("old")},
		{ModuleKey: types.StringValue("dns"), Filename: types.StringValue("dns.yaml"), Content: types.StringValue("old")},
	})
	if diags.HasError() {
		t.Fatal(diags)
	}

	// The API returns the files in a different order, with one extra file.
	got, diags := flattenPlaybookFiles(ctx, []sdk.SdkAutomationModuleFile{
		file("syslog", "syslog.yaml", "c"),
		file("dns", "dns.yaml", "b"),
		file("ntp", "ntp.yaml", "a"),
	}, prior)
	if diags.HasError() {
		t.Fatal(diags)
	}
	var models []playbookModuleFileModel
	if diags := got.ElementsAs(ctx, &models, false); diags.HasError() {
		t.Fatal(diags)
	}

	want := []struct{ moduleKey, content string }{{"ntp", "a"}, {"dns", "b"}, {"syslog", "c"}}
	if len(models) != len(want) {
		t.Fatalf("got %d files, want %d", len(models), len(want))
	}
	for i, w := range want {
		if models[i].ModuleKey.ValueString() != w.moduleKey || models[i].Content.ValueString() != w.content {
			t.Errorf("file %d = %s/%s, want %s/%s", i, models[i].ModuleKey.ValueString(), models[i].Content.ValueString(), w.moduleKey, w.content)
		}
	}
}

func TestTimestampValue(t *testing.T) {
	skipUnitTestInGitHubActions(t)
	if v := timestampValue(nil); !v.IsNull() {
		t.Errorf("nil timestamp = %v, want null", v)
	}
	ts := &sdk.GoogleProtobufTimestamp{Seconds: sdk.PtrInt64(1790000000)}
	if got, want := timestampValue(ts).ValueString(), "2026-09-21T14:13:20Z"; got != want {
		t.Errorf("timestampValue = %q, want %q", got, want)
	}
}

func TestPlaybookJobPhaseChecks(t *testing.T) {
	skipUnitTestInGitHubActions(t)
	at := func(sec int64, nanos int32) *sdk.GoogleProtobufTimestamp {
		return &sdk.GoogleProtobufTimestamp{Seconds: sdk.PtrInt64(sec), Nanos: sdk.PtrInt32(nanos)}
	}

	dryRunOnly := &sdk.SdkAutomationPlaybookJob{Status: sdk.PtrString(playbookJobStatusDryRunComplete), DryRunEndedAt: at(100, 0), RunEndedAt: at(100, 0)}
	if !playbookDryRunDone(dryRunOnly) {
		t.Error("a DRY_RUN_COMPLETE job should be done with its dry-run")
	}
	if playbookDeployDone(dryRunOnly) {
		t.Error("a job that never deployed should not be deploy-done")
	}
	if playbookJobFailure(dryRunOnly) != "" {
		t.Error("a passed dry-run should not report a failure")
	}

	// A dry-run that ended in any other status must never count as passed.
	endedOther := &sdk.SdkAutomationPlaybookJob{Status: sdk.PtrString("DRY_RUN_FAILED"), DryRunEndedAt: at(100, 0)}
	if playbookDryRunDone(endedOther) {
		t.Error("only DRY_RUN_COMPLETE should count as a passed dry-run")
	}
	if playbookJobFailure(endedOther) == "" {
		t.Error("a *_FAILED status should report a failure even without failed_phase")
	}
	if playbookJobFailure(&sdk.SdkAutomationPlaybookJob{Status: sdk.PtrString("ABORTED")}) == "" {
		t.Error("ABORTED should report a failure")
	}
	if playbookJobFailure(&sdk.SdkAutomationPlaybookJob{FailedPhase: sdk.PtrString("dry_run")}) == "" {
		t.Error("a job with failed_phase should report a failure")
	}

	// run_ended_at stamped earlier in the same second as deploy end is stale.
	deploying := &sdk.SdkAutomationPlaybookJob{DeployEndedAt: at(200, 500), RunEndedAt: at(200, 100)}
	if playbookDeployDone(deploying) {
		t.Error("deploy should not be done while run_ended_at predates deploy_ended_at")
	}
	// Post-deploy check still running after the deploy.
	deploying.RunEndedAt = at(200, 900)
	deploying.PostDeployCheckStartedAt = at(201, 0)
	if playbookDeployDone(deploying) {
		t.Error("deploy should not be done while the post-deploy check started after run_ended_at")
	}
	deploying.RunEndedAt = at(250, 0)
	if !playbookDeployDone(deploying) {
		t.Error("deploy should be done once the run ends after the deploy and post-deploy check")
	}
}
