HOST=localhost:9001

fn="${1:-test.aif}"

if ! [ -x "$(command -v noscl)" ]; then
  echo 'Error: noscl is not installed.'
  exit 1
fi

pk=$(noscl public | head -n 1)
if [[ -z "$pk" ]] ; then
  echo 'Error: no pubkey set in noscl'
  exit 1
fi

# Get the file size and shasum
size=$(cat $fn | wc -c | sed 's/ //g')
sum=$(shasum -a 256 $fn | awk '{ print $1 }')

echo "Fetching a quote for uploading $fn"
sleep 1
echo "filesize: $size"
echo "shasum: $sum"
sleep 1

quoteData() {
  cat <<EOF
{
  "pk":"$pk",
  "size":$size,
  "sum":"$sum"
}
EOF
}

event() {
	stream_url=$1
	download_url=$1
  cat <<EOF
{
    "id": "",
    "pubkey": "000005f8bc46b589ace6db0c6f7cf8b1b88dc55595886976e53bbd91423e267e",
    "created_at": $(date +%s),
    "kind": 1,
    "tags": [
      ["t","hiphop"],
      ["t","funky"],
      ["stream_url","$stream_url"],
      ["download_url","$download_url"]
    ],
    "content": "and anotha one",
    "sig": ""
  }
EOF
}

# Get a quote and grab the returned event
quote_id=$(curl -s -XPOST -H "Content-Type: application/json" -d "$(quoteData)" "${HOST}/upload/quote" | jq '.quote_id')

echo "creating event"
sleep 1
echo "signing..."
sleep 1

# Sign the event
event=$(event)
sign=$(noscl sign "$event")
id=$(echo $sign | awk '{ print $2 }')
sig=$(echo $sign | awk '{ print $4 }')
signedEvent=$(echo $event | jq '.sig = "'$sig'" | .id = "'$id'"')
base64SignedEvent=$(echo $signedEvent | jq -c . | base64)

echo "event id: $id"
echo "event sig: $sig"

echo "uploading $fn"
sleep 1

# Upload
curl \
  -F "pk=$pk" \
  -F "size=$size" \
  -F "sum=$sum" \
  -F "quoteId=$quote_id" \
  -F "event=$base64SignedEvent" \
  -F "fileName=$fn" \
  -F "file=@$fn" \
  "${HOST}/upload"
