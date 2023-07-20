FROM --platform=$TARGETPLATFORM alpine:latest
MAINTAINER SJ Zhou <morooiu@gmail.com>

WORKDIR /app
COPY asset/bot-$TARGETOS-$TARGETARCH /app/bot
COPY telegram.db /app/telegram.db

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

ENTRYPOINT ["/app/bot"]