FROM --platform=${BUILDPLATFORM:-linux/amd64} harness0.harness.io/oci/docker_artifacts/hhi/go:1.26.10@sha256:718fddf012b3a305e936ae9964e5a642036bcb7b4c6c0786e2f78f238d637db8 AS hhi-go-toolchain-0
FROM --platform=${BUILDPLATFORM:-linux/amd64} harness0.harness.io/oci/docker_artifacts/hhi/harnessos-alpine:1@sha256:290a37c85b4284d2c11fa0778e5da79643d912a1f5a57d6f701b77644503b7e7 AS build
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

FROM amazoncorretto:17-alpine3.22@sha256:a5858cc0468afe691d84dbb2577606fce1f0c29acb267f7e7385737ce6b87409
ENV NODE_OPTIONS="--use-openssl-ca"
ENV REQUESTS_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt PIP_CERT=/etc/ssl/certs/ca-certificates.crt CURL_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt

ARG SONAR_VERSION=5.0.2.4997
ARG SONAR_SCANNER_CLI=sonar-scanner-cli-${SONAR_VERSION}
ARG SONAR_SCANNER=sonar-scanner-${SONAR_VERSION}

# RUN apt-get update \
#     && apt-get install -y nodejs curl \
#     && apt-get clean

RUN apk --no-cache --update add 'ca-certificates>=20261020-r0' java-cacerts 'nodejs>=22.23.4' npm curl unzip git 'php82>=8.2.35' python3 py3-pip maven gcc g++ make ruby ruby-dev perl \
    && ln -s /usr/bin/php82 /usr/bin/php \
    && php -r 'exit(PHP_VERSION_ID >= 80235 && PHP_MAJOR_VERSION === 8 && PHP_MINOR_VERSION === 2 ? 0 : 1);' \
    && python3 -m pip install --no-cache-dir --break-system-packages 'pip==26.2.2' 'certifi==2026.10.19'

# Keep Java's default store and keytool imports on the native CA update path.
RUN update-ca-certificates \
    && ln -sf /etc/ssl/certs/java/cacerts "$JAVA_HOME/lib/security/cacerts"

COPY --from=build /go/src/github.com/diegopereiraeng/harness-cie-sonarqube-scanner/harness-sonar /bin/
WORKDIR /bin

RUN curl --fail --show-error --location https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/${SONAR_SCANNER_CLI}.zip -so /bin/${SONAR_SCANNER_CLI}.zip
RUN unzip ${SONAR_SCANNER_CLI}.zip \
    && rm ${SONAR_SCANNER_CLI}.zip 

ENV PATH $PATH:/bin/${SONAR_SCANNER}/bin

ENTRYPOINT /bin/harness-sonar
