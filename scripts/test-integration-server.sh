#!/bin/zsh

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT_DIR"

POLICY_URI="https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2"
SERVER_PORT="${SERVER_PORT:-18086}"
TEST_HOME="${TEST_HOME:-/tmp/java-runner-server-home}"

CLI_SIGN_RAW="/tmp/test-integration-server-sign.raw"
CLI_SIGN_JSON="/tmp/test-integration-server-sign.json"
CLI_SIGNATURE_DATA="/tmp/test-integration-server-signature.txt"
CLI_VALIDATE_RAW="/tmp/test-integration-server-validate.raw"
CLI_VALIDATE_JSON="/tmp/test-integration-server-validate.json"

cleanup() {
  HOME="$TEST_HOME" ./assinatura stop --port "$SERVER_PORT" >/dev/null 2>&1 || true
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

pathlib.Path(sys.argv[2]).write_text(src[start:end + 1], encoding="utf-8")
PY
}

echo "==> Subindo assinador via CLI na porta ${SERVER_PORT}"
trap cleanup EXIT INT TERM

HOME="$TEST_HOME" ./assinatura start --port "$SERVER_PORT" --timeout 5

echo "==> Conferindo reuso da instância gerenciada"
HOME="$TEST_HOME" ./assinatura start --port "$SERVER_PORT" --timeout 5 | grep -q "reutilizando instância existente"

echo "==> Executando sign via CLI -> HTTP"
HOME="$TEST_HOME" ./assinatura sign \
  --bundle /tmp/bundle.json \
  --provenance /tmp/provenance.json \
  --timestamp "$(date +%s)" \
  --strategy iat \
  --policy "$POLICY_URI" \
  --cert /tmp/chain.json \
  --crypto-type PEM \
  --crypto-pem /tmp/key.pem \
  --config /tmp/config.json >"$CLI_SIGN_RAW" 2>&1

extract_json "$CLI_SIGN_RAW" "$CLI_SIGN_JSON"

python3 - "$CLI_SIGN_JSON" "$CLI_SIGNATURE_DATA" <<'PY'
import json
import pathlib
import sys

src, dst = sys.argv[1], sys.argv[2]
data = json.loads(pathlib.Path(src).read_text(encoding="utf-8"))
if data.get("resourceType") != "Signature":
    raise SystemExit("sign nao retornou resourceType=Signature")
signature_data = data.get("data")
if not signature_data:
    raise SystemExit("sign nao retornou campo data")
pathlib.Path(dst).write_text(signature_data, encoding="utf-8")
PY

echo "==> Executando validate via CLI -> HTTP"
HOME="$TEST_HOME" ./assinatura validate \
  --signature "$CLI_SIGNATURE_DATA" \
  --timestamp "$(date +%s)" \
  --policy "$POLICY_URI" \
  --config /tmp/config.json >"$CLI_VALIDATE_RAW" 2>&1

extract_json "$CLI_VALIDATE_RAW" "$CLI_VALIDATE_JSON"

python3 - "$CLI_VALIDATE_JSON" <<'PY'
import json
import pathlib
import sys

data = json.loads(pathlib.Path(sys.argv[1]).read_text(encoding="utf-8"))
issues = data.get("issue") or []
if not issues:
    raise SystemExit("validate nao retornou issue")
coding = (((issues[0].get("details") or {}).get("coding")) or [{}])[0]
if coding.get("code") != "VALIDATION.SUCCESS":
    raise SystemExit("validate nao retornou VALIDATION.SUCCESS")
PY

echo "==> Encerrando assinador via CLI"
HOME="$TEST_HOME" ./assinatura stop --port "$SERVER_PORT"
trap - EXIT INT TERM

echo "==> Integração servidor validada com sucesso"
echo "sign success     : $CLI_SIGN_JSON"
echo "validate success : $CLI_VALIDATE_JSON"
