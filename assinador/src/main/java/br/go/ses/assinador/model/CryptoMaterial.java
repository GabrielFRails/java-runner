package br.go.ses.assinador.model;

import com.fasterxml.jackson.annotation.JsonIgnoreProperties;

/**
 * Material criptográfico do signatário.
 * Discriminado pelo campo {@code type}: PEM, PKCS12, SMARTCARD ou TOKEN.
 */
@JsonIgnoreProperties(ignoreUnknown = true)
public class CryptoMaterial {

    public enum Type { PEM, PKCS12, SMARTCARD, TOKEN }

    private Type type;

    // PEM
    private String privateKeyPem;
    private String password; // opcional para PEM, obrigatório para PKCS12

    // PKCS12
    private String pkcs12Base64;
    private String alias;

    // SMARTCARD / TOKEN
    private String pin;
    private String identifier;
    private Integer slotId;       // opcional
    private String tokenLabel;    // opcional, máx 32 chars

    public Type getType() { return type; }
    public void setType(Type type) { this.type = type; }

    public String getPrivateKeyPem() { return privateKeyPem; }
    public void setPrivateKeyPem(String privateKeyPem) { this.privateKeyPem = privateKeyPem; }

    public String getPassword() { return password; }
    public void setPassword(String password) { this.password = password; }

    public String getPkcs12Base64() { return pkcs12Base64; }
    public void setPkcs12Base64(String pkcs12Base64) { this.pkcs12Base64 = pkcs12Base64; }

    public String getAlias() { return alias; }
    public void setAlias(String alias) { this.alias = alias; }

    public String getPin() { return pin; }
    public void setPin(String pin) { this.pin = pin; }

    public String getIdentifier() { return identifier; }
    public void setIdentifier(String identifier) { this.identifier = identifier; }

    public Integer getSlotId() { return slotId; }
    public void setSlotId(Integer slotId) { this.slotId = slotId; }

    public String getTokenLabel() { return tokenLabel; }
    public void setTokenLabel(String tokenLabel) { this.tokenLabel = tokenLabel; }
}