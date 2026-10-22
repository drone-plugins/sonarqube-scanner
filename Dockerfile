# BuildKit cross-compiles natively; older builders retain the existing amd64 target.
FROM --platform=${BUILDPLATFORM:-linux/amd64} golang:1.26-alpine AS build
ARG TARGETOS
ARG TARGETARCH
RUN apk add --no-cache --update git
RUN mkdir -p /go/src/github.com/diegopereiraeng/harness-cie-sonarqube-scanner
WORKDIR /go/src/github.com/diegopereiraeng/harness-cie-sonarqube-scanner 
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} CGO_ENABLED=0 go build -mod=readonly -o harness-sonar

FROM amazoncorretto:17-alpine3.22

ARG SONAR_VERSION=5.0.2.4997
ARG SONAR_SCANNER_CLI=sonar-scanner-cli-${SONAR_VERSION}
ARG SONAR_SCANNER=sonar-scanner-${SONAR_VERSION}

# RUN apt-get update \
#     && apt-get install -y nodejs curl \
#     && apt-get clean

RUN apk --no-cache --update add nodejs-current npm curl unzip git 'php82>=8.2.35' python3 py3-pip maven gcc g++ make ruby ruby-dev perl \
    && ln -s /usr/bin/php82 /usr/bin/php \
    && php -r 'exit(PHP_VERSION_ID >= 80235 && PHP_MAJOR_VERSION === 8 && PHP_MINOR_VERSION === 2 ? 0 : 1);'

COPY --from=build /go/src/github.com/diegopereiraeng/harness-cie-sonarqube-scanner/harness-sonar /bin/
WORKDIR /bin

RUN curl --fail --show-error --location https://binaries.sonarsource.com/Distribution/sonar-scanner-cli/${SONAR_SCANNER_CLI}.zip -so /bin/${SONAR_SCANNER_CLI}.zip
RUN unzip ${SONAR_SCANNER_CLI}.zip \
    && rm ${SONAR_SCANNER_CLI}.zip 

ENV PATH $PATH:/bin/${SONAR_SCANNER}/bin

ENTRYPOINT /bin/harness-sonar
