#!/bin/sh
set -eu

image=${1:?Usage: smoke-image.sh IMAGE}

docker run --rm --entrypoint php "$image" -r '
    if (PHP_MAJOR_VERSION !== 8 || PHP_MINOR_VERSION !== 2 || PHP_VERSION_ID < 80235) {
        fwrite(STDERR, "Expected patched PHP 8.2.35 or later on the 8.2 branch.\n");
        exit(1);
    }
    echo PHP_VERSION, "\n";
'
java_version=$(docker run --rm --entrypoint java "$image" -version 2>&1)
printf '%s\n' "$java_version"
printf '%s\n' "$java_version" | grep 'version "17[.]'

# Emulation can hide a mismatched Go binary, so inspect its ELF target as well.
architecture=$(docker image inspect --format '{{.Architecture}}' "$image")
case "$architecture" in
    amd64) machine='Advanced Micro Devices X86-64' ;;
    arm64) machine='AArch64' ;;
    *) echo "Unsupported smoke-test architecture: $architecture" >&2; exit 1 ;;
esac
elf_header=$(docker run --rm --entrypoint readelf "$image" -h /bin/harness-sonar)
printf '%s\n' "$elf_header" | grep "Machine:.*$machine"
docker run --rm --entrypoint node "$image" --version
docker run --rm --entrypoint /bin/harness-sonar "$image" --help
docker run --rm --entrypoint sonar-scanner "$image" --version
