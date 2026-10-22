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
docker run --rm --entrypoint java "$image" -version
docker run --rm --entrypoint node "$image" --version
docker run --rm --entrypoint /bin/harness-sonar "$image" --help
docker run --rm --entrypoint sonar-scanner "$image" --version
