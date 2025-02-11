# Build application from source
ARG PKG
ARG VERSION
ARG BUILD_TIME

FROM golang:1.23-alpine3.21 AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . ./

RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags "-w -X ${PKG}.version=${VERSION} -X ${PKG}.buildTime=${BUILD_TIME}" -o tests/links

# Run unit tests
FROM build-stage AS run-unit-tests-stage
RUN go test -v ./internal

# Run e2e tests
FROM build-stage AS run-e2e-tests-stage
RUN go test -v ./tests

# Deploy application binary into a lean image
FROM gcr.io/distroless/base-debian12 AS build-release-stage

WORKDIR /

COPY --from=build-stage /app/tests/links /links

USER nonroot:nonroot

ENTRYPOINT ["/links"]