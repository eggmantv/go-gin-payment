# STAGE 1
FROM golang:1.22.1 AS build_host
# compile
RUN mkdir -p /app/go-gin-payment
WORKDIR /app/go-gin-payment
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -mod=vendor -o go-gin-payment -v cmd/main.go
RUN sh build_cmds.sh

# STAGE 2
# deploy
FROM golang:1.22.1
ARG APP_ROOT
RUN echo $APP_ROOT
RUN mkdir -p $APP_ROOT/static
WORKDIR $APP_ROOT
COPY --from=build_host /app/go-gin-payment/static ./static
COPY --from=build_host /app/go-gin-payment/go-gin-payment .
COPY --from=build_host /app/go-gin-payment/cmd-* .

EXPOSE 5011

ENV IS_IN_DOCKER=1
ENV _IS_CHILD=1
CMD ["./go-gin-payment", "-e", "production"]
