SHELL := /bin/zsh

APP_NAME := assinatura
GO_CMD := go
MVN_CMD := mvn
JAVA_DIR := assinador
JAVA_TARGET_DIR := $(JAVA_DIR)/target
ROOT_JAR := assinador.jar
JAVA_JAR_PATTERN := $(JAVA_TARGET_DIR)/assinador-*.jar

.PHONY: help go-deps go-build go-test java-test java-build java-run sync-jar build test clean version status

help:
	@echo "Alvos disponíveis:"
	@echo "  make go-deps    - instala/atualiza dependências Go do projeto"
	@echo "  make go-build   - compila o CLI Go na raiz do repositório"
	@echo "  make go-test    - executa os testes Go"
	@echo "  make java-test  - executa os testes do assinador Java"
	@echo "  make java-build - gera o jar do assinador"
	@echo "  make java-run   - empacota e sobe o servidor Java"
	@echo "  make sync-jar   - copia o jar gerado para ./assinador.jar"
	@echo "  make build      - build completo: Java + Go + cópia do jar"
	@echo "  make test       - executa testes Java e Go"
	@echo "  make version    - executa ./assinatura version"
	@echo "  make status     - executa ./assinatura status"
	@echo "  make clean      - remove artefatos locais de build"

go-deps:
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

build: java-build go-build sync-jar

test: java-test go-test

version: go-build
	./$(APP_NAME) version

status: build
	./$(APP_NAME) status

clean:
	rm -f $(APP_NAME) $(ROOT_JAR)
	rm -rf $(JAVA_TARGET_DIR)
