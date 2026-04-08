package br.go.ses.assinador.model;

/**
 * Exceção lançada quando a validação de parâmetros falha.
 * Carrega o código FHIR padronizado e uma mensagem descritiva.
 */
public class ValidationException extends RuntimeException {

    private final String fhirCode;

    public ValidationException(String fhirCode, String message) {
        super(message);
        this.fhirCode = fhirCode;
    }

    public String getFhirCode() {
        return fhirCode;
    }
}