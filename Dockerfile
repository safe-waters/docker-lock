FROM alpine AS build
ARG TARGETPLATFORM
WORKDIR build
COPY dist/ dist/
RUN TARGETPLATFORM="${TARGETPLATFORM/\//_}" && \
    mkdir prod && \
    mv "dist/docker-lock_${TARGETPLATFORM}/docker-lock" prod/

FROM scratch AS prod
ARG TARGETPLATFORM
COPY --from=build /build/prod/docker-lock .
ENTRYPOINT ["./docker-lock"]
