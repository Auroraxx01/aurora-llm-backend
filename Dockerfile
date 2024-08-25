FROM golang:1.23.0 as builder
LABEL authors="aurora"

ENV PROJECT_PATH $GOPATH/src/aurora-llm
WORKDIR $PROJECT_PATH

COPY . $PROJECT_PATH/
RUN cd $PROJECT_PATH/ && go build -o aurora-llm $PROJECT_PATH/cmd/server.go
EXPOSE 8080
CMD ["./aurora-llm"]