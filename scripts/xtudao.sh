#!/bin/zsh

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT_DIR"

API_URL="http://127.0.0.1:8080"
POLICY_URI="https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2"

CLI_SIGN_RESPONSE="/tmp/xtudao-cli-sign.json"
CLI_VALIDATE_RESPONSE="/tmp/xtudao-cli-validate.json"
CLI_SIGNATURE_DATA="/tmp/xtudao-cli-signature.txt"
CLI_SIGN_RAW="/tmp/xtudao-cli-sign.raw"
CLI_VALIDATE_RAW="/tmp/xtudao-cli-validate.raw"

API_SIGN_REQUEST="/tmp/xtudao-api-sign-request.json"
API_SIGN_RESPONSE="/tmp/xtudao-api-sign-response.json"
API_SIGNATURE_DATA="/tmp/xtudao-api-signature.txt"
API_VALIDATE_REQUEST="/tmp/xtudao-api-validate-request.json"
API_VALIDATE_RESPONSE="/tmp/xtudao-api-validate-response.json"
API_LOG="/tmp/xtudao-api.log"

cleanup() {
  if [[ -n "${API_PID:-}" ]]; then
    kill "$API_PID" 2>/dev/null || true
    wait "$API_PID" 2>/dev/null || true
  fi
}

extract_json() {
  local src="$1"
  local dst="$2"

  python3 - "$src" "$dst" <<'PY'
import pathlib
import sys

src = pathlib.Path(sys.argv[1]).read_text(encoding="utf-8")
start = src.find("{")
end = src.rfind("}")

if start == -1 or end == -1 or end < start:
    raise SystemExit(f"nao foi possivel localizar JSON em {sys.argv[1]}")

json_payload = src[start:end + 1]
pathlib.Path(sys.argv[2]).write_text(json_payload, encoding="utf-8")
PY
}

echo "==> Subindo API em ${API_URL}"
java -jar "$ROOT_DIR/assinador.jar" >"$API_LOG" 2>&1 &
API_PID=$!
trap cleanup EXIT INT TERM

for _ in {1..30}; do
  if curl -fsS "${API_URL}/health" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done

curl -fsS "${API_URL}/health"
echo

echo "==> Executando sign via CLI"
./assinatura sign \
  --bundle /tmp/bundle.json \
  --provenance /tmp/provenance.json \
  --timestamp "$(date +%s)" \
  --strategy iat \
  --policy "$POLICY_URI" \
  --cert /tmp/chain.json \
  --crypto-type PEM \
  --crypto-pem /tmp/key.pem \
  --config /tmp/config.json >"$CLI_SIGN_RAW" 2>&1

extract_json "$CLI_SIGN_RAW" "$CLI_SIGN_RESPONSE"

cat "$CLI_SIGN_RESPONSE"
echo

python3 - "$CLI_SIGN_RESPONSE" "$CLI_SIGNATURE_DATA" <<'PY'
import json
import sys

src, dst = sys.argv[1], sys.argv[2]
with open(src, "r", encoding="utf-8") as fh:
    data = json.load(fh)
with open(dst, "w", encoding="utf-8") as fh:
    fh.write(data["data"])
PY

echo "==> Executando validate via CLI"
./assinatura validate \
  --signature "$CLI_SIGNATURE_DATA" \
  --timestamp "$(date +%s)" \
  --policy "$POLICY_URI" \
  --config /tmp/config.json >"$CLI_VALIDATE_RAW" 2>&1

extract_json "$CLI_VALIDATE_RAW" "$CLI_VALIDATE_RESPONSE"

cat "$CLI_VALIDATE_RESPONSE"
echo

echo "==> Montando payload para /sign"
python3 - "$API_SIGN_REQUEST" <<'PY'
import json
import time

with open("/tmp/bundle.json", "r", encoding="utf-8") as fh:
    bundle = fh.read()
with open("/tmp/provenance.json", "r", encoding="utf-8") as fh:
    provenance = fh.read()
with open("/tmp/chain.json", "r", encoding="utf-8") as fh:
    chain = json.load(fh)
with open("/tmp/key.pem", "r", encoding="utf-8") as fh:
    key = fh.read()
with open("/tmp/config.json", "r", encoding="utf-8") as fh:
    config = json.load(fh)

payload = {
    "bundle": bundle,
    "provenance": provenance,
    "referenceTimestamp": int(time.time()),
    "strategy": "iat",
    "policyUri": "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2",
    "certificateChain": chain,
    "cryptoMaterial": {
        "type": "PEM",
        "privateKeyPem": key,
    },
    "config": config,
}

with open(__import__("sys").argv[1], "w", encoding="utf-8") as fh:
    json.dump(payload, fh)
PY

echo "==> Executando /sign via HTTP"
curl -fsS -X POST "${API_URL}/sign" \
  -H "Content-Type: application/json" \
  --data @"$API_SIGN_REQUEST" >"$API_SIGN_RESPONSE"

cat "$API_SIGN_RESPONSE"
echo

python3 - "$API_SIGN_RESPONSE" "$API_SIGNATURE_DATA" <<'PY'
import json
import sys

src, dst = sys.argv[1], sys.argv[2]
with open(src, "r", encoding="utf-8") as fh:
    data = json.load(fh)
with open(dst, "w", encoding="utf-8") as fh:
    fh.write(data["data"])
PY

echo "==> Montando payload para /validate"
python3 - "$API_VALIDATE_REQUEST" "$API_SIGNATURE_DATA" <<'PY'
import json
import sys
import time

request_path, signature_path = sys.argv[1], sys.argv[2]

with open(signature_path, "r", encoding="utf-8") as fh:
    signature = fh.read()
with open("/tmp/config.json", "r", encoding="utf-8") as fh:
    config = json.load(fh)

payload = {
    "signatureData": signature,
    "referenceTimestamp": int(time.time()),
    "policyUri": "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2",
    "config": config,
}

with open(request_path, "w", encoding="utf-8") as fh:
    json.dump(payload, fh)
PY

echo "==> Executando /validate via HTTP"
curl -fsS -X POST "${API_URL}/validate" \
  -H "Content-Type: application/json" \
  --data @"$API_VALIDATE_REQUEST" >"$API_VALIDATE_RESPONSE"

cat "$API_VALIDATE_RESPONSE"
echo

echo "==> Fluxo completo finalizado"
echo "CLI sign      : $CLI_SIGN_RESPONSE"
echo "CLI validate  : $CLI_VALIDATE_RESPONSE"
echo "API sign      : $API_SIGN_RESPONSE"
echo "API validate  : $API_VALIDATE_RESPONSE"
echo "API log       : $API_LOG"
