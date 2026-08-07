package service

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
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

type notificationRequestSpec struct {
	endpoint   string
	body       []byte
	headers    map[string]string
	lockedHost string
}

func buildNotificationHTTPRequest(channel *model.NotificationChannel, message outboundMessage) (*notificationRequestSpec, error) {
	config, secret := notificationChannelConfiguration(channel)
	headers := map[string]string{"Content-Type": "application/json", "Accept": "application/json"}
	var endpoint string
	var payload interface{}
	switch channel.Type {
	case "webhook":
		endpoint, _ = secret["url"].(string)
		if values, ok := secret["headers"].(map[string]interface{}); ok {
			for key, value := range values {
				headers[key] = fmt.Sprint(value)
			}
		}
		payload = map[string]interface{}{
			"type":        "seshat.notification",
			"app":         map[string]interface{}{"code": message.AppCode},
			"integration": map[string]interface{}{"id": message.IntegrationID, "name": message.IntegrationName},
			"event": map[string]interface{}{
				"id": message.EventID, "type": message.EventType, "title": message.Title,
				"summary": message.Body, "severity": message.Severity, "received_at": message.ReceivedAt,
			},
		}
	case "telegram":
		token, _ := secret["bot_token"].(string)
		endpoint = "https://api.telegram.org/bot" + token + "/sendMessage"
		payload = map[string]interface{}{"chat_id": config["chat_id"], "text": notificationPlainText(message), "disable_notification": config["silent"]}
		if threadID := config["message_thread_id"]; threadID != nil && fmt.Sprint(threadID) != "" {
			payload.(map[string]interface{})["message_thread_id"] = threadID
		}
	case "apprise":
		baseURL, _ := config["base_url"].(string)
		mode, _ := config["mode"].(string)
		if mode == "stateless" {
			endpoint = strings.TrimRight(baseURL, "/") + "/notify"
			payload = map[string]interface{}{"urls": secret["urls"], "title": message.Title, "body": message.Body, "type": appriseSeverity(message.Severity), "format": "text"}
		} else {
			key, _ := secret["key"].(string)
			endpoint = strings.TrimRight(baseURL, "/") + "/notify/" + url.PathEscape(key)
			payload = map[string]interface{}{"title": message.Title, "body": message.Body, "type": appriseSeverity(message.Severity), "format": "text", "tag": config["tag"]}
		}
	case "serverchan":
		sendKey, _ := secret["send_key"].(string)
		var err error
		endpoint, err = serverChanEndpoint(sendKey)
		if err != nil {
			return nil, &deliveryError{message: err.Error()}
		}
		form := url.Values{"title": []string{truncateRunes(message.Title, 32)}, "desp": []string{message.Body}}
		headers["Content-Type"] = "application/x-www-form-urlencoded"
		return requestSpec(endpoint, []byte(form.Encode()), headers, true), nil
	case "bark":
		baseURL, _ := config["base_url"].(string)
		endpoint = strings.TrimRight(baseURL, "/") + "/push"
		barkPayload := map[string]interface{}{"device_key": secret["device_key"], "title": message.Title, "body": message.Body}
		if group, _ := config["group"].(string); group != "" {
			barkPayload["group"] = group
		}
		if sound, _ := config["sound"].(string); sound != "" {
			barkPayload["sound"] = sound
		}
		payload = barkPayload
	case "dingtalk":
		endpoint, _ = secret["webhook_url"].(string)
		if signingSecret, _ := secret["signing_secret"].(string); signingSecret != "" {
			endpoint = signDingTalkURL(endpoint, signingSecret)
		}
		payload = map[string]interface{}{"msgtype": "markdown", "markdown": map[string]interface{}{"title": singleLineTitle(message.Title), "text": "### " + escapeDingTalkMarkdown(message.Title) + "\n\n" + escapeDingTalkMarkdown(message.Body)}}
	case "feishu":
		endpoint, _ = secret["webhook_url"].(string)
		payload = map[string]interface{}{"msg_type": "text", "content": map[string]interface{}{"text": notificationPlainText(message)}}
		if signingSecret, _ := secret["signing_secret"].(string); signingSecret != "" {
			timestamp, signature := signFeishu(signingSecret)
			payload.(map[string]interface{})["timestamp"] = timestamp
			payload.(map[string]interface{})["sign"] = signature
		}
	case "whatsapp":
		version, _ := config["api_version"].(string)
		phoneNumberID, _ := secret["phone_number_id"].(string)
		endpoint = "https://graph.facebook.com/" + version + "/" + url.PathEscape(phoneNumberID) + "/messages"
		headers["Authorization"] = "Bearer " + fmt.Sprint(secret["access_token"])
		payload = map[string]interface{}{"messaging_product": "whatsapp", "recipient_type": "individual", "to": secret["recipient"], "type": "text", "text": map[string]interface{}{"preview_url": false, "body": notificationPlainText(message)}}
	case "wxpusher":
		endpoint = "https://wxpusher.zjiecode.com/api/send/message"
		topicIDs, err := commaSeparatedInts(fmt.Sprint(config["topic_ids"]))
		if err != nil {
			return nil, &deliveryError{message: "WxPusher Topic ID 配置无效"}
		}
		payload = map[string]interface{}{"appToken": secret["app_token"], "content": notificationPlainText(message), "summary": truncateRunes(message.Title, 100), "contentType": 1, "uids": commaSeparatedStrings(fmt.Sprint(config["uids"])), "topicIds": topicIDs}
	default:
		return nil, &deliveryError{message: "不支持的推送渠道类型"}
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, &deliveryError{message: "无法生成推送内容"}
	}
	return requestSpec(endpoint, body, headers, channel.Type != "webhook"), nil
}

func requestSpec(endpoint string, body []byte, headers map[string]string, lockHost bool) *notificationRequestSpec {
	spec := &notificationRequestSpec{endpoint: endpoint, body: body, headers: headers}
	if lockHost {
		if parsed, err := url.Parse(endpoint); err == nil {
			spec.lockedHost = parsed.Hostname()
		}
	}
	return spec
}

func notificationChannelConfiguration(channel *model.NotificationChannel) (map[string]interface{}, map[string]interface{}) {
	config, secret := map[string]interface{}{}, map[string]interface{}{}
	_ = json.Unmarshal(channel.Config, &config)
	_ = json.Unmarshal(channel.SecretConfig, &secret)
	return config, secret
}

func notificationPlainText(message outboundMessage) string {
	if message.Body == "" {
		return message.Title
	}
	return message.Title + "\n\n" + message.Body
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}

func singleLineTitle(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r", " "), "\n", " "))
}

func escapeDingTalkMarkdown(value string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "{", "\\{", "}", "\\}",
		"[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)", "#", "\\#", "+", "\\+",
		"-", "\\-", ".", "\\.", "!", "\\!", "|", "\\|", ">", "\\>",
	)
	return replacer.Replace(value)
}

func serverChanEndpoint(sendKey string) (string, error) {
	if strings.HasPrefix(sendKey, "sctp") {
		remainder := strings.TrimPrefix(sendKey, "sctp")
		separator := strings.IndexByte(remainder, 't')
		if separator <= 0 {
			return "", errors.New("Server酱³ SendKey 格式无效")
		}
		uid := remainder[:separator]
		if _, err := strconv.ParseUint(uid, 10, 64); err != nil {
			return "", errors.New("Server酱³ SendKey 格式无效")
		}
		return "https://" + uid + ".push.ft07.com/send/" + url.PathEscape(sendKey) + ".send", nil
	}
	return "https://sctapi.ftqq.com/" + url.PathEscape(sendKey) + ".send", nil
}

func signDingTalkURL(endpoint, secret string) string {
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "\n" + secret))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return endpoint
	}
	query := parsed.Query()
	query.Set("timestamp", timestamp)
	query.Set("sign", signature)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func signFeishu(secret string) (string, string) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	stringToSign := timestamp + "\n" + secret
	mac := hmac.New(sha256.New, []byte(stringToSign))
	return timestamp, base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func commaSeparatedStrings(value string) []string {
	parts := strings.FieldsFunc(value, func(character rune) bool { return character == ',' || character == ';' || character == '\n' })
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			result = append(result, item)
		}
	}
	return result
}

func commaSeparatedInts(value string) ([]int64, error) {
	stringsList := commaSeparatedStrings(value)
	result := make([]int64, 0, len(stringsList))
	for _, item := range stringsList {
		number, err := strconv.ParseInt(item, 10, 64)
		if err != nil || number <= 0 {
			return nil, errors.New("invalid integer list")
		}
		result = append(result, number)
	}
	return result, nil
}

func validateNotificationAPIResponse(channelType string, body []byte) error {
	if channelType == "webhook" || channelType == "apprise" {
		return nil
	}
	var response map[string]interface{}
	if len(body) == 0 || json.Unmarshal(body, &response) != nil {
		return &deliveryError{message: "推送渠道返回了无法识别的响应"}
	}
	number := func(key string) int64 {
		switch value := response[key].(type) {
		case float64:
			return int64(value)
		case string:
			result, _ := strconv.ParseInt(value, 10, 64)
			return result
		default:
			return -1
		}
	}
	success := false
	switch channelType {
	case "telegram":
		success, _ = response["ok"].(bool)
	case "serverchan", "dingtalk", "feishu":
		success = number("code") == 0 || number("errcode") == 0 || number("StatusCode") == 0
	case "bark":
		success = number("code") == 200
	case "whatsapp":
		messages, _ := response["messages"].([]interface{})
		success = len(messages) > 0 && response["error"] == nil
	case "wxpusher":
		success = number("code") == 1000
	}
	if !success {
		return &deliveryError{message: "推送渠道 API 返回失败状态"}
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
	to = strings.ReplaceAll(to, ";", ",")
	recipients, err := mail.ParseAddressList(to)
	if err != nil || len(recipients) == 0 {
		return "", nil, errors.New("邮件收件人地址无效")
	}
	result := make([]string, 0, len(recipients))
	for _, recipient := range recipients {
		result = append(result, recipient.Address)
	}
	return fromAddress.Address, result, nil
}

func sendEmailChannel(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, allowPrivate bool, proxyURL string) error {
	config, secret := notificationChannelConfiguration(channel)
	host := strings.TrimSpace(fmt.Sprint(config["smtp_host"]))
	port, err := notificationSMTPPort(config)
	if err != nil {
		return &deliveryError{message: err.Error()}
	}
	encryption := strings.TrimSpace(fmt.Sprint(config["encryption"]))
	if encryption == "" {
		encryption = "starttls"
	}
	if encryption == "none" && !allowPrivate {
		return &deliveryError{message: "未加密 SMTP 仅允许在私有网络推送开启后使用"}
	}
	from, recipients, err := parseEmailRecipients(fmt.Sprint(config["from"]), fmt.Sprint(config["to"]))
	if err != nil {
		return &deliveryError{message: err.Error()}
	}
	address := net.JoinHostPort(host, strconv.Itoa(port))
	var connection net.Conn
	if proxyURL != "" {
		if err := validateOutboundHost(host, allowPrivate); err != nil {
			return &deliveryError{message: err.Error()}
		}
		connection, err = dialHTTPProxyTunnel(ctx, proxyURL, address)
	} else {
		connection, err = dialValidatedAddress(ctx, host, strconv.Itoa(port), allowPrivate)
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
	username, _ := secret["username"].(string)
	password, _ := secret["password"].(string)
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
