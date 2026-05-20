# PKCS#11 e SoftHSM2

Este projeto suporta um fluxo mínimo de PKCS#11 para `TOKEN` e `SMARTCARD`.
Nesta etapa, o `assinador.jar` valida que consegue carregar o provider
`SunPKCS11`, abrir o token com PIN e encontrar o identificador solicitado. A
assinatura retornada ainda é simulada.

## Parâmetros

Use `--crypto-type TOKEN` ou `--crypto-type SMARTCARD` com:

- `--crypto-pin`: PIN do token
- `--crypto-identifier`: alias/identificador da chave no token
- `--pkcs11-library`: caminho da biblioteca PKCS#11
- `--pkcs11-slot`: slot do token, opcional
- `--token-label`: rótulo do token, opcional

Exemplo:

```bash
./assinatura sign \
  --bundle /tmp/bundle.json \
  --provenance /tmp/provenance.json \
  --timestamp "$(date +%s)" \
  --strategy iat \
  --policy "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2" \
  --cert /tmp/chain.json \
  --crypto-type TOKEN \
  --crypto-pin 1234 \
  --crypto-identifier hubsaude-key \
  --pkcs11-library /usr/local/lib/softhsm/libsofthsm2.so \
  --config /tmp/config.json
```

## SoftHSM2

Instale o SoftHSM2 conforme sua plataforma e inicialize um token:

```bash
softhsm2-util --init-token --free --label hubsaude --pin 1234 --so-pin 123456
softhsm2-util --show-slots
```

Depois importe ou gere uma chave/certificado com o alias que será usado em
`--crypto-identifier`. O caminho exato da biblioteca varia por sistema, por
exemplo:

- macOS Homebrew: `/opt/homebrew/lib/softhsm/libsofthsm2.so`
- Linux: `/usr/lib/softhsm/libsofthsm2.so`

## Erros esperados

- `PKCS11.LIBRARY-MISSING`: parâmetro `pkcs11LibraryPath` ausente
- `PKCS11.LIBRARY-NOT-FOUND`: biblioteca PKCS#11 não encontrada
- `PKCS11.PROVIDER-UNAVAILABLE`: JDK sem provider `SunPKCS11`
- `PKCS11.DEVICE-UNAVAILABLE`: token indisponível, PIN inválido ou falha ao abrir o dispositivo
- `PKCS11.KEY-NOT-FOUND`: identificador não encontrado no token
