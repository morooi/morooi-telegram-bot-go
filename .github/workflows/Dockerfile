FROM --platform=$TARGETPLATFORM alpine:latest
MAINTAINER SJ Zhou <morooiu@gmail.com>

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY asset/bot-${TARGETOS}-${TARGETARCH} /app/bot

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone

ENTRYPOINT ["/app/bot"]