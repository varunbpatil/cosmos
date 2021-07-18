FROM golang:alpine AS builder
WORKDIR /cosmos-backend
COPY ./cosmos-backend/ .
RUN CGO_ENABLED=0 go build ./cmd/cosmosd/
RUN CGO_ENABLED=0 go build ./cmd/temporald/

FROM alpine:3.14
ADD https://github.com/ufoscout/docker-compose-wait/releases/download/2.9.0/wait /wait
RUN chmod +x /wait
RUN apk update && apk add --no-cache docker-cli tini
COPY --from=builder /cosmos-backend/cosmosd /cosmosd
COPY --from=builder /cosmos-backend/temporald /temporald
COPY ./cosmos-frontend/dist /dist
ENTRYPOINT ["/sbin/tini", "-g", "--"]