plugins {
    id("com.android.application")
    kotlin("android")
}

android {
    namespace = "io.github.autowifibits.app"
    compileSdk = 35
    defaultConfig {
        applicationId = "io.github.autowifibits.app"
        minSdk = 26
        targetSdk = 35
        versionCode = (System.getenv("APP_VERSION_CODE") ?: "1").toInt()
        versionName = System.getenv("APP_VERSION_NAME") ?: "dev"
    }
    // Release signing comes from the environment (CI secrets); never from files in the repo.
    val ks = System.getenv("ANDROID_KEYSTORE_FILE")
    signingConfigs {
        if (ks != null) create("release") {
            storeFile = file(ks)
            storePassword = System.getenv("ANDROID_KEYSTORE_PASSWORD")
            keyAlias = System.getenv("ANDROID_KEY_ALIAS")
            keyPassword = System.getenv("ANDROID_KEY_PASSWORD")
        }
    }
    buildTypes {
        release {
            isMinifyEnabled = false
            if (ks != null) signingConfig = signingConfigs.getByName("release")
        }
    }
    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }
    kotlinOptions { jvmTarget = "17" }
    lint { abortOnError = true; checkReleaseBuilds = false }
}

dependencies {
    implementation(project(":core"))
    // Background checks: periodic (15 min), delayed keepalive/retry, after a Wi-Fi join (spec).
    implementation("androidx.work:work-runtime:2.10.0")
    testImplementation(kotlin("test"))
    testImplementation("junit:junit:4.13.2")
}
