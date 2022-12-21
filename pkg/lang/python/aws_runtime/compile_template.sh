#!/bin/sh

alias ksed='sed -i'
if [ 'Darwin' = "$(uname -s)" ]; then
    alias ksed='sed -i ""'
fi

for var in "$@"
do
    cp ${var}.py ${var}.py.tmpl
    ksed 's:#TMPL ::g' ${var}.py.tmpl
    echo "generated ${var}.py.tmpl"
done
