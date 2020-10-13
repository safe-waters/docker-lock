FROM dockerlocktestaccount/golang AS base
COPY . .
FROM base AS final
ADD . .
FROM dockerlocktestaccount/ubuntu:focal
