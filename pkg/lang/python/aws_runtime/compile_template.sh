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
    echo ${var}.py.tmpl >> .gitignore
done

sort -u .gitignore > gitignore-tmp
mv gitignore-tmp .gitignore
