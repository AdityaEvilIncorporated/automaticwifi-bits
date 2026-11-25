pluginManagement {
    repositories { gradlePluginPortal(); google(); mavenCentral() }
}
dependencyResolutionManagement {
    repositories { google(); mavenCentral() }
}
rootProject.name = "autowifi-bits"

// :core is plain Kotlin/JVM (protocol, client, engine) and is tested on any JDK.
include(":core")

// :app needs the Android SDK; it is included only when one is configured, so the core tests
// run in a plain JDK container (scripts/test-android.sh).
val sdk = System.getenv("ANDROID_HOME") ?: System.getenv("ANDROID_SDK_ROOT")
if (sdk != null || file("local.properties").exists()) include(":app")
