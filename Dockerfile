FROM golang:alpine AS builder

ENV USER=vnc \
    GROUP=vnc \
    UID=${UID:-1000} \
    GID=${GID:-1000}

WORKDIR $GOPATH/src/vncrecorder

RUN apk update > /dev/null 2>&1 && \
    apk add git --no-cache ca-certificates tzdata > /dev/null 2>&1 && \
    update-ca-certificates

RUN addgroup "${GROUP}" --gid "${GID}"

RUN adduser \
    -D \
    -g "" \
    -h "/nonexistent" \
    -s "/sbin/nologin" \
    -H \
    -G "${GROUP}" \
    -u "${UID}" \
    "${USER}"


RUN mkdir -p /recordings && \
    chown -R ${USER}:${GROUP} /recordings

COPY . .

RUN go mod download > /dev/null 2>&1

RUN go mod verify > /dev/null 2>&1

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /go/bin/vncrecorder > /dev/null 2>&1

FROM jrottenberg/ffmpeg:4.1-alpine

ENV UID=${UID:-1000} \
    GID=${GID:-1000} \
    LOG_FILE=/recordings/vnc.log

COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /recordings /recordings
COPY --from=builder /go/bin/vncrecorder .

USER ${UID}:${GID}

VOLUME /recordings

ENTRYPOINT ["/vncrecorder"]

CMD [""]
