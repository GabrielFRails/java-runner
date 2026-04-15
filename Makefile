SHELL := /bin/zsh

APP_NAME := assinatura
GO_CMD := go
MVN_CMD := mvn
JAVA_DIR := assinador
JAVA_TARGET_DIR := $(JAVA_DIR)/target
ROOT_JAR := assinador.jar
JAVA_JAR_PATTERN := $(JAVA_TARGET_DIR)/assinador-*.jar
TMP_BUNDLE := /tmp/bundle.json
TMP_PROVENANCE := /tmp/provenance.json
TMP_CONFIG := /tmp/config.json
TMP_CHAIN := /tmp/chain.json
TMP_KEY := /tmp/key.pem
SIGN_POLICY := https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2
TEST_CERT_BASE64 := MIIDEzCCAfugAwIBAgIUIpafd10rpjmI4ug26Rzbv55qFp4wDQYJKoZIhvcNAQELBQAwGTEXMBUGA1UEAwwOQXNzaW5hZG9yIFRlc3QwHhcNMjYwNDAyMDEzMTQyWhcNMjYwNDAzMDEzMTQyWjAZMRcwFQYDVQQDDA5Bc3NpbmFkb3IgVGVzdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAJpCscpAJ8WS2EmQmLUZeFQzDPUxEgx+rHNRGdNzsSPGSX+U/PBbI1qJIZI0m3J8jCGmCyVoPZpmsMEQBy7v4pA+KqDdQYXsNZJbwCZWNDCycJp5qoOH4TbhrLcjli+eAwd41eLwMgriLkw2DiOykaP1C9L1n4VlAPMNBsUV7I3ZcwIDjvZ78veu/MKobmVzyp/DlRZ5FtXzBADaQl4TiZfsUUBTp+F4//Ew5FNBii5Ti6iA3lktnsH5G1pKDzvQDe6gr2a4zSXUpq7aBMR8fbeHpcJFtV2GeHJrhHVNZzqgW0X1VjuP0oBmr+nJU0GfHqHcSVScs8+ZHzum/CvuGnECAwEAAaNTMFEwHQYDVR0OBBYEFJWuXTtR46jX7IQZLMkFOZSDao78MB8GA1UdIwQYMBaAFJWuXTtR46jX7IQZLMkFOZSDao78MA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBABm134KBnybk7z5wkwZCKW273brHVAq459Npq8x603mbHvssJQNXpUFjU1tEB9vxqvbYroElPoxUPSWFLIOPF9+nx23q+juQiTAYr4IvyEs3UD75WWjRoHZQv5s1GMhvmZ4Ed/l8Nh9tNM6+qQyyfs2nhIKBjEfIJcQIA77hemfslBT/P8UADHL5onZXhOhdCswS9RdamKPZ4zYwTFClCfOO8wiFZ6jTw73dpF1A1J87Kg9gUUP0ilIkhG/867BJxJWHqFT6wAGcMeM7yKQZAvvr4GGqWsMett03zbpkiITIVWDJbt/kpjpoA00t2J+6vR0EZKJnEvTxEZwzfQBM0eA=

.PHONY: help go-deps go-build go-test java-test java-build java-run sync-jar sample-files sample-sign build test clean version status

help:
	@echo "Alvos disponíveis:"
	@echo "  make go-deps    - instala/atualiza dependências Go do projeto"
	@echo "  make go-build   - compila o CLI Go na raiz do repositório"
	@echo "  make go-test    - executa os testes Go"
	@echo "  make java-test  - executa os testes do assinador Java"
	@echo "  make java-build - gera o jar do assinador"
	@echo "  make java-run   - empacota e sobe o servidor Java"
	@echo "  make sync-jar   - copia o jar gerado para ./assinador.jar"
	@echo "  make sample-files - cria os arquivos de teste do exemplo em /tmp"
	@echo "  make sample-sign  - executa o exemplo completo de assinatura pela CLI"
	@echo "  make build      - build completo: Java + Go + cópia do jar"
	@echo "  make test       - executa testes Java e Go"
	@echo "  make version    - executa ./assinatura version"
	@echo "  make status     - executa ./assinatura status"
	@echo "  make clean      - remove artefatos locais de build"

go-deps:
	$(GO_CMD) install github.com/spf13/cobra-cli@latest
	$(GO_CMD) get github.com/spf13/cobra
	$(GO_CMD) get github.com/mattn/go-sqlite3
	$(GO_CMD) mod tidy

go-build:
	$(GO_CMD) build -o $(APP_NAME) ./cmd/assinatura

go-test:
	$(GO_CMD) test ./...

java-test:
	cd $(JAVA_DIR) && $(MVN_CMD) test

java-build:
	cd $(JAVA_DIR) && $(MVN_CMD) clean package

java-run: java-build sync-jar
	java -jar $(ROOT_JAR)

sync-jar: java-build
	@jar_path=$$(ls $(JAVA_JAR_PATTERN) 2>/dev/null | grep -v '\.original$$' | head -n 1); \
	if [[ -z "$$jar_path" ]]; then \
		echo "Nenhum jar encontrado em $(JAVA_TARGET_DIR)." >&2; \
		exit 1; \
	fi; \
	cp "$$jar_path" $(ROOT_JAR); \
	echo "Jar copiado para $(ROOT_JAR)"

sample-files:
	@printf '%s\n' '{"resourceType":"Bundle","type":"collection","entry":[{"fullUrl":"urn:uuid:550e8400-e29b-41d4-a716-446655440000","resource":{"resourceType":"Patient","id":"patient-1"}}]}' > $(TMP_BUNDLE)
	@printf '%s\n' '{"resourceType":"Provenance","target":[{"reference":"urn:uuid:550e8400-e29b-41d4-a716-446655440000"}],"recorded":"2025-01-01T00:00:00Z","agent":[{"who":{"reference":"Practitioner/1"}}]}' > $(TMP_PROVENANCE)
	@printf '%s\n' '{"ocspCacheTtl":3600,"crlCacheTtl":3600,"ocspTimeout":20,"crlTimeout":20,"tsaTimeout":20,"maxRetries":3,"retryInterval":2,"maxEntriesBundle":1000,"maxBundleSize":52428800,"timeoutVerificationBundle":10}' > $(TMP_CONFIG)
	@printf '%s\n' '["$(TEST_CERT_BASE64)","$(TEST_CERT_BASE64)"]' > $(TMP_CHAIN)
	@printf '%s\n' '-----BEGIN PRIVATE KEY-----' 'MIIEvgIBADANBgkqhkiG9w0BAQEFAASCBKgwggSkAgEAAoIBAQC7' '-----END PRIVATE KEY-----' > $(TMP_KEY)
	@echo "Arquivos de teste criados:"
	@echo "  $(TMP_BUNDLE)"
	@echo "  $(TMP_PROVENANCE)"
	@echo "  $(TMP_CONFIG)"
	@echo "  $(TMP_CHAIN)"
	@echo "  $(TMP_KEY)"

sample-sign: build sample-files
	./$(APP_NAME) sign \
		--bundle $(TMP_BUNDLE) \
		--provenance $(TMP_PROVENANCE) \
		--timestamp $$(date +%s) \
		--strategy iat \
		--policy "$(SIGN_POLICY)" \
		--cert $(TMP_CHAIN) \
		--crypto-type PEM \
		--crypto-pem $(TMP_KEY) \
		--config $(TMP_CONFIG)

build: java-build go-build sync-jar

test: java-test go-test

version: go-build
	./$(APP_NAME) version

status: build
	./$(APP_NAME) status

clean:
	rm -f $(APP_NAME) $(ROOT_JAR)
	rm -rf $(JAVA_TARGET_DIR)
