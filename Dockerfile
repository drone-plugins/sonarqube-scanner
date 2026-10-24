FROM harness0.harness.io/oci/docker_artifacts/hhi/harnessos-alpine:1 AS hhi-ca-certificates
RUN apk add --no-cache 'ca-certificates>=20261020-r0'

FROM --platform=${BUILDPLATFORM:-linux/amd64} harness0.harness.io/oci/docker_artifacts/hhi/go:1.26.10 AS hhi-go-toolchain-0
FROM --platform=${BUILDPLATFORM:-linux/amd64} harness0.harness.io/oci/docker_artifacts/hhi/harnessos-alpine:1 AS build
ARG TARGETOS
ARG TARGETARCH
COPY --from=hhi-go-toolchain-0 /usr/lib/go /usr/lib/go
ENV PATH=/usr/lib/go/bin:$PATH GOPATH=/go GOTOOLCHAIN=local
RUN apk add --no-cache --update git
RUN mkdir -p /go/src/github.com/diegopereiraeng/harness-cie-sonarqube-scanner
WORKDIR /go/src/github.com/diegopereiraeng/harness-cie-sonarqube-scanner 
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} CGO_ENABLED=0 go build -mod=readonly -o harness-sonar

FROM amazoncorretto:17-alpine3.22
COPY --from=hhi-ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/hhi/certs/ca-certificates.crt
ENV SSL_CERT_FILE=/etc/hhi/certs/ca-certificates.crt SSL_CERT_DIR=/etc/hhi/certs

# Java and the scanner use the same patched roots as the Go wrapper.
RUN awk 'BEGIN {n=0} /-----BEGIN CERTIFICATE-----/ {n++; file="/tmp/hhi-root-" n ".crt"} file {print > file} /-----END CERTIFICATE-----/ {close(file); file=""}' /etc/hhi/certs/ca-certificates.crt \
    && for cert in /tmp/hhi-root-*.crt; do \
        keytool -importcert -trustcacerts -noprompt -alias "$(basename "$cert" .crt)" \
            -file "$cert" -keystore /etc/hhi/cacerts -storepass changeit || exit 1; \
    done \
    && rm /tmp/hhi-root-*.crt
ENV JAVA_TOOL_OPTIONS="-Djavax.net.ssl.trustStore=/etc/hhi/cacerts -Djavax.net.ssl.trustStorePassword=changeit"


ENV REQUESTS_CA_BUNDLE=/etc/hhi/certs/ca-certificates.crt PIP_CERT=/etc/hhi/certs/ca-certificates.crt CURL_CA_BUNDLE=/etc/hhi/certs/ca-certificates.crt

ARG SONAR_VERSION=5.0.2.4997
ARG SONAR_SCANNER_CLI=sonar-scanner-cli-${SONAR_VERSION}
ARG SONAR_SCANNER=sonar-scanner-${SONAR_VERSION}

# RUN apt-get update \
#     && apt-get install -y nodejs curl \
#     && apt-get clean

RUN apk --no-cache --update add nodejs-current npm curl unzip git 'php82>=8.2.35' python3 py3-pip maven gcc g++ make ruby ruby-dev perl \
    && ln -s /usr/bin/php82 /usr/bin/php \
    && php -r 'exit(PHP_VERSION_ID >= 80235 && PHP_MAJOR_VERSION === 8 && PHP_MINOR_VERSION === 2 ? 0 : 1);'

# PHP curl also loads the system certificate directory.
RUN apk add --no-cache ca-certificates \
    && sed -i -e 's|^mozilla/emSign_ECC_Root_CA_-_G3.crt|!&|' \
    -e 's|^mozilla/Comodo_AAA_Services_root.crt|!&|' \
    -e 's|^mozilla/emSign_Root_TLS_CA_-_G3.crt|!&|' \
    -e 's|^mozilla/emSign_Root_CS_CA_-_G3.crt|!&|' /etc/ca-certificates.conf \
    && update-ca-certificates

COPY --from=build /go/src/github.com/diegopereiraeng/harness-cie-sonarqube-scanner/harness-sonar /bin/
WORKDIR /bin

RUN curl --fail --show-error --location https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/${SONAR_SCANNER_CLI}.zip -so /bin/${SONAR_SCANNER_CLI}.zip
RUN unzip ${SONAR_SCANNER_CLI}.zip \
    && rm ${SONAR_SCANNER_CLI}.zip 

ENV PATH $PATH:/bin/${SONAR_SCANNER}/bin

ENTRYPOINT /bin/harness-sonar

# Supply the incident-patched system roots from the HHI package.
COPY --from=hhi-ca-certificates /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
