package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	statsService "github.com/xtls/xray-core/app/stats/command"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"os"
	"regexp"
	"strconv"
	"time"
)

type XrayApi struct {
	Host string
	Port int
}

type Traffic struct {
	User string `json:"user"`
	Up   int64  `json:"up"`
	Down int64  `json:"down"`
}

var xrayApi *XrayApi

func InitXrayStats() {
	InitXrayApi()
	InitStatsJob()
}

func InitXrayApi() {
	xrayApiHost := os.Getenv("XRAY_API_HOST")
	xrayApiPort := os.Getenv("XRAY_API_PORT")
	if len(xrayApiHost) == 0 {
		xrayApiHost = "127.0.0.1"
	}
	if len(xrayApiPort) == 0 {
		xrayApiPort = "8080"
	}
	xrayApiPortInt, _ := strconv.Atoi(xrayApiPort)
	xrayApi = &XrayApi{Host: xrayApiHost, Port: xrayApiPortInt}
}

func InitStatsJob() {
	c := cron.New()

	_, err := c.AddFunc("*/5 * * * *", func() {
		checkAndUpdateXrayTraffic()
	})
	if err != nil {
		fmt.Println("添加定时任务失败:", err)
		return
	}

	c.Start()
}

func checkAndUpdateXrayTraffic() {
	traffics, err := GetTraffic(true)
	if err != nil {
		log.Error("获取 Xray 流量异常", err)
		return
	}

	if traffics == nil {
		log.Error("获取 Xray 流量为空")
		return
	}

	jsonString, _ := json.Marshal(traffics)
	log.Infof("获取到 Xray 流量: %s", jsonString)

	thisHour := time.Now().Truncate(time.Hour)
	formattedDate := thisHour.Format(DateFormat)
	formattedTime := thisHour.Format(TimeFormat)

	for _, traffic := range traffics {
		if traffic.Up == 0 && traffic.Down == 0 {
			continue
		}
		data, err := SelectXrayUserStatsByUserAndDateTime(traffic.User, formattedDate, formattedTime)
		if err != nil {
			continue
		}
		if data == nil {
			// 新增
			xrayUserStats := &XrayUserStats{
				User: traffic.User,
				Date: formattedDate,
				Time: formattedTime,
				Down: traffic.Down,
				Up:   traffic.Up,
			}
			err := InsertXrayUserStats(xrayUserStats)
			if err != nil {
				log.Error("保存 Xray 流量异常", err)
			}
		} else {
			// 更新
			xrayUserStats := &XrayUserStats{
				User: traffic.User,
				Date: formattedDate,
				Time: formattedTime,
				Down: traffic.Down + data.Down,
				Up:   traffic.Up + data.Up,
			}
			err := UpdateXrayUserStats(xrayUserStats)
			if err != nil {
				log.Error("保存 Xray 流量异常", err)
			}
		}
	}
}

var trafficRegex = regexp.MustCompile("user>>>([^>]+)>>>traffic>>>(downlink|uplink)")

func GetTraffic(reset bool) ([]*Traffic, error) {
	conn, err := grpc.Dial(fmt.Sprintf("%s:%d", xrayApi.Host, xrayApi.Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	client := statsService.NewStatsServiceClient(conn)
	response, err := client.QueryStats(ctx, &statsService.QueryStatsRequest{Reset_: reset})
	if err != nil {
		return nil, err
	}
	userTrafficMap := map[string]*Traffic{}
	traffics := make([]*Traffic, 0)
	for _, stat := range response.GetStat() {
		matches := trafficRegex.FindStringSubmatch(stat.GetName())
		user := matches[1]
		isDown := matches[2] == "downlink"
		traffic, ok := userTrafficMap[user]
		if !ok {
			traffic = &Traffic{
				User: user,
			}
			userTrafficMap[user] = traffic
			traffics = append(traffics, traffic)
		}
		if isDown {
			traffic.Down = stat.GetValue()
		} else {
			traffic.Up = stat.GetValue()
		}
	}
	return traffics, nil
}
