FROM --platform=$BUILDPLATFORM golang:1.27.3-trixie AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod *.go index.html ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /web .

FROM scratch
COPY --from=build /web /web
USER 10001
EXPOSE 8080
ENTRYPOINT ["/web"]
