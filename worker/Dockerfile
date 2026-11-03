FROM --platform=$BUILDPLATFORM golang:1.27.3-trixie AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod *.go ./
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /worker .

FROM scratch
COPY --from=build /worker /worker
USER 10001
EXPOSE 8080
ENTRYPOINT ["/worker"]
