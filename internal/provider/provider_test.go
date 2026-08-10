package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func TestResourceSchemas(t *testing.T) {
	ctx := context.Background()
	p := New("test")()

	for _, f := range p.(*GraphiantProvider).Resources(ctx) {
		r := f()
		var metaResp resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "graphiant"}, &metaResp)

		var schemaResp resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Errorf("%s: schema diagnostics: %v", metaResp.TypeName, schemaResp.Diagnostics)
		}
		if diags := schemaResp.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("%s: invalid schema: %v", metaResp.TypeName, diags)
		}
	}
}

func TestDataSourceSchemas(t *testing.T) {
	ctx := context.Background()
	p := New("test")()

	for _, f := range p.(*GraphiantProvider).DataSources(ctx) {
		d := f()
		var metaResp datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "graphiant"}, &metaResp)

		var schemaResp datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
		if schemaResp.Diagnostics.HasError() {
			t.Errorf("%s: schema diagnostics: %v", metaResp.TypeName, schemaResp.Diagnostics)
		}
		if diags := schemaResp.Schema.ValidateImplementation(ctx); diags.HasError() {
			t.Errorf("%s: invalid schema: %v", metaResp.TypeName, diags)
		}
	}
}
