package data

import (
	"strings"

	"go-stock/backend/logger"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/go-resty/resty/v2"
)

// @Author spark
// @Date 2025/1/3 13:53
// @Desc
//-----------------------------------------------------------------------------------

type DingDingAPI struct {
	client *resty.Client
}

func NewDingDingAPI() *DingDingAPI {
	return &DingDingAPI{
		client: SharedHTTPClient,
	}
}

func (DingDingAPI) SendDingDingMessage(message string) string {
	if GetSettingConfig().DingPushEnable == false {
		//logger.SugaredLogger.Info("钉钉推送未开启")
		return "钉钉推送未开启"
	}
	// 发送钉钉消息
	resp, err := SharedHTTPClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(message).
		Post(getApiURL())
	if err != nil {
		logger.SugaredLogger.Error(err.Error())
		return "发送钉钉消息失败"
	}
	logger.SugaredLogger.Infof("send dingding message: %s", resp.String())
	return "发送钉钉消息成功"
}

// SendDingDingMessageWithConfig 使用指定配置发送钉钉消息（Web端用户隔离）
func (DingDingAPI) SendDingDingMessageWithConfig(message string, enabled bool, robotURL string) string {
	if !enabled {
		return "钉钉推送未开启"
	}
	if strings.TrimSpace(robotURL) == "" {
		return "钉钉推送未配置机器人地址"
	}
	resp, err := SharedHTTPClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(message).
		Post(robotURL)
	if err != nil {
		logger.SugaredLogger.Error(err.Error())
		return "发送钉钉消息失败"
	}
	logger.SugaredLogger.Infof("send dingding message: %s", resp.String())
	return "发送钉钉消息成功"
}

func getApiURL() string {
	return GetSettingConfig().DingRobot
}

func (DingDingAPI) SendToDingDing(title, message string) string {
	message = strutil.ReplaceWithMap(message, map[string]string{
		"\\n":   "\n",
		"\\r":   "\r",
		"\\t":   "\t",
		"\\\\n": "\n",
		"\\\\r": "\r",
		"\\\\t": "\t",
	})
	// 发送钉钉消息
	resp, err := SharedHTTPClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(&Message{
			Msgtype: "markdown",
			Markdown: Markdown{
				Title: "go-stock " + title,
				Text:  message,
			},
			At: At{
				IsAtAll: true,
			},
		}).
		Post(getApiURL())
	if err != nil {
		logger.SugaredLogger.Error(err.Error())
		return "发送钉钉消息失败"
	}
	logger.SugaredLogger.Infof("send dingding message: %s", resp.String())
	return "发送钉钉消息成功"
}

// SendToDingDingWithConfig 使用指定配置发送钉钉消息（Web端用户隔离）
func (DingDingAPI) SendToDingDingWithConfig(title, message string, enabled bool, robotURL string) string {
	if !enabled {
		return "钉钉推送未开启"
	}
	if strings.TrimSpace(robotURL) == "" {
		return "钉钉推送未配置机器人地址"
	}
	message = strutil.ReplaceWithMap(message, map[string]string{
		"\\n":   "\n",
		"\\r":   "\r",
		"\\t":   "\t",
		"\\\\n": "\n",
		"\\\\r": "\r",
		"\\\\t": "\t",
	})
	resp, err := SharedHTTPClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(&Message{
			Msgtype: "markdown",
			Markdown: Markdown{
				Title: "股海智擎 " + title,
				Text:  message,
			},
			At: At{
				IsAtAll: true,
			},
		}).
		Post(robotURL)
	if err != nil {
		logger.SugaredLogger.Error(err.Error())
		return "发送钉钉消息失败"
	}
	logger.SugaredLogger.Infof("send dingding message: %s", resp.String())
	return "发送钉钉消息成功"
}

type Message struct {
	Msgtype  string   `json:"msgtype"`
	Markdown Markdown `json:"markdown"`
	At       At       `json:"at"`
}

type Markdown struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type At struct {
	AtMobiles []string `json:"atMobiles"`
	AtUserIds []string `json:"atUserIds"`
	IsAtAll   bool     `json:"isAtAll"`
}
