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
    restart: unless-stopped
```

### 运行

```shell
docker compose up -d 
```