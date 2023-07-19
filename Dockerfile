FROM alpine:latest
MAINTAINER SJ Zhou <morooiu@gmail.com>

WORKDIR /app
COPY morooi-telegram-bot-go /app/morooi-telegram-bot-go

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

ENTRYPOINT ["/app/morooi-telegram-bot-go"]