package br.go.ses.assinador.service;

import br.go.ses.assinador.model.SignRequest;
import br.go.ses.assinador.model.ValidateRequest;

/**
 * Serviço de assinatura digital.
 * A implementação real seria criptográfica; aqui usamos simulação.
 */
public interface SignatureService {

    /**
     * Simula a criação de uma assinatura digital.
     * Parâmetros já foram validados antes desta chamada.
     *
     * @param request parâmetros da operação
     * @return JSON representando um FHIR Signature simulado
     */
    String sign(SignRequest request);

    /**
     * Simula a validação de uma assinatura digital.
     * Parâmetros já foram validados antes desta chamada.
     *
     * @param request parâmetros da operação
     * @return JSON representando um FHIR OperationOutcome com resultado
     */
    String validate(ValidateRequest request);
}