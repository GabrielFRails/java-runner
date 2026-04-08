package br.go.ses.assinador.service;

import br.go.ses.assinador.model.SignRequest;
import br.go.ses.assinador.model.ValidateRequest;
import org.springframework.stereotype.Service;

/**
 * Implementação simulada do {@link SignatureService}.
 *
 * <p>Retorna respostas pré-construídas sem realizar operações criptográficas reais.
 * O foco desta implementação está na validação correta dos parâmetros de entrada,
 * realizada pelo {@link br.go.ses.assinador.validation.SignatureValidator} antes
 * de qualquer chamada a este serviço.
 */
@Service
public class FakeSignatureService implements SignatureService {

    private static final String CODESYSTEM =
        "https://fhir.saude.go.gov.br/r4/seguranca/CodeSystem/situacao-excepcional-assinatura";

    /**
     * Retorna uma assinatura FHIR simulada.
     * Em um sistema real, aqui ocorreria a operação criptográfica JAdES/JWS.
     */
    @Override
    public String sign(SignRequest request) {
        // Em produção: executaria as 14 etapas de criação de assinatura JAdES
        // Aqui: retorna resposta simulada pré-construída
        return """
            {
              "resourceType": "Signature",
              "type": [{
                "system": "urn:iso-astm:E1762-95:2013",
                "code": "1.2.840.10065.1.12.1.1",
                "display": "Author's Signature"
              }],
              "when": "2025-01-01T00:00:00Z",
              "who": {
                "identifier": {
                  "system": "urn:brasil:cpf",
                  "value": "00000000000"
                }
              },
              "sigFormat": "application/jose",
              "targetFormat": "application/octet-stream",
              "data": "SIMULATED_SIGNATURE_BASE64_eyJhbGciOiJSUzI1NiIsIng1YyI6W10sInNpZ1BJZCI6eyJpZCI6InNpbXVsYWRvIn19.U0lNVUxBVEVE.U0lNVUxBVEVE"
            }
            """;
    }

    /**
     * Retorna resultado de validação simulado (sempre sucesso).
     * Em um sistema real, aqui ocorreriam as 7 etapas de validação JAdES.
     */
    @Override
    public String validate(ValidateRequest request) {
        // Em produção: executaria as 7 etapas de validação
        // Aqui: retorna sucesso simulado
        return """
            {
              "resourceType": "OperationOutcome",
              "issue": [{
                "severity": "information",
                "code": "informational",
                "details": {
                  "coding": [{
                    "system": "%s",
                    "code": "VALIDATION.SUCCESS"
                  }]
                },
                "diagnostics": "Assinatura digital validada com sucesso (simulação)"
              }]
            }
            """.formatted(CODESYSTEM);
    }
}