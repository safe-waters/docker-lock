FROM alpine AS build
ARG TARGETPLATFORM
WORKDIR build
COPY dist/ dist/
RUN apk --no-cache add ca-certificates && \
    TARGETPLATFORM="${TARGETPLATFORM//\//_}" && \
    TARGETPLATFORM="${TARGETPLATFORM//v/}" && \
    mkdir prod && \
    mv "dist/docker-lock_${TARGETPLATFORM}/docker-lock" prod/

FROM scratch AS prod
ARG TARGETPLATFORM
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /build/prod /prod
WORKDIR /run
ENTRYPOINT ["/prod/docker-lock"]
