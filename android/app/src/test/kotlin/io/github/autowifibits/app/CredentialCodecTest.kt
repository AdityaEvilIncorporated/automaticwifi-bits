package io.github.autowifibits.app

import io.github.autowifibits.core.Credentials
import org.junit.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertNull

class CredentialCodecTest {
    @Test fun roundTrip() {
        for (c in listOf(Credentials("f20250001", "p"), Credentials("f2025", " spaces ok "), Credentials("ü", "pässwörd\u0000x"))) {
            assertEquals(c, CredentialCodec.decode(CredentialCodec.encode(c)))
        }
    }

    @Test fun trimsUsername() = assertEquals(Credentials("id", "p"), CredentialCodec.decode(CredentialCodec.encode(Credentials("  id ", "p"))))

    @Test fun rejectsEmptyAndNul() {
        for (c in listOf(Credentials("", "p"), Credentials(" ", "p"), Credentials("u", ""), Credentials("a\u0000b", "p"))) {
            assertFailsWith<IllegalArgumentException> { CredentialCodec.encode(c) }
        }
    }

    @Test fun badPlaintext() {
        for (s in listOf("", "nouser", "\u0000pw", "user\u0000")) assertNull(CredentialCodec.decode(s.toByteArray()))
    }
}
