HOST=localhost:9001

pk="${1:-fixmefixmefixme}"
fn="${2:-test.aif}"
size=$(cat $fn | wc -c | sed 's/ //g')
sig=$(shasum -a 256 $fn | awk '{ print $1 }')

curl "${HOST}/upload/quote?pk=${pk}&size=${size}&sig=${sig}"
