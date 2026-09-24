# syntax=docker/dockerfile:1

# ---------- build stage ----------
FROM golang:1.26.5 AS build

WORKDIR /src

# Copy the module files alone, then download. An edit to a .go file leaves this
# layer untouched, so the download runs one time and not once per build.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

# CGO_ENABLED=0 makes one static binary. With CGO on, the net and os/user
# packages link against libc, and a base image with no libc cannot start it.
# -trimpath drops the build machine paths. -s -w drop the symbol table and
# the DWARF data, which is about 4 MB here.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

# ---------- run stage ----------
# distroless static holds the CA certificates, the timezone data and an
# /etc/passwd entry. It holds no shell and no package manager, so an attacker
# who reaches the container finds no tools. static-debian13 also exists.
FROM gcr.io/distroless/static-debian12:nonroot

# The page and the migrations already sit inside the binary, from go:embed.
COPY --from=build /server /server

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]
