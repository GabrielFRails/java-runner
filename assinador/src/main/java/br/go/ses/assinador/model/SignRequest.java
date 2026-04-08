package br.go.ses.assinador.model;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;
import java.util.List;

/**
 * Entradas da operação de criação de assinatura digital.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class SignRequest {

    /** Bundle FHIR R4 serializado em JSON */
    private String bundle;

    /** Provenance FHIR R4 serializado em JSON */
    private String provenance;

    /** Material criptográfico do signatário */
    private CryptoMaterial cryptoMaterial;

    /** Cadeia de certificados — base64 DER, mínimo 2 (folha + raiz ICP-Brasil) */
    private List<String> certificateChain;

    /** Timestamp de referência Unix UTC — intervalo [1751328000, 4102444800] */
    private Long referenceTimestamp;

    /** Estratégia de timestamp: "iat" ou "tsa" */
    private String strategy;

    /** URI da política de assinatura com versão semântica */
    private String policyUri;

    /** Configurações operacionais */
    private OperationalConfig config;

    public String getBundle() { return bundle; }
    public void setBundle(String bundle) { this.bundle = bundle; }

    public String getProvenance() { return provenance; }
    public void setProvenance(String provenance) { this.provenance = provenance; }

    public CryptoMaterial getCryptoMaterial() { return cryptoMaterial; }
    public void setCryptoMaterial(CryptoMaterial cryptoMaterial) { this.cryptoMaterial = cryptoMaterial; }

    public List<String> getCertificateChain() { return certificateChain; }
    public void setCertificateChain(List<String> certificateChain) { this.certificateChain = certificateChain; }

    public Long getReferenceTimestamp() { return referenceTimestamp; }
    public void setReferenceTimestamp(Long referenceTimestamp) { this.referenceTimestamp = referenceTimestamp; }

    public String getStrategy() { return strategy; }
    public void setStrategy(String strategy) { this.strategy = strategy; }

    public String getPolicyUri() { return policyUri; }
    public void setPolicyUri(String policyUri) { this.policyUri = policyUri; }

    public OperationalConfig getConfig() { return config; }
    public void setConfig(OperationalConfig config) { this.config = config; }
}