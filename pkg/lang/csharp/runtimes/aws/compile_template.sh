#!/bin/sh

alias ksed='sed -i'
if [ 'Darwin' = "$(uname -s)" ]; then
    alias ksed='sed -i ""'
fi

for var in "$@"
do
    cp ${var}.cs ${var}.cs.tmpl
    ksed 's://TMPL ::g' ${var}.cs.tmpl
    echo "generated ${var}.cs.tmpl"
done
