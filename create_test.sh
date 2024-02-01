#!/bin/sh

set -e

out_dir=$(mktemp -d)
export KLOTHO_DEBUG_DIR=$out_dir/debug
mkdir -p $KLOTHO_DEBUG_DIR

name=$(basename $1)
name=${name%.*}
name=${name%.input}

echo "Running $name"

# Run the engine
echo "Using $out_dir as output directory"
set +e
go run ./cmd/engine Run \
  -i "$1" \
  -c "$1" \
  -o "$out_dir" > $out_dir/error_details.json 2> $out_dir/err.log

code=$?
echo "Ran $name return code $code, copying results to testdata"
set -e

test_dir="pkg/engine2/testdata"

if [ ! "$test_dir/$name.input.yaml" -ef "$1" ]; then
  cp $1 "$test_dir/$name.input.yaml"
fi

cp "$out_dir/resources.yaml" "$test_dir/$name.expect.yaml"
cp "$out_dir/dataflow-topology.yaml" "$test_dir/$name.dataflow-viz.yaml"
cp "$out_dir/iac-topology.yaml" "$test_dir/$name.iac-viz.yaml"
cp "$out_dir/error_details.json" "$test_dir/$name.err.json"

rm -rf $out_dir
