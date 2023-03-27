#HOST=https://api.stemstr.app
HOST=localhost:9001

fn="${1:-test.aif}"

pk=$(noscl public | head -n 1)
if [[ -z "$pk" ]] ; then
  echo 'Error: no pubkey set in noscl'
  exit 1
fi

sum=$(shasum -a 256 $fn | awk '{ print $1 }')

curl \
  -F "pk=$pk" \
  -F "sum=$sum" \
  -F "filename=$fn" \
  -F "file=@$fn" \
  "${HOST}/upload"
