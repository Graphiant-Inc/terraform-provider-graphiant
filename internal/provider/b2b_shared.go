package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

// Shared model types and schema/conversion helpers for the four B2B ("partner" data
// exchange) resources: producer service, customer, match, consumer. All four accept
// a nat_translation_mode block and most accept ManaV2B2bSiteInformation-shaped site
// lists, so those are defined once here rather than duplicated per resource.

type b2bSiteInfoModel struct {
	Sites     types.List `tfsdk:"sites"`
	SiteLists types.List `tfsdk:"site_lists"`
}

var b2bSiteInfoAttrTypes = map[string]attrType{
	"sites":      types.ListType{ElemType: types.Int64Type},
	"site_lists": types.ListType{ElemType: types.Int64Type},
}

var b2bSiteInfoListType = types.ObjectType{AttrTypes: b2bSiteInfoAttrTypes}

func b2bSitesAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"sites":      schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
				"site_lists": schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
			},
		},
	}
}

func buildB2bSiteInfoList(ctx context.Context, list types.List) ([]sdk.ManaV2B2bSiteInformation, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	var models []b2bSiteInfoModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]sdk.ManaV2B2bSiteInformation, 0, len(models))
	for _, m := range models {
		var info sdk.ManaV2B2bSiteInformation
		if !m.Sites.IsNull() && !m.Sites.IsUnknown() {
			diags.Append(m.Sites.ElementsAs(ctx, &info.Sites, false)...)
		}
		if !m.SiteLists.IsNull() && !m.SiteLists.IsUnknown() {
			diags.Append(m.SiteLists.ElementsAs(ctx, &info.SiteLists, false)...)
		}
		out = append(out, info)
	}
	return out, diags
}

func applyB2bSiteInfoList(ctx context.Context, in []sdk.ManaV2B2bSiteInformation) (types.List, diag.Diagnostics) {
	if len(in) == 0 {
		return types.ListNull(b2bSiteInfoListType), nil
	}
	models := make([]b2bSiteInfoModel, 0, len(in))
	var diags diag.Diagnostics
	for _, info := range in {
		sites, d := types.ListValueFrom(ctx, types.Int64Type, info.Sites)
		diags.Append(d...)
		siteLists, d2 := types.ListValueFrom(ctx, types.Int64Type, info.SiteLists)
		diags.Append(d2...)
		models = append(models, b2bSiteInfoModel{Sites: sites, SiteLists: siteLists})
	}
	if diags.HasError() {
		return types.ListNull(b2bSiteInfoListType), diags
	}
	return types.ListValueFrom(ctx, b2bSiteInfoListType, models)
}

type b2bPrefixTagModel struct {
	Prefix types.String `tfsdk:"prefix"`
	Tag    types.String `tfsdk:"tag"`
}

var b2bPrefixTagAttrTypes = map[string]attrType{
	"prefix": types.StringType,
	"tag":    types.StringType,
}

var b2bPrefixTagListType = types.ObjectType{AttrTypes: b2bPrefixTagAttrTypes}

func b2bPrefixTagsAttribute() schema.ListNestedAttribute {
	return schema.ListNestedAttribute{
		Optional: true,
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"prefix": schema.StringAttribute{Required: true},
				"tag":    schema.StringAttribute{Optional: true},
			},
		},
	}
}

func buildB2bPrefixTags(ctx context.Context, list types.List) ([]sdk.ManaV2B2bExtranetPrefixTag, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}
	var models []b2bPrefixTagModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}
	out := make([]sdk.ManaV2B2bExtranetPrefixTag, 0, len(models))
	for _, m := range models {
		out = append(out, sdk.ManaV2B2bExtranetPrefixTag{Prefix: m.Prefix.ValueString(), Tag: m.Tag.ValueStringPointer()})
	}
	return out, diags
}

func applyB2bPrefixTags(ctx context.Context, in []sdk.ManaV2B2bExtranetPrefixTag) (types.List, diag.Diagnostics) {
	models := make([]b2bPrefixTagModel, 0, len(in))
	for _, t := range in {
		models = append(models, b2bPrefixTagModel{Prefix: types.StringValue(t.Prefix), Tag: types.StringPointerValue(t.Tag)})
	}
	return types.ListValueFrom(ctx, b2bPrefixTagListType, models)
}

// nat_translation_mode: a oneof of centralized / decentralized (both map[string]{prefixes})
// / peer_to_peer (a list of {prefix, outside_nat_prefix}).

var b2bDevicePrefixesAttrTypes = map[string]attrType{
	"prefixes": types.ListType{ElemType: types.StringType},
}

type b2bDevicePrefixesModel struct {
	Prefixes types.List `tfsdk:"prefixes"`
}

var b2bNatSideAttrTypes = map[string]attrType{
	"prefixes": types.MapType{ElemType: types.ObjectType{AttrTypes: b2bDevicePrefixesAttrTypes}},
}

type b2bNatSideModel struct {
	Prefixes types.Map `tfsdk:"prefixes"`
}

var b2bNatPeerToPeerPrefixAttrTypes = map[string]attrType{
	"prefix":             types.StringType,
	"outside_nat_prefix": types.StringType,
}

type b2bNatPeerToPeerPrefixModel struct {
	Prefix           types.String `tfsdk:"prefix"`
	OutsideNatPrefix types.String `tfsdk:"outside_nat_prefix"`
}

var b2bNatModeAttrTypes = map[string]attrType{
	"centralized":   types.ObjectType{AttrTypes: b2bNatSideAttrTypes},
	"decentralized": types.ObjectType{AttrTypes: b2bNatSideAttrTypes},
	"peer_to_peer":  types.ListType{ElemType: types.ObjectType{AttrTypes: b2bNatPeerToPeerPrefixAttrTypes}},
}

func b2bNatModeAttribute() schema.SingleNestedAttribute {
	natSide := func() schema.SingleNestedAttribute {
		return schema.SingleNestedAttribute{
			Optional: true,
			Attributes: map[string]schema.Attribute{
				"prefixes": schema.MapNestedAttribute{
					Optional:    true,
					Description: "Keyed by device id.",
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"prefixes": schema.ListAttribute{Optional: true, ElementType: types.StringType},
						},
					},
				},
			},
		}
	}
	return schema.SingleNestedAttribute{
		Optional:    true,
		Description: "Exactly one of centralized / decentralized / peer_to_peer should be set (not enforced by a validator).",
		Attributes: map[string]schema.Attribute{
			"centralized":   natSide(),
			"decentralized": natSide(),
			"peer_to_peer": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"prefix":             schema.StringAttribute{Optional: true},
						"outside_nat_prefix": schema.StringAttribute{Optional: true},
					},
				},
			},
		},
	}
}

func buildB2bNatSide(ctx context.Context, obj types.Object) (*map[string]sdk.ManaV2ExtranetNatTranslationDevicePrefixes, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}
	var m b2bNatSideModel
	diags.Append(obj.As(ctx, &m, objectAsOptions)...)
	if diags.HasError() {
		return nil, diags
	}
	result := map[string]sdk.ManaV2ExtranetNatTranslationDevicePrefixes{}
	if !m.Prefixes.IsNull() && !m.Prefixes.IsUnknown() {
		var byDevice map[string]b2bDevicePrefixesModel
		diags.Append(m.Prefixes.ElementsAs(ctx, &byDevice, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for k, v := range byDevice {
			var prefixes []string
			if !v.Prefixes.IsNull() && !v.Prefixes.IsUnknown() {
				diags.Append(v.Prefixes.ElementsAs(ctx, &prefixes, false)...)
			}
			result[k] = sdk.ManaV2ExtranetNatTranslationDevicePrefixes{Prefixes: prefixes}
		}
	}
	return &result, diags
}

func applyB2bNatSide(ctx context.Context, in *map[string]sdk.ManaV2ExtranetNatTranslationDevicePrefixes) (types.Object, diag.Diagnostics) {
	if in == nil {
		return types.ObjectNull(b2bNatSideAttrTypes), nil
	}
	byDevice := make(map[string]b2bDevicePrefixesModel, len(*in))
	var diags diag.Diagnostics
	for k, v := range *in {
		list, d := types.ListValueFrom(ctx, types.StringType, v.Prefixes)
		diags.Append(d...)
		byDevice[k] = b2bDevicePrefixesModel{Prefixes: list}
	}
	if diags.HasError() {
		return types.ObjectNull(b2bNatSideAttrTypes), diags
	}
	prefixesMap, d2 := types.MapValueFrom(ctx, types.ObjectType{AttrTypes: b2bDevicePrefixesAttrTypes}, byDevice)
	diags.Append(d2...)
	if diags.HasError() {
		return types.ObjectNull(b2bNatSideAttrTypes), diags
	}
	return types.ObjectValueFrom(ctx, b2bNatSideAttrTypes, b2bNatSideModel{Prefixes: prefixesMap})
}

func buildB2bNatMode(ctx context.Context, obj types.Object) (*sdk.ManaV2ExtranetNatTranslationMode, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}
	var m struct {
		Centralized   types.Object `tfsdk:"centralized"`
		Decentralized types.Object `tfsdk:"decentralized"`
		PeerToPeer    types.List   `tfsdk:"peer_to_peer"`
	}
	diags.Append(obj.As(ctx, &m, objectAsOptions)...)
	if diags.HasError() {
		return nil, diags
	}

	mode := &sdk.ManaV2ExtranetNatTranslationMode{}

	centralized, d := buildB2bNatSide(ctx, m.Centralized)
	diags.Append(d...)
	if centralized != nil {
		mode.Centralized = &sdk.ManaV2ExtranetNatTranslationCentralized{Prefixes: centralized}
	}

	decentralized, d2 := buildB2bNatSide(ctx, m.Decentralized)
	diags.Append(d2...)
	if decentralized != nil {
		mode.Decentralized = &sdk.ManaV2ExtranetNatTranslationDecentralized{Prefixes: decentralized}
	}

	if !m.PeerToPeer.IsNull() && !m.PeerToPeer.IsUnknown() {
		var p2pModels []b2bNatPeerToPeerPrefixModel
		diags.Append(m.PeerToPeer.ElementsAs(ctx, &p2pModels, false)...)
		if !diags.HasError() {
			prefixes := make([]sdk.ManaV2ExtranetNatTranslationPeerToPeerPrefix, 0, len(p2pModels))
			for _, p := range p2pModels {
				prefixes = append(prefixes, sdk.ManaV2ExtranetNatTranslationPeerToPeerPrefix{
					Prefix:           p.Prefix.ValueStringPointer(),
					OutsideNatPrefix: p.OutsideNatPrefix.ValueStringPointer(),
				})
			}
			mode.PeerToPeer = &sdk.ManaV2ExtranetNatTranslationPeerToPeer{Prefixes: prefixes}
		}
	}

	if diags.HasError() {
		return nil, diags
	}
	return mode, diags
}

func applyB2bNatMode(ctx context.Context, mode *sdk.ManaV2ExtranetNatTranslationMode) (types.Object, diag.Diagnostics) {
	if mode == nil {
		return types.ObjectNull(b2bNatModeAttrTypes), nil
	}
	var diags diag.Diagnostics

	var centralizedPrefixes *map[string]sdk.ManaV2ExtranetNatTranslationDevicePrefixes
	if mode.Centralized != nil {
		centralizedPrefixes = mode.Centralized.Prefixes
	}
	centralized, d := applyB2bNatSide(ctx, centralizedPrefixes)
	diags.Append(d...)

	var decentralizedPrefixes *map[string]sdk.ManaV2ExtranetNatTranslationDevicePrefixes
	if mode.Decentralized != nil {
		decentralizedPrefixes = mode.Decentralized.Prefixes
	}
	decentralized, d2 := applyB2bNatSide(ctx, decentralizedPrefixes)
	diags.Append(d2...)

	peerToPeer := types.ListNull(types.ObjectType{AttrTypes: b2bNatPeerToPeerPrefixAttrTypes})
	if mode.PeerToPeer != nil {
		models := make([]b2bNatPeerToPeerPrefixModel, 0, len(mode.PeerToPeer.Prefixes))
		for _, p := range mode.PeerToPeer.Prefixes {
			models = append(models, b2bNatPeerToPeerPrefixModel{
				Prefix:           types.StringPointerValue(p.Prefix),
				OutsideNatPrefix: types.StringPointerValue(p.OutsideNatPrefix),
			})
		}
		list, d3 := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: b2bNatPeerToPeerPrefixAttrTypes}, models)
		diags.Append(d3...)
		peerToPeer = list
	}

	if diags.HasError() {
		return types.ObjectNull(b2bNatModeAttrTypes), diags
	}

	return types.ObjectValueFrom(ctx, b2bNatModeAttrTypes, struct {
		Centralized   types.Object `tfsdk:"centralized"`
		Decentralized types.Object `tfsdk:"decentralized"`
		PeerToPeer    types.List   `tfsdk:"peer_to_peer"`
	}{Centralized: centralized, Decentralized: decentralized, PeerToPeer: peerToPeer})
}
