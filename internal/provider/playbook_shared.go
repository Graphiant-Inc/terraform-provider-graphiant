package provider

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

// Shared helpers for the SDK automation (Automation Workflows) playbook API:
// graphiant_playbook_config, graphiant_playbook_job and the graphiant_playbook_*
// data sources.

// Job statuses named in the API's own parameter/field descriptions. The full enum is
// undocumented, so the checks below are deliberately conservative: a dry-run only
// counts as passed on the exact DRY_RUN_COMPLETE status (the state the approve
// endpoint requires), and any status mentioning FAIL or ABORT counts as a failure.
const (
	playbookJobStatusDryRunComplete = "DRY_RUN_COMPLETE"
	playbookJobStatusAwaitingReauth = "AWAITING_REAUTH"
)

const playbookJobPollInterval = 10 * time.Second

// playbookJobNotFoundGrace is how long a freshly started job may 404 before the
// wait gives up, to tolerate read-after-write lag right after the dry-run POST.
const playbookJobNotFoundGrace = 1 * time.Minute

var playbookModuleFileAttrTypes = map[string]attrType{
	"module_key": types.StringType,
	"filename":   types.StringType,
	"content":    types.StringType,
}

type playbookModuleFileModel struct {
	ModuleKey types.String `tfsdk:"module_key"`
	Filename  types.String `tfsdk:"filename"`
	Content   types.String `tfsdk:"content"`
}

func expandPlaybookFiles(ctx context.Context, list types.List) ([]sdk.SdkAutomationModuleFile, diag.Diagnostics) {
	var models []playbookModuleFileModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}
	files := make([]sdk.SdkAutomationModuleFile, 0, len(models))
	for _, m := range models {
		files = append(files, sdk.SdkAutomationModuleFile{
			ModuleKey: m.ModuleKey.ValueStringPointer(),
			Filename:  m.Filename.ValueStringPointer(),
			Content:   m.Content.ValueStringPointer(),
		})
	}
	return files, diags
}

// flattenPlaybookFiles converts API module files to state, ordered to match prior
// (the files already in plan/state) where the same module_key+filename appears in
// both, so a server-side reordering doesn't show up as a diff. Files only the API
// returns are appended in API order.
func flattenPlaybookFiles(ctx context.Context, files []sdk.SdkAutomationModuleFile, prior types.List) (types.List, diag.Diagnostics) {
	key := func(moduleKey, filename string) string { return moduleKey + "\x00" + filename }

	byKey := make(map[string]sdk.SdkAutomationModuleFile, len(files))
	for _, f := range files {
		byKey[key(f.GetModuleKey(), f.GetFilename())] = f
	}

	models := make([]playbookModuleFileModel, 0, len(files))
	used := make(map[string]bool, len(files))
	if !prior.IsNull() && !prior.IsUnknown() {
		var priorModels []playbookModuleFileModel
		if diags := prior.ElementsAs(ctx, &priorModels, false); diags.HasError() {
			return types.ListNull(types.ObjectType{AttrTypes: playbookModuleFileAttrTypes}), diags
		}
		for _, p := range priorModels {
			k := key(p.ModuleKey.ValueString(), p.Filename.ValueString())
			if f, ok := byKey[k]; ok && !used[k] {
				models = append(models, playbookModuleFileModelFrom(f))
				used[k] = true
			}
		}
	}
	for _, f := range files {
		k := key(f.GetModuleKey(), f.GetFilename())
		if !used[k] {
			models = append(models, playbookModuleFileModelFrom(f))
			used[k] = true
		}
	}
	return types.ListValueFrom(ctx, types.ObjectType{AttrTypes: playbookModuleFileAttrTypes}, models)
}

func playbookModuleFileModelFrom(f sdk.SdkAutomationModuleFile) playbookModuleFileModel {
	return playbookModuleFileModel{
		ModuleKey: types.StringPointerValue(f.ModuleKey),
		Filename:  types.StringPointerValue(f.Filename),
		Content:   types.StringPointerValue(f.Content),
	}
}

// playbookValidationErrorsDetail renders validation errors returned by config
// create/update into a single diagnostic detail.
func playbookValidationErrorsDetail(errs []sdk.SdkAutomationValidationError) string {
	lines := make([]string, 0, len(errs))
	for _, e := range errs {
		var b strings.Builder
		if e.GetModuleKey() != "" {
			fmt.Fprintf(&b, "[%s] ", e.GetModuleKey())
		}
		if e.GetType() != "" {
			fmt.Fprintf(&b, "%s: ", e.GetType())
		}
		b.WriteString(e.GetMessage())
		lines = append(lines, "- "+b.String())
	}
	return "The playbook config failed validation:\n" + strings.Join(lines, "\n")
}

// timestampValue formats an API timestamp as RFC 3339 (UTC), or null when unset.
func timestampValue(ts *sdk.GoogleProtobufTimestamp) types.String {
	if ts == nil || ts.Seconds == nil {
		return types.StringNull()
	}
	return types.StringValue(time.Unix(ts.GetSeconds(), int64(ts.GetNanos())).UTC().Format(time.RFC3339))
}

// optionalStringValue maps an API string to state, keeping null when the API
// echoes back "" for a field the configuration never set (prior is null).
func optionalStringValue(v *string, prior types.String) types.String {
	if (v == nil || *v == "") && prior.IsNull() {
		return types.StringNull()
	}
	return types.StringPointerValue(v)
}

var playbookJobAttrTypes = map[string]attrType{
	"job_id":                       types.StringType,
	"config_id":                    types.StringType,
	"config_name":                  types.StringType,
	"config_description":           types.StringType,
	"status":                       types.StringType,
	"failed_phase":                 types.StringType,
	"run_number":                   types.Int64Type,
	"started_by_user_id":           types.StringType,
	"verbose_logs":                 types.BoolType,
	"logs_available":               types.BoolType,
	"sdk_version":                  types.StringType,
	"collection_version":           types.StringType,
	"dry_run_started_at":           types.StringType,
	"dry_run_ended_at":             types.StringType,
	"deploy_started_at":            types.StringType,
	"deploy_ended_at":              types.StringType,
	"post_deploy_check_started_at": types.StringType,
	"run_ended_at":                 types.StringType,
	"run_duration_ms":              types.Int64Type,
}

type playbookJobModel struct {
	JobID                    types.String `tfsdk:"job_id"`
	ConfigID                 types.String `tfsdk:"config_id"`
	ConfigName               types.String `tfsdk:"config_name"`
	ConfigDescription        types.String `tfsdk:"config_description"`
	Status                   types.String `tfsdk:"status"`
	FailedPhase              types.String `tfsdk:"failed_phase"`
	RunNumber                types.Int64  `tfsdk:"run_number"`
	StartedByUserID          types.String `tfsdk:"started_by_user_id"`
	VerboseLogs              types.Bool   `tfsdk:"verbose_logs"`
	LogsAvailable            types.Bool   `tfsdk:"logs_available"`
	SdkVersion               types.String `tfsdk:"sdk_version"`
	CollectionVersion        types.String `tfsdk:"collection_version"`
	DryRunStartedAt          types.String `tfsdk:"dry_run_started_at"`
	DryRunEndedAt            types.String `tfsdk:"dry_run_ended_at"`
	DeployStartedAt          types.String `tfsdk:"deploy_started_at"`
	DeployEndedAt            types.String `tfsdk:"deploy_ended_at"`
	PostDeployCheckStartedAt types.String `tfsdk:"post_deploy_check_started_at"`
	RunEndedAt               types.String `tfsdk:"run_ended_at"`
	RunDurationMs            types.Int64  `tfsdk:"run_duration_ms"`
}

func playbookJobModelFrom(j *sdk.SdkAutomationPlaybookJob) playbookJobModel {
	return playbookJobModel{
		JobID:                    types.StringPointerValue(j.JobId),
		ConfigID:                 types.StringPointerValue(j.ConfigId),
		ConfigName:               types.StringPointerValue(j.ConfigName),
		ConfigDescription:        types.StringPointerValue(j.ConfigDescription),
		Status:                   types.StringPointerValue(j.Status),
		FailedPhase:              types.StringPointerValue(j.FailedPhase),
		RunNumber:                types.Int64PointerValue(intPtr32To64(j.RunNumber)),
		StartedByUserID:          types.StringPointerValue(j.StartedByUserId),
		VerboseLogs:              types.BoolPointerValue(j.VerboseLogs),
		LogsAvailable:            types.BoolPointerValue(j.LogsAvailable),
		SdkVersion:               types.StringPointerValue(j.SdkVersion),
		CollectionVersion:        types.StringPointerValue(j.CollectionVersion),
		DryRunStartedAt:          timestampValue(j.DryRunStartedAt),
		DryRunEndedAt:            timestampValue(j.DryRunEndedAt),
		DeployStartedAt:          timestampValue(j.DeployStartedAt),
		DeployEndedAt:            timestampValue(j.DeployEndedAt),
		PostDeployCheckStartedAt: timestampValue(j.PostDeployCheckStartedAt),
		RunEndedAt:               timestampValue(j.RunEndedAt),
		RunDurationMs:            types.Int64PointerValue(j.RunDurationMs),
	}
}

// getPlaybookJob fetches a job by id. found is false (with no error) on HTTP 404.
func getPlaybookJob(ctx context.Context, pd *providerData, jobID string) (*sdk.SdkAutomationPlaybookJob, bool, error) {
	out, httpResp, err := pd.api.DefaultAPI.V1SdkAutomationPlaybookJobsJobIdGet(ctx, jobID).Authorization(pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		if isNotFound(httpResp) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("%s", apiErrorDetail(err))
	}
	if out == nil || out.Job == nil {
		return nil, false, nil
	}
	return out.Job, true, nil
}

// playbookJobFailure returns a non-empty reason when the job has failed, been
// aborted, or is parked waiting for the user to re-authenticate.
func playbookJobFailure(j *sdk.SdkAutomationPlaybookJob) string {
	switch {
	case j.GetFailedPhase() != "":
		return fmt.Sprintf("job failed in phase %q (status %q)", j.GetFailedPhase(), j.GetStatus())
	case j.GetStatus() == playbookJobStatusAwaitingReauth:
		return "job is paused awaiting re-authentication; resume it from the Graphiant Portal (Automation Workflows)"
	case playbookJobStatusFailedOrAborted(j.GetStatus()):
		return fmt.Sprintf("job ended with status %q", j.GetStatus())
	}
	return ""
}

// playbookJobStatusFailedOrAborted matches FAILED/ABORTED and any phase-specific
// variant (e.g. DRY_RUN_FAILED), since the full status enum is undocumented.
func playbookJobStatusFailedOrAborted(status string) bool {
	s := strings.ToUpper(status)
	return strings.Contains(s, "FAIL") || strings.Contains(s, "ABORT")
}

// playbookDryRunDone reports whether the job's mandatory dry-run has passed and the
// job is waiting for approval. Only the exact status counts: a dry-run that ended in
// any other state is never treated as passed, so it can't lead to an approval.
func playbookDryRunDone(j *sdk.SdkAutomationPlaybookJob) bool {
	return j.GetStatus() == playbookJobStatusDryRunComplete
}

// playbookDeployDone reports whether an approved job has finished its deploy and
// any post-deploy check: the run end is stamped after both (compared at full
// precision, so a stale same-second run end from the dry-run doesn't count).
func playbookDeployDone(j *sdk.SdkAutomationPlaybookJob) bool {
	if j.DeployEndedAt == nil || j.RunEndedAt == nil {
		return false
	}
	if timestampBefore(j.RunEndedAt, j.DeployEndedAt) {
		return false
	}
	return j.PostDeployCheckStartedAt == nil || !timestampBefore(j.RunEndedAt, j.PostDeployCheckStartedAt)
}

func timestampBefore(a, b *sdk.GoogleProtobufTimestamp) bool {
	if a.GetSeconds() != b.GetSeconds() {
		return a.GetSeconds() < b.GetSeconds()
	}
	return a.GetNanos() < b.GetNanos()
}

// waitForPlaybookJob polls a job until done reports true, the job fails, or timeout
// elapses. It always returns the last job read, if any, so callers can save state.
func waitForPlaybookJob(ctx context.Context, pd *providerData, jobID string, timeout time.Duration, done func(*sdk.SdkAutomationPlaybookJob) bool) (*sdk.SdkAutomationPlaybookJob, error) {
	start := time.Now()
	deadline := start.Add(timeout)
	var last *sdk.SdkAutomationPlaybookJob
	for {
		job, found, err := getPlaybookJob(ctx, pd, jobID)
		if err != nil {
			return last, fmt.Errorf("polling job %s: %w", jobID, err)
		}
		if !found {
			if last != nil || time.Since(start) > playbookJobNotFoundGrace {
				return last, fmt.Errorf("job %s no longer exists", jobID)
			}
		} else {
			last = job
		}
		if job == nil {
			if err := sleepCtx(ctx, playbookJobPollInterval); err != nil {
				return last, err
			}
			continue
		}
		if reason := playbookJobFailure(job); reason != "" {
			return last, fmt.Errorf("job %s: %s", jobID, reason)
		}
		if done(job) {
			return last, nil
		}
		if time.Now().After(deadline) {
			return last, fmt.Errorf("job %s did not finish within %s (last status %q)", jobID, timeout, job.GetStatus())
		}
		if err := sleepCtx(ctx, playbookJobPollInterval); err != nil {
			return last, err
		}
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}
