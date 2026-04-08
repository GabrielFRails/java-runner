package br.go.ses.assinador.model;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Configurações operacionais para controle do processo de assinatura/validação.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class OperationalConfig {

    // Cache
    private Integer ocspCacheTtl = 3600;   // [300, 86400]
    private Integer crlCacheTtl  = 3600;   // [300, 86400]

    // Timeouts
    private Integer ocspTimeout  = 20;     // [5, 120]
    private Integer crlTimeout   = 20;     // [5, 120]
    private Integer tsaTimeout   = 20;     // [5, 120]

    // Retry
    private Integer maxRetries    = 3;     // [1, 5]
    private Integer retryInterval = 2;     // [1, 10]

    // TSA (obrigatório se strategy=tsa)
    private String tsaUrl;
    private String tsaUsername;
    private String tsaPassword;

    // Limites de segurança
    private Integer maxEntriesBundle          = 1000;     // [100, 10000]
    private Integer maxBundleSize             = 52428800; // [1048576, 209715200] (50MB)
    private Integer timeoutVerificationBundle = 10;       // [5, 300]

    // Getters e Setters
    public Integer getOcspCacheTtl() { return ocspCacheTtl; }
    public void setOcspCacheTtl(Integer ocspCacheTtl) { this.ocspCacheTtl = ocspCacheTtl; }

    public Integer getCrlCacheTtl() { return crlCacheTtl; }
    public void setCrlCacheTtl(Integer crlCacheTtl) { this.crlCacheTtl = crlCacheTtl; }

    public Integer getOcspTimeout() { return ocspTimeout; }
    public void setOcspTimeout(Integer ocspTimeout) { this.ocspTimeout = ocspTimeout; }

    public Integer getCrlTimeout() { return crlTimeout; }
    public void setCrlTimeout(Integer crlTimeout) { this.crlTimeout = crlTimeout; }

    public Integer getTsaTimeout() { return tsaTimeout; }
    public void setTsaTimeout(Integer tsaTimeout) { this.tsaTimeout = tsaTimeout; }

    public Integer getMaxRetries() { return maxRetries; }
    public void setMaxRetries(Integer maxRetries) { this.maxRetries = maxRetries; }

    public Integer getRetryInterval() { return retryInterval; }
    public void setRetryInterval(Integer retryInterval) { this.retryInterval = retryInterval; }

    public String getTsaUrl() { return tsaUrl; }
    public void setTsaUrl(String tsaUrl) { this.tsaUrl = tsaUrl; }

    public String getTsaUsername() { return tsaUsername; }
    public void setTsaUsername(String tsaUsername) { this.tsaUsername = tsaUsername; }

    public String getTsaPassword() { return tsaPassword; }
    public void setTsaPassword(String tsaPassword) { this.tsaPassword = tsaPassword; }

    public Integer getMaxEntriesBundle() { return maxEntriesBundle; }
    public void setMaxEntriesBundle(Integer maxEntriesBundle) { this.maxEntriesBundle = maxEntriesBundle; }

    public Integer getMaxBundleSize() { return maxBundleSize; }
    public void setMaxBundleSize(Integer maxBundleSize) { this.maxBundleSize = maxBundleSize; }

    public Integer getTimeoutVerificationBundle() { return timeoutVerificationBundle; }
    public void setTimeoutVerificationBundle(Integer t) { this.timeoutVerificationBundle = t; }
}