FROM golang:1.24.0-alpine
COPY . /src
WORKDIR /src
RUN echo $GOPATH
ENV CGO_ENABLED=0 GOPROXY="https://goproxy.cn,direct"
RUN go mod tidy
RUN go build -o goBuild
CMD ["/src/goBuild", "--env", "dev"]