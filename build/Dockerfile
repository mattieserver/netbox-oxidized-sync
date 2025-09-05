FROM golang:1.22 AS build

WORKDIR /app
COPY ../ ./
RUN CGO_ENABLED=0 GOOS=linux  go build cmd/netbox-oxidized-sync/netbox-oxidized-sync.go

FROM alpine:3.22.1

RUN apk --no-cache add tzdata

COPY --from=build /app/netbox-oxidized-sync /opt/netbox-oxidized-sync
RUN echo '0  1  *  *  *    /opt/netbox-oxidized-sync' > /etc/crontabs/root

CMD ["crond", "-f"]
