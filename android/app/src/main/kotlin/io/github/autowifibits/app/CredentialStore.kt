package io.github.autowifibits.app

import android.content.Context
import android.security.keystore.KeyGenParameterSpec
import android.security.keystore.KeyProperties
import android.util.Base64
import io.github.autowifibits.core.Credentials
import java.security.KeyStore
import javax.crypto.Cipher
import javax.crypto.KeyGenerator
import javax.crypto.SecretKey
import javax.crypto.spec.GCMParameterSpec

/** Pure encoding of credentials into the plaintext that gets encrypted (unit-tested). */
object CredentialCodec {
    private const val SEP = '\u0000'

    fun encode(c: Credentials): ByteArray {
        require(c.username.isNotBlank() && c.password.isNotEmpty()) { "username and password are required" }
        require(SEP !in c.username) { "invalid username" }
        return (c.username.trim() + SEP + c.password).toByteArray(Charsets.UTF_8)
    }

    fun decode(b: ByteArray): Credentials? {
        val s = String(b, Charsets.UTF_8)
        val i = s.indexOf(SEP)
        if (i <= 0 || i == s.length - 1) return null
        return Credentials(s.substring(0, i), s.substring(i + 1))
    }
}

/**
 * Credentials encrypted with an AES-256-GCM key that never leaves the Android Keystore. The
 * ciphertext and IV live in app-private SharedPreferences, excluded from backups.
 */
class CredentialStore(context: Context) {
    private val prefs = context.getSharedPreferences("secure", Context.MODE_PRIVATE)

    private fun key(): SecretKey {
        val ks = KeyStore.getInstance("AndroidKeyStore").apply { load(null) }
        (ks.getKey(ALIAS, null) as? SecretKey)?.let { return it }
        val gen = KeyGenerator.getInstance(KeyProperties.KEY_ALGORITHM_AES, "AndroidKeyStore")
        gen.init(
            KeyGenParameterSpec.Builder(ALIAS, KeyProperties.PURPOSE_ENCRYPT or KeyProperties.PURPOSE_DECRYPT)
                .setBlockModes(KeyProperties.BLOCK_MODE_GCM)
                .setEncryptionPaddings(KeyProperties.ENCRYPTION_PADDING_NONE)
                .setKeySize(256)
                .build(),
        )
        return gen.generateKey()
    }

    fun get(): Credentials? {
        val ct = prefs.getString("ct", null) ?: return null
        val iv = prefs.getString("iv", null) ?: return null
        return try {
            val c = Cipher.getInstance(TRANSFORM)
            c.init(Cipher.DECRYPT_MODE, key(), GCMParameterSpec(128, Base64.decode(iv, Base64.NO_WRAP)))
            CredentialCodec.decode(c.doFinal(Base64.decode(ct, Base64.NO_WRAP)))
        } catch (e: Exception) {
            null // key lost (e.g. restored device) or tampered: ask again
        }
    }

    fun set(cr: Credentials) {
        val c = Cipher.getInstance(TRANSFORM)
        c.init(Cipher.ENCRYPT_MODE, key())
        val ct = c.doFinal(CredentialCodec.encode(cr))
        prefs.edit()
            .putString("ct", Base64.encodeToString(ct, Base64.NO_WRAP))
            .putString("iv", Base64.encodeToString(c.iv, Base64.NO_WRAP))
            .apply()
    }

    fun delete() {
        prefs.edit().clear().apply()
    }

    private companion object {
        const val ALIAS = "autowifi-creds"
        const val TRANSFORM = "AES/GCM/NoPadding"
    }
}
