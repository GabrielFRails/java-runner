package br.go.ses.assinador.controller;

import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.autoconfigure.web.servlet.AutoConfigureMockMvc;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.http.MediaType;
import org.springframework.test.web.servlet.MockMvc;

import static org.springframework.test.web.servlet.request.MockMvcRequestBuilders.post;
import static org.springframework.test.web.servlet.result.MockMvcResultMatchers.*;

/**
 * Testes de integração dos endpoints HTTP.
 * Sobem o contexto Spring completo e validam o fluxo ponta-a-ponta.
 *
 * Ambos os testes são "must fail" — verificam que requisições inválidas
 * retornam HTTP 422 com o código FHIR correto.
 */
@SpringBootTest
@AutoConfigureMockMvc
class SignatureControllerIntegrationTest {

    @Autowired
    private MockMvc mockMvc;

    private static final String CODESYSTEM =
        "https://fhir.saude.go.gov.br/r4/seguranca/CodeSystem/situacao-excepcional-assinatura";

    private static final String POLICY_VALIDA =
        "https://fhir.saude.go.gov.br/r4/seguranca/ImplementationGuide/br.go.ses.seguranca|0.1.2";

    private static final String CONFIG_PADRAO = """
        {
          "ocspCacheTtl": 3600,
          "crlCacheTtl": 3600,
          "ocspTimeout": 20,
          "crlTimeout": 20,
          "tsaTimeout": 20,
          "maxRetries": 3,
          "retryInterval": 2,
          "maxEntriesBundle": 1000,
          "maxBundleSize": 52428800,
          "timeoutVerificationBundle": 10
        }
        """;

    /**
     * Testa que certificateChain com apenas 1 elemento é rejeitado.
     * Equivalente ao curl: certificateChain: ["MAMB"]
     */
    @Test
    @DisplayName("MUST FAIL — certificateChain com 1 elemento deve retornar CERT.CHAIN-INCOMPLETE")
    void deveRejeitarCertChainIncompleta() throws Exception {
        long agora = System.currentTimeMillis() / 1000;

        String payload = """
            {
              "bundle": "{\\"resourceType\\":\\"Bundle\\",\\"type\\":\\"collection\\",\\"entry\\":[{\\"fullUrl\\":\\"urn:uuid:550e8400-e29b-41d4-a716-446655440000\\",\\"resource\\":{\\"resourceType\\":\\"Patient\\"}}]}",
              "provenance": "{\\"resourceType\\":\\"Provenance\\",\\"target\\":[{\\"reference\\":\\"urn:uuid:550e8400-e29b-41d4-a716-446655440000\\"}],\\"recorded\\":\\"2025-01-01T00:00:00Z\\",\\"agent\\":[{\\"who\\":{\\"reference\\":\\"Practitioner/1\\"}}]}",
              "referenceTimestamp": %d,
              "strategy": "iat",
              "policyUri": "%s",
              "certificateChain": ["MAMB"],
              "cryptoMaterial": {
                "type": "PEM",
                "privateKeyPem": "-----BEGIN PRIVATE KEY-----\\nMIIE\\n-----END PRIVATE KEY-----"
              },
              "config": %s
            }
            """.formatted(agora, POLICY_VALIDA, CONFIG_PADRAO);

        mockMvc.perform(post("/sign")
                .contentType(MediaType.APPLICATION_JSON)
                .content(payload))
            .andExpect(status().isUnprocessableEntity())
            .andExpect(jsonPath("$.resourceType").value("OperationOutcome"))
            .andExpect(jsonPath("$.issue[0].severity").value("error"))
            .andExpect(jsonPath("$.issue[0].details.coding[0].system").value(CODESYSTEM))
            .andExpect(jsonPath("$.issue[0].details.coding[0].code").value("CERT.CHAIN-INCOMPLETE"));
    }

    /**
     * Testa que strategy com valor inválido é rejeitada.
     * Equivalente ao curl: strategy: "invalida"
     */
    @Test
    @DisplayName("MUST FAIL — strategy inválida deve retornar CONFIG.INVALID-STRATEGY")
    void deveRejeitarStrategyInvalida() throws Exception {
        long agora = System.currentTimeMillis() / 1000;

        String payload = """
            {
              "referenceTimestamp": %d,
              "strategy": "invalida",
              "policyUri": "%s",
              "config": %s
            }
            """.formatted(agora, POLICY_VALIDA, CONFIG_PADRAO);

        mockMvc.perform(post("/sign")
                .contentType(MediaType.APPLICATION_JSON)
                .content(payload))
            .andExpect(status().isUnprocessableEntity())
            .andExpect(jsonPath("$.resourceType").value("OperationOutcome"))
            .andExpect(jsonPath("$.issue[0].severity").value("error"))
            .andExpect(jsonPath("$.issue[0].details.coding[0].system").value(CODESYSTEM))
            .andExpect(jsonPath("$.issue[0].details.coding[0].code").value("CONFIG.INVALID-STRATEGY"));
    }
}