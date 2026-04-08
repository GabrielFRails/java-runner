# java-runner
java runner

## bibliotecas go para instalar:

```bash
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
