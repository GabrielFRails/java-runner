package br.go.ses.assinador.validation;

import br.go.ses.assinador.model.*;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Nested;
import org.junit.jupiter.api.Test;
import org.springframework.test.util.ReflectionTestUtils;

import java.util.Base64;
import java.util.List;

import static org.assertj.core.api.Assertions.*;

class SignatureValidatorTest {

    private SignatureValidator validator;

    // Bundle FHIR R4 mínimo válido
    private static final String BUNDLE_VALIDO = """
        {
          "resourceType": "Bundle",
          "type": "collection",
          "entry": [{
            "fullUrl": "urn:uuid:550e8400-e29b-41d4-a716-446655440000",
            "resource": {
              "resourceType": "Patient",
              "id": "exemplo"
            }
          }]
        }
        """;

    // Provenance FHIR R4 mínimo válido
    private static final String PROVENANCE_VALIDO = """
        {
          "resourceType": "Provenance",
          "target": [{
            "reference": "urn:uuid:550e8400-e29b-41d4-a716-446655440000"
          }],
          "recorded": "2025-01-01T00:00:00Z",
          "agent": [{"who": {"reference": "Practitioner/1"}}]
        }
        """;

    // Certificado DER mínimo em base64 (bytes 0x30 0x00 — SEQUENCE vazia, apenas para teste)
    private static final String CERT_BASE64_VALIDO =
        Base64.getEncoder().encodeToString(new byte[]{0x30, 0x00, 0x01, 0x02});

    @BeforeEach
    void setUp() {
        validator = new SignatureValidator();
        ReflectionTestUtils.setField(validator, "supportedPolicyVersion", "0.1.2");
    }

    // -------------------------------------------------------------------------
    // Política de assinatura
    // -------------------------------------------------------------------------

    @Nested
    @DisplayName("Validação de policyUri")
    class PolicyTests {

        @Test
        @DisplayName("Deve lançar POLICY.MISSING quando policyUri é nulo")
        void deveRejeitarPolicyNula() {
            var ex = catchThrowableOfType(
                () -> validator.validatePolicy(null),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("POLICY.MISSING");
        }

        @Test
        @DisplayName("Deve lançar POLICY.MISSING quando policyUri está em branco")
        void deveRejeitarPolicyEmBranco() {
            var ex = catchThrowableOfType(
                () -> validator.validatePolicy("   "),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("POLICY.MISSING");
        }

        @Test
        @DisplayName("Deve lançar POLICY.URI-INVALID quando URI não inicia com base correta")
        void deveRejeitarUriInvalida() {
            var ex = catchThrowableOfType(
                () -> validator.validatePolicy("https://outro.dominio/policy|0.1.2"),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("POLICY.URI-INVALID");
        }

        @Test
        @DisplayName("Deve lançar POLICY.URI-INVALID quando versão não segue semver")
        void deveRejeitarVersaoInvalida() {
            var base = "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|";
            var ex = catchThrowableOfType(
                () -> validator.validatePolicy(base + "versao-errada"),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("POLICY.URI-INVALID");
        }

        @Test
        @DisplayName("Deve lançar POLICY.VERSION-UNSUPPORTED para versão não suportada")
        void deveRejeitarVersaoNaoSuportada() {
            var base = "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|";
            var ex = catchThrowableOfType(
                () -> validator.validatePolicy(base + "9.9.9"),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("POLICY.VERSION-UNSUPPORTED");
        }

        @Test
        @DisplayName("Deve aceitar URI de política válida")
        void deveAceitarPolicyValida() {
            var base = "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|";
            assertThatNoException().isThrownBy(() -> validator.validatePolicy(base + "0.1.2"));
        }
    }

    // -------------------------------------------------------------------------
    // Timestamp
    // -------------------------------------------------------------------------

    @Nested
    @DisplayName("Validação de timestamp")
    class TimestampTests {

        @Test
        @DisplayName("Deve lançar CONFIG.INVALID-TIMESTAMP-FORMAT quando timestamp é nulo")
        void deveRejeitarTimestampNulo() {
            var ex = catchThrowableOfType(
                () -> validator.validateTimestamp(null),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("CONFIG.INVALID-TIMESTAMP-FORMAT");
        }

        @Test
        @DisplayName("Deve lançar CONFIG.TIMESTAMP-OUT-OF-RANGE para valor abaixo do mínimo")
        void deveRejeitarTimestampAbaixoDoMinimo() {
            var ex = catchThrowableOfType(
                () -> validator.validateTimestamp(1_000_000L),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("CONFIG.TIMESTAMP-OUT-OF-RANGE");
        }

        @Test
        @DisplayName("Deve lançar TIMESTAMP.OUT-OF-TOLERANCE-WINDOW para timestamp muito distante do atual")
        void deveRejeitarTimestampForaDaJanela() {
            // Timestamp muito no futuro mas dentro do intervalo válido
            long futuro = (System.currentTimeMillis() / 1000) + 3600; // 1 hora no futuro
            var ex = catchThrowableOfType(
                () -> validator.validateTimestamp(futuro),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("TIMESTAMP.OUT-OF-TOLERANCE-WINDOW");
        }

        @Test
        @DisplayName("Deve aceitar timestamp atual dentro da janela de tolerância")
        void deveAceitarTimestampAtual() {
            long agora = System.currentTimeMillis() / 1000;
            assertThatNoException().isThrownBy(() -> validator.validateTimestamp(agora));
        }
    }

    // -------------------------------------------------------------------------
    // Estratégia
    // -------------------------------------------------------------------------

    @Nested
    @DisplayName("Validação de strategy")
    class StrategyTests {

        @Test
        @DisplayName("Deve lançar CONFIG.INVALID-STRATEGY para valor inválido")
        void deveRejeitarStrategyInvalida() {
            var ex = catchThrowableOfType(
                () -> validator.validateStrategy("blockchain", configPadrao()),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("CONFIG.INVALID-STRATEGY");
        }

        @Test
        @DisplayName("Deve lançar CONFIG.TSA-CONFIG-MISSING quando strategy=tsa sem tsaUrl")
        void deveRejeitarTsaSemUrl() {
            var config = configPadrao();
            config.setTsaUrl(null);
            var ex = catchThrowableOfType(
                () -> validator.validateStrategy("tsa", config),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("CONFIG.TSA-CONFIG-MISSING");
        }

        @Test
        @DisplayName("Deve aceitar strategy=iat")
        void deveAceitarIat() {
            assertThatNoException()
                .isThrownBy(() -> validator.validateStrategy("iat", configPadrao()));
        }

        @Test
        @DisplayName("Deve aceitar strategy=tsa com tsaUrl válida")
        void deveAceitarTsaComUrl() {
            var config = configPadrao();
            config.setTsaUrl("https://tsa.exemplo.com");
            assertThatNoException()
                .isThrownBy(() -> validator.validateStrategy("tsa", config));
        }
    }

    // -------------------------------------------------------------------------
    // Cadeia de certificados
    // -------------------------------------------------------------------------

    @Nested
    @DisplayName("Validação de certificateChain")
    class CertChainTests {

        @Test
        @DisplayName("Deve lançar CERT.CHAIN-INCOMPLETE quando chain tem menos de 2 certs")
        void deveRejeitarChainIncompleta() {
            var ex = catchThrowableOfType(
                () -> validator.validateCertificateChain(List.of(CERT_BASE64_VALIDO)),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("CERT.CHAIN-INCOMPLETE");
        }

        @Test
        @DisplayName("Deve lançar CERT.BASE64-INVALID para base64 inválido")
        void deveRejeitarBase64Invalido() {
            var ex = catchThrowableOfType(
                () -> validator.validateCertificateChain(List.of("não-é-base64!!!", CERT_BASE64_VALIDO)),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("CERT.BASE64-INVALID");
        }

        @Test
        @DisplayName("Deve aceitar chain com pelo menos 2 certificados válidos")
        void deveAceitarChainValida() {
            assertThatNoException().isThrownBy(() ->
                validator.validateCertificateChain(List.of(CERT_BASE64_VALIDO, CERT_BASE64_VALIDO))
            );
        }
    }

    // -------------------------------------------------------------------------
    // Configurações operacionais
    // -------------------------------------------------------------------------

    @Nested
    @DisplayName("Validação de OperationalConfig")
    class ConfigTests {

        @Test
        @DisplayName("Deve lançar CONFIG.MISSING-PARAMETER quando config é nulo")
        void deveRejeitarConfigNula() {
            var ex = catchThrowableOfType(
                () -> validator.validateConfig(null),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("CONFIG.MISSING-PARAMETER");
        }

        @Test
        @DisplayName("Deve lançar CONFIG.TTL-OUT-OF-RANGE para ocspCacheTtl inválido")
        void deveRejeitarTtlFoDoRange() {
            var config = configPadrao();
            config.setOcspCacheTtl(10); // abaixo do mínimo 300
            var ex = catchThrowableOfType(
                () -> validator.validateConfig(config),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("CONFIG.TTL-OUT-OF-RANGE");
        }

        @Test
        @DisplayName("Deve lançar CONFIG.TIMEOUT-OUT-OF-RANGE para ocspTimeout inválido")
        void deveRejeitarTimeoutForaDoRange() {
            var config = configPadrao();
            config.setOcspTimeout(200); // acima do máximo 120
            var ex = catchThrowableOfType(
                () -> validator.validateConfig(config),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("CONFIG.TIMEOUT-OUT-OF-RANGE");
        }

        @Test
        @DisplayName("Deve aceitar configuração padrão válida")
        void deveAceitarConfigPadrao() {
            assertThatNoException().isThrownBy(() -> validator.validateConfig(configPadrao()));
        }
    }

    // -------------------------------------------------------------------------
    // Material criptográfico
    // -------------------------------------------------------------------------

    @Nested
    @DisplayName("Validação de CryptoMaterial")
    class CryptoMaterialTests {

        @Test
        @DisplayName("Deve lançar CONFIG.MISSING-PARAMETER quando type é nulo")
        void deveRejeitarTypeNulo() {
            var crypto = new CryptoMaterial();
            var ex = catchThrowableOfType(
                () -> validator.validateCryptoMaterial(crypto),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("CONFIG.MISSING-PARAMETER");
        }

        @Test
        @DisplayName("Deve lançar CONFIG.MISSING-PARAMETER para PEM sem privateKeyPem")
        void deveRejeitarPemSemChave() {
            var crypto = new CryptoMaterial();
            crypto.setType(CryptoMaterial.Type.PEM);
            var ex = catchThrowableOfType(
                () -> validator.validateCryptoMaterial(crypto),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("CONFIG.MISSING-PARAMETER");
        }

        @Test
        @DisplayName("Deve lançar CONFIG.MISSING-PARAMETER para PKCS12 sem password")
        void deveRejeitarPkcs12SemPassword() {
            var crypto = new CryptoMaterial();
            crypto.setType(CryptoMaterial.Type.PKCS12);
            crypto.setPkcs12Base64(Base64.getEncoder().encodeToString(new byte[]{1, 2, 3}));
            crypto.setAlias("minha-chave");
            // password ausente
            var ex = catchThrowableOfType(
                () -> validator.validateCryptoMaterial(crypto),
                ValidationException.class
            );
            assertThat(ex.getFhirCode()).isEqualTo("CONFIG.MISSING-PARAMETER");
        }

        @Test
        @DisplayName("Deve aceitar PEM com privateKeyPem preenchido")
        void deveAceitarPemValido() {
            var crypto = new CryptoMaterial();
            crypto.setType(CryptoMaterial.Type.PEM);
            crypto.setPrivateKeyPem("-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----");
            assertThatNoException().isThrownBy(() -> validator.validateCryptoMaterial(crypto));
        }
    }

    // -------------------------------------------------------------------------
    // Helpers
    // -------------------------------------------------------------------------

    private OperationalConfig configPadrao() {
        var config = new OperationalConfig();
        // Todos os valores dentro dos intervalos válidos
        config.setOcspCacheTtl(3600);
        config.setCrlCacheTtl(3600);
        config.setOcspTimeout(20);
        config.setCrlTimeout(20);
        config.setTsaTimeout(20);
        config.setMaxRetries(3);
        config.setRetryInterval(2);
        config.setMaxEntriesBundle(1000);
        config.setMaxBundleSize(52_428_800);
        config.setTimeoutVerificationBundle(10);
        return config;
    }
}