# Scanner image security and compatibility

The Java 17 image keeps PHP on the 8.2 branch and requires 8.2.35 or later for
CVE-2026-109618 and CVE-2026-109619. Alpine 3.22 supplies the fixed PHP packages.
The `php` command and existing plugin flags remain available.

The Go builder uses HHI Go 1.26.10, including the fix for
[CVE-2025-68121](https://pkg.go.dev/vuln/GO-2026-4337). It builds the committed
module versions with `-mod=readonly`; `go.mod` retains its existing language
version. BuildKit cross-compiles on the builder's native platform. Legacy builds
retain the existing Linux/amd64 target.

SonarScanner stays on the 5.0 release branch. The default is
[5.0.2.4997](https://github.com/SonarSource/sonar-scanner-cli/releases/tag/5.0.2.4997),
which updates its OkHttp/Okio libraries and resolves the old Okio finding. This avoids a scanner major-version or
Java-major migration. Existing `SONAR_VERSION` build overrides still work and
must be assessed and scanned separately when used.

Build and check the final image:

```sh
docker build --pull --no-cache -t sonarqube-scanner:test .
sh tests/smoke-image.sh sonarqube-scanner:test
trivy image --scanners vuln sonarqube-scanner:test
```

A package scan can report
[CVE-2026-39824](https://pkg.go.dev/vuln/GO-2026-5024) against the older `x/sys`
module in the binary's dependency metadata. The affected function is
`golang.org/x/sys/windows.NewNTUnicodeString`; this Linux image links `x/sys/unix`
and excludes `x/sys/windows`. Record this platform-specific disposition with the
raw scan instead of suppressing it globally. Reassess it before any Windows build
or dependency changes.

The exact 5.0.2 image scan still reports **CVE-2021-0341 (HIGH)** for shaded
OkHttp 3.14.9. The [upstream maintainer's assessment](https://github.com/square/okhttp/issues/6724)
says the normal client path guards non-ASCII input before the affected verifier;
exposure depends on direct verifier use. Inspection of the bundled scanner JAR
found references only in OkHttpClient.Builder, RealConnection and the verifier
itself. This supports further reachability review, but does not constitute a
Security-approved clearance. Keep this finding open for Security disposition or
a validated upstream scanner/library upgrade before publication. Do not replace
shaded JAR classes or suppress the alert to obtain a clean scan.

Before publication, also run a real analysis against the supported SonarQube
server versions, scan the exact release digest and follow the normal tag/fleet
rollout. CLI startup checks alone do not establish end-to-end server compatibility.

The image also uses the incident-patched HHI roots for Go, Java, PHP and the
system certificate bundle. The PHP/runtime and locked-module build changes
follow [PR #114](https://github.com/drone-plugins/sonarqube-scanner/pull/114).
Review the overlapping changes together when merging these incident fixes.

The default entrypoint remains the scanner wrapper; this image does not install
PHP-FPM or Apache. PHP's [October 23 session-upload-progress advisory](https://www.php.net/archive/2026.php)
also requires configuration review wherever PHP is used to serve HTTP requests.
The PHP patch floor above does not establish that separate service-level review
or the hosted-image rollout tracked in CI-25471.

All base indexes are pinned to verified existing multi-platform digests. The final
image uses the available patched Node 22.23.4 package and OpenSSL system roots,
plus pip 26.2.2 and certifi 2026.10.19. Java remains the patched Corretto 17.0.21
release on the original Java 17 branch. Fresh validation uses `--pull --no-cache`.
