package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	tele "gopkg.in/telebot.v3"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Command = string
type CommandHandler struct {
	command Command
	handler tele.HandlerFunc
}

const (
	Info           Command = "/info"
	Start          Command = "/start"
	BwgBind        Command = "/bwg_bind"
	BwgInfo        Command = "/bwg_info"
	QueryXrayStats Command = "/xray_stats"
)

var commandHandlers map[Command]CommandHandler

func InitCommandHandler() {
	commandHandlers = make(map[Command]CommandHandler, 5)

	commandHandlers[Start] = CommandHandler{Start, StartHandler}
	commandHandlers[Info] = CommandHandler{Info, InfoHandler}
	commandHandlers[BwgBind] = CommandHandler{BwgBind, BwgBindHandler}
	commandHandlers[BwgInfo] = CommandHandler{BwgInfo, BwgInfoHandler}
	commandHandlers[QueryXrayStats] = CommandHandler{QueryXrayStats, QueryXrayStatsHandler}
}

func StartHandler(c tele.Context) error {
	message := c.Message()
	firstName := message.Sender.FirstName
	lastName := message.Sender.LastName

	reply := fmt.Sprintf("%s %s，欢迎使用 morooi's Bot", firstName, lastName)
	return c.Send(reply)
}

func InfoHandler(c tele.Context) error {
	message := c.Message()
	id := message.Sender.ID
	firstName := message.Sender.FirstName
	lastName := message.Sender.LastName

	reply := fmt.Sprintf("*INFO*\nfirstName: %s\nlastName: %s\nuserId: %d", firstName, lastName, id)
	return c.Send(reply)
}

func BwgBindHandler(c tele.Context) error {
	message := c.Message()
	args := c.Args()
	if len(args) != 2 {
		return c.Send("请在命令后指定您的 VEID 和 API KEY，用空格分隔\n如：`/bwg_bind VEID API_KEY`")
	}

	userId := message.Sender.ID

	encryptedVeid, _ := encryptString(args[0])
	encryptedApiKey, _ := encryptString(args[1])
	bwgApiKey := &BwgApiKey{UserId: userId, Veid: encryptedVeid, ApiKey: encryptedApiKey}

	_, err := SelectBwgKeyByUserId(userId)
	if err != nil {
		insertErr := InsertBwgKey(bwgApiKey)
		if insertErr != nil {
			return c.Send("*绑定失败*！\n请稍后再试")
		}
	} else {
		updateErr := UpdateBwgKeyByUserId(bwgApiKey)
		if updateErr != nil {
			return c.Send("*绑定失败*！\n请稍后再试")
		}
	}

	return c.Send("*绑定成功*！\n请使用 /bwg\\_info 命令获取信息")
}

func BwgInfoHandler(c tele.Context) error {
	message := c.Message()
	userId := message.Sender.ID

	bwgApiKey, err := SelectBwgKeyByUserId(userId)
	if err != nil {
		return c.Send("请先使用 /bwg\\_bind 命令绑定 VEID 和 API KEY")
	}

	veid, _ := decryptString(bwgApiKey.Veid)
	apiKey, _ := decryptString(bwgApiKey.ApiKey)

	info, err := GetBwgServerInfo(veid, apiKey)
	if err != nil || info == nil || info.Error != 0 {
		return c.Send("获取服务器信息失败，请确认 VEID 和 API KEY 是否正确\n确认后重新使用 /bwg\\_bind 命令更新信息")
	}

	reply := buildServerInfoMessage(info)

	return c.Send(reply)
}

func buildServerInfoMessage(info *BwgServerInfo) string {
	hostname := ReplaceForMarkdownV2(info.HostName)
	ipAddresses := ReplaceForMarkdownV2(strings.Join(info.IpAddresses, ", "))
	nodeDatacenter := ReplaceForMarkdownV2(info.NodeDataCenter)
	dataCounter := ReplaceForMarkdownV2(decimal.NewFromInt(info.DataCounter >> 20).Div(decimal.NewFromInt(1024)).Round(2).String())
	planMonthlyData := ReplaceForMarkdownV2(strconv.FormatInt(info.PlanMonthlyData>>30, 10))
	dataPercent := ReplaceForMarkdownV2(decimal.NewFromInt(info.DataCounter).Mul(decimal.NewFromInt(100)).Div(decimal.NewFromInt(info.PlanMonthlyData)).Round(2).String())
	nextDateResetDate := time.Unix(info.DataNextReset, 0)
	dataNextReset := ReplaceForMarkdownV2(nextDateResetDate.Format(DateTimeFormat))
	duration := GetDuration(nextDateResetDate)

	reply := fmt.Sprintf("*主机名*：%s\n*IP*：`%s`\n*数据中心*：%s\n*流量使用情况*：%s GB / %s GB \\(%s %%\\)\n*流量重置时间*：%s\n*距离重置还有*：%s",
		hostname, ipAddresses, nodeDatacenter, dataCounter, planMonthlyData, dataPercent, dataNextReset, duration)
	return reply
}

func QueryXrayStatsHandler(c tele.Context) error {
	// 判断查询权限
	userId := c.Message().Sender.ID
	xrayStatsAdmin := os.Getenv("XRAY_STATS_ADMIN")
	if !strings.Contains(xrayStatsAdmin, strconv.FormatInt(userId, 10)) {
		return c.Send("无查询权限")
	}

	var date string
	args := c.Args()
	if len(args) > 0 {
		parsedDate, err := time.Parse("20060102", args[0])
		if err != nil {
			log.Error("日期解析错误: ", err)
			return c.Send("获取流量情况失败")
		}
		date = parsedDate.Format(DateFormat)
	}

	if len(date) == 0 {
		date = time.Now().Format(DateFormat)
	}

	xrayUserStatsList, err := SelectXrayUserStatsByDate(date)
	if err != nil {
		log.Error("获取流量情况失败", err)
		return c.Send("获取流量情况失败")
	}

	isToday := time.Now().Format(DateFormat) == date
	if len(*xrayUserStatsList) == 0 && !isToday {
		return c.Send(ReplaceForMarkdownV2(fmt.Sprintf("%s 流量信息为空", date)))
	}

	userTrafficList := make([]*Traffic, 0)
	userTrafficMap := map[string]*Traffic{}
	for _, xrayUserStats := range *xrayUserStatsList {
		userTraffic, ok := userTrafficMap[xrayUserStats.User]
		if !ok {
			userTraffic = &Traffic{
				User: xrayUserStats.User,
				Down: xrayUserStats.Down,
				Up:   xrayUserStats.Up,
			}
			userTrafficMap[xrayUserStats.User] = userTraffic
			userTrafficList = append(userTrafficList, userTraffic)
		} else {
			userTraffic.Down = userTraffic.Down + xrayUserStats.Down
			userTraffic.Up = userTraffic.Up + xrayUserStats.Up
		}
	}

	// 如果是当天 统计还未落库的数据
	if isToday {
		trafficList, _ := GetTraffic(false)
		for _, traffic := range trafficList {
			userTraffic, ok := userTrafficMap[traffic.User]
			if !ok {
				userTraffic = &Traffic{
					User: traffic.User,
					Down: traffic.Down,
					Up:   traffic.Up,
				}
				userTrafficMap[traffic.User] = userTraffic
				userTrafficList = append(userTrafficList, userTraffic)
			} else {
				userTraffic.Down = userTraffic.Down + traffic.Down
				userTraffic.Up = userTraffic.Up + traffic.Up
			}
		}
	}

	// 排序
	sort.Slice(userTrafficList, func(i, j int) bool {
		iTotal := userTrafficList[i].Down + userTrafficList[i].Up
		jTotal := userTrafficList[j].Down + userTrafficList[j].Up
		return iTotal > jTotal
	})

	var total int64 = 0
	msgSlice := make([]string, 0)
	msgSlice = append(msgSlice, fmt.Sprintf("*%s 流量使用情况*", ReplaceForMarkdownV2(date)))
	for _, traffic := range userTrafficList {
		userTotal := traffic.Up + traffic.Down
		total = total + userTotal
		if userTotal == 0 {
			continue
		}
		trafficInfo := fmt.Sprintf("*%s*：%s", ReplaceForMarkdownV2(traffic.User), ReplaceForMarkdownV2(calculateTraffic(userTotal)))
		msgSlice = append(msgSlice, trafficInfo)
	}
	msgSlice = append(msgSlice, fmt.Sprintf("*总流量*：%s", ReplaceForMarkdownV2(calculateTraffic(total))))

	return c.Send(strings.Join(msgSlice, "\n"))
}

func TextHandler(c tele.Context) error {
	jsonMessage, _ := json.Marshal(c.Message())
	log.Infof("收到请求：%s", jsonMessage)

	if IsCommand(c.Message()) {
		return c.Send(fmt.Sprint("未知的命令：", c.Text()))
	} else {
		return c.Send("只支持输入命令")
	}
}

type BwgServerInfo struct {
	HostName        string   `json:"hostname"`
	NodeDataCenter  string   `json:"node_datacenter"`
	IpAddresses     []string `json:"ip_addresses"`
	PlanMonthlyData int64    `json:"plan_monthly_data"`
	DataCounter     int64    `json:"data_counter"`
	DataNextReset   int64    `json:"data_next_reset"`
	Error           int      `json:"error"`
}

func GetBwgServerInfo(veid string, apiKey string) (*BwgServerInfo, error) {
	if len(veid) == 0 || len(apiKey) == 0 {
		return nil, errors.New("veid 或 apiKey 不可为空")
	}

	resp, err := http.Get(fmt.Sprintf("https://api.64clouds.com/v1/getServiceInfo?veid=%s&api_key=%s", veid, apiKey))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var bwgServerInfo BwgServerInfo
	err = json.Unmarshal(body, &bwgServerInfo)
	if err != nil {
		return nil, err
	}

	return &bwgServerInfo, nil
}
