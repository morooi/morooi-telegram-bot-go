package main

import (
	"errors"
	"github.com/jmoiron/sqlx"
	log "github.com/sirupsen/logrus"
)

const schema = `
	CREATE TABLE IF NOT EXISTS bwg_api_key (
		pid integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		user_id integer NOT NULL,
		veid text(20) NOT NULL,
		api_key text(50) NOT NULL);
	CREATE UNIQUE INDEX uniq_user_id ON bwg_api_key (user_id ASC);
`

var db *sqlx.DB

type BwgApiKey struct {
	Pid    int64  `db:"pid"`
	UserId int64  `db:"user_id"`
	Veid   string `db:"veid"`
	ApiKey string `db:"api_key"`
}

func InitSqlite() {
	db, _ = sqlx.Open("sqlite3", "telegram.db")

	var name string
	err := db.Get(&name, "SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'bwg_api_key'")
	if err != nil {
		log.Info("初始化数据库...")
		_, _ = db.Exec(schema)
		log.Info("初始化数据库完成")
	}
}

func SelectByUserId(userId int64) (*BwgApiKey, error) {
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

func Insert(bwgApiKey *BwgApiKey) error {
	if bwgApiKey == nil {
		return nil
	}
	_, err := db.Exec("insert into bwg_api_key (user_id, veid, api_key) VALUES (?, ?, ?)", bwgApiKey.UserId, bwgApiKey.Veid, bwgApiKey.ApiKey)
	if err != nil {
		log.Warn("插入数据库错误, err: ", err)
		return err
	}
	return nil
}

func UpdateByUserId(bwgApiKey *BwgApiKey) error {
	if bwgApiKey == nil {
		return nil
	}
	_, err := db.Exec("update bwg_api_key set veid = ?, api_key = ? where user_id = ?", bwgApiKey.Veid, bwgApiKey.ApiKey, bwgApiKey.UserId)
	if err != nil {
		log.Warn("更新数据库错误, err: ", err)
		return err
	}
	return nil
}
