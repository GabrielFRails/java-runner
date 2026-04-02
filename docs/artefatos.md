# Verificação de Artefatos

Todos os binários distribuídos nas releases do projeto são assinados com
[Cosign](https://docs.sigstore.dev/cosign/overview/), parte do ecossistema
[Sigstore](https://www.sigstore.dev/). Isso permite verificar a autenticidade
e integridade de qualquer artefato antes de executá-lo.

## Pré-requisitos

Instale o Cosign seguindo as instruções oficiais:
https://docs.sigstore.dev/cosign/system_config/installation/

## Arquivos disponíveis em cada release

Para cada binário distribuído, a release contém três arquivos:

```
assinatura-<versão>-<os>-<arch>       # binário
assinatura-<versão>-<os>-<arch>.sig   # assinatura
assinatura-<versão>-<os>-<arch>.pem   # certificado
```

Além de um arquivo consolidado de checksums:

```
checksums.txt
```

## Como verificar um artefato

### 1. Verificar a assinatura com Cosign

Substitua `<artefato>` pelo nome do binário que deseja verificar:

```bash
cosign verify-blob \
  --certificate <artefato>.pem \
  --signature <artefato>.sig \
  --certificate-identity-regexp="https://github.com/kyriosdata/assinatura" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  <artefato>
```

Exemplo para Linux:

```bash
cosign verify-blob \
  --certificate assinatura-v1.0.0-linux-amd64.pem \
  --signature assinatura-v1.0.0-linux-amd64.sig \
  --certificate-identity-regexp="https://github.com/GabrielFRails/java-runner" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  assinatura-v1.0.0-linux-amd64
```

Exemplo macos:

```bash
cosign verify-blob \
  --certificate assinatura-v0.1.3-darwin-arm64.pem \
  --signature assinatura-v0.1.3-darwin-arm64.sig \
  --certificate-identity-regexp="https://github.com/GabrielFRails/java-runner" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  assinatura-v0.1.3-darwin-arm64
```

Se a verificação for bem-sucedida, o Cosign exibirá:

```
Verified OK
```

### 2. Verificar o checksum SHA256

```bash
# Linux / macOS
sha256sum --check checksums.txt

# Windows (PowerShell)
Get-FileHash assinatura-v1.0.0-windows-amd64.exe -Algorithm SHA256
```

## Como funciona a assinatura

A assinatura é gerada automaticamente pelo pipeline de CI/CD do GitHub Actions
usando identidade OIDC — sem chave privada armazenada. Cada assinatura é
registrada no transparency log público do Sigstore (Rekor), o que garante
rastreabilidade e permite auditoria independente da origem do artefato.