## morooi's Telegram Bot

https://t.me/morooi_bot

### 获取 Docker 镜像

```shell
docker pull morooi/morooi-telegram-bot-go:latest
```

### 编辑 docker-compose 

```yaml
services:
  morooi-telegram-bot-go:
    container_name: morooi-telegram-bot-go
    image: morooi/morooi-telegram-bot-go:latest
    volumes:
      - /etc/localtime:/etc/localtime:ro
      - /home/user/telegram.db:/app/telegram.db
    environment:
      TOKEN: "YOUR TELEGRAM BOT API TOKEN"  # <-- 更改成你的 token
      KEY: "YOUR TELEGRAM BOT API KEY"  # <-- 更改成你的 key
      XRAY_API_HOST: "127.0.0.1"  # <-- 更改成你的 Xray API 监听地址
      XRAY_API_PORT: "8080"  # <-- 更改成你的 Xray API 监听端口
      XRAY_STATS_ADMIN: "XXXXXXX"  # <-- 更改成可以查询流量的 Telegram 用户 ID
      XRAY_STATS_CRON: "*/5 * * * *"  # <-- 数据收集的频率
      SERVER_TRAFFIC_CRON: "0 */6 * * *"  # <-- 服务器流量检测的频率
      BWG_VEID: "XXXXXXX"  # <-- 更改成你的搬瓦工 VEID
      BWG_API_KEY: "XXXXXXX"  # <-- 更改成你的搬瓦工 API KEY
      SEND_MESSAGE_CHANNEL_ID: "XXXXXXX"  # <-- 更改成你的 Telegram 频道 ID
    restart: unless-stopped
```

### 运行

```shell
docker compose up -d 
```