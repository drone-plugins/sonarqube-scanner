#!/bin/sh
set -eu

image=${1:?Usage: smoke-image.sh IMAGE}

docker run --rm --network none --entrypoint php "$image" -r '
    if (PHP_MAJOR_VERSION !== 8 || PHP_MINOR_VERSION !== 2 || PHP_VERSION_ID < 80235) {
        fwrite(STDERR, "Expected patched PHP 8.2.35 or later on the 8.2 branch.\n");
        exit(1);
    }
    echo PHP_VERSION, "\n";
'
java_version=$(docker run --rm --network none --entrypoint java "$image" -version 2>&1)
printf '%s\n' "$java_version"
printf '%s\n' "$java_version" | grep 'version "17[.]0[.]21'

# Emulation can hide a mismatched Go binary, so inspect its ELF target as well.
architecture=$(docker image inspect --format '{{.Architecture}}' "$image")
case "$architecture" in
    amd64) machine='Advanced Micro Devices X86-64' ;;
    arm64) machine='AArch64' ;;
    *) echo "Unsupported smoke-test architecture: $architecture" >&2; exit 1 ;;
esac
elf_header=$(docker run --rm --network none --entrypoint readelf "$image" -h /bin/harness-sonar)
printf '%s\n' "$elf_header" | grep "Machine:.*$machine"
docker run --rm --network none --entrypoint node "$image" -e '
    const [major, minor, patch] = process.versions.node.split(".").map(Number);
    if (major !== 22 || minor < 23 || (minor === 23 && patch < 4)) process.exit(1);
    console.log(process.versions.node);
'
docker run --rm --network none --entrypoint python3 "$image" -c '
import certifi, pip
assert pip.__version__ == "26.2.2"
assert certifi.__version__ == "2026.10.19"
print("pip", pip.__version__, "certifi", certifi.__version__)
'
docker run --rm --network none --entrypoint /bin/harness-sonar "$image" --help
docker run --rm --network none --entrypoint sonar-scanner "$image" --version
