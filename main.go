package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	appId           = "gamedemo"
	defaultApiToken = "valid_gamedemo_api_token"
)

var cfg = &config{
	OpenServerStatus:          http.StatusOK,
	OpenServerStatusSleep:     15,
	OpenServerCallbackSuccess: true,
	OpenServerCallbackMessage: "whoops",
	NotifyStatus:              http.StatusOK,
	NotifyStatusSleep:         15,
	NotifyCallbackSuccess:     true,
	NotifyCallbackMessage:     "oops",
	ZonelistStatus:            http.StatusOK,
	ZonelistStatusSleep:       15,
	ApiToken:                  defaultApiToken,
	Env:                       os.Getenv("env"),
}

type config struct {
	OpenServerStatus          int    `json:"open_server_status"`
	OpenServerStatusSleep     int    `json:"open_server_status_sleep"`
	OpenServerCallbackSuccess bool   `json:"open_server_callback_success"`
	OpenServerCallbackMessage string `json:"open_server_callback_message"`
	NotifyStatus              int    `json:"notify_status"`
	NotifyStatusSleep         int    `json:"notify_status_sleep"`
	NotifyCallbackSuccess     bool   `json:"notify_callback_success"`
	NotifyCallbackMessage     string `json:"notify_callback_message"`
	ZonelistStatus            int    `json:"zonelist_status"`
	ZonelistStatusSleep       int    `json:"zonelist_status_sleep"`
	ApiToken                  string `json:"api_token"`
	Env                       string `json:"env"`
}

type errReq struct {
	Error string `json:"error"`
}

func main() {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	r.GET("/api/config", getConfig)
	r.POST("/api/config", editConfig)

	r.POST("/api/panik", func(c *gin.Context) {
		panic("panik")
	})

	authorized := r.Group("/")
	authorized.Use(authRequired())
	{
		authorized.POST("/api/server", server)
		authorized.POST("/api/notify", notify)
		authorized.GET("/api/zonelist", zonelist)
	}

	r.Run(":8080")
}

func authRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")

		if token != cfg.ApiToken {
			err := fmt.Errorf("expected authorization token to be %s got %s", cfg.ApiToken, token)
			c.JSON(http.StatusUnauthorized, errReq{Error: err.Error()})
			c.Abort()
			return
		}
		c.Next()
	}
}

func getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, cfg)
}

func editConfig(c *gin.Context) {
	var body config
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errReq{Error: err.Error()})
		return
	}

	cfg = &body
	c.JSON(http.StatusOK, body)
}

type serverAndNotifyReq struct {
	ActionId       string   `json:"actionId"`
	ServerUsers    int      `json:"serverUsers"`
	ServerId       int      `json:"serverId"`
	NewServerNames []string `json:"newServerNames"`
	CallbackToken  string   `json:"callbackToken"`
}

type serverAndNotifyCallbackReq struct {
	AppId         string `json:"appId"`
	CallbackToken string `json:"callbackToken"`
	ActionId      string `json:"actionId"`
	Success       bool   `json:"success"`
	Message       string `json:"message"`
}

func server(c *gin.Context) {
	var body serverAndNotifyReq
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errReq{Error: err.Error()})
		return
	}

	time.Sleep(time.Duration(cfg.OpenServerStatusSleep) * time.Second)

	c.JSON(cfg.OpenServerStatus, gin.H{
		"message": "ok",
		"status":  cfg.OpenServerStatus,
	})

	go func() {
		if cfg.OpenServerStatus != http.StatusAccepted {
			return
		}
		req := serverAndNotifyCallbackReq{
			AppId:         appId,
			CallbackToken: body.CallbackToken,
			ActionId:      body.ActionId,
			Success:       cfg.OpenServerCallbackSuccess,
			Message:       cfg.OpenServerCallbackMessage,
		}
		sendCallback(getServerCallbackUrl(), req)
	}()
}

func notify(c *gin.Context) {
	var body serverAndNotifyReq
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, errReq{Error: err.Error()})
		return
	}

	time.Sleep(time.Duration(cfg.NotifyStatusSleep) * time.Second)

	c.JSON(cfg.NotifyStatus, gin.H{
		"message": "ok",
		"status":  cfg.NotifyStatus,
	})

	go func() {
		if cfg.NotifyStatus != http.StatusAccepted {
			return
		}
		req := serverAndNotifyCallbackReq{
			AppId:         appId,
			CallbackToken: body.CallbackToken,
			ActionId:      body.ActionId,
			Success:       cfg.NotifyCallbackSuccess,
			Message:       cfg.NotifyCallbackMessage,
		}
		sendCallback(getServerCallbackUrl(), req)
	}()
}

func zonelist(c *gin.Context) {
	time.Sleep(time.Duration(cfg.ZonelistStatusSleep) * time.Second)

	c.JSON(cfg.ZonelistStatus, gin.H{
		"message": "ok",
		"status":  cfg.ZonelistStatus,
	})
}

func getServerCallbackUrl() string {
	if os.Getenv("env") == "prod" {
		return "https://game-cloud.g123.jp/cp/api/v1/new_server/callback"
	}
	return "https://game-cloud.stg.g123.jp/cp/api/v1/open_server/callback"
}

func getNotifyCallbackUrl() string {
	if os.Getenv("env") == "prod" {
		return "https://game-cloud.g123.jp/cp/api/v1/new_server/callback"
	}
	return "https://game-cloud.stg.g123.jp/cp/api/v1/new_server/callback"
}

func sendCallback(url string, req serverAndNotifyCallbackReq) {
	postBody, err := json.Marshal(req)
	if err != nil {
		log.Fatalf("error when marshal postbody %v", err)
	}

	responseBody := bytes.NewBuffer(postBody)
	_, err = http.Post(url, "application/json", responseBody)
	if err != nil {
		log.Fatalf("error when sending a request %v", err)
	}
}
