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
go run ./cmd/engine Run \
  -i "$1" \
  -c "$1" \
  -o "$out_dir" > $out_dir/out.log 2> $out_dir/err.log

echo "Successfully ran $name, copying results to testdata"

test_dir="pkg/engine2/testdata"

if [ ! "$test_dir/$name.input.yaml" -ef "$1" ]; then
  cp $1 "$test_dir/$name.input.yaml"
fi

cp "$out_dir/resources.yaml" "$test_dir/$name.expect.yaml"
cp "$out_dir/dataflow-topology.yaml" "$test_dir/$name.dataflow-viz.yaml"
cp "$out_dir/iac-topology.yaml" "$test_dir/$name.iac-viz.yaml"

rm -rf $out_dir
