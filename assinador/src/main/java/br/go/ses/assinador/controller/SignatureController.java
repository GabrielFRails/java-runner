package br.go.ses.assinador.controller;

import br.go.ses.assinador.model.SignRequest;
import br.go.ses.assinador.model.ValidateRequest;
import br.go.ses.assinador.model.ValidationException;
import br.go.ses.assinador.service.SignatureService;
import br.go.ses.assinador.validation.SignatureValidator;
import org.springframework.http.HttpStatus;
import org.springframework.http.MediaType;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

/**
 * Endpoints HTTP do assinador.
 *
 * <p>POST /sign    — cria assinatura digital simulada
 * <p>POST /validate — valida assinatura digital simulada
 */
@RestController
public class SignatureController {

    private static final String CODESYSTEM =
        "https://fhir.saude.go.gov.br/r4/seguranca/CodeSystem/situacao-excepcional-assinatura";

    private final SignatureService signatureService;
    private final SignatureValidator validator;

    public SignatureController(SignatureService signatureService, SignatureValidator validator) {
        this.signatureService = signatureService;
        this.validator = validator;
    }

    @PostMapping(
        value = "/sign",
        consumes = MediaType.APPLICATION_JSON_VALUE,
        produces = MediaType.APPLICATION_JSON_VALUE
    )
    public ResponseEntity<String> sign(@RequestBody SignRequest request) {
        try {
            validator.validateSignRequest(request);
            String result = signatureService.sign(request);
            return ResponseEntity.ok(result);
        } catch (ValidationException e) {
            return ResponseEntity
                .status(HttpStatus.UNPROCESSABLE_ENTITY)
                .body(buildOperationOutcome(e.getFhirCode(), e.getMessage()));
        }
    }

    @PostMapping(
        value = "/validate",
        consumes = MediaType.APPLICATION_JSON_VALUE,
        produces = MediaType.APPLICATION_JSON_VALUE
    )
    public ResponseEntity<String> validate(@RequestBody ValidateRequest request) {
        try {
            validator.validateValidateRequest(request);
            String result = signatureService.validate(request);
            return ResponseEntity.ok(result);
        } catch (ValidationException e) {
            return ResponseEntity
                .status(HttpStatus.UNPROCESSABLE_ENTITY)
                .body(buildOperationOutcome(e.getFhirCode(), e.getMessage()));
        }
    }

    @GetMapping(value = "/health", produces = MediaType.APPLICATION_JSON_VALUE)
    public ResponseEntity<String> health() {
        return ResponseEntity.ok("""
            {"status": "UP", "service": "assinador"}
            """);
    }

    private String buildOperationOutcome(String code, String diagnostics) {
        return """
            {
              "resourceType": "OperationOutcome",
              "issue": [{
                "severity": "error",
                "code": "invalid",
                "details": {
                  "coding": [{
                    "system": "%s",
                    "code": "%s"
                  }]
                },
                "diagnostics": "%s"
              }]
            }
            """.formatted(CODESYSTEM, code, diagnostics.replace("\"", "\\\""));
    }
}