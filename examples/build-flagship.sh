#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "$0")/.."

examples=(
	./examples/workbench
	./examples/bridge-coverage
	./examples/scenes
	./examples/table-outline
	./examples/native-table-outline
	./examples/layout
	./examples/accessibility
	./examples/media-transfer
	./examples/charts
)

outdir="${TMPDIR:-/tmp}/swiftui-flagship-build"
mkdir -p "$outdir"

for example in "${examples[@]}"; do
	echo "BUILD: $example"
	name="${example#./examples/}"
	name="${name//\//-}"
	go build -o "$outdir/$name" "$example"
done

echo "OK: flagship examples built"
