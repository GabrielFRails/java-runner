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
    // Bundle
    // -------------------------------------------------------------------------

    @Nested
    @DisplayName("Validação de Bundle")
    class BundleTests {

        @Test
        @DisplayName("Deve lançar FORMAT.BUNDLE-MALFORMED quando bundle está ausente")
        void deveRejeitarBundleAusente() {
            var ex = catchThrowableOfType(
                () -> validator.validateBundle(null, configPadrao()),
                ValidationException.class
            );

            assertThat(ex.getFhirCode()).isEqualTo("FORMAT.BUNDLE-MALFORMED");
        }

        @Test
        @DisplayName("Deve lançar FORMAT.BUNDLE-EMPTY quando bundle não possui entries")
        void deveRejeitarBundleSemEntries() {
            String bundleSemEntries = """
                {
                  "resourceType": "Bundle",
                  "type": "collection",
                  "entry": []
                }
                """;

            var ex = catchThrowableOfType(
                () -> validator.validateBundle(bundleSemEntries, configPadrao()),
                ValidationException.class
            );

            assertThat(ex.getFhirCode()).isEqualTo("FORMAT.BUNDLE-EMPTY");
        }

        @Test
        @DisplayName("Deve aceitar bundle válido")
        void deveAceitarBundleValido() {
            assertThatNoException()
                .isThrownBy(() -> validator.validateBundle(BUNDLE_VALIDO, configPadrao()));
        }
    }

    // -------------------------------------------------------------------------
    // Provenance
    // -------------------------------------------------------------------------

    @Nested
    @DisplayName("Validação de Provenance")
    class ProvenanceTests {

        @Test
        @DisplayName("Deve lançar FORMAT.PROVENANCE-INVALID quando provenance está ausente")
        void deveRejeitarProvenanceAusente() {
            var ex = catchThrowableOfType(
                () -> validator.validateProvenance(null, configPadrao()),
                ValidationException.class
            );

            assertThat(ex.getFhirCode()).isEqualTo("FORMAT.PROVENANCE-INVALID");
        }

        @Test
        @DisplayName("Deve lançar FORMAT.PROVENANCE-INVALID quando target está ausente")
        void deveRejeitarProvenanceSemTarget() {
            String provenanceSemTarget = """
                {
                  "resourceType": "Provenance",
                  "target": []
                }
                """;

            var ex = catchThrowableOfType(
                () -> validator.validateProvenance(provenanceSemTarget, configPadrao()),
                ValidationException.class
            );

            assertThat(ex.getFhirCode()).isEqualTo("FORMAT.PROVENANCE-INVALID");
        }

        @Test
        @DisplayName("Deve aceitar provenance válido")
        void deveAceitarProvenanceValido() {
            assertThatNoException()
                .isThrownBy(() -> validator.validateProvenance(PROVENANCE_VALIDO, configPadrao()));
        }
    }

    // -------------------------------------------------------------------------
    // Cruzamento Bundle x Provenance
    // -------------------------------------------------------------------------

    @Nested
    @DisplayName("Validação cruzada de Bundle e Provenance")
    class BundleProvenanceCrossTests {

        @Test
        @DisplayName("Deve lançar FORMAT.TARGET-REFERENCE-MISSING quando target não existe no bundle")
        void deveRejeitarTargetAusenteNoBundle() {
            String provenanceComTargetInexistente = """
                {
                  "resourceType": "Provenance",
                  "target": [{
                    "reference": "urn:uuid:11111111-1111-1111-1111-111111111111"
                  }],
                  "recorded": "2025-01-01T00:00:00Z",
                  "agent": [{"who": {"reference": "Practitioner/1"}}]
                }
                """;

            var ex = catchThrowableOfType(
                () -> validator.validateBundleProvenanceCross(BUNDLE_VALIDO, provenanceComTargetInexistente),
                ValidationException.class
            );

            assertThat(ex.getFhirCode()).isEqualTo("FORMAT.TARGET-REFERENCE-MISSING");
        }

        @Test
        @DisplayName("Deve aceitar bundle e provenance compatíveis")
        void deveAceitarBundleEProvenanceCompativeis() {
            assertThatNoException()
                .isThrownBy(() -> validator.validateBundleProvenanceCross(BUNDLE_VALIDO, PROVENANCE_VALIDO));
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
    // ValidateRequest
    // -------------------------------------------------------------------------

    @Nested
    @DisplayName("Validação de ValidateRequest")
    class ValidateRequestTests {

        @Test
        @DisplayName("Deve lançar FORMAT.JWS-MALFORMED quando signatureData está ausente")
        void deveRejeitarSignatureAusente() {
            var request = validateRequestPadrao();
            request.setSignatureData(null);

            var ex = catchThrowableOfType(
                () -> validator.validateValidateRequest(request),
                ValidationException.class
            );

            assertThat(ex.getFhirCode()).isEqualTo("FORMAT.JWS-MALFORMED");
        }

        @Test
        @DisplayName("Deve lançar FORMAT.BASE64-INVALID quando signatureData não é base64")
        void deveRejeitarSignatureBase64Invalida() {
            var request = validateRequestPadrao();
            request.setSignatureData("não-base64");

            var ex = catchThrowableOfType(
                () -> validator.validateValidateRequest(request),
                ValidationException.class
            );

            assertThat(ex.getFhirCode()).isEqualTo("FORMAT.BASE64-INVALID");
        }

        @Test
        @DisplayName("Deve aceitar ValidateRequest com assinatura base64 válida")
        void deveAceitarValidateRequestValido() {
            assertThatNoException()
                .isThrownBy(() -> validator.validateValidateRequest(validateRequestPadrao()));
        }
    }

    // -------------------------------------------------------------------------
    // SignRequest
    // -------------------------------------------------------------------------

    @Nested
    @DisplayName("Validação de SignRequest")
    class SignRequestTests {

        @Test
        @DisplayName("Deve aceitar SignRequest completo e válido")
        void deveAceitarSignRequestValido() {
            assertThatNoException()
                .isThrownBy(() -> validator.validateSignRequest(signRequestPadrao()));
        }

        @Test
        @DisplayName("Deve rejeitar SignRequest com policy ausente")
        void deveRejeitarSignRequestComPolicyAusente() {
            var request = signRequestPadrao();
            request.setPolicyUri(null);

            var ex = catchThrowableOfType(
                () -> validator.validateSignRequest(request),
                ValidationException.class
            );

            assertThat(ex.getFhirCode()).isEqualTo("POLICY.MISSING");
        }

        @Test
        @DisplayName("Deve rejeitar SignRequest com provenance incompatível com o bundle")
        void deveRejeitarSignRequestComProvenanceIncompativel() {
            var request = signRequestPadrao();
            request.setProvenance("""
                {
                  "resourceType": "Provenance",
                  "target": [{
                    "reference": "urn:uuid:11111111-1111-1111-1111-111111111111"
                  }],
                  "recorded": "2025-01-01T00:00:00Z",
                  "agent": [{"who": {"reference": "Practitioner/1"}}]
                }
                """);

            var ex = catchThrowableOfType(
                () -> validator.validateSignRequest(request),
                ValidationException.class
            );

            assertThat(ex.getFhirCode()).isEqualTo("FORMAT.TARGET-REFERENCE-MISSING");
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

    private ValidateRequest validateRequestPadrao() {
        var request = new ValidateRequest();
        request.setSignatureData(Base64.getEncoder().encodeToString("SIMULATED_SIGNATURE".getBytes()));
        request.setReferenceTimestamp(System.currentTimeMillis() / 1000);
        request.setPolicyUri(
            "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2"
        );
        request.setConfig(configPadrao());
        return request;
    }

    private SignRequest signRequestPadrao() {
        var request = new SignRequest();
        request.setBundle(BUNDLE_VALIDO);
        request.setProvenance(PROVENANCE_VALIDO);
        request.setReferenceTimestamp(System.currentTimeMillis() / 1000);
        request.setStrategy("iat");
        request.setPolicyUri(
            "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2"
        );
        request.setCertificateChain(List.of(CERT_BASE64_VALIDO, CERT_BASE64_VALIDO));

        var crypto = new CryptoMaterial();
        crypto.setType(CryptoMaterial.Type.PEM);
        crypto.setPrivateKeyPem("-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----");
        request.setCryptoMaterial(crypto);
        request.setConfig(configPadrao());

        return request;
    }
}
