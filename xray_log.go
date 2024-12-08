package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var XrayServerName string

type RequestTime struct {
	time.Time
}

func (ct RequestTime) MarshalJSON() ([]byte, error) {
	// 定义格式化的时间字符串
	formatted := ct.Format("2006-01-02 15:04:05")
	return json.Marshal(formatted)
}

type XrayLog struct {
	Pid         int64       `db:"pid" json:"-"`
	User        string      `db:"user" json:"user"`
	IP          string      `db:"ip" json:"ip"`
	Target      string      `db:"target" json:"target"`
	Inbound     string      `db:"inbound" json:"inbound"`
	Outbound    string      `db:"outbound" json:"outbound"`
	RequestTime RequestTime `db:"timestamp" json:"request_time"`
	Server      string      `db:"server" json:"server"`
}

func init() {
	logChannel := make(chan XrayLog, 300)
	logFilePath := os.Getenv("XRAY_LOG_PATH")
	if len(logFilePath) == 0 {
		log.Info("未设置 Xray 日志文件路径")
		return
	}

	XrayServerName = os.Getenv("XRAY_SERVER_NAME")

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
		Server:      XrayServerName,
	}

	return entry, true
}
