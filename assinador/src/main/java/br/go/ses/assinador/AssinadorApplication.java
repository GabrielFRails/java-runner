package br.go.ses.assinador;

import br.go.ses.assinador.cli.AssinadorCli;
import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

@SpringBootApplication
public class AssinadorApplication {

    public static void main(String[] args) {
        if (args.length > 0 && (args[0].equals("sign") || args[0].equals("validate"))) {
            AssinadorCli.run(args);
            return;
        }
        SpringApplication.run(AssinadorApplication.class, args);
    }
}
