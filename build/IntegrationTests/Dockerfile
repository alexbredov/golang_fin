FROM golang:1.23-alpine3.21

LABEL authors="Alexander"
LABEL organization="OTUS"
LABEL service="integration_tests"

ENV BIN_FILE /opt/antibf/integrationtests
ENV SRC_DIR /go/src/

WORKDIR ${SRC_DIR}

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
ENV CONFIG_FILE /etc/antibf/config.env
COPY ./configs/docker/config.env ${CONFIG_FILE}
CMD go test -v ./integrationtests/ > integrationtestsLog.txt

# FROM alpine:3.21
# LABEL authors="Alexander"
# LABEL organization="OTUS"
# LABEL service="integration_tests"
#
# ENV BIN_FILE "/opt/antibf/integrationtests"
# COPY --from=build ${BIN_FILE} ${BIN_FILE}
#
# ENV CONFIG_FILE /etc/antibf/config.env
# COPY ./configs/docker/config.env ${CONFIG_FILE}
#
# RUN chmod +x ${BIN_FILE}
#
# CMD ${BIN_FILE} -config ${CONFIG_FILE} -test.v > integrationtestsLog.txt
#