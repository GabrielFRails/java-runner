package br.go.ses.assinador.model;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Entradas da operação de validação de assinatura digital.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class ValidateRequest {

    /** Signature.data: JWS JSON Serialization em base64 (padrão FHIR) */
    private String signatureData;

    /** Timestamp de referência Unix UTC — intervalo [1751328000, 4102444800] */
    private Long referenceTimestamp;

    /** URI da política de assinatura com versão semântica */
    private String policyUri;

    /** Configurações operacionais */
    private OperationalConfig config;

    // Opcional — necessário apenas para verificação de integridade do conteúdo
    private String bundle;
    private String provenance;

    public String getSignatureData() { return signatureData; }
    public void setSignatureData(String signatureData) { this.signatureData = signatureData; }

    public Long getReferenceTimestamp() { return referenceTimestamp; }
    public void setReferenceTimestamp(Long referenceTimestamp) { this.referenceTimestamp = referenceTimestamp; }

    public String getPolicyUri() { return policyUri; }
    public void setPolicyUri(String policyUri) { this.policyUri = policyUri; }

    public OperationalConfig getConfig() { return config; }
    public void setConfig(OperationalConfig config) { this.config = config; }

    public String getBundle() { return bundle; }
    public void setBundle(String bundle) { this.bundle = bundle; }

    public String getProvenance() { return provenance; }
    public void setProvenance(String provenance) { this.provenance = provenance; }
}