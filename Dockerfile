FROM golang:1.20-alpine3.17

ARG GOBUILD_ARGS=""

WORKDIR /work

COPY go.* ./
RUN go mod download

COPY . ./

RUN VERSION=$(cat ./VERSION) && \
  go build  ${GOBUILD_ARGS} \
  -ldflags "-X 'github.com/findy-network/findy-agent-vault/utils.Version=$VERSION'"\
  -o /go/bin/findy-agent-vault

FROM alpine:3.17

LABEL org.opencontainers.image.source https://github.com/findy-network/findy-agent-vault

EXPOSE 8085

# used when running instrumented binary
ENV GOCOVERDIR /coverage

# override when running
ENV FAV_JWT_KEY "mySuperSecretKeyLol"
ENV FAV_DB_HOST "vault-db"
ENV FAV_DB_PASSWORD "my-secret-password"
ENV FAV_AGENCY_HOST "localhost"
ENV FAV_AGENCY_PORT "50051"
ENV FAV_AGENCY_CERT_PATH "/grpc"
ENV FAV_AGENCY_ADMIN_ID "findy-root"
ENV FAV_AGENCY_INSECURE "false"

COPY --from=0 /work/db/migrations /db/migrations
COPY --from=0 /go/bin/findy-agent-vault /findy-agent-vault

# keep this for now
# if previous docker-compose-files still refer to the start script
RUN echo '#!/bin/sh' > /start.sh && \
  echo '/findy-agent-vault' >> /start.sh && chmod a+x /start.sh

ENTRYPOINT ["/findy-agent-vault"]
