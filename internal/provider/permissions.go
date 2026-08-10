package provider

import (
	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// permissionsModel mirrors commonPermissions. Every field is a free-form
// string in the API (observed values include things like "read", "write",
// "none"), so it is modeled here as string rather than an enum.
type permissionsModel struct {
	AssetManager                 types.String `tfsdk:"asset_manager"`
	B2b                          types.String `tfsdk:"b2b"`
	B2bSecurityProfileExternal   types.String `tfsdk:"b2b_security_profile_external"`
	BillingAndInvoicing          types.String `tfsdk:"billing_and_invoicing"`
	Compliance                   types.String `tfsdk:"compliance"`
	DeveloperTools               types.String `tfsdk:"developer_tools"`
	Gateway                      types.String `tfsdk:"gateway"`
	GlobalServices               types.String `tfsdk:"global_services"`
	Insights                     types.String `tfsdk:"insights"`
	Licensing                    types.String `tfsdk:"licensing"`
	Logs                         types.String `tfsdk:"logs"`
	MonitoringAndTroubleshooting types.String `tfsdk:"monitoring_and_troubleshooting"`
	NetworkConfiguration         types.String `tfsdk:"network_configuration"`
	OrderStatus                  types.String `tfsdk:"order_status"`
	Reports                      types.String `tfsdk:"reports"`
	SafetyAndSecurity            types.String `tfsdk:"safety_and_security"`
	ServicePolicies              types.String `tfsdk:"service_policies"`
	Support                      types.String `tfsdk:"support"`
	UserAndTenantManagement      types.String `tfsdk:"user_and_tenant_management"`
}

// permissionsFields lists the tfsdk attribute names for every permission,
// used to build the schema without repeating each attribute by hand.
var permissionsFields = []string{
	"asset_manager",
	"b2b",
	"b2b_security_profile_external",
	"billing_and_invoicing",
	"compliance",
	"developer_tools",
	"gateway",
	"global_services",
	"insights",
	"licensing",
	"logs",
	"monitoring_and_troubleshooting",
	"network_configuration",
	"order_status",
	"reports",
	"safety_and_security",
	"service_policies",
	"support",
	"user_and_tenant_management",
}

func permissionsSchemaAttribute(computed bool) schema.SingleNestedAttribute {
	attrs := make(map[string]schema.Attribute, len(permissionsFields))
	for _, name := range permissionsFields {
		attrs[name] = schema.StringAttribute{
			Optional:    !computed,
			Computed:    computed,
			Description: "Access level for this permission area (e.g. \"none\", \"read\", \"write\").",
		}
	}
	return schema.SingleNestedAttribute{
		Optional:    !computed,
		Computed:    computed,
		Attributes:  attrs,
		Description: "Per-area role permissions.",
	}
}

func expandPermissions(m *permissionsModel) *graphiant.CommonPermissions {
	if m == nil {
		return nil
	}
	p := graphiant.NewCommonPermissionsWithDefaults()
	if v := strPtr(m.AssetManager); v != nil {
		p.SetAssetManager(*v)
	}
	if v := strPtr(m.B2b); v != nil {
		p.SetB2b(*v)
	}
	if v := strPtr(m.B2bSecurityProfileExternal); v != nil {
		p.SetB2bSecurityProfileExternal(*v)
	}
	if v := strPtr(m.BillingAndInvoicing); v != nil {
		p.SetBillingAndInvoicing(*v)
	}
	if v := strPtr(m.Compliance); v != nil {
		p.SetCompliance(*v)
	}
	if v := strPtr(m.DeveloperTools); v != nil {
		p.SetDeveloperTools(*v)
	}
	if v := strPtr(m.Gateway); v != nil {
		p.SetGateway(*v)
	}
	if v := strPtr(m.GlobalServices); v != nil {
		p.SetGlobalServices(*v)
	}
	if v := strPtr(m.Insights); v != nil {
		p.SetInsights(*v)
	}
	if v := strPtr(m.Licensing); v != nil {
		p.SetLicensing(*v)
	}
	if v := strPtr(m.Logs); v != nil {
		p.SetLogs(*v)
	}
	if v := strPtr(m.MonitoringAndTroubleshooting); v != nil {
		p.SetMonitoringAndTroubleshooting(*v)
	}
	if v := strPtr(m.NetworkConfiguration); v != nil {
		p.SetNetworkConfiguration(*v)
	}
	if v := strPtr(m.OrderStatus); v != nil {
		p.SetOrderStatus(*v)
	}
	if v := strPtr(m.Reports); v != nil {
		p.SetReports(*v)
	}
	if v := strPtr(m.SafetyAndSecurity); v != nil {
		p.SetSafetyAndSecurity(*v)
	}
	if v := strPtr(m.ServicePolicies); v != nil {
		p.SetServicePolicies(*v)
	}
	if v := strPtr(m.Support); v != nil {
		p.SetSupport(*v)
	}
	if v := strPtr(m.UserAndTenantManagement); v != nil {
		p.SetUserAndTenantManagement(*v)
	}
	return p
}

func flattenPermissions(p *graphiant.CommonPermissions) *permissionsModel {
	if p == nil {
		return nil
	}
	return &permissionsModel{
		AssetManager:                 strValue(p.AssetManager),
		B2b:                          strValue(p.B2b),
		B2bSecurityProfileExternal:   strValue(p.B2bSecurityProfileExternal),
		BillingAndInvoicing:          strValue(p.BillingAndInvoicing),
		Compliance:                   strValue(p.Compliance),
		DeveloperTools:               strValue(p.DeveloperTools),
		Gateway:                      strValue(p.Gateway),
		GlobalServices:               strValue(p.GlobalServices),
		Insights:                     strValue(p.Insights),
		Licensing:                    strValue(p.Licensing),
		Logs:                         strValue(p.Logs),
		MonitoringAndTroubleshooting: strValue(p.MonitoringAndTroubleshooting),
		NetworkConfiguration:         strValue(p.NetworkConfiguration),
		OrderStatus:                  strValue(p.OrderStatus),
		Reports:                      strValue(p.Reports),
		SafetyAndSecurity:            strValue(p.SafetyAndSecurity),
		ServicePolicies:              strValue(p.ServicePolicies),
		Support:                      strValue(p.Support),
		UserAndTenantManagement:      strValue(p.UserAndTenantManagement),
	}
}
