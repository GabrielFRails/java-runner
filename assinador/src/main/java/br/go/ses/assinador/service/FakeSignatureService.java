package br.go.ses.assinador.service;

import br.go.ses.assinador.model.SignRequest;
import br.go.ses.assinador.model.ValidateRequest;
import org.springframework.stereotype.Service;

import java.nio.charset.StandardCharsets;
import java.util.Base64;

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
     * Retorna resultado de validação simulado baseado em critério simples.
     * Em um sistema real, aqui ocorreriam as 7 etapas de validação JAdES.
     */
    @Override
    public String validate(ValidateRequest request) {
        // Em produção: executaria as 7 etapas de validação.
        // Aqui: a assinatura é considerada inválida quando o conteúdo decodificado
        // contém a palavra INVALID; nos demais casos, retorna sucesso.
        String decoded = new String(
            Base64.getDecoder().decode(request.getSignatureData()),
            StandardCharsets.UTF_8
        );
        boolean valid = !decoded.toUpperCase().contains("INVALID");

        return """
            {
              "resourceType": "OperationOutcome",
              "issue": [{
                "severity": "%s",
                "code": "%s",
                "details": {
                  "coding": [{
                    "system": "%s",
                    "code": "%s"
                  }]
                },
                "diagnostics": "%s"
              }]
            }
            """.formatted(
                valid ? "information" : "error",
                valid ? "informational" : "invalid",
                CODESYSTEM,
                valid ? "VALIDATION.SUCCESS" : "VALIDATION.FAILURE",
                valid
                    ? "Assinatura digital validada com sucesso (simulação)"
                    : "Assinatura digital inválida conforme critério simples da simulação"
            );
    }
}
