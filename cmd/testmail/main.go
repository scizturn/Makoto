package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"strings"
)

func main() {
	// Load .env file
	loadEnvFile(".env")

	smtpHost := "smtp.kirimemail.com"
	smtpPort := "587"
	
	senderEmail := os.Getenv("SMTP_USERNAME") 
	smtpPassword := os.Getenv("SMTP_PASSWORD")

	if senderEmail == "" || smtpPassword == "" {
		log.Fatal("Please set SMTP_USERNAME and SMTP_PASSWORD in your .env file.")
	}

	recipients := []string{
		"sebastianshanr@gmail.com",
		"tegaramir327@gmail.com",
	}

	for _, to := range recipients {
		// Construct the email message
		subject := "Subject: Test Email dari Makoto (SMTP)\r\n"
		mime := "MIME-version: 1.0;\nContent-Type: text/plain; charset=\"UTF-8\";\n\n"
		body := "Halo! Ini adalah test email dari Makoto menggunakan koneksi langsung SMTP.\r\n"
		
		msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\n%s%s%s", senderEmail, to, subject, mime, body))

		// Setup authentication
		auth := smtp.PlainAuth("", senderEmail, smtpPassword, smtpHost)

		// Create TLS configuration
		tlsconfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         smtpHost,
		}

		log.Printf("Sending test email to %s via SMTP...", to)
		
		// Connect to the SMTP Server
		conn, err := smtp.Dial(smtpHost + ":" + smtpPort)
		if err != nil {
			log.Printf("Failed to connect to SMTP server: %v", err)
			continue
		}

		// Start TLS
		if err = conn.StartTLS(tlsconfig); err != nil {
			log.Printf("Failed to start TLS: %v", err)
			conn.Close()
			continue
		}

		// Authenticate
		if err = conn.Auth(auth); err != nil {
			log.Printf("Failed to authenticate: %v", err)
			conn.Close()
			continue
		}

		// Set the sender and recipient first
		if err = conn.Mail(senderEmail); err != nil {
			log.Printf("Failed to set sender: %v", err)
			conn.Close()
			continue
		}

		if err = conn.Rcpt(to); err != nil {
			log.Printf("Failed to set recipient: %v", err)
			conn.Close()
			continue
		}

		// Send the email body
		w, err := conn.Data()
		if err != nil {
			log.Printf("Failed to issue DATA command: %v", err)
			conn.Close()
			continue
		}

		_, err = w.Write(msg)
		if err != nil {
			log.Printf("Failed to write message body: %v", err)
			w.Close()
			conn.Close()
			continue
		}

		err = w.Close()
		if err != nil {
			log.Printf("Failed to close message writer: %v", err)
			conn.Close()
			continue
		}

		conn.Quit()
		
		fmt.Printf("[%s] SUCCESS: Email sent via SMTP\n", to)
	}

	log.Println("Done.")
}

// loadEnvFile reads a .env file and sets environment variables.
func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return // silently skip if no .env file
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if os.Getenv(key) == "" {
			os.Setenv(key, value)
		}
	}
}
