package br.go.ses.assinador.cli;

import br.go.ses.assinador.model.*;
import br.go.ses.assinador.service.FakeSignatureService;
import br.go.ses.assinador.validation.SignatureValidator;
import com.fasterxml.jackson.databind.ObjectMapper;

import java.io.IOException;
import java.nio.file.Files;
import java.nio.file.Path;
import java.util.ArrayList;
import java.util.List;

/**
 * Modo CLI do assinador.
 *
 * <p>Uso:
 * <pre>
 *   java -jar assinador.jar sign \
 *     --bundle bundle.json \
 *     --provenance provenance.json \
 *     --timestamp 1751328000 \
 *     --strategy iat \
 *     --policy "https://...seguranca|0.1.2" \
 *     --cert chain.json \
 *     --crypto-type PEM \
 *     --crypto-pem key.pem \
 *     --config config.json
 *
 *   java -jar assinador.jar validate \
 *     --signature signature.json \
 *     --timestamp 1751328000 \
 *     --policy "https://...seguranca|0.1.2" \
 *     --config config.json
 * </pre>
 */
public class AssinadorCli {

    private static final ObjectMapper mapper = new ObjectMapper();

    public static void run(String[] args) {
        if (args.length == 0) {
            printUsage();
            System.exit(1);
        }

        String command = args[0];
        try {
            switch (command) {
                case "sign"     -> runSign(args);
                case "validate" -> runValidate(args);
                default -> {
                    System.err.println("Comando desconhecido: " + command);
                    printUsage();
                    System.exit(1);
                }
            }
        } catch (ValidationException e) {
            String outcome = buildOperationOutcome(e.getFhirCode(), e.getMessage());
            System.err.println(outcome);
            System.exit(1);
        } catch (Exception e) {
            System.err.println("Erro inesperado: " + e.getMessage());
            System.exit(1);
        }
    }

    private static void runSign(String[] args) throws Exception {
        SignRequest request = new SignRequest();

        for (int i = 1; i < args.length; i++) {
            switch (args[i]) {
                case "--bundle"     -> request.setBundle(readFile(args[++i]));
                case "--provenance" -> request.setProvenance(readFile(args[++i]));
                case "--timestamp" -> request.setReferenceTimestamp(Long.parseLong(args[++i]));
                case "--strategy"   -> request.setStrategy(args[++i]);
                case "--policy"     -> request.setPolicyUri(args[++i]);
                case "--cert"       -> request.setCertificateChain(parseCertChain(args[++i]));
                case "--config"     -> request.setConfig(
                    mapper.readValue(readFile(args[++i]), OperationalConfig.class));
                case "--crypto-type" -> {
                    if (request.getCryptoMaterial() == null)
                        request.setCryptoMaterial(new CryptoMaterial());
                    request.getCryptoMaterial().setType(
                        CryptoMaterial.Type.valueOf(args[++i].toUpperCase()));
                }
                case "--crypto-pem" -> {
                    if (request.getCryptoMaterial() == null)
                        request.setCryptoMaterial(new CryptoMaterial());
                    request.getCryptoMaterial().setPrivateKeyPem(readFile(args[++i]));
                }
                case "--crypto-password" -> {
                    if (request.getCryptoMaterial() == null)
                        request.setCryptoMaterial(new CryptoMaterial());
                    request.getCryptoMaterial().setPassword(args[++i]);
                }
                case "--crypto-pkcs12" -> {
                    if (request.getCryptoMaterial() == null)
                        request.setCryptoMaterial(new CryptoMaterial());
                    request.getCryptoMaterial().setPkcs12Base64(readFile(args[++i]));
                }
                case "--crypto-alias" -> {
                    if (request.getCryptoMaterial() == null)
                        request.setCryptoMaterial(new CryptoMaterial());
                    request.getCryptoMaterial().setAlias(args[++i]);
                }
            }
        }

        // Validação e execução
        SignatureValidator validator = new SignatureValidator();
        validator.validateSignRequest(request);

        FakeSignatureService service = new FakeSignatureService();
        System.out.println(service.sign(request));
    }

    private static void runValidate(String[] args) throws Exception {
        ValidateRequest request = new ValidateRequest();

        for (int i = 1; i < args.length; i++) {
            switch (args[i]) {
                case "--signature"  -> request.setSignatureData(readFile(args[++i]));
                case "--timestamp"  -> request.setReferenceTimestamp(Long.parseLong(args[++i]));
                case "--policy"     -> request.setPolicyUri(args[++i]);
                case "--config"     -> request.setConfig(
                    mapper.readValue(readFile(args[++i]), OperationalConfig.class));
                case "--bundle"     -> request.setBundle(readFile(args[++i]));
                case "--provenance" -> request.setProvenance(readFile(args[++i]));
            }
        }

        SignatureValidator validator = new SignatureValidator();
        validator.validateValidateRequest(request);

        FakeSignatureService service = new FakeSignatureService();
        System.out.println(service.validate(request));
    }

    private static String readFile(String path) throws IOException {
        return Files.readString(Path.of(path));
    }

    private static List<String> parseCertChain(String path) throws IOException {
        // Espera um arquivo JSON contendo um array de strings base64
        String json = readFile(path);
        var type = mapper.getTypeFactory().constructCollectionType(List.class, String.class);
        return mapper.readValue(json, type);
    }

    private static String buildOperationOutcome(String code, String diagnostics) {
        return """
            {
              "resourceType": "OperationOutcome",
              "issue": [{
                "severity": "error",
                "code": "invalid",
                "details": {
                  "coding": [{
                    "system": "https://fhir.saude.go.gov.br/r4/seguranca/CodeSystem/situacao-excepcional-assinatura",
                    "code": "%s"
                  }]
                },
                "diagnostics": "%s"
              }]
            }
            """.formatted(code, diagnostics.replace("\"", "\\\""));
    }

    private static void printUsage() {
        System.err.println("""
            Uso:
              java -jar assinador.jar sign \\
                --bundle <arquivo> \\
                --provenance <arquivo> \\
                --timestamp <unix-utc> \\
                --strategy <iat|tsa> \\
                --policy <uri> \\
                --cert <arquivo-json-array> \\
                --crypto-type <PEM|PKCS12|SMARTCARD|TOKEN> \\
                --crypto-pem <arquivo>  (para type=PEM) \\
                --config <arquivo>

              java -jar assinador.jar validate \\
                --signature <arquivo> \\
                --timestamp <unix-utc> \\
                --policy <uri> \\
                --config <arquivo>
            """);
    }
}