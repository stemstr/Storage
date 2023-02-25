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

# Get a quote and grab the returned event
event=$(curl -s "${HOST}/upload/quote?pk=${pk}&size=${size}&sig=${sum}" | jq '.event')

echo "received event to sign"
sleep 1
echo "signing..."
sleep 1

# Sign the event
sign=$(noscl sign "$event")
id=$(echo $sign | awk '{ print $2 }')
sig=$(echo $sign | awk '{ print $4 }')
signedEvent=$(echo $event | jq '.sig = "'$sig'" | .id = "'$id'"')
base64SignedEvent=$(echo $signedEvent | jq -c . | base64)

echo "uploading $fn"
sleep 1

# Upload
_=$(curl \
  -F "pk=$pk" \
  -F "size=$size" \
  -F "sum=$sum" \
  -F "event=$base64SignedEvent" \
  -F "file=@$fn" \
  "${HOST}/upload")
