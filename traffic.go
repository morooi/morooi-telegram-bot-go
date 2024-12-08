package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/robfig/cron/v3"
	log "github.com/sirupsen/logrus"
	statsService "github.com/xtls/xray-core/app/stats/command"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
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
	cronStr := os.Getenv("XRAY_STATS_CRON")
	if len(cronStr) == 0 {
		cronStr = "*/5 * * * *"
	}

	_, err := c.AddFunc(cronStr, func() {
		CheckAndUpdateXrayTraffic()
	})
	if err != nil {
		fmt.Println("添加定时任务失败:", err)
		return
	}

	c.Start()
}

func InitXrayLog() {
	logChannel := make(chan XrayLog, 300)
	logFilePath := os.Getenv("XRAY_LOG_PATH")
	if len(logFilePath) == 0 {
		logFilePath = "/var/log/xray/access.log"
	}

	// 启动日志文件监听的 goroutine
	go watchXrayLogFile(logFilePath, logChannel)

	// 启动日志处理 goroutine
	go saveXrayLogEntries(logChannel)
}

func watchXrayLogFile(logFilePath string, logChannel chan XrayLog) {
	file, err := os.Open(logFilePath)
	if err != nil {
		log.Fatalf("无法打开日志文件: %v", err)
	}
	defer file.Close()

	// 定位到文件末尾，模拟 `tail -f`
	_, _ = file.Seek(0, io.SeekEnd)
	reader := bufio.NewReader(file)

	for {
		// 每次读取一行
		line, err := reader.ReadString('\n')
		if err != nil {
			// EOF 时暂停读取一段时间，等待新日志写入
			time.Sleep(1 * time.Second)
			continue
		}

		// 解析日志行
		entry, ok := parseXrayLogEntry(line)
		if ok {
			// 将解析结果发送到 logChannel
			logChannel <- entry
		}
	}
}

func saveXrayLogEntries(logChannel chan XrayLog) {
	var entries []XrayLog
	count := 0
	for entry := range logChannel {
		// 数据插入数据库
		_ = InsertXrayLog(&entry)
		entries = append(entries, entry)
		count++
		if count == 10 {
			// 保存到 CF D1 数据库
			saveToCloudFlareD1(entries)
			entries = []XrayLog{}
			count = 0
		}
	}
}

func saveToCloudFlareD1(entries []XrayLog) {
	records := map[string]interface{}{
		"records": entries,
	}
	jsonData, err := json.Marshal(records)
	if err != nil {
		log.Error("JSON 序列化错误:", err)
		return
	}

	url := os.Getenv("CF_D1_INSERT_URL")
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Error("创建 CF D1 请求错误:", err)
		return
	}

	token := os.Getenv("CF_D1_REQUEST_TOKEN")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Error("发送 CF D1 请求错误:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Errorf("CF D1 请求失败，状态码: %d，返回内容: %s", resp.StatusCode, string(body))
	}
}

func parseXrayLogEntry(line string) (XrayLog, bool) {
	re := regexp.MustCompile(`(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2}) from (?:tcp:|udp:)?\[?([0-9a-fA-F:]+|\d+\.\d+\.\d+\.\d+)]?(?::\d+)? accepted (?:tcp:|udp:)?([\w.-]+)(?::\d+)? \[(.+?) [->]+ (.+?)] email: (.+)`)
	match := re.FindStringSubmatch(line)

	if match == nil {
		return XrayLog{}, false
	}

	// 如果匹配不完整，跳过
	if len(match) < 7 {
		return XrayLog{}, false
	}

	ip := match[2]
	if ip == "127.0.0.1" || ip == "1.1.1.1" || ip == "8.8.8.8" {
		return XrayLog{}, false
	}

	requestTime, err := time.Parse("2006/01/02 15:04:05", match[1])
	if err != nil {
		log.Println("时间解析错误:", err)
		return XrayLog{}, false
	}

	user := strings.Split(match[6], "-")[0]

	entry := XrayLog{
		User:        user,
		IP:          ip,
		Target:      match[3],
		Inbound:     match[4],
		Outbound:    match[5],
		RequestTime: RequestTime{requestTime},
	}

	return entry, true
}

func CheckAndUpdateXrayTraffic() {
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

	thisHour := time.Now().Add(-time.Minute).Truncate(time.Hour)
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
	conn, err := grpc.NewClient(fmt.Sprintf("%s:%d", xrayApi.Host, xrayApi.Port), grpc.WithTransportCredentials(insecure.NewCredentials()))
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
