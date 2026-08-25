#!/usr/bin/env bash
# Regenerates internal/provider/generated/** from api/graphiant_api_docs_v26.7.0.json.
#
# Given the generator toolchain's known rough edges (see api/patch_ir.py and
# api/dedupe_generated.py), review the diff after running this rather than
# assuming a clean regeneration -- particularly for any resource/data source
# whose generator_config.yml entry or augment_spec.py handling changed.
set -euo pipefail
cd "$(dirname "$0")/.."

python3 api/augment_spec.py
go tool tfplugingen-openapi generate \
	--config api/generator_config.yml \
	--output api/provider-code-spec.json \
	api/graphiant_api_docs_v26.7.0.codegen.json
python3 api/patch_ir.py
rm -rf internal/provider/generated
go tool tfplugingen-framework generate all \
	--input api/provider-code-spec.json \
	--output internal/provider/generated
python3 api/dedupe_generated.py
gofmt -l -w internal/provider/generated/
