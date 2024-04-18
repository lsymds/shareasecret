FROM debian:latest

ARG TARGETOS
ARG TARGETARCH

VOLUME [ "/data" ]

ENV UID=1000
ENV GID=1000
ENV SHAREASECRET_DB_PATH=/data/shareasecret.db
ENV SHAREASECRET_LISTENING_ADDR=0.0.0.0:8994

USER ${UID}:${GID}

COPY ./build/tmp/shareasecret-${TARGETOS}-${TARGETARCH}-* /shareasecret

CMD [ "/shareasecret" ]
