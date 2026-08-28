package provider

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// attrType is a short alias used when building types.Object attribute-type maps.
type attrType = attr.Type

// objectAsOptions is the zero-value options used by every types.Object.As(...) call
// in this provider; our nested model structs use types.X fields throughout, which
// natively represent null/unknown, so no special unhandled-null/unknown behavior is needed.
var objectAsOptions = basetypes.ObjectAsOptions{}

// int64ID formats an int64 API identifier as the string form Terraform stores in state.
func int64ID(v int64) string {
	return strconv.FormatInt(v, 10)
}

// int64PtrID formats an optional int64 API identifier, returning "" when nil.
func int64PtrID(v *int64) string {
	if v == nil {
		return ""
	}
	return int64ID(*v)
}

// parseInt64ID parses a Terraform string id back into the int64 form the API expects.
func parseInt64ID(id string) (int64, error) {
	return strconv.ParseInt(id, 10, 64)
}

// intPtr32To64 widens an optional int32 (common on SDK response models) to the
// int64 this provider stores in Terraform state, nil-safe.
func intPtr32To64(v *int32) *int64 {
	if v == nil {
		return nil
	}
	w := int64(*v)
	return &w
}

// closeBody safely drains-free closes an SDK call's raw *http.Response, which every
// generated method returns alongside the typed result/error.
func closeBody(httpResp *http.Response) {
	if httpResp != nil && httpResp.Body != nil {
		_ = httpResp.Body.Close()
	}
}

// isNotFound reports whether an SDK call failed with HTTP 404, the signal a
// point-lookup-by-id Read() uses to remove the resource from state.
func isNotFound(httpResp *http.Response) bool {
	return httpResp != nil && httpResp.StatusCode == http.StatusNotFound
}

// configurePD extracts the shared *providerData from a resource/data source's
// Configure request, appending a diagnostic if the provider hasn't been configured
// yet (raw == nil, e.g. during terraform validate) or the type is unexpected.
func configurePD(raw interface{}, diags *diag.Diagnostics) *providerData {
	if raw == nil {
		return nil
	}
	pd, ok := raw.(*providerData)
	if !ok {
		diags.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *provider.providerData, got: %T. Please report this issue to the provider developers.", raw),
		)
		return nil
	}
	return pd
}
