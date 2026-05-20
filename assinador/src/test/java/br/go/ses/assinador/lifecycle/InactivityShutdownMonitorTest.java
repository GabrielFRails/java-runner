package br.go.ses.assinador.lifecycle;

import org.junit.jupiter.api.Test;

import java.time.Clock;
import java.time.Instant;
import java.time.ZoneOffset;
import java.util.concurrent.atomic.AtomicLong;
import java.util.concurrent.atomic.AtomicBoolean;

import static org.assertj.core.api.Assertions.assertThat;

class InactivityShutdownMonitorTest {

    @Test
    void timeoutZeroMantemMonitorDesabilitado() {
        var monitor = new InactivityShutdownMonitor(0, fixedClock(0), () -> {});

        assertThat(monitor.isEnabled()).isFalse();
        assertThat(monitor.shouldShutdown()).isFalse();
    }

    @Test
    void executaShutdownQuandoTempoDeInatividadeExpira() {
        var called = new AtomicBoolean(false);
        var clock = new MutableClock(0);
        var monitor = new InactivityShutdownMonitor(1, clock, () -> called.set(true));

        clock.setMillis(61_000);

        monitor.shutdownIfInactive();

        assertThat(called).isTrue();
    }

    private Clock fixedClock(long millis) {
        return Clock.fixed(Instant.ofEpochMilli(millis), ZoneOffset.UTC);
    }

    private static class MutableClock extends Clock {
        private final AtomicLong millis;

        MutableClock(long millis) {
            this.millis = new AtomicLong(millis);
        }

        void setMillis(long millis) {
            this.millis.set(millis);
        }

        @Override
        public ZoneOffset getZone() {
            return ZoneOffset.UTC;
        }

        @Override
        public Clock withZone(java.time.ZoneId zone) {
            return this;
        }

        @Override
        public Instant instant() {
            return Instant.ofEpochMilli(millis());
        }

        @Override
        public long millis() {
            return millis.get();
        }
    }
}
