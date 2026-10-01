package provider

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ resource.Resource                = &playbookJobResource{}
	_ resource.ResourceWithConfigure   = &playbookJobResource{}
	_ resource.ResourceWithImportState = &playbookJobResource{}
	_ resource.ResourceWithModifyPlan  = &playbookJobResource{}
)

func NewPlaybookJobResource() resource.Resource {
	return &playbookJobResource{}
}

type playbookJobResource struct {
	pd *providerData
}

type playbookJobResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	ConfigID           types.String `tfsdk:"config_id"`
	Approve            types.Bool   `tfsdk:"approve"`
	VerboseLogs        types.Bool   `tfsdk:"verbose_logs"`
	Triggers           types.Map    `tfsdk:"triggers"`
	WaitTimeoutMinutes types.Int64  `tfsdk:"wait_timeout_minutes"`
	Status             types.String `tfsdk:"status"`
	FailedPhase        types.String `tfsdk:"failed_phase"`
	RunNumber          types.Int64  `tfsdk:"run_number"`
	LogsAvailable      types.Bool   `tfsdk:"logs_available"`
	DryRunEndedAt      types.String `tfsdk:"dry_run_ended_at"`
	DeployStartedAt    types.String `tfsdk:"deploy_started_at"`
	DeployEndedAt      types.String `tfsdk:"deploy_ended_at"`
	RunEndedAt         types.String `tfsdk:"run_ended_at"`
}

func (r *playbookJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook_job"
}

func (r *playbookJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A run of a staged graphiant_playbook_config. Creating it starts the mandatory dry-run " +
			"(ansible --check) and waits for it to finish; nothing is deployed until approve is true, so the " +
			"deploy step is always an explicit change in a reviewed plan (Automation Workflows' Manual Approval " +
			"gate). Setting approve to true on an existing job deploys it in place and waits for the deploy and " +
			"post-deploy check. A failed dry-run or deploy fails the apply, and the next plan replaces the job " +
			"with a fresh dry-run (as it does for a job that failed outside Terraform). Changing config_id or triggers starts a new job; use triggers (e.g. a " +
			"hash of the config's files) to re-run after the config changes. Destroying the resource aborts the job " +
			"if it is still in progress or awaiting approval; completed jobs remain in the Portal's job history and " +
			"audit log. Re-run with skip_dry_run and resume-after-reauth are deliberately not exposed: re-run here " +
			"always goes through a new dry-run, and a job parked in AWAITING_REAUTH must be resumed from the Portal.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Job id.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"config_id": schema.StringAttribute{
				Required:      true,
				Description:   "Id of the staged graphiant_playbook_config to run. Changing it starts a new job.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"approve": schema.BoolAttribute{
				Optional: true,
				Description: "Approve the job for deploy once its dry-run has passed. Defaults to false (dry-run " +
					"only). Approval cannot be undone: setting it back to false does not roll back a deploy.",
			},
			"verbose_logs": schema.BoolAttribute{
				Optional:    true,
				Description: "Enable verbose (debug) ansible logging for the dry-run and deploy.",
			},
			"triggers": schema.MapAttribute{
				Optional:      true,
				ElementType:   types.StringType,
				Description:   "Arbitrary values that start a new job (fresh dry-run) whenever they change.",
				PlanModifiers: []planmodifier.Map{mapplanmodifier.RequiresReplace()},
			},
			"wait_timeout_minutes": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(30),
				Description: "How long to wait for the dry-run, and separately for the deploy, to finish. Defaults to 30.",
				Validators:  []validator.Int64{int64validator.AtLeast(1)},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Job status, e.g. DRY_RUN_COMPLETE while awaiting approval.",
			},
			"failed_phase": schema.StringAttribute{
				Computed:    true,
				Description: "Phase the job failed in (dry_run, deploy or post_deploy_check), if any.",
			},
			"run_number": schema.Int64Attribute{
				Computed:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"logs_available": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether logs can be read with the graphiant_playbook_job_logs data source.",
			},
			"dry_run_ended_at": schema.StringAttribute{
				Computed: true,
			},
			"deploy_started_at": schema.StringAttribute{
				Computed: true,
			},
			"deploy_ended_at": schema.StringAttribute{
				Computed: true,
			},
			"run_ended_at": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *playbookJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (m *playbookJobResourceModel) applyJob(j *sdk.SdkAutomationPlaybookJob) {
	if j == nil {
		return
	}
	if j.GetConfigId() != "" {
		m.ConfigID = types.StringValue(j.GetConfigId())
	}
	m.Status = types.StringPointerValue(j.Status)
	m.FailedPhase = types.StringPointerValue(j.FailedPhase)
	m.RunNumber = types.Int64PointerValue(intPtr32To64(j.RunNumber))
	m.LogsAvailable = types.BoolPointerValue(j.LogsAvailable)
	m.DryRunEndedAt = timestampValue(j.DryRunEndedAt)
	m.DeployStartedAt = timestampValue(j.DeployStartedAt)
	m.DeployEndedAt = timestampValue(j.DeployEndedAt)
	m.RunEndedAt = timestampValue(j.RunEndedAt)
}

// clearUnknowns nulls any computed attribute still unknown, e.g. when a wait fails
// before the job could be read back, so the state saved alongside the error is valid.
func (m *playbookJobResourceModel) clearUnknowns() {
	for _, s := range []*types.String{&m.Status, &m.FailedPhase, &m.DryRunEndedAt, &m.DeployStartedAt, &m.DeployEndedAt, &m.RunEndedAt} {
		if s.IsUnknown() {
			*s = types.StringNull()
		}
	}
	if m.RunNumber.IsUnknown() {
		m.RunNumber = types.Int64Null()
	}
	if m.LogsAvailable.IsUnknown() {
		m.LogsAvailable = types.BoolNull()
	}
}

func (m *playbookJobResourceModel) waitTimeout() time.Duration {
	return time.Duration(m.WaitTimeoutMinutes.ValueInt64()) * time.Minute
}

// approveAndWait sends the deploy approval and waits for the deploy to finish.
// approved reports whether the approval request itself was accepted, so a caller
// can avoid recording approve = true for a job that was never approved.
func (r *playbookJobResource) approveAndWait(ctx context.Context, m *playbookJobResourceModel) (job *sdk.SdkAutomationPlaybookJob, approved bool, summary string, err error) {
	body := sdk.V1SdkAutomationPlaybookJobsJobIdApprovePutRequest{VerboseLogs: m.VerboseLogs.ValueBoolPointer()}
	_, httpResp, err := r.pd.api.DefaultAPI.V1SdkAutomationPlaybookJobsJobIdApprovePut(ctx, m.ID.ValueString()).
		Authorization(r.pd.token).
		V1SdkAutomationPlaybookJobsJobIdApprovePutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		return nil, false, "Unable to approve playbook job", errors.New(apiErrorDetail(err))
	}
	job, err = waitForPlaybookJob(ctx, r.pd, m.ID.ValueString(), m.waitTimeout(), playbookDeployDone)
	if err != nil {
		return job, true, "Playbook job deploy did not succeed", err
	}
	return job, true, "", nil
}

func (r *playbookJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan playbookJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1SdkAutomationPlaybookConfigsConfigIdDryRunPostRequest{VerboseLogs: plan.VerboseLogs.ValueBoolPointer()}
	out, httpResp, err := r.pd.api.DefaultAPI.V1SdkAutomationPlaybookConfigsConfigIdDryRunPost(ctx, plan.ConfigID.ValueString()).
		Authorization(r.pd.token).
		V1SdkAutomationPlaybookConfigsConfigIdDryRunPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to start playbook dry-run", apiErrorDetail(err))
		return
	}
	if out == nil || out.GetJobId() == "" {
		resp.Diagnostics.AddError("Unable to start playbook dry-run", "API response did not include a job id")
		return
	}
	plan.ID = types.StringValue(out.GetJobId())

	// From here on, state is saved even on failure so Terraform tracks (and taints)
	// the job rather than losing it.
	defer func() {
		plan.clearUnknowns()
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}()

	job, err := waitForPlaybookJob(ctx, r.pd, plan.ID.ValueString(), plan.waitTimeout(), playbookDryRunDone)
	plan.applyJob(job)
	if err != nil {
		resp.Diagnostics.AddError("Playbook dry-run did not succeed", err.Error())
		return
	}

	if plan.Approve.ValueBool() {
		job, _, summary, err := r.approveAndWait(ctx, &plan)
		plan.applyJob(job)
		if err != nil {
			resp.Diagnostics.AddError(summary, err.Error())
			return
		}
	}
}

func (r *playbookJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state playbookJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, found, err := getPlaybookJob(ctx, r.pd, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read playbook job", err.Error())
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	// approve and verbose_logs are configuration, not refreshed here: an approval
	// made in the Portal must not fight an approve-less config on every plan.
	// ImportState derives them once instead.
	state.applyJob(job)
	if state.WaitTimeoutMinutes.IsNull() {
		state.WaitTimeoutMinutes = types.Int64Value(30)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *playbookJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state playbookJobResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	defer func() {
		plan.clearUnknowns()
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}()

	// Carry server-side fields forward; only an approval below changes them.
	job, found, err := getPlaybookJob(ctx, r.pd, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read playbook job", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update playbook job", "job no longer exists")
		return
	}
	plan.applyJob(job)

	switch {
	case plan.Approve.ValueBool() && !state.Approve.ValueBool():
		job, approved, summary, err := r.approveAndWait(ctx, &plan)
		plan.applyJob(job)
		if err != nil {
			resp.Diagnostics.AddError(summary, err.Error())
			// If the approval was never accepted, keep the prior value so the next
			// apply retries it. If it was accepted and the deploy then failed,
			// ModifyPlan replaces the failed job with a fresh dry-run instead.
			if !approved {
				plan.Approve = state.Approve
			}
		}
	case !plan.Approve.ValueBool() && state.Approve.ValueBool():
		resp.Diagnostics.AddWarning("Playbook job approval cannot be undone",
			"approve was set back to false, but the job has already been approved for deploy. Changes it applied "+
				"to the network are not rolled back; apply a corrected config with a new job instead.")
	}
}

func (r *playbookJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state playbookJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	job, found, err := getPlaybookJob(ctx, r.pd, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to read playbook job", err.Error())
		return
	}
	if !found {
		return
	}
	// Finished jobs stay in the job history / audit log; there's nothing to delete.
	status := job.GetStatus()
	if job.GetFailedPhase() != "" || playbookJobStatusFailedOrAborted(status) || playbookDeployDone(job) {
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1SdkAutomationPlaybookJobsJobIdAbortPut(ctx, state.ID.ValueString()).
		Authorization(r.pd.token).
		Body(map[string]interface{}{}).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddWarning("Unable to abort playbook job",
			"The job was removed from Terraform state but could not be aborted (status "+status+"): "+apiErrorDetail(err))
	}
}

// ModifyPlan replaces a job that has failed or been aborted (during a Terraform
// apply or outside it), so the next apply starts a fresh dry-run instead of
// leaving a dead job in state.
func (r *playbookJobResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var state playbookJobResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.FailedPhase.ValueString() == "" && !playbookJobStatusFailedOrAborted(state.Status.ValueString()) {
		return
	}
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("status"), types.StringUnknown())...)
	resp.RequiresReplace = append(resp.RequiresReplace, path.Root("status"))
}

// ImportState derives approve and verbose_logs from the job once, since Read
// deliberately leaves them as configured.
func (r *playbookJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	job, found, err := getPlaybookJob(ctx, r.pd, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read playbook job", err.Error())
		return
	}
	if !found {
		resp.Diagnostics.AddError("Playbook job not found", "no job with id "+req.ID)
		return
	}
	if job.DeployStartedAt != nil {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("approve"), true)...)
	}
	if job.GetVerboseLogs() {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("verbose_logs"), true)...)
	}
}
