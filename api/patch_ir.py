#!/usr/bin/env python3
"""Hand-patches provider-code-spec.json for things generator_config.yml can't express:
plan modifiers, immutability (RequiresReplace), and a couple of type corrections
where the raw API shape (a {seconds,nanos} timestamp object) doesn't match the
RFC3339-string UX the hand-written provider already exposes.

Regenerate with: python3 api/patch_ir.py (run after tfplugingen-openapi generate,
before tfplugingen-framework generate).
"""
import json
from pathlib import Path

SPEC = Path(__file__).parent / "provider-code-spec.json"

STRINGPLANMOD = "github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
INT64PLANMOD = "github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
BOOLPLANMOD = "github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"


def requires_replace(pkg_path, ctor):
    return {"custom": {"imports": [{"path": pkg_path}], "schema_definition": f"{ctor}()"}}


def use_state_for_unknown(pkg_path, ctor):
    return {"custom": {"imports": [{"path": pkg_path}], "schema_definition": f"{ctor}()"}}


def find_type_block(attr):
    for k, v in attr.items():
        if k != "name" and isinstance(v, dict):
            return k, v
    raise KeyError(f"no type block on attribute {attr!r}")


def attrs_by_name(attributes):
    return {a["name"]: a for a in attributes}


def get_resource(spec, name):
    return next(r for r in spec["resources"] if r["name"] == name)


def get_datasource(spec, name):
    return next(r for r in spec["datasources"] if r["name"] == name)


def nested_attrs(attr):
    _, block = find_type_block(attr)
    return attrs_by_name(block["nested_object"]["attributes"])


def set_computed(attr):
    _, block = find_type_block(attr)
    block["computed_optional_required"] = "computed"


def to_plain_string(attr, computed_optional_required="computed"):
    attr.pop("single_nested", None)
    attr["string"] = {"computed_optional_required": computed_optional_required}


def add_plan_modifier(attr, modifier):
    _, block = find_type_block(attr)
    block.setdefault("plan_modifiers", []).append(modifier)


def main():
    spec = json.loads(SPEC.read_text())

    # -- id attributes merged in purely via a path-param alias default to
    # computed_optional; they're server-assigned only. --
    for res_name in ("app_list", "content_filter", "custom_app"):
        attrs = attrs_by_name(get_resource(spec, res_name)["schema"]["attributes"])
        set_computed(attrs["id"])

    # -- timestamp fields: expose as RFC3339 strings (matching the existing
    # hand-written util.go timestampValue() conversion), not the raw
    # {seconds, nanos} object shape. --
    site_attrs = attrs_by_name(get_resource(spec, "site")["schema"]["attributes"])
    to_plain_string(site_attrs["created_at"])
    add_plan_modifier(site_attrs["created_at"], use_state_for_unknown(STRINGPLANMOD, "stringplanmodifier.UseStateForUnknown"))
    to_plain_string(site_attrs["updated_at"])

    user_attrs = attrs_by_name(get_resource(spec, "user")["schema"]["attributes"])
    to_plain_string(user_attrs["last_active_at"])

    ds_site_attrs = attrs_by_name(get_datasource(spec, "site")["schema"]["attributes"])
    to_plain_string(ds_site_attrs["created_at"])
    to_plain_string(ds_site_attrs["updated_at"])

    ds_user_attrs = attrs_by_name(get_datasource(spec, "user")["schema"]["attributes"])
    to_plain_string(ds_user_attrs["last_active_at"])

    # sites/users (plural data sources): same timestamp-shape fix, one level
    # down inside each list item.
    sites_item = nested_attrs(attrs_by_name(get_datasource(spec, "sites")["schema"]["attributes"])["sites"])
    to_plain_string(sites_item["created_at"])
    to_plain_string(sites_item["updated_at"])

    users_item = nested_attrs(attrs_by_name(get_datasource(spec, "users")["schema"]["attributes"])["users"])
    to_plain_string(users_item["last_active_at"])

    # -- immutable fields: no rename/field-level update support server-side. --
    site_list_attrs = attrs_by_name(get_resource(spec, "site_list")["schema"]["attributes"])
    add_plan_modifier(site_list_attrs["name"], requires_replace(STRINGPLANMOD, "stringplanmodifier.RequiresReplace"))

    add_plan_modifier(user_attrs["email"], requires_replace(STRINGPLANMOD, "stringplanmodifier.RequiresReplace"))

    add_plan_modifier(site_attrs["enterprise_id"], requires_replace(INT64PLANMOD, "int64planmodifier.RequiresReplace"))

    group_attrs = attrs_by_name(get_resource(spec, "group")["schema"]["attributes"])
    add_plan_modifier(group_attrs["manages_enterprises"], requires_replace(BOOLPLANMOD, "boolplanmodifier.RequiresReplace"))
    add_plan_modifier(group_attrs["group_id"], requires_replace(STRINGPLANMOD, "stringplanmodifier.RequiresReplace"))

    # content_filters (plural data source): the raw API shape ("rows" of
    # {global_content_filter_id, global_content_filter_name, lans: [{lan_name}],
    # sites: [{site_name}], ...}) doesn't match the existing, friendlier
    # hand-written data source UX (content_filters of {id, name, lan_names:
    # [string], site_names: [string], ...}). Reshape the IR to match it,
    # rather than force a breaking rename on every user of this data source.
    cf_rows_attr = attrs_by_name(get_datasource(spec, "content_filters")["schema"]["attributes"])["rows"]
    cf_rows_attr["name"] = "content_filters"
    cf_rows = nested_attrs(cf_rows_attr)
    to_plain_string(cf_rows["created_at"])
    to_plain_string(cf_rows["updated_at"])
    cf_rows["global_content_filter_id"]["name"] = "id"
    cf_rows["global_content_filter_name"]["name"] = "name"
    cf_rows["lans"]["name"] = "lan_names"
    cf_rows["lans"]["list"] = {"computed_optional_required": "computed", "element_type": {"string": {}}}
    del cf_rows["lans"]["list_nested"]
    cf_rows["sites"]["name"] = "site_names"
    cf_rows["sites"]["list"] = {"computed_optional_required": "computed", "element_type": {"string": {}}}
    del cf_rows["sites"]["list_nested"]

    # site_list (resource + singular data source): manaV2SiteListEntry's raw
    # field is "regular"; rename to "site_id" to match the existing
    # hand-written model/UX (a plain site reference, as opposed to "tag").
    site_list_entries = nested_attrs(attrs_by_name(get_resource(spec, "site_list")["schema"]["attributes"])["entries"])
    site_list_entries["regular"]["name"] = "site_id"

    ds_site_list_entries = nested_attrs(attrs_by_name(get_datasource(spec, "site_list")["schema"]["attributes"])["entries"])
    ds_site_list_entries["regular"]["name"] = "site_id"

    # site_lists (plural data source): top-level "entries" here is the list
    # of site-list summaries, not one site list's members -- rename to avoid
    # confusion with the singular data source's "entries", and fix up its
    # created_at the same way as elsewhere.
    site_lists_attr = attrs_by_name(get_datasource(spec, "site_lists")["schema"]["attributes"])["entries"]
    site_lists_attr["name"] = "site_lists"
    site_lists_item = nested_attrs(site_lists_attr)
    to_plain_string(site_lists_item["created_at"])

    SPEC.write_text(json.dumps(spec, indent=1))
    print(f"patched {SPEC}")


if __name__ == "__main__":
    main()
