package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &assuranceDnsproxyEntriesDataSource{}
	_ datasource.DataSourceWithConfigure = &assuranceDnsproxyEntriesDataSource{}
)

func NewAssuranceDnsproxyEntriesDataSource() datasource.DataSource {
	return &assuranceDnsproxyEntriesDataSource{}
}

type assuranceDnsproxyEntriesDataSource struct {
	pd *providerData
}

var dnsproxyEntryAttrTypes = map[string]attrType{
	"dnsproxy_entry_id": types.StringType,
	"name":              types.StringType,
	"name_text":         types.StringType,
	"ip_list":           types.ListType{ElemType: types.StringType},
	"port_list":         types.ListType{ElemType: types.StringType},
	"protocol":          types.StringType,
}

type dnsproxyEntryModel struct {
	DnsproxyEntryID types.String `tfsdk:"dnsproxy_entry_id"`
	Name            types.String `tfsdk:"name"`
	NameText        types.String `tfsdk:"name_text"`
	IPList          types.List   `tfsdk:"ip_list"`
	PortList        types.List   `tfsdk:"port_list"`
	Protocol        types.String `tfsdk:"protocol"`
}

type assuranceDnsproxyEntriesDataSourceModel struct {
	Entries types.List `tfsdk:"entries"`
}

func (d *assuranceDnsproxyEntriesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_assurance_dnsproxy_entries"
}

func (d *assuranceDnsproxyEntriesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Current DNS proxy filter entries for data assurance. Read-only: the API's delete endpoint " +
			"(V2AssuranceDeleteDnsproxyEntryDelete) has no way to target a specific entry — no path, query, or " +
			"body parameter identifies which entry to delete — so entries created via this provider would leak " +
			"permanently with no cleanup path. This is a data source rather than a resource for that reason; " +
			"manage entries via the portal until the API gains a working delete.",
		Attributes: map[string]schema.Attribute{
			"entries": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: dnsproxyEntryAttrTypes},
			},
		},
	}
}

func (d *assuranceDnsproxyEntriesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *assuranceDnsproxyEntriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg assuranceDnsproxyEntriesDataSourceModel

	out, httpResp, err := d.pd.api.DefaultAPI.V2AssuranceReadDnsproxyListGet(ctx).Authorization(d.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read DNS proxy entries", apiErrorDetail(err))
		return
	}

	models := make([]dnsproxyEntryModel, 0)
	if out != nil {
		for _, e := range out.DnsproxyList {
			ipList, diags := types.ListValueFrom(ctx, types.StringType, e.IpList)
			resp.Diagnostics.Append(diags...)
			portList, diags2 := types.ListValueFrom(ctx, types.StringType, e.PortList)
			resp.Diagnostics.Append(diags2...)
			models = append(models, dnsproxyEntryModel{
				DnsproxyEntryID: types.StringValue(e.DnsproxyEntryId),
				Name:            types.StringValue(e.Name),
				NameText:        types.StringValue(e.NameText),
				IPList:          ipList,
				PortList:        portList,
				Protocol:        types.StringValue(e.Protocol),
			})
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: dnsproxyEntryAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Entries = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
