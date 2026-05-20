package br.go.ses.assinador.crypto;

import br.go.ses.assinador.model.CryptoMaterial;
import br.go.ses.assinador.model.ValidationException;
import org.springframework.stereotype.Service;

import java.nio.file.Files;
import java.nio.file.Path;
import java.security.KeyStore;
import java.security.Provider;
import java.security.Security;

@Service
public class Pkcs11TokenService {

    public void assertTokenAvailable(CryptoMaterial crypto) {
        Path libraryPath = Path.of(crypto.getPkcs11LibraryPath());
        if (!Files.isRegularFile(libraryPath)) {
            throw new ValidationException("PKCS11.LIBRARY-NOT-FOUND",
                "Biblioteca PKCS#11 não encontrada: " + libraryPath);
        }

        Provider baseProvider = Security.getProvider("SunPKCS11");
        if (baseProvider == null) {
            throw new ValidationException("PKCS11.PROVIDER-UNAVAILABLE",
                "Provider SunPKCS11 não está disponível neste JDK");
        }

        Path configPath = null;
        Provider provider = null;
        try {
            configPath = Files.createTempFile("hubsaude-pkcs11-", ".cfg");
            Files.writeString(configPath, buildProviderConfig(crypto, libraryPath));

            provider = baseProvider.configure(configPath.toString());
            Security.addProvider(provider);

            KeyStore keyStore = KeyStore.getInstance("PKCS11", provider);
            keyStore.load(null, crypto.getPin().toCharArray());

            if (!keyStore.containsAlias(crypto.getIdentifier())) {
                throw new ValidationException("PKCS11.KEY-NOT-FOUND",
                    "Identificador não encontrado no dispositivo PKCS#11: "
                    + crypto.getIdentifier());
            }
        } catch (ValidationException e) {
            throw e;
        } catch (Exception e) {
            throw new ValidationException("PKCS11.DEVICE-UNAVAILABLE",
                "Não foi possível acessar o dispositivo PKCS#11: " + e.getMessage());
        } finally {
            if (provider != null) {
                Security.removeProvider(provider.getName());
            }
            if (configPath != null) {
                try {
                    Files.deleteIfExists(configPath);
                } catch (Exception ignored) {
                    // arquivo temporário; falha de limpeza não altera o resultado da operação
                }
            }
        }
    }

    private String buildProviderConfig(CryptoMaterial crypto, Path libraryPath) {
        StringBuilder config = new StringBuilder();
        config.append("name = HubSaudeToken\n");
        config.append("library = ").append(libraryPath).append("\n");
        if (crypto.getSlotId() != null) {
            config.append("slot = ").append(crypto.getSlotId()).append("\n");
        }
        return config.toString();
    }
}
