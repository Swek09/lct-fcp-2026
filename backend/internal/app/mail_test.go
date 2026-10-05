package app

import (
	"net"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

// A loopback SMTP server checks the actual wire protocol without external mail.
func smtpServer(t *testing.T) (Config, <-chan string) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	messages := make(chan string, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		protocol := textproto.NewConn(conn)
		_ = protocol.PrintfLine("220 localhost ESMTP")
		for {
			line, err := protocol.ReadLine()
			if err != nil {
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO "), strings.HasPrefix(line, "HELO "):
				_ = protocol.PrintfLine("250 localhost")
			case strings.HasPrefix(line, "MAIL FROM:"), strings.HasPrefix(line, "RCPT TO:"):
				_ = protocol.PrintfLine("250 OK")
			case line == "DATA":
				_ = protocol.PrintfLine("354 Send message")
				raw, err := protocol.ReadDotBytes()
				if err != nil {
					return
				}
				messages <- string(raw)
				_ = protocol.PrintfLine("250 Queued")
			case line == "QUIT":
				_ = protocol.PrintfLine("221 Bye")
				return
			default:
				_ = protocol.PrintfLine("500 Unsupported")
			}
		}
	}()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	return Config{SMTPHost: host, SMTPPort: port, SMTPFrom: "platform@example.test", Env: "test"}, messages
}
func TestSMTP(t *testing.T) {
	t.Run("delivers_utf8_message", func(t *testing.T) {
		cfg, messages := smtpServer(t)
		a := &App{Config: cfg}
		err := a.sendMail(nil, "recipient@example.test", []byte("From: platform@example.test\r\nTo: recipient@example.test\r\nSubject: Verification\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nПодтверждение почты\r\n"))
		if err != nil {
			t.Fatal(err)
		}
		select {
		case message := <-messages:
			if !strings.Contains(message, "Подтверждение почты") || !strings.Contains(message, "recipient@example.test") {
				t.Fatalf("unexpected SMTP payload: %s", message)
			}
		case <-time.After(time.Second):
			t.Fatal("SMTP DATA was not delivered")
		}
	})
	t.Run("production_refuses_plaintext", func(t *testing.T) {
		cfg, messages := smtpServer(t)
		cfg.Env = "production"
		a := &App{Config: cfg}
		if err := a.sendMail(nil, "recipient@example.test", []byte("secret token")); err == nil || !strings.Contains(err.Error(), "STARTTLS") {
			t.Fatalf("expected STARTTLS requirement, got %v", err)
		}
		select {
		case <-messages:
			t.Fatal("plaintext secret was delivered")
		default:
		}
	})
}
