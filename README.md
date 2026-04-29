# java-runner

Trabalho prático da disciplina de Implementação e Integração (BES/UFG 2026).

Desenvolvido para a plataforma **HubSaúde** (SES-GO / UFG), o Sistema Runner
facilita o acesso a operações de assinatura digital via linha de comandos, sem
que o usuário precise conhecer detalhes de configuração do ambiente Java.

Envolve particularidades como a verificação da versão java automaticamente e,
caso não esteja instalado, instalar para o usuário.

---

## bibliotecas go para instalar:

```bash
go mod init github.com/kyriosdata/assinatura

# Ferramenta de scaffold do Cobra
go install github.com/spf13/cobra-cli@latest

# CLI framework (equivalente ao cobra/click de outras linguagens)
go get github.com/spf13/cobra

# SQLite para o banco local
go get github.com/mattn/go-sqlite3

go mod tidy
```

## para rodar o servidor java:

```bash
mvn clean package -q && java -jar target/assinador-0.1.0.jar
```

## Estrutura do repositório

```
java-runner/
├── assinador/          # Aplicação Java (assinador.jar) — Spring Boot
├── cmd/assinatura/     # CLI Go — ponto de entrada do usuário
├── internal/
│   ├── environment/    # Detecção do Java no sistema
│   ├── jar/            # Localização e invocação do assinador.jar
│   ├── runner/         # Sequência de inicialização (startup)
│   └── storage/        # Banco de dados local (~/.hubsaude)
└── docs/
	└── verificacao-artefatos.md  # Como verificar binários com Cosign
```

---

## Pré-requisitos (para trabalhar no repo)

| Ferramenta | Versão mínima | Como verificar |
|---|---|---|
| Go | 1.21 | `go version` |
| Java (JDK) | 17 | `java -version` |
| Maven | 3.8 | `mvn -version` |

---

## Compilando o assinador.jar

O `assinador.jar` é a aplicação Java que realiza (simula) as operações de
assinatura digital. Ele precisa ser compilado antes de usar o CLI.

```bash
cd assinador
mvn clean package
```

O jar compilado fica em `assinador/target/assinador-0.1.0.jar`.

Para rodar os testes Java:

```bash
cd assinador
mvn test
```

---

## Compilando o CLI Go

O CLI `assinatura` é o ponto de entrada do usuário. Ele localiza o Java,
localiza o `assinador.jar` e delega as operações para ele.

```bash
# na raiz do repositório
go mod tidy
go build -o assinatura ./cmd/assinatura
```

O binário `assinatura` será gerado na raiz do repositório.

O projeto usa o módulo Go `github.com/kyriosdata/assinatura` e o comando
`assinatura version` é exposto pela estrutura baseada em Cobra em
`cmd/assinatura/main.go`.

---

## Configuração inicial

O `assinador.jar` deve estar no mesmo diretório que o executável `assinatura`.
Após compilar os dois, copie o jar:

```bash
cp assinador/target/assinador-0.1.0.jar ./assinador.jar
```

Na primeira execução de qualquer comando, o CLI cria automaticamente o
diretório `~/.hubsaude` com o banco de dados local de estado.

---

## Comandos disponíveis

### `assinatura version`

Exibe a versão atual do CLI.

```bash
./assinatura version
# assinatura dev
```

### `assinatura status`

Exibe informações sobre o ambiente: sistema operacional, arquitetura,
Java detectado e localização do `assinador.jar`.

```bash
./assinatura status
# Diretório de trabalho : /Users/usuario/.hubsaude
# Sistema operacional   : macOS
# Arquitetura           : arm64
# Java                  : 23.0.1
# Java path             : /usr/bin/java
# Java compatível       : sim (mínimo: 21)
# assinador.jar         : /Users/usuario/code/java-runner/assinador.jar
```

### `assinatura sign`

Cria uma assinatura digital simulada invocando o `assinador.jar`.

```bash
./assinatura sign \
  --bundle <arquivo>       \  # Bundle FHIR R4 em JSON
  --provenance <arquivo>   \  # Provenance FHIR R4 em JSON
  --timestamp <unix-utc>   \  # Padrão: instante atual
  --strategy <iat|tsa>     \  # Padrão: iat
  --policy <uri>           \  # URI da política de assinatura
  --cert <arquivo>         \  # Array JSON de certificados base64 DER
  --crypto-type <tipo>     \  # PEM, PKCS12, SMARTCARD ou TOKEN
  --crypto-pem <arquivo>   \  # Chave privada PEM (para --crypto-type PEM)
  --config <arquivo>           # Configurações operacionais em JSON
```

Exemplo completo com arquivos de teste:

```bash
# 1. Criar arquivos de entrada
cat > /tmp/bundle.json << 'EOF'
{"resourceType":"Bundle","type":"collection","entry":[{"fullUrl":"urn:uuid:550e8400-e29b-41d4-a716-446655440000","resource":{"resourceType":"Patient"}}]}
EOF

cat > /tmp/provenance.json << 'EOF'
{"resourceType":"Provenance","target":[{"reference":"urn:uuid:550e8400-e29b-41d4-a716-446655440000"}],"recorded":"2025-01-01T00:00:00Z","agent":[{"who":{"reference":"Practitioner/1"}}]}
EOF

cat > /tmp/config.json << 'EOF'
{"ocspCacheTtl":3600,"crlCacheTtl":3600,"ocspTimeout":20,"crlTimeout":20,"tsaTimeout":20,"maxRetries":3,"retryInterval":2,"maxEntriesBundle":1000,"maxBundleSize":52428800,"timeoutVerificationBundle":10}
EOF

cat > /tmp/chain.json << 'EOF'
["MAMA","MAMA"]
EOF

cat > /tmp/key.pem << 'EOF'
-----BEGIN PRIVATE KEY-----
MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC7
-----END PRIVATE KEY-----
EOF

# 2. Invocar o sign
./assinatura sign \
  --bundle /tmp/bundle.json \
  --provenance /tmp/provenance.json \
  --timestamp $(date +%s) \
  --strategy iat \
  --policy "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2" \
  --cert /tmp/chain.json \
  --crypto-type PEM \
  --crypto-pem /tmp/key.pem \
  --config /tmp/config.json
```

### `assinatura validate`

Valida uma assinatura digital simulada invocando o `assinador.jar`.

```bash
./assinatura validate \
  --signature <arquivo>  \  # Signature.data em base64
  --timestamp <unix-utc> \  # Padrão: instante atual
  --policy <uri>         \  # URI da política de assinatura
  --config <arquivo>         # Configurações operacionais em JSON
```

---

## Modo servidor HTTP (assinador.jar)

O `assinador.jar` também pode ser executado como servidor HTTP,
expondo endpoints REST para integração com outros sistemas.

```bash
# Subir o servidor na porta 8080
java -jar assinador.jar

# Verificar se está no ar
curl http://localhost:8080/health

# Criar assinatura via HTTP
curl -s -X POST http://localhost:8080/sign \
  -H "Content-Type: application/json" \
  -d '{ ... }' | python3 -m json.tool

# Validar assinatura via HTTP
curl -s -X POST http://localhost:8080/validate \
  -H "Content-Type: application/json" \
  -d '{ ... }' | python3 -m json.tool
```

Endpoints disponíveis:

| Endpoint | Método | Descrição |
|---|---|---|
| `/health` | GET | Status do servidor |
| `/sign` | POST | Cria assinatura simulada |
| `/validate` | POST | Valida assinatura simulada |

---

## Erros e códigos FHIR

Quando os parâmetros são inválidos, o sistema retorna um `OperationOutcome`
FHIR com um código padronizado. Exemplos:

| Código | Causa |
|---|---|
| `POLICY.MISSING` | `policyUri` ausente |
| `POLICY.URI-INVALID` | Formato da URI inválido |
| `POLICY.VERSION-UNSUPPORTED` | Versão da política não suportada |
| `CONFIG.INVALID-TIMESTAMP-FORMAT` | Timestamp ausente ou inválido |
| `CONFIG.TIMESTAMP-OUT-OF-RANGE` | Timestamp fora do intervalo permitido |
| `TIMESTAMP.OUT-OF-TOLERANCE-WINDOW` | Timestamp muito distante do horário atual |
| `CONFIG.INVALID-STRATEGY` | Strategy diferente de `iat` ou `tsa` |
| `FORMAT.BUNDLE-MALFORMED` | Bundle não é FHIR R4 válido |
| `FORMAT.BUNDLE-EMPTY` | Bundle sem entradas |
| `FORMAT.PROVENANCE-INVALID` | Provenance não é FHIR R4 válido |
| `FORMAT.TARGET-REFERENCE-MISSING` | Target do Provenance não existe no Bundle |
| `CERT.CHAIN-INCOMPLETE` | Cadeia de certificados com menos de 2 elementos |
| `CERT.BASE64-INVALID` | Certificado não é base64 válido |
| `CONFIG.MISSING-PARAMETER` | Parâmetro obrigatório ausente |

---

## Integridade dos binários

Os binários distribuídos via GitHub Releases são assinados com Cosign.
Consulte [`docs/verificacao-artefatos.md`](docs/verificacao-artefatos.md)
para instruções de verificação.

---

## Referências

- [Especificação FHIR — Criar Assinatura](https://fhir.saude.go.gov.br/r4/seguranca/caso-de-uso-criar-assinatura.html)
- [Especificação FHIR — Validar Assinatura](https://fhir.saude.go.gov.br/r4/seguranca/caso-de-uso-validar-assinatura.html)
- [Sigstore / Cosign](https://docs.sigstore.dev/cosign/overview/)
- [HAPI FHIR](https://hapifhir.io/)

## Status atual do projeto:

### Sprint 1

| requirement/task/US | status |
|---|---|
| Sprint 1 overall | DONE |
| US-01.1 - Estrutura base do CLI em Go | DONE |
| `go mod init github.com/kyriosdata/assinatura` | DONE |
| Evidência de instalação/uso de `go install github.com/spf13/cobra-cli@latest` | DONE |
| Estrutura de pacotes definida e documentada | DONE |
| Aplicação compila e executa nas três plataformas (Windows, Linux, macOS) | DONE |
| `assinatura version` exibe a versão atual do CLI | DONE |
| US-05.1 - Pipeline CI/CD multiplataforma | DONE |
| GitHub Actions configurado com workflow de build | DONE |
| Cross-compilation para `windows/amd64`, `linux/amd64` e `darwin/amd64` | DONE |
| Build executado a cada push na branch principal | DONE |
| Artefatos de build disponíveis como artifacts do workflow | DONE |
| US-05.2 - Publicação de releases com versionamento semântico | DONE |
| Tags de versão seguem SemVer | DONE |
| Workflow de release gera binários nomeados por plataforma | DONE |
| Binários publicados automaticamente no GitHub Releases ao criar tag | DONE |
| Nome dos artefatos segue `assinatura-<versão>-<os>-<arch>` | DONE |
| US-05.3 - Checksums SHA256 e assinatura de artefatos com Cosign | DONE |
| Cada release inclui arquivo de checksums SHA256 para todos os binários | DONE |
| Artefatos assinados com Cosign (identidade OIDC + transparency log) | DONE |
| Cada artefato acompanhado de `.sig` e `.pem` | DONE |
| Processo de assinatura automatizado no pipeline CI/CD | DONE |
| Documentação de como verificar artefatos com `cosign verify-blob` | DONE |

### Sprint 2

sobre o diretorio, decidi deixar aonde está atualmente mesmo visto que estou avançado no projeto

| requirement/task/US | status |
|---|---|
| Sprint 2 overall | TODO |
| US-02.1 - Simulação de criação de assinatura digital | DONE |
| Projeto Java base inicializado no diretório `projetos/assinador-java` | DONE |
| Interface `SignatureService` definida com métodos `sign` e `validate` | DONE |
| Implementação `FakeSignatureService` retorna assinatura pré-construída para parâmetros válidos | DONE |
| Resposta simulada inclui os campos esperados conforme especificação | DONE |
| Testes unitários cobrem o cenário de sucesso | DONE |
| US-02.2 - Validação de parâmetros de criação de assinatura | DONE |
| Todos os parâmetros obrigatórios são verificados (presença e formato) | DONE |
| Mensagens de erro indicam qual parâmetro está inválido e o motivo | DONE |
| Parâmetros inválidos são rejeitados antes de qualquer processamento | DONE |
| Testes unitários cobrem todos os cenários de validação | DONE (porém vamos ver ao longo do projeto) |
| US-02.3 - Simulação e validação de parâmetros de validação de assinatura | DONE |
| Parâmetros de validação são verificados (presença e formato) | DONE |
| Resultado pré-determinado (válido/inválido) retornado baseado em critérios simples | DONE |
| Mensagens de erro claras para parâmetros inválidos | DONE |
| Testes unitários cobrem cenários de sucesso e falha | DONE |
| US-01.2 - Parsing de comandos e parâmetros no CLI | DONE |
| CLI aceita o comando `sign` com os parâmetros necessários | DONE |
| CLI aceita o comando `validate` com os parâmetros necessários | DONE |
| Mensagem de ajuda (`--help`) documenta os comandos e parâmetros disponíveis | DONE |
| Parâmetros ausentes ou inválidos geram mensagem de erro orientativa | DONE |
| Testes cobrem o parsing de comandos e parâmetros | DONE |
| US-01.3 - Invocação do assinador.jar no modo local | TODO |
| CLI localiza o `java` disponível (provisionado ou do sistema) | DONE |
| CLI constrói e executa `java -jar assinador.jar` com parâmetros corretamente mapeados | DONE |
| Saída do assinador.jar é capturada e repassada ao usuário | DONE |
| Erros de execução são tratados com mensagens claras | DONE |
| Testes de integração validam o fluxo CLI -> assinador.jar | TODO |
| US-01.4 - Exibição legível de resultados | DONE |
| Resultado de criação de assinatura é formatado de forma legível | DONE |
| Resultado de validação de assinatura indica claramente se é válida ou inválida | DONE |
| Erros são apresentados com mensagem descritiva e orientação para correção | DONE |
| Saída é adequada para uso em terminal (não requer pós-processamento) | DONE |
| US-04.1 - Detecção e provisionamento automático do JDK | TODO |
| Sistema verifica se JDK 21 está disponível no `PATH` ou em diretório gerenciado (`~/.hubsaude/`) | TODO |
| Se ausente, JDK é baixado automaticamente da distribuição adequada para a plataforma | TODO |
| JDK baixado é armazenado em `~/.hubsaude/jdk/` para reuso | TODO |
| Download não é repetido se JDK já estiver provisionado | TODO |
| Testes cobrem detecção de JDK presente e ausente nas três plataformas | TODO |
