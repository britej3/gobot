// Package platform provides secure Telegram bot implementation per technical specifications
package platform

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// SecureBot provides authenticated Telegram bot command handling
type SecureBot struct {
	bot      *tgbotapi.BotAPI
	authID   int64
	commands map[string]CommandHandler
	mu       sync.RWMutex
}

// CommandHandler defines the function signature for bot commands
type CommandHandler func(update tgbotapi.Update) error

// NewSecureBot creates a new secure Telegram bot with ChatID whitelisting
func NewSecureBot() (*SecureBot, error) {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_TOKEN not set in .env")
	}
	
	authIDStr := os.Getenv("AUTHORIZED_CHAT_ID")
	if authIDStr == "" {
		return nil, fmt.Errorf("AUTHORIZED_CHAT_ID not set in .env")
	}
	
	authID, err := strconv.ParseInt(authIDStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid AUTHORIZED_CHAT_ID: %w", err)
	}
	
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}
	
	return &SecureBot{
		bot:      bot,
		authID:   authID,
		commands: make(map[string]CommandHandler),
	}, nil
}

// RegisterCommand adds a command handler
func (b *SecureBot) RegisterCommand(name string, handler CommandHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.commands[name] = handler
}

// Start begins the secure bot with middleware ChatID validation
func (b *SecureBot) Start() {
	u := tgbotapi.NewUpdate(0)
	updates := b.bot.GetUpdatesChan(u)
	
	log.Printf("✅ Secure Telegram bot started. Whitelisted ChatID: %d", b.authID)
	
	for update := range updates {
		if update.Message == nil {
			continue
		}
		
		// SECURITY CHECK: Whitelist ChatID (per reply_unknown.md)
		if update.Message.Chat.ID != b.authID {
			log.Printf("⛔ UNAUTHORIZED ACCESS ATTEMPT from ChatID: %d", update.Message.Chat.ID)
			// Silently ignore or send "Not Authorized" message
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "⛔ Unauthorized access. Your ChatID is not whitelisted.")
			b.bot.Send(msg)
			continue
		}
		
		// Handle authorized commands
		command := update.Message.Command()
		if command != "" {
			b.mu.RLock()
			handler, exists := b.commands[command]
			b.mu.RUnlock()
			
			if !exists {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❓ Unknown command. Available: /panic, /status")
				b.bot.Send(msg)
				continue
			}
			
			// Execute command handler
			if err := handler(update); err != nil {
				log.Printf("❌ Command handler error: %v", err)
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("❌ Error: %v", err))
				b.bot.Send(msg)
			}
		}
	}
}

// SendMessage sends a message to the authorized chat
func (b *SecureBot) SendMessage(text string) error {
	msg := tgbotapi.NewMessage(b.authID, text)
	_, err := b.bot.Send(msg)
	return err
}

// GetBot returns the underlying bot API for advanced usage
func (b *SecureBot) GetBot() *tgbotapi.BotAPI {
	return b.bot
}
