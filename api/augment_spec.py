#!/usr/bin/env python3
"""Produces a codegen-only variant of the Graphiant OpenAPI spec.

tfplugingen-openapi maps each Terraform attribute straight from the OpenAPI
request/response schema shape. Several Graphiant endpoints wrap their real
payload in a single-key envelope object (e.g. POST .../app-lists takes
`{"appListConfig": {...}}`), and a couple of resources (groups, users) have
no single-object GET at all -- only a list endpoint. Both would force
unwanted nesting or an unusable collection-shaped resource schema onto the
generated code. This script produces a sibling spec, used only as codegen
input (never for real API calls), that:

  1. Replaces enveloped request/response schemas with flat equivalents that
     match the resource's real top-level Terraform attributes.
  2. Adds synthetic single-object GET operations for groups/users so they
     have a proper "read" source.
  3. Flattens list-endpoint items that nest their real fields one level
     deeper (app-lists, custom apps) so the plural data sources come out
     flat instead of doubly-nested.

Regenerate the derived spec with: python3 api/augment_spec.py
"""
import copy
import json
from pathlib import Path

SRC = Path(__file__).parent / "graphiant_api_docs_v26.7.0.json"
DST = Path(__file__).parent / "graphiant_api_docs_v26.7.0.codegen.json"


def ref(name):
    return {"$ref": f"#/components/schemas/{name}"}


def json_schema(schema):
    return {"content": {"application/json": {"schema": schema}}}


def set_request_body(op, schema):
    op["requestBody"] = {"required": True, **json_schema(schema)}


def set_response(op, schema, status="200", description="OK"):
    op["responses"][status] = {"description": description, **json_schema(schema)}


def add_path_param_get(paths, path, param_name, response_schema):
    paths[path]["get"] = {
        "description": "Synthetic single-object read, added for OpenAPI codegen only.",
        "parameters": [
            {
                "name": param_name,
                "in": "path",
                "required": True,
                "schema": {"type": "string"},
            }
        ],
        "responses": {"200": {"description": "OK", **json_schema(response_schema)}},
    }


def main():
    spec = json.loads(SRC.read_text())
    schemas = spec["components"]["schemas"]
    paths = spec["paths"]

    EMPTY = {"type": "object", "properties": {}}
    schemas["codegenEmpty"] = EMPTY

    # Site Lists' request body doesn't declare "required" in the source spec
    # even though name/entries are mandatory in practice.
    schemas["v1GlobalSiteListsPostRequest"]["required"] = ["name", "entries"]
    schemas["v1GlobalSiteListsIdPutRequest"]["required"] = ["entries"]

    # -- App Lists: flatten the "appListConfig" envelope --
    schemas["codegenAppListRequest"] = {
        "type": "object",
        "required": ["name", "apps"],
        "properties": {
            "name": {"type": "string"},
            "description": {"type": "string"},
            "apps": {
                "type": "array",
                "items": {
                    "type": "object",
                    "required": ["id", "type"],
                    "properties": {
                        "id": {"type": "integer", "format": "int64"},
                        "type": {"type": "string"},
                    },
                },
            },
        },
    }
    schemas["codegenAppListReadResponse"] = copy.deepcopy(schemas["codegenAppListRequest"])
    schemas["codegenAppListReadResponse"].pop("required")

    p = paths["/v1/global/apps/app-lists"]
    set_request_body(p["post"], ref("codegenAppListRequest"))
    set_response(p["post"], EMPTY)
    p_id = paths["/v1/global/apps/app-lists/{appListId}"]
    set_response(p_id["get"], ref("codegenAppListReadResponse"))
    set_request_body(p_id["put"], ref("codegenAppListRequest"))

    # -- App Lists (plural): flatten entries[].appList.identifier.id one level --
    schemas["codegenAppListSummary"] = {
        "type": "object",
        "properties": {
            "id": {"type": "integer", "format": "int64"},
            "name": {"type": "string"},
            "description": {"type": "string"},
            "appCount": {"type": "integer", "format": "int32"},
            "policyReferenceCount": {"type": "integer", "format": "int32"},
        },
    }
    schemas["codegenAppListsListResponse"] = {
        "type": "object",
        "properties": {
            "appLists": {"type": "array", "items": ref("codegenAppListSummary")}
        },
    }
    set_response(paths["/v1/global/apps/app-lists"]["get"], ref("codegenAppListsListResponse"))

    # -- Content Filters: flatten the "config" envelope --
    schemas["codegenContentFilterRequest"] = {
        "type": "object",
        "required": ["name"],
        "properties": {
            "name": {"type": "string"},
            "lanNames": {"type": "array", "items": {"type": "string"}},
            "rules": {
                "type": "array",
                "items": {
                    "type": "object",
                    "required": ["domainCategoryId"],
                    "properties": {
                        "domainCategoryId": {"type": "integer", "format": "int64"},
                        "exceptionWildcards": {"type": "array", "items": {"type": "string"}},
                    },
                },
            },
            "siteListId": {"type": "integer", "format": "int64"},
            "useAllSites": {"type": "boolean"},
        },
    }
    schemas["codegenContentFilterReadResponse"] = copy.deepcopy(schemas["codegenContentFilterRequest"])
    schemas["codegenContentFilterReadResponse"].pop("required")

    p = paths["/v1/global/content-filters"]
    set_request_body(p["post"], ref("codegenContentFilterRequest"))
    set_response(p["post"], EMPTY)
    p_id = paths["/v1/global/content-filters/{globalContentFilterId}"]
    set_response(p_id["get"], ref("codegenContentFilterReadResponse"))
    set_request_body(p_id["put"], ref("codegenContentFilterRequest"))

    # -- Custom Apps: flatten the "appConfig" envelope --
    schemas["codegenCustomAppRequest"] = {
        "type": "object",
        "required": ["name"],
        "properties": {
            "name": {"type": "string"},
            "description": {"type": "string"},
            "url": {"type": "string"},
            "ipProtocol": {"type": "string"},
            "ipLists": {"type": "array", "items": {"type": "string"}},
            "ipPrefixes": {"type": "array", "items": {"type": "string"}},
            "portRanges": {
                "type": "array",
                "items": {
                    "type": "object",
                    "required": ["lower", "upper"],
                    "properties": {
                        "lower": {"type": "integer", "format": "int32"},
                        "upper": {"type": "integer", "format": "int32"},
                    },
                },
            },
        },
    }
    schemas["codegenCustomAppReadResponse"] = copy.deepcopy(schemas["codegenCustomAppRequest"])
    schemas["codegenCustomAppReadResponse"].pop("required")

    p = paths["/v1/global/apps/custom"]
    set_request_body(p["post"], ref("codegenCustomAppRequest"))
    set_response(p["post"], EMPTY)
    p_id = paths["/v1/global/apps/custom/{appId}"]
    set_response(p_id["get"], ref("codegenCustomAppReadResponse"))
    set_request_body(p_id["put"], ref("codegenCustomAppRequest"))

    # -- Custom Apps (plural): flatten entries[].app + entries[].appConfig --
    schemas["codegenCustomAppSummary"] = copy.deepcopy(schemas["codegenCustomAppRequest"])
    schemas["codegenCustomAppSummary"].pop("required")
    schemas["codegenCustomAppSummary"]["properties"]["id"] = {"type": "integer", "format": "int64"}
    schemas["codegenCustomAppSummary"]["properties"]["appListReferenceCount"] = {"type": "integer", "format": "int32"}
    schemas["codegenCustomAppSummary"]["properties"]["policyReferenceCount"] = {"type": "integer", "format": "int32"}
    schemas["codegenCustomAppsListResponse"] = {
        "type": "object",
        "properties": {
            "customApps": {"type": "array", "items": ref("codegenCustomAppSummary")}
        },
    }
    set_response(paths["/v1/global/apps/custom"]["get"], ref("codegenCustomAppsListResponse"))

    # -- Sites: flatten the "site" envelope (and drop unsupported *Ops fields) --
    schemas["codegenSiteCreateRequest"] = {
        "type": "object",
        "required": ["name", "enterpriseId"],
        "properties": {
            "enterpriseId": {"type": "integer", "format": "int64"},
            "name": {"type": "string"},
            "notes": {"type": "string"},
            "location": ref("manaV2Location"),
        },
    }
    schemas["codegenSiteUpdateRequest"] = {
        "type": "object",
        "required": ["name"],
        "properties": {
            "name": {"type": "string"},
            "notes": {"type": "string"},
            "location": ref("manaV2Location"),
        },
    }
    schemas["codegenSiteResponse"] = {
        "type": "object",
        "properties": {
            "id": {"type": "integer", "format": "int64"},
            "name": {"type": "string"},
            "notes": {"type": "string"},
            "location": ref("manaV2Location"),
            "address": {"type": "string"},
            "edgeCount": {"type": "integer", "format": "int32"},
            "segmentCount": {"type": "integer", "format": "int32"},
            "policyReferenceCount": {"type": "integer", "format": "int32"},
            "siteListReferenceCount": {"type": "integer", "format": "int32"},
            "tags": {"type": "array", "items": {"type": "string"}},
            "createdAt": ref("googleProtobufTimestamp"),
            "updatedAt": ref("googleProtobufTimestamp"),
        },
    }

    p = paths["/v1/sites"]
    set_request_body(p["post"], ref("codegenSiteCreateRequest"))
    set_response(p["post"], ref("codegenSiteResponse"))
    p_id = paths["/v1/sites/{siteId}"]
    set_request_body(p_id["post"], ref("codegenSiteUpdateRequest"))
    set_response(p_id["post"], ref("codegenSiteResponse"))

    # -- Groups / Users: synthesize a single-object GET (list-only in the real API) --
    add_path_param_get(paths, "/v1/groups/{id}", "id", ref("iamGroup"))
    add_path_param_get(paths, "/v1/users/{id}", "id", ref("commonUser"))

    DST.write_text(json.dumps(spec, indent=1))
    print(f"wrote {DST}")


if __name__ == "__main__":
    main()
