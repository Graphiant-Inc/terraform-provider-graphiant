package provider

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
)

func insecureHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 60 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 -- opt-in via provider.insecure_skip_verify for lab/on-prem controllers
		},
	}
}

// closeBody closes an SDK call's HTTP response body, if any. The error is
// intentionally discarded: by the time we're closing it, the body has
// already been fully read (or the request failed before one arrived), so a
// Close error carries no actionable information.
func closeBody(httpRes *http.Response) {
	if httpRes != nil {
		_ = httpRes.Body.Close()
	}
}

// apiErrorDetail expands an SDK call error with the API's response body.
// graphiant-sdk-go's GenericOpenAPIError.Error() returns only the HTTP
// status line (e.g. "400 Bad Request") — the actual validation/error detail
// the API returned lives in its Body(), which callers must opt into reading.
func apiErrorDetail(err error) string {
	var apiErr *graphiant.GenericOpenAPIError
	if errors.As(err, &apiErr) {
		if body := strings.TrimSpace(string(apiErr.Body())); body != "" {
			return fmt.Sprintf("%s: %s", err.Error(), body)
		}
	}
	return err.Error()
}

// strPtr returns nil for an empty/unknown/null types.String, otherwise a pointer to its value.
func strPtr(v types.String) *string {
	if v.IsNull() || v.IsUnknown() || v.ValueString() == "" {
		return nil
	}
	s := v.ValueString()
	return &s
}

// strValue converts an SDK *string to types.String, mapping nil to null.
func strValue(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

// int64Ptr returns nil for a null/unknown types.Int64, otherwise a pointer to its value.
func int64Ptr(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := v.ValueInt64()
	return &i
}

func int32Value(v *int32) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}

func int64Value(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

func boolPtr(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

func boolValue(v *bool) types.Bool {
	if v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}

func float64Ptr(v types.Float64) *float64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	f := v.ValueFloat64()
	return &f
}

func float64Value(v *float64) types.Float64 {
	if v == nil {
		return types.Float64Null()
	}
	return types.Float64Value(*v)
}

// stringListValue converts an SDK []string into a types.List of strings,
// mapping nil to an empty (not null) list so computed attributes stay consistent.
func stringListValue(ctx context.Context, v []string) (types.List, diag.Diagnostics) {
	elems := v
	if elems == nil {
		elems = []string{}
	}
	return types.ListValueFrom(ctx, types.StringType, elems)
}

// timestampValue formats an SDK protobuf timestamp as RFC3339 UTC, mapping nil/zero to null.
func timestampValue(ts *graphiant.GoogleProtobufTimestamp) types.String {
	if ts == nil || ts.Seconds == nil {
		return types.StringNull()
	}
	var nanos int32
	if ts.Nanos != nil {
		nanos = *ts.Nanos
	}
	return types.StringValue(time.Unix(*ts.Seconds, int64(nanos)).UTC().Format(time.RFC3339))
}
