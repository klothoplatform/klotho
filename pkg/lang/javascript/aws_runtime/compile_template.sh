#!/bin/sh

alias ksed='sed -i'
if [ 'Darwin' = "$(uname -s)" ]; then
    alias ksed='sed -i ""'
fi

npm install
for var in "$@"
do
    echo '{"extends": "./tsconfig.json", "include": ["'_${var}.ts'"]}' > tmp_tsconfig.json
    tsc --project tmp_tsconfig.json
    rm tmp_tsconfig.json

    mv _${var}.js ${var}.js.tmpl
    ksed 's://TMPL ::g' ${var}.js.tmpl 
    echo "generated ${var}.js.tmpl"
done
