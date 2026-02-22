ARG BUILDPLATFORM

FROM --platform=$BUILDPLATFORM golang:1.25.5 AS build

WORKDIR /app
ARG TARGETOS
ARG TARGETARCH
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /app/server .

FROM alpine:3.20
WORKDIR /app
COPY --from=build /app/server ./server
EXPOSE 8080
CMD ["./server"]
