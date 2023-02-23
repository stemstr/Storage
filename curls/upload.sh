HOST=localhost:9001

fn="${1:-test.aif}"

curl \
  -F "filename=$fn" \
  -F "file=@$fn" \
  "${HOST}/upload"
