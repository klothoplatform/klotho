#!/bin/sh

# This script takes in an input yaml, and runs the engine on it twice.
# It saves the log output in out.log and out2.log.
# It then filters the logs for important messages and outputs to out.queue.log and out2.queue.log.
# These files can then be diffed to see if the engine is behaving the same way on the same input.

file=$1
if [ ! -f "$file" ]; then
    file="./pkg/engine2/testdata/$1"
fi

rm out.log out2.log
export NO_COLOR=1
export COLUMNS=80
go run ./cmd/engine Run -i "$file" -o "./out/$(basename $file .yaml)" -v 2> out.log
sleep 2
go run ./cmd/engine Run -i "$file" -o "./out/$(basename $file .yaml)2" -v 2> out2.log

rm out.queue.log out2.queue.log
grep -E -e 'op: dequeue|eval|poll-deps' -e 'Satisfied' out.log > out.queue.log
grep -E -e 'op: dequeue|eval|poll-deps' -e 'Satisfied' out2.log > out2.queue.log

