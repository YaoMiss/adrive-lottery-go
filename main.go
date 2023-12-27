package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	TokenUrl       = "https://auth.aliyundrive.com/v2/account/token"
	SignUrl        = "https://member.aliyundrive.com/v1/activity/sign_in_list"
	RetryHours     = 3
	Date2DayFormat = "2006/01/02"
)

func main() {
	tgToken := os.Getenv("TG_BOT_TOKEN")
	tgUserId, _ := strconv.ParseInt(os.Getenv("TG_USER_ID"), 10, 64)
	aliRefreshToken := os.Getenv("ALIPAN_REFRESH_TOKENS")

	if len(tgToken) == 0 || len(aliRefreshToken) == 0 {
		log.Fatalln(fmt.Sprintf("Plz fill out the .env file"))
	}
	bot, err := tgbotapi.NewBotAPI(tgToken)
	if err != nil {
		log.Fatalln("TG BOT is initialized failed", err)
	}
	tgMsg(tgUserId, fmt.Sprintf(" 🏷🏷🏷 TG BOT %s auth successfully && will sign in automatally", bot.Self.UserName), bot)

	for {
		for _, refreshToken := range strings.Split(aliRefreshToken, "&") {
			md5binStr := md5.Sum([]byte(fmt.Sprintf("%s-%s", refreshToken, time.Now().Format(Date2DayFormat))))
			filename := hex.EncodeToString(md5binStr[:])

			if fileIsExisting(filename) {
				timezone, _ := time.LoadLocation("Asia/Shanghai")
				addTime := time.Now().In(timezone).Add(time.Hour * RetryHours)
				msg := fmt.Sprintf(`
%s Already signed in and file was locked. 
📢📢📢 If you wanna force to sign >>>> 
Please clear the file : ./log/%s 
NOTE : Job will be retried at %s`, time.Now().Format(Date2DayFormat), filename, addTime.Format(time.RFC3339))
				go tgMsg(tgUserId, msg, bot)

				log.Println(fmt.Sprintf("%s is existing", filename))
				time.Sleep(time.Hour * RetryHours)
				continue
			}

			var tokenResMap = acquireToken(refreshToken)
			userName, _ := tokenResMap["user_name"]

			if accessToken, isSet := tokenResMap["access_token"]; isSet {
				signRes := sign(accessToken.(string))
				if isSuccess, isSet := signRes["success"]; isSet && isSuccess == true {
					dir, _ := filepath.Abs("./log")
					filePointer, _ := os.OpenFile(fmt.Sprintf("%s/%s", dir, filename), os.O_RDONLY|os.O_CREATE, 0766)
					defer filePointer.Close()

					msg := fmt.Sprintf("⭐️⭐️⭐️ %s️ ALIPAN user_name = %s sign in successfully", time.Now().Format(Date2DayFormat), userName)
					go tgMsg(tgUserId, msg, bot)
					log.Println(msg)

					time.Sleep(time.Second * 5)
				}
			} else {
				log.Println("📢📢📢 Exchange token encountered an error")
				time.Sleep(time.Second * 5)
			}
		}
	}

}

func acquireToken(refreshToken string) (tokenMap map[string]interface{}) {
	var resBufferData = make([]byte, 1<<20)
	var resMap map[string]interface{}

	client := &http.Client{}

	jsonParams := make(map[string]interface{})
	jsonParams["grant_type"] = "refresh_token"
	jsonParams["refresh_token"] = refreshToken
	byteData, _ := json.Marshal(jsonParams)

	request, _ := http.NewRequest("POST", TokenUrl, bytes.NewReader(byteData))
	request.Header.Set("Content-Type", "application/json")
	res, _ := client.Do(request)
	defer res.Body.Close()
	n, _ := io.ReadFull(res.Body, resBufferData)

	if err := json.Unmarshal(resBufferData[:n], &resMap); err != nil {
		log.Panic(err)
	}

	return resMap
}

func sign(token string) (signRes map[string]interface{}) {
	var resBufferBytes = make([]byte, 1<<20)
	var resMap map[string]interface{}

	client := &http.Client{}
	jsonParams := make(map[string]interface{})
	jsonParams["isReward"] = false
	byteData, _ := json.Marshal(jsonParams)

	request, _ := http.NewRequest("POST", SignUrl, bytes.NewReader(byteData))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	res, _ := client.Do(request)
	defer res.Body.Close()
	n, _ := io.ReadFull(res.Body, resBufferBytes)

	if err := json.Unmarshal(resBufferBytes[:n], &resMap); err != nil {
		log.Panic(err)
	}

	return resMap
}

func tgMsg(chatId int64, msg string, bot *tgbotapi.BotAPI) {
	newMsg := tgbotapi.NewMessage(chatId, msg)
	_, _ = bot.Send(newMsg)
}

func fileIsExisting(filename string) bool {
	dir, _ := filepath.Abs("./log")
	path := fmt.Sprintf("%s/%s", dir, filename)

	_, err := os.Stat(path)

	if err == nil {
		return true
	}

	if os.IsNotExist(err) {
		return false
	}
	return false
}
