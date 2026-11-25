package io.github.autowifibits.core

import org.junit.jupiter.api.AfterAll
import org.junit.jupiter.api.Assumptions.assumeTrue
import org.junit.jupiter.api.BeforeAll
import org.junit.jupiter.api.TestInstance
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue

/** Mirrors internal/tlsconf/tlsconf_test.go against the Go mock portal's certificate. */
@TestInstance(TestInstance.Lifecycle.PER_CLASS)
class PortalTlsTest {
    private lateinit var mock: MockPortal

    @BeforeAll fun up() { assumeTrue(MockPortal.available(), "MOCKPORTAL_BIN not set"); mock = MockPortal.start() }
    @AfterAll fun down() { if (::mock.isInitialized) mock.close() }

    private val good = Credentials("f20250001", "correct-horse")

    private fun client(factory: javax.net.ssl.SSLSocketFactory) =
        PortalClient(mock.config(), UrlConnectionHttp(), UrlConnectionHttp(sslSocketFactory = factory), Resolver { true })

    @Test fun spkiMatchesGo() {
        val k = PortalTls.spkiSha256(mock.cert())
        assertEquals(64, k.length)
        assertEquals(mock.goSpki, k, "Kotlin and Go must compute the same pin")
    }

    @Test fun trustedStoreLogsIn() {
        mock.control("reset")
        val r = client(PortalTls.socketFactory("127.0.0.1", "", mock.trustStore())).checkOnce { good }
        assertEquals(Outcome.LOGGED_IN, r.outcome, r.error)
    }

    @Test fun untrustedWithoutPinIsRefused() {
        mock.control("reset")
        val r = client(PortalTls.socketFactory("127.0.0.1", "")).checkOnce { good }
        assertEquals(Outcome.PORTAL_ERROR, r.outcome)
        assertEquals(0, mock.control("state").getInt("logins_ok") + mock.control("state").getInt("logins_failed"))
    }

    @Test fun matchingPinIsAccepted() {
        mock.control("reset")
        val r = client(PortalTls.socketFactory("127.0.0.1", mock.goSpki)).checkOnce { good }
        assertEquals(Outcome.LOGGED_IN, r.outcome, r.error)
    }

    @Test fun wrongPinIsRefused() {
        mock.control("reset")
        val r = client(PortalTls.socketFactory("127.0.0.1", "ab".repeat(32))).checkOnce { good }
        assertEquals(Outcome.PORTAL_ERROR, r.outcome)
        assertEquals(0, mock.control("state").getInt("logins_failed"))
    }

    @Test fun factoryIsBoundToItsHost() {
        mock.control("reset")
        // pin matches, but the factory is for another host
        val r = client(PortalTls.socketFactory("fw.bits-pilani.ac.in", mock.goSpki)).checkOnce { good }
        assertEquals(Outcome.PORTAL_ERROR, r.outcome)
    }

    @Test fun badPinRejected() {
        for (p in listOf("abc", "G".repeat(64), "AB".repeat(32))) {
            assertFailsWith<IllegalArgumentException>(p) { PortalTls.socketFactory("h", p) }
        }
        assertTrue(PortalTls.socketFactory("h", "").javaClass.name.isNotEmpty())
    }
}
