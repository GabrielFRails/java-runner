package br.go.ses.assinador.service;

import br.go.ses.assinador.model.SignRequest;
import br.go.ses.assinador.model.ValidateRequest;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.*;

class FakeSignatureServiceTest {

    private FakeSignatureService service;

    @BeforeEach
    void setUp() {
        service = new FakeSignatureService();
    }

    @Test
    @DisplayName("sign deve retornar JSON com resourceType Signature")
    void signDeveRetornarSignature() {
        String resultado = service.sign(new SignRequest());
        assertThat(resultado).contains("\"resourceType\": \"Signature\"");
    }

    @Test
    @DisplayName("sign deve retornar campo data preenchido")
    void signDeveRetornarDataPreenchido() {
        String resultado = service.sign(new SignRequest());
        assertThat(resultado).contains("\"data\":");
        assertThat(resultado).contains("SIMULATED_SIGNATURE");
    }

    @Test
    @DisplayName("sign deve retornar sigFormat application/jose")
    void signDeveRetornarSigFormat() {
        String resultado = service.sign(new SignRequest());
        assertThat(resultado).contains("\"sigFormat\": \"application/jose\"");
    }

    @Test
    @DisplayName("validate deve retornar OperationOutcome com VALIDATION.SUCCESS")
    void validateDeveRetornarSucesso() {
        String resultado = service.validate(new ValidateRequest());
        assertThat(resultado).contains("\"resourceType\": \"OperationOutcome\"");
        assertThat(resultado).contains("VALIDATION.SUCCESS");
    }

    @Test
    @DisplayName("validate deve retornar severity information")
    void validateDeveRetornarSeverityInformation() {
        String resultado = service.validate(new ValidateRequest());
        assertThat(resultado).contains("\"severity\": \"information\"");
    }
}