FROM golang:alpine as builder
WORKDIR /home
# 利用build指定上下文，且COPY和ADD只能拷贝上下文内的文件到容器中
COPY . .
RUN go build --mod=vendor -o chatapp ./cmd/root.go

FROM alpine as runner
ENV WORKDIR=/home
WORKDIR $WORKDIR

RUN apk add tzdata --no-cache \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone \
    && apk del tzdata
RUN mkdir -p $WORKDIR/static \
    && mkdir -p $WORKDIR/config \
    && mkdir -p $WORKDIR/cmd
COPY --from=builder $WORKDIR/chatapp $WORKDIR
COPY --from=builder $WORKDIR/config $WORKDIR/config
COPY --from=builder $WORKDIR/cmd $WORKDIR/cmd

# 需要有空格
CMD ["./chatapp"]