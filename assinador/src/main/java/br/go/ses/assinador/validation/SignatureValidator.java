package br.go.ses.assinador.validation;

import br.go.ses.assinador.model.*;
import ca.uhn.fhir.context.FhirContext;
import ca.uhn.fhir.parser.DataFormatException;
import org.hl7.fhir.r4.model.Bundle;
import org.hl7.fhir.r4.model.Provenance;
import org.springframework.beans.factory.annotation.Value;
import org.springframework.stereotype.Component;

import java.util.Base64;
import java.util.HashSet;
import java.util.List;
import java.util.Set;
import java.util.regex.Pattern;

/**
 * Valida os parâmetros de entrada das operações sign e validate.
 * Cada método lança {@link ValidationException} com o código FHIR correspondente.
 * As validações seguem a ordem definida na especificação FHIR da SES-GO.
 */
@Component
public class SignatureValidator {

    private static final long TIMESTAMP_MIN = 1_751_328_000L;
    private static final long TIMESTAMP_MAX = 4_102_444_800L;
    private static final long TOLERANCE_SECONDS = 300L;

    private static final String POLICY_BASE_URI =
        "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|";

    private static final Pattern SEMVER = Pattern.compile("^\\d+\\.\\d+\\.\\d+$");
    private static final Pattern UUID_URN = Pattern.compile(
        "^urn:uuid:[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$"
    );

    @Value("${assinador.policy.supported-version}")
    private String supportedPolicyVersion;

    private final FhirContext fhirContext = FhirContext.forR4();

    // -------------------------------------------------------------------------
    // Ponto de entrada principal
    // -------------------------------------------------------------------------

    /** Valida todos os parâmetros de uma requisição sign. */
    public void validateSignRequest(SignRequest req) {
        validatePolicy(req.getPolicyUri());
        validateTimestamp(req.getReferenceTimestamp());
        validateStrategy(req.getStrategy(), req.getConfig());
        validateConfig(req.getConfig());
        validateBundle(req.getBundle(), req.getConfig());
        validateProvenance(req.getProvenance(), req.getConfig());
        validateBundleProvenanceCross(req.getBundle(), req.getProvenance());
        validateCertificateChain(req.getCertificateChain());
        validateCryptoMaterial(req.getCryptoMaterial());
    }

    /** Valida todos os parâmetros de uma requisição validate. */
    public void validateValidateRequest(ValidateRequest req) {
        validatePolicy(req.getPolicyUri());
        validateTimestamp(req.getReferenceTimestamp());
        validateConfig(req.getConfig());

        if (req.getSignatureData() == null || req.getSignatureData().isBlank()) {
            throw new ValidationException("FORMAT.JWS-MALFORMED",
                "signatureData é obrigatório e não pode ser vazio");
        }
        // Verifica se é base64 válido
        try {
            Base64.getDecoder().decode(req.getSignatureData());
        } catch (IllegalArgumentException e) {
            throw new ValidationException("FORMAT.BASE64-INVALID",
                "signatureData não é uma string base64 válida");
        }
    }

    // -------------------------------------------------------------------------
    // 1. Política de assinatura
    // -------------------------------------------------------------------------

    public void validatePolicy(String policyUri) {
        if (policyUri == null || policyUri.isBlank()) {
            throw new ValidationException("POLICY.MISSING",
                "policyUri é obrigatório");
        }
        if (!policyUri.startsWith(POLICY_BASE_URI)) {
            throw new ValidationException("POLICY.URI-INVALID",
                "policyUri deve iniciar com: " + POLICY_BASE_URI);
        }
        long pipeCount = policyUri.chars().filter(c -> c == '|').count();
        if (pipeCount != 1) {
            throw new ValidationException("POLICY.URI-INVALID",
                "policyUri deve conter exatamente um separador '|'");
        }
        String version = policyUri.substring(POLICY_BASE_URI.length());
        if (!SEMVER.matcher(version).matches()) {
            throw new ValidationException("POLICY.URI-INVALID",
                "Versão da política '" + version + "' não segue o formato major.minor.patch");
        }
        if (!version.equals(supportedPolicyVersion)) {
            throw new ValidationException("POLICY.VERSION-UNSUPPORTED",
                "Versão da política '" + version + "' não suportada. Versão suportada: "
                + supportedPolicyVersion);
        }
    }

    // -------------------------------------------------------------------------
    // 2. Timestamp de referência
    // -------------------------------------------------------------------------

    public void validateTimestamp(Long timestamp) {
        if (timestamp == null) {
            throw new ValidationException("CONFIG.INVALID-TIMESTAMP-FORMAT",
                "referenceTimestamp é obrigatório");
        }
        if (timestamp < TIMESTAMP_MIN || timestamp > TIMESTAMP_MAX) {
            throw new ValidationException("CONFIG.TIMESTAMP-OUT-OF-RANGE",
                "referenceTimestamp deve estar no intervalo [" + TIMESTAMP_MIN + ", "
                + TIMESTAMP_MAX + "]. Valor recebido: " + timestamp);
        }
        long now = System.currentTimeMillis() / 1000;
        if (Math.abs(timestamp - now) > TOLERANCE_SECONDS) {
            throw new ValidationException("TIMESTAMP.OUT-OF-TOLERANCE-WINDOW",
                "referenceTimestamp está fora da janela de tolerância de ±" + TOLERANCE_SECONDS
                + " segundos em relação ao instante atual do servidor");
        }
    }

    // -------------------------------------------------------------------------
    // 3. Estratégia
    // -------------------------------------------------------------------------

    public void validateStrategy(String strategy, OperationalConfig config) {
        if (strategy == null || (!strategy.equals("iat") && !strategy.equals("tsa"))) {
            throw new ValidationException("CONFIG.INVALID-STRATEGY",
                "strategy deve ser 'iat' ou 'tsa'. Valor recebido: " + strategy);
        }
        if ("tsa".equals(strategy)) {
            if (config == null || config.getTsaUrl() == null || config.getTsaUrl().isBlank()) {
                throw new ValidationException("CONFIG.TSA-CONFIG-MISSING",
                    "config.tsaUrl é obrigatório quando strategy='tsa'");
            }
            if (!config.getTsaUrl().startsWith("https://")) {
                throw new ValidationException("CONFIG.TSA-URL-INVALID",
                    "config.tsaUrl deve usar protocolo HTTPS");
            }
        }
    }

    // -------------------------------------------------------------------------
    // 4. Bundle FHIR
    // -------------------------------------------------------------------------

    public void validateBundle(String bundleJson, OperationalConfig config) {
        if (bundleJson == null || bundleJson.isBlank()) {
            throw new ValidationException("FORMAT.BUNDLE-MALFORMED",
                "bundle é obrigatório");
        }

        // Limites de segurança — verificados antes do parsing
        int cfg = config != null && config.getMaxBundleSize() != null
            ? config.getMaxBundleSize() : 52_428_800;
        if (bundleJson.getBytes().length > cfg) {
            throw new ValidationException("SECURITY.BUNDLE-MEMORY-LIMIT-EXCEEDED",
                "Bundle excede o tamanho máximo permitido de " + cfg + " bytes");
        }

        Bundle bundle;
        try {
            bundle = fhirContext.newJsonParser().parseResource(Bundle.class, bundleJson);
        } catch (DataFormatException e) {
            throw new ValidationException("FORMAT.BUNDLE-MALFORMED",
                "Bundle não é um recurso FHIR R4 válido: " + e.getMessage());
        }

        if (bundle.getEntry() == null || bundle.getEntry().isEmpty()) {
            throw new ValidationException("FORMAT.BUNDLE-EMPTY",
                "Bundle.entry deve conter pelo menos uma entrada");
        }

        int maxEntries = config != null && config.getMaxEntriesBundle() != null
            ? config.getMaxEntriesBundle() : 1000;
        if (bundle.getEntry().size() > maxEntries) {
            throw new ValidationException("SECURITY.BUNDLE-SIZE-LIMIT-EXCEEDED",
                "Bundle.entry excede o máximo de " + maxEntries + " entradas");
        }

        Set<String> fullUrls = new HashSet<>();
        for (var entry : bundle.getEntry()) {
            String fullUrl = entry.getFullUrl();
            if (fullUrl == null || fullUrl.isBlank()) {
                throw new ValidationException("FORMAT.BUNDLE-MALFORMED",
                    "Todas as entradas do Bundle devem ter fullUrl preenchido");
            }
            if (!UUID_URN.matcher(fullUrl).matches()) {
                throw new ValidationException("FORMAT.BUNDLE-MALFORMED",
                    "Bundle.entry.fullUrl deve seguir o formato urn:uuid:<UUID RFC 4122>. " +
                    "Valor inválido: " + fullUrl);
            }
            if (!fullUrls.add(fullUrl)) {
                throw new ValidationException("FORMAT.BUNDLE-MALFORMED",
                    "Bundle.entry contém fullUrl duplicado: " + fullUrl);
            }
        }
    }

    // -------------------------------------------------------------------------
    // 5. Provenance FHIR
    // -------------------------------------------------------------------------

    public void validateProvenance(String provenanceJson, OperationalConfig config) {
        if (provenanceJson == null || provenanceJson.isBlank()) {
            throw new ValidationException("FORMAT.PROVENANCE-INVALID",
                "provenance é obrigatório");
        }

        Provenance provenance;
        try {
            provenance = fhirContext.newJsonParser().parseResource(Provenance.class, provenanceJson);
        } catch (DataFormatException e) {
            throw new ValidationException("FORMAT.PROVENANCE-INVALID",
                "Provenance não é um recurso FHIR R4 válido: " + e.getMessage());
        }

        if (provenance.getTarget() == null || provenance.getTarget().isEmpty()) {
            throw new ValidationException("FORMAT.PROVENANCE-INVALID",
                "Provenance.target deve conter pelo menos uma referência");
        }

        int maxEntries = config != null && config.getMaxEntriesBundle() != null
            ? config.getMaxEntriesBundle() : 1000;
        if (provenance.getTarget().size() > maxEntries) {
            throw new ValidationException("SECURITY.PROVENANCE-SIZE-LIMIT-EXCEEDED",
                "Provenance.target excede o máximo de " + maxEntries + " entradas");
        }

        Set<String> targets = new HashSet<>();
        for (var ref : provenance.getTarget()) {
            String refValue = ref.getReference();
            if (refValue == null || !UUID_URN.matcher(refValue).matches()) {
                throw new ValidationException("FORMAT.PROVENANCE-TARGET-INVALID",
                    "Provenance.target deve ser urn:uuid:<UUID RFC 4122>. Valor inválido: "
                    + refValue);
            }
            if (!targets.add(refValue)) {
                throw new ValidationException("FORMAT.PROVENANCE-TARGET-DUPLICATE",
                    "Provenance.target contém referência duplicada: " + refValue);
            }
        }
    }

    // -------------------------------------------------------------------------
    // 6. Cruzamento Bundle x Provenance
    // -------------------------------------------------------------------------

    public void validateBundleProvenanceCross(String bundleJson, String provenanceJson) {
        if (bundleJson == null || provenanceJson == null) return;

        Bundle bundle;
        Provenance provenance;
        try {
            bundle = fhirContext.newJsonParser().parseResource(Bundle.class, bundleJson);
            provenance = fhirContext.newJsonParser().parseResource(Provenance.class, provenanceJson);
        } catch (DataFormatException e) {
            return; // já validado anteriormente
        }

        // Mapeia fullUrls disponíveis no Bundle
        Set<String> bundleFullUrls = new HashSet<>();
        for (var entry : bundle.getEntry()) {
            String fullUrl = entry.getFullUrl();
            if (fullUrl != null) {
                if (!bundleFullUrls.add(fullUrl)) {
                    throw new ValidationException("FORMAT.BUNDLE-DUPLICATE-ENTRY",
                        "Bundle.entry contém fullUrl duplicado: " + fullUrl);
                }
            }
        }

        // Verifica que cada target do Provenance existe no Bundle
        for (var ref : provenance.getTarget()) {
            String target = ref.getReference();
            if (!bundleFullUrls.contains(target)) {
                throw new ValidationException("FORMAT.TARGET-REFERENCE-MISSING",
                    "Provenance.target '" + target + "' não encontrado em Bundle.entry.fullUrl");
            }
        }

        // Verifica que as entries correspondentes têm resource preenchido
        for (var entry : bundle.getEntry()) {
            if (provenance.getTarget().stream()
                    .anyMatch(t -> entry.getFullUrl().equals(t.getReference()))) {
                if (entry.getResource() == null) {
                    throw new ValidationException("FORMAT.BUNDLE-RESOURCE-MISSING",
                        "Bundle.entry com fullUrl '" + entry.getFullUrl()
                        + "' está referenciado em Provenance.target mas não tem resource");
                }
            }
        }
    }

    // -------------------------------------------------------------------------
    // 7. Cadeia de certificados
    // -------------------------------------------------------------------------

    public void validateCertificateChain(List<String> chain) {
        if (chain == null || chain.size() < 2) {
            throw new ValidationException("CERT.CHAIN-INCOMPLETE",
                "certificateChain deve conter pelo menos 2 certificados " +
                "(certificado do signatário + AC raiz ICP-Brasil)");
        }
        for (int i = 0; i < chain.size(); i++) {
            String cert = chain.get(i);
            if (cert == null || cert.isBlank()) {
                throw new ValidationException("CERT.BASE64-INVALID",
                    "certificateChain[" + i + "] está vazio");
            }
            try {
                byte[] decoded = Base64.getDecoder().decode(cert);
                // Verificação mínima de estrutura DER: deve iniciar com 0x30 (SEQUENCE)
                if (decoded.length < 2 || decoded[0] != 0x30) {
                    throw new ValidationException("CERT.INVALID-FORMAT",
                        "certificateChain[" + i + "] não parece ser um certificado DER válido");
                }
            } catch (IllegalArgumentException e) {
                throw new ValidationException("CERT.BASE64-INVALID",
                    "certificateChain[" + i + "] não é base64 válido");
            }
        }
    }

    // -------------------------------------------------------------------------
    // 8. Material criptográfico
    // -------------------------------------------------------------------------

    public void validateCryptoMaterial(CryptoMaterial crypto) {
        if (crypto == null || crypto.getType() == null) {
            throw new ValidationException("CONFIG.MISSING-PARAMETER",
                "cryptoMaterial e cryptoMaterial.type são obrigatórios");
        }
        switch (crypto.getType()) {
            case PEM -> {
                if (crypto.getPrivateKeyPem() == null || crypto.getPrivateKeyPem().isBlank()) {
                    throw new ValidationException("CONFIG.MISSING-PARAMETER",
                        "cryptoMaterial.privateKeyPem é obrigatório para type=PEM");
                }
            }
            case PKCS12 -> {
                if (crypto.getPkcs12Base64() == null || crypto.getPkcs12Base64().isBlank()) {
                    throw new ValidationException("CONFIG.MISSING-PARAMETER",
                        "cryptoMaterial.pkcs12Base64 é obrigatório para type=PKCS12");
                }
                if (crypto.getPassword() == null || crypto.getPassword().isBlank()) {
                    throw new ValidationException("CONFIG.MISSING-PARAMETER",
                        "cryptoMaterial.password é obrigatório para type=PKCS12");
                }
                if (crypto.getAlias() == null || crypto.getAlias().isBlank()) {
                    throw new ValidationException("CONFIG.MISSING-PARAMETER",
                        "cryptoMaterial.alias é obrigatório para type=PKCS12");
                }
                try {
                    Base64.getDecoder().decode(crypto.getPkcs12Base64());
                } catch (IllegalArgumentException e) {
                    throw new ValidationException("FORMAT.BASE64-INVALID",
                        "cryptoMaterial.pkcs12Base64 não é base64 válido");
                }
            }
            case SMARTCARD, TOKEN -> {
                if (crypto.getPin() == null || crypto.getPin().isBlank()) {
                    throw new ValidationException("CONFIG.MISSING-PARAMETER",
                        "cryptoMaterial.pin é obrigatório para type=" + crypto.getType());
                }
                if (crypto.getIdentifier() == null || crypto.getIdentifier().isBlank()) {
                    throw new ValidationException("CONFIG.MISSING-PARAMETER",
                        "cryptoMaterial.identifier é obrigatório para type=" + crypto.getType());
                }
                if (crypto.getTokenLabel() != null && crypto.getTokenLabel().length() > 32) {
                    throw new ValidationException("MIDDLEWARE.TOKEN-LABEL-INVALID",
                        "cryptoMaterial.tokenLabel não pode exceder 32 caracteres");
                }
            }
        }
    }

    // -------------------------------------------------------------------------
    // Configurações operacionais
    // -------------------------------------------------------------------------

    public void validateConfig(OperationalConfig config) {
        if (config == null) {
            throw new ValidationException("CONFIG.MISSING-PARAMETER",
                "config (configurações operacionais) é obrigatório");
        }
        validateRange(config.getOcspCacheTtl(), 300, 86400, "config.ocspCacheTtl",
            "CONFIG.TTL-OUT-OF-RANGE");
        validateRange(config.getCrlCacheTtl(), 300, 86400, "config.crlCacheTtl",
            "CONFIG.TTL-OUT-OF-RANGE");
        validateRange(config.getOcspTimeout(), 5, 120, "config.ocspTimeout",
            "CONFIG.TIMEOUT-OUT-OF-RANGE");
        validateRange(config.getCrlTimeout(), 5, 120, "config.crlTimeout",
            "CONFIG.TIMEOUT-OUT-OF-RANGE");
        validateRange(config.getTsaTimeout(), 5, 120, "config.tsaTimeout",
            "CONFIG.TIMEOUT-OUT-OF-RANGE");
        validateRange(config.getMaxRetries(), 1, 5, "config.maxRetries",
            "CONFIG.INVALID-PARAMETER");
        validateRange(config.getRetryInterval(), 1, 10, "config.retryInterval",
            "CONFIG.INVALID-PARAMETER");
        validateRange(config.getMaxEntriesBundle(), 100, 10000, "config.maxEntriesBundle",
            "CONFIG.BUNDLE-SIZE-LIMIT-OUT-OF-RANGE");
        validateRange(config.getMaxBundleSize(), 1_048_576, 209_715_200, "config.maxBundleSize",
            "CONFIG.BUNDLE-MEMORY-LIMIT-OUT-OF-RANGE");
        validateRange(config.getTimeoutVerificationBundle(), 5, 300,
            "config.timeoutVerificationBundle", "CONFIG.BUNDLE-TIMEOUT-OUT-OF-RANGE");
    }

    private void validateRange(Integer value, int min, int max, String field, String code) {
        if (value == null) {
            throw new ValidationException("CONFIG.MISSING-PARAMETER",
                field + " é obrigatório");
        }
        if (value < min || value > max) {
            throw new ValidationException(code,
                field + " deve estar no intervalo [" + min + ", " + max
                + "]. Valor recebido: " + value);
        }
    }
}