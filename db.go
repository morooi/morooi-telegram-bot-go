package main

import (
	"database/sql"
	"errors"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

const bwgApiKeySchema = `
	CREATE TABLE IF NOT EXISTS bwg_api_key (
		pid integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		user_id integer NOT NULL,
		veid text(200) NOT NULL,
		api_key text(200) NOT NULL);
	CREATE UNIQUE INDEX uniq_user_id ON bwg_api_key (user_id ASC);
`

const xrayUserStatsSchema = `
	CREATE TABLE IF NOT EXISTS xray_user_stats (
		pid integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		user text(20) NOT NULL,
		date text(30) NOT NULL,
		time text(10) NOT NULL,
		down integer NOT NULL,
		up integer NOT NULL);
`

const xrayLogSchema = `
	CREATE TABLE IF NOT EXISTS xray_log (
		pid integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		user text NOT NULL,
		ip text NOT NULL,
		target text NOT NULL,
		inbound text NOT NULL,
		outbound text NOT NULL,
		timestamp DATETIME NOT NULL);
`

var db *sqlx.DB

type BwgApiKey struct {
	Pid    int64  `db:"pid"`
	UserId int64  `db:"user_id"`
	Veid   string `db:"veid"`
	ApiKey string `db:"api_key"`
}

type XrayUserStats struct {
	Pid  int64  `db:"pid" json:"pid"`
	User string `db:"user" json:"user"`
	Date string `db:"date" json:"date"`
	Time string `db:"time" json:"time"`
	Down int64  `db:"down" json:"down"`
	Up   int64  `db:"up" json:"up"`
}

func InitSqlite() {
	db, _ = sqlx.Open("sqlite", "./telegram.db")

	var name string
	err := db.Get(&name, "SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'bwg_api_key'")
	if err != nil {
		log.Info("初始化搬瓦工 KEY 数据库...")
		_, _ = db.Exec(bwgApiKeySchema)
		log.Info("初始化搬瓦工 KEY 数据库完成")
	}

	err = db.Get(&name, "SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'xray_user_stats'")
	if err != nil {
		log.Info("初始化 Xray 用户信息数据库...")
		_, _ = db.Exec(xrayUserStatsSchema)
		log.Info("初始化 Xray 用户信息数据库完成")
	}

	err = db.Get(&name, "SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'xray_log'")
	if err != nil {
		log.Info("初始化 Xray 日志数据库...")
		_, _ = db.Exec(xrayLogSchema)
		log.Info("初始化 Xray 日志数据库完成")
	}
}

func SelectBwgKeyByUserId(userId int64) (*BwgApiKey, error) {
	if &userId == nil {
		return nil, errors.New("userId 不可为空")
	}
	bwgApiKey := BwgApiKey{}
	err := db.Get(&bwgApiKey, "select pid, user_id, veid, api_key from bwg_api_key where user_id = ?", userId)
	if err != nil {
		return nil, err
	}
	return &bwgApiKey, nil
}

func InsertBwgKey(bwgApiKey *BwgApiKey) error {
	if bwgApiKey == nil {
		return nil
	}
	_, err := db.Exec("insert into bwg_api_key (user_id, veid, api_key) VALUES (?, ?, ?)", bwgApiKey.UserId, bwgApiKey.Veid, bwgApiKey.ApiKey)
	if err != nil {
		log.Warn("插入 bwg_api_key 错误, err: ", err)
		return err
	}
	return nil
}

func UpdateBwgKeyByUserId(bwgApiKey *BwgApiKey) error {
	if bwgApiKey == nil {
		return nil
	}
	_, err := db.Exec("update bwg_api_key set veid = ?, api_key = ? where user_id = ?", bwgApiKey.Veid, bwgApiKey.ApiKey, bwgApiKey.UserId)
	if err != nil {
		log.Warn("更新 bwg_api_key 错误, err: ", err)
		return err
	}
	return nil
}

func SelectXrayUserStatsByDate(date string) (*[]XrayUserStats, error) {
	if len(date) == 0 {
		return nil, errors.New("时间不可为空")
	}
	xrayUserStatsList := make([]XrayUserStats, 0)
	err := db.Select(&xrayUserStatsList, "select pid, user, date, time, down, up from xray_user_stats where date = ?", date)
	if err != nil {
		return nil, err
	}
	return &xrayUserStatsList, nil
}

func InsertXrayUserStats(xrayUserStats *XrayUserStats) error {
	if xrayUserStats == nil {
		return nil
	}

	_, err := db.NamedExec("INSERT INTO xray_user_stats (user, date, time, down, up) VALUES (:user, :date, :time, :down, :up)", xrayUserStats)
	return err
}

func UpdateXrayUserStats(xrayUserStats *XrayUserStats) error {
	if xrayUserStats == nil {
		return nil
	}
	_, err := db.Exec("update xray_user_stats set down = ?, up = ? where user = ? and date = ? and time = ?",
		xrayUserStats.Down, xrayUserStats.Up, xrayUserStats.User, xrayUserStats.Date, xrayUserStats.Time)
	return err
}

func SelectXrayUserStatsByUserAndDateTime(user string, date string, time string) (*XrayUserStats, error) {
	if len(date) == 0 {
		return nil, errors.New("时间不可为空")
	}
	xrayUserStats := &XrayUserStats{}
	err := db.Get(xrayUserStats, "select pid, user, date, time, down, up from xray_user_stats where user = ? and date = ? and time = ?", user, date, time)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return xrayUserStats, nil
}

func InsertXrayLog(xrayLog *XrayLog) error {
	if xrayLog == nil {
		return nil
	}

	insertSQL := `
		INSERT INTO xray_log 
		    (user, ip, target, inbound, outbound, timestamp)
		VALUES 
		    (:user, :ip, :target, :inbound, :outbound, :timestamp)
	`
	_, err := db.NamedExec(insertSQL, xrayLog)
	return err
}
