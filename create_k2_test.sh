#!/bin/sh

set -e

out_dir=$(mktemp -d)
export KLOTHO_DEBUG_DIR=$out_dir/debug
mkdir -p $KLOTHO_DEBUG_DIR

if [ -d $1 ]; then
  name=$(basename $1)
  infrapy=$1/infra.py
else
  name=${1%.py}
  infrapy=$1
fi

echo "Running $name"

# Run the engine
set +e
echo "Using $out_dir as output directory"
go run ./cmd/klotho up \
  -n=3 \
  --state-directory "$out_dir" \
  "$infrapy" > $out_dir/out.log 2> $out_dir/err.log

code=$?
set -e
if [ $code -ne 0 ]; then
  echo "Engine failed with exit code $code"
  cat $out_dir/err.log
  exit 1
fi

# note: 'go run' always returns exit code 1 if the program returns any non-zero
# so using $? to capture it won't work. We'd need to build and run the binary

echo "Ran $name, copying results to testdata"

test_dir="pkg/k2/testdata"

if [ ! "$test_dir/$name/infra.py" -ef "$infrapy" ]; then
  cp "$infrapy" "$test_dir/$name/infra.py"
fi

for constr_path in $(find $out_dir -maxdepth 4 -mindepth 4 -type d); do
  constr=$(basename $constr_path)

  for f in engine_input.yaml resources.yaml index.ts; do
    if [ -e "$constr_path/$f" ]; then
      cp "$constr_path/$f" "$test_dir/$name/$constr.$f"
    fi
  done
done

rm -rf $out_dir
