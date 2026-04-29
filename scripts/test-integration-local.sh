#!/bin/zsh

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "$0")/.." && pwd)
cd "$ROOT_DIR"

POLICY_URI="https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2"

CLI_SIGN_RAW="/tmp/test-integration-local-sign.raw"
CLI_SIGN_JSON="/tmp/test-integration-local-sign.json"
CLI_SIGNATURE_DATA="/tmp/test-integration-local-signature.txt"
CLI_VALIDATE_RAW="/tmp/test-integration-local-validate.raw"
CLI_VALIDATE_JSON="/tmp/test-integration-local-validate.json"
CLI_JAR_MISSING_RAW="/tmp/test-integration-local-missing-jar.raw"

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

echo "==> Cenário 1: sign via CLI"
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

echo "==> Cenário 2: validate via CLI"
./assinatura validate \
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

echo "==> Cenário 3: erro claro quando assinador.jar está ausente"
mv "$ROOT_DIR/assinador.jar" "$ROOT_DIR/assinador.jar.bak"
trap 'mv "$ROOT_DIR/assinador.jar.bak" "$ROOT_DIR/assinador.jar" 2>/dev/null || true' EXIT INT TERM

if ./assinatura sign \
  --bundle /tmp/bundle.json \
  --provenance /tmp/provenance.json \
  --timestamp "$(date +%s)" \
  --strategy iat \
  --policy "$POLICY_URI" \
  --cert /tmp/chain.json \
  --crypto-type PEM \
  --crypto-pem /tmp/key.pem \
  --config /tmp/config.json >"$CLI_JAR_MISSING_RAW" 2>&1; then
  echo "esperava falha quando assinador.jar estivesse ausente" >&2
  exit 1
fi

if ! grep -Eq "assinador.jar não encontrado|assinador.jar nao encontrado" "$CLI_JAR_MISSING_RAW"; then
  echo "mensagem esperada para jar ausente não foi encontrada" >&2
  cat "$CLI_JAR_MISSING_RAW" >&2
  exit 1
fi

mv "$ROOT_DIR/assinador.jar.bak" "$ROOT_DIR/assinador.jar"
trap - EXIT INT TERM

echo "==> Integração local validada com sucesso"
echo "sign success     : $CLI_SIGN_JSON"
echo "validate success : $CLI_VALIDATE_JSON"
echo "jar ausente      : $CLI_JAR_MISSING_RAW"
