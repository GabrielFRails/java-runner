package br.go.ses.assinador.crypto;

import br.go.ses.assinador.model.CryptoMaterial;
import br.go.ses.assinador.model.ValidationException;
import org.junit.jupiter.api.Test;

import static org.assertj.core.api.Assertions.assertThat;
import static org.assertj.core.api.Assertions.catchThrowableOfType;

class Pkcs11TokenServiceTest {

    @Test
    void deveRetornarErroClaroQuandoBibliotecaNaoExiste() {
        var crypto = new CryptoMaterial();
        crypto.setType(CryptoMaterial.Type.TOKEN);
        crypto.setPin("1234");
        crypto.setIdentifier("alias");
        crypto.setPkcs11LibraryPath("/caminho/inexistente/libpkcs11.so");

        var ex = catchThrowableOfType(
            () -> new Pkcs11TokenService().assertTokenAvailable(crypto),
            ValidationException.class
        );

        assertThat(ex.getFhirCode()).isEqualTo("PKCS11.LIBRARY-NOT-FOUND");
    }
}
