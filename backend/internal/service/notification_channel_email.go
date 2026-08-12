package service

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/http"
	"net/mail"
	"net/smtp"
	"net/url"
	"strconv"
	"strings"
	"time"

	"seshat/internal/model"
)

type emailNotificationAdapter struct{}

func (emailNotificationAdapter) Type() string { return "email" }

func (emailNotificationAdapter) Sanitize(config, credentials map[string]interface{}) (map[string]interface{}, map[string]interface{}, error) {
	cleanConfig, cleanCredentials, err := sanitizeNotificationChannelFields(config, credentials, []string{"smtp_host", "smtp_port", "encryption", "from", "to"}, []string{"username", "password"})
	if err != nil {
		return nil, nil, err
	}
	encryption, _ := cleanConfig["encryption"].(string)
	if encryption == "" {
		cleanConfig["encryption"] = "starttls"
	} else if encryption != "starttls" && encryption != "tls" && encryption != "none" {
		return nil, nil, errors.New("邮件加密方式必须是 starttls、tls 或 none")
	}
	return cleanConfig, cleanCredentials, nil
}

func (emailNotificationAdapter) Validate(config, credentials map[string]interface{}) error {
	if !requiredNotificationString(config, "smtp_host") || !requiredNotificationString(config, "from") || !requiredNotificationString(config, "to") {
		return errors.New("SMTP 主机、发件人和收件人不能为空")
	}
	if _, err := notificationSMTPPort(config); err != nil {
		return err
	}
	if _, _, err := parseEmailRecipients(fmt.Sprint(config["from"]), fmt.Sprint(config["to"])); err != nil {
		return err
	}
	if requiredNotificationString(credentials, "username") != requiredNotificationString(credentials, "password") {
		return errors.New("SMTP 用户名和密码必须同时填写")
	}
	return nil
}

func (emailNotificationAdapter) Send(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, options notificationSendOptions) error {
	config, credentials := notificationChannelConfiguration(channel)
	host := strings.TrimSpace(fmt.Sprint(config["smtp_host"]))
	port, err := notificationSMTPPort(config)
	if err != nil {
		return &deliveryError{message: err.Error()}
	}
	encryption := strings.TrimSpace(fmt.Sprint(config["encryption"]))
	if encryption == "" {
		encryption = "starttls"
	}
	if encryption == "none" && !options.allowPrivate {
		return &deliveryError{message: "未加密 SMTP 仅允许在私有网络推送开启后使用"}
	}
	from, recipients, err := parseEmailRecipients(fmt.Sprint(config["from"]), fmt.Sprint(config["to"]))
	if err != nil {
		return &deliveryError{message: err.Error()}
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))
	var connection net.Conn
	if options.proxyURL != "" {
		if err := validateOutboundHost(host, options.allowPrivate); err != nil {
			return &deliveryError{message: err.Error()}
		}
		connection, err = dialHTTPProxyTunnel(ctx, options.proxyURL, address)
	} else {
		connection, err = dialValidatedAddress(ctx, host, strconv.Itoa(port), options.allowPrivate)
	}
	if err != nil {
		return &deliveryError{message: "连接 SMTP 服务器失败", retryable: true}
	}
	defer connection.Close()
	deadline := time.Now().Add(10 * time.Second)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	_ = connection.SetDeadline(deadline)
	tlsConfig := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if encryption == "tls" {
		tlsConnection := tls.Client(connection, tlsConfig)
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			return &deliveryError{message: "SMTP TLS 握手失败", retryable: true}
		}
		connection = tlsConnection
	}
	client, err := smtp.NewClient(connection, host)
	if err != nil {
		return &deliveryError{message: "创建 SMTP 会话失败", retryable: true}
	}
	defer client.Close()
	if encryption == "starttls" {
		if supported, _ := client.Extension("STARTTLS"); !supported {
			return &deliveryError{message: "SMTP 服务器不支持 STARTTLS"}
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return &deliveryError{message: "SMTP STARTTLS 失败", retryable: true}
		}
	}
	username, _ := credentials["username"].(string)
	password, _ := credentials["password"].(string)
	if username != "" {
		if err := client.Auth(smtp.PlainAuth("", username, password, host)); err != nil {
			return &deliveryError{message: "SMTP 身份验证失败"}
		}
	}
	if err := client.Mail(from); err != nil {
		return &deliveryError{message: "SMTP 发件人被拒绝"}
	}
	for _, recipient := range recipients {
		if err := client.Rcpt(recipient); err != nil {
			return &deliveryError{message: "SMTP 收件人被拒绝"}
		}
	}
	writer, err := client.Data()
	if err != nil {
		return &deliveryError{message: "SMTP 无法写入邮件内容", retryable: true}
	}
	subject := mime.QEncoding.Encode("UTF-8", strings.ReplaceAll(strings.ReplaceAll(message.Title, "\r", " "), "\n", " "))
	bodyText := strings.ReplaceAll(strings.ReplaceAll(notificationPlainText(message), "\r\n", "\n"), "\r", "\n")
	bodyText = strings.ReplaceAll(bodyText, "\n", "\r\n")
	mailBody := "From: " + from + "\r\nTo: " + strings.Join(recipients, ", ") + "\r\nSubject: " + subject + "\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n" + bodyText
	if _, err := writer.Write([]byte(mailBody)); err != nil {
		_ = writer.Close()
		return &deliveryError{message: "SMTP 写入邮件内容失败", retryable: true}
	}
	if err := writer.Close(); err != nil {
		return &deliveryError{message: "SMTP 发送邮件失败", retryable: true}
	}
	if err := client.Quit(); err != nil {
		return &deliveryError{message: "SMTP 结束会话失败", retryable: true}
	}
	return nil
}

func notificationSMTPPort(config map[string]interface{}) (int, error) {
	value, exists := config["smtp_port"]
	if !exists || value == nil || fmt.Sprint(value) == "" {
		return 587, nil
	}
	var port int
	switch typed := value.(type) {
	case float64:
		if typed != float64(int(typed)) {
			return 0, errors.New("SMTP 端口必须是整数")
		}
		port = int(typed)
	case int:
		port = typed
	case string:
		port, _ = strconv.Atoi(typed)
	}
	if port < 1 || port > 65535 {
		return 0, errors.New("SMTP 端口必须在 1 到 65535 之间")
	}
	return port, nil
}

func parseEmailRecipients(from, to string) (string, []string, error) {
	fromAddress, err := mail.ParseAddress(strings.TrimSpace(from))
	if err != nil {
		return "", nil, errors.New("邮件发件人地址无效")
	}
	recipients, err := mail.ParseAddressList(strings.ReplaceAll(to, ";", ","))
	if err != nil || len(recipients) == 0 {
		return "", nil, errors.New("邮件收件人地址无效")
	}
	result := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		result = append(result, recipient.Address)
	}
	return fromAddress.Address, result, nil
}

func validateOutboundHost(host string, allowPrivate bool) error {
	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return errors.New("无法解析推送目标地址")
	}
	for _, ip := range ips {
		if err := validateOutboundIP(ip, allowPrivate); err != nil {
			return err
		}
	}
	return nil
}

func dialValidatedAddress(ctx context.Context, host, port string, allowPrivate bool) (net.Conn, error) {
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil || len(ips) == 0 {
		return nil, errors.New("无法解析推送目标地址")
	}
	for _, ip := range ips {
		if err := validateOutboundIP(ip, allowPrivate); err != nil {
			return nil, err
		}
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	return dialer.DialContext(ctx, "tcp", net.JoinHostPort(ips[0].String(), port))
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (connection *bufferedConn) Read(buffer []byte) (int, error) {
	return connection.reader.Read(buffer)
}

func dialHTTPProxyTunnel(ctx context.Context, rawProxyURL, targetAddress string) (net.Conn, error) {
	proxyURL, err := url.Parse(rawProxyURL)
	if err != nil || proxyURL.Hostname() == "" || (proxyURL.Scheme != "http" && proxyURL.Scheme != "https") {
		return nil, ErrInvalidHTTPProxyURL
	}
	port := proxyURL.Port()
	if port == "" {
		if proxyURL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}
	connection, err := dialValidatedAddress(ctx, proxyURL.Hostname(), port, true)
	if err != nil {
		return nil, err
	}
	if proxyURL.Scheme == "https" {
		tlsConnection := tls.Client(connection, &tls.Config{ServerName: proxyURL.Hostname(), MinVersion: tls.VersionTLS12})
		if err := tlsConnection.HandshakeContext(ctx); err != nil {
			_ = connection.Close()
			return nil, err
		}
		connection = tlsConnection
	}
	request := &http.Request{Method: http.MethodConnect, URL: &url.URL{Opaque: targetAddress}, Host: targetAddress, Header: make(http.Header)}
	if proxyURL.User != nil {
		password, _ := proxyURL.User.Password()
		credentials := base64.StdEncoding.EncodeToString([]byte(proxyURL.User.Username() + ":" + password))
		request.Header.Set("Proxy-Authorization", "Basic "+credentials)
	}
	if err := request.Write(connection); err != nil {
		_ = connection.Close()
		return nil, err
	}
	reader := bufio.NewReader(connection)
	response, err := http.ReadResponse(reader, request)
	if err != nil {
		_ = connection.Close()
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		if response.Body != nil {
			_ = response.Body.Close()
		}
		_ = connection.Close()
		return nil, errors.New("HTTP 代理拒绝 CONNECT 请求")
	}
	return &bufferedConn{Conn: connection, reader: reader}, nil
}
