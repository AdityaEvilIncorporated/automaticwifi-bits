plugins {
    kotlin("jvm")
}

kotlin { jvmToolchain(17) }

dependencies {
    // HTML form parsing for the login page (see ARCHITECTURE.md "Dependencies").
    implementation("org.jsoup:jsoup:1.21.2")
    testImplementation(kotlin("test"))
    testImplementation("org.junit.jupiter:junit-jupiter:5.11.4")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
    // JSON only for reading protocol/testcases in tests.
    testImplementation("org.json:json:20250517")
}

tasks.test {
    useJUnitPlatform()
    // Shared cases live at <repo>/protocol/testcases.
    systemProperty("testcases.dir", rootProject.projectDir.resolve("../protocol/testcases").canonicalPath)
    // Go mock portal binary (scripts/test-android.sh builds it); scenario tests skip without it.
    System.getenv("MOCKPORTAL_BIN")?.let { environment("MOCKPORTAL_BIN", it) }
    testLogging { events("failed"); exceptionFormat = org.gradle.api.tasks.testing.logging.TestExceptionFormat.FULL }
}
