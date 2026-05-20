package br.go.ses.assinador.lifecycle;

import org.springframework.beans.factory.annotation.Value;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.scheduling.annotation.Scheduled;
import org.springframework.stereotype.Component;

import java.time.Clock;
import java.util.concurrent.atomic.AtomicLong;

@Component
public class InactivityShutdownMonitor {

    private final long timeoutMillis;
    private final Clock clock;
    private final Runnable shutdownAction;
    private final AtomicLong lastActivityMillis;

    @Autowired
    public InactivityShutdownMonitor(
        @Value("${hubsaude.inactivity-timeout-minutes:0}") long timeoutMinutes
    ) {
        this(timeoutMinutes, Clock.systemUTC(), () -> System.exit(0));
    }

    InactivityShutdownMonitor(long timeoutMinutes, Clock clock, Runnable shutdownAction) {
        this.timeoutMillis = Math.max(0, timeoutMinutes) * 60_000;
        this.clock = clock;
        this.shutdownAction = shutdownAction;
        this.lastActivityMillis = new AtomicLong(clock.millis());
    }

    public void touch() {
        lastActivityMillis.set(clock.millis());
    }

    public boolean isEnabled() {
        return timeoutMillis > 0;
    }

    boolean shouldShutdown() {
        return isEnabled() && clock.millis() - lastActivityMillis.get() >= timeoutMillis;
    }

    @Scheduled(fixedDelay = 30_000)
    public void shutdownIfInactive() {
        if (shouldShutdown()) {
            shutdownAction.run();
        }
    }
}
