#!/bin/sh

alias ksed='sed -i'
if [ 'Darwin' = "$(uname -s)" ]; then
    alias ksed='sed -i ""'
fi

npm install

echo "$*" | jq -R '. | split(" ") | map("_" + . + ".ts") | {extends: "./tsconfig.json", include: .}' > tmp_tsconfig.json
npx tsc --project tmp_tsconfig.json
rm tmp_tsconfig.json

for var in "$@"
do
    mv _${var}.js ${var}.js.tmpl
    ksed 's://TMPL ::g' ${var}.js.tmpl
    echo "generated ${var}.js.tmpl"
done
