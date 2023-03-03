HOST=localhost:9001

if ! [ -x "$(command -v noscl)" ]; then
  echo 'Error: noscl is not installed.'
  exit 1
fi

pk=$(noscl public | head -n 1)
if [[ -z "$pk" ]] ; then
  echo 'Error: no pubkey set in noscl'
  exit 1
fi
event() {
  cat <<EOF
{
    "id": "",
    "pubkey": "000005f8bc46b589ace6db0c6f7cf8b1b88dc55595886976e53bbd91423e267e",
    "created_at": 1677376043,
    "kind": 1,
    "tags": [
      ["t","hiphop"],
      ["t","funky"]
    ],
    "content": "random event",
    "sig": ""
  }
EOF
}

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

echo "event id: $id"
echo "event sig: $sig"

echo "proxying event"
sleep 1

echo $signedEvent

# Upload
curl -XPOST \
  -H "Content-Type: application/json" \
  -d "$signedEvent" \
  "${HOST}/event"
