package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"seshat/internal/logging"
	"seshat/internal/model"
	"seshat/internal/webhook"
	"seshat/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	maxNotificationRetries    = 6
	notificationResponseLimit = 32 << 10
	staleDeliveryThreshold    = time.Minute
)

var (
	errDeliveryAlreadyClaimed = errors.New("推送任务已被其他 Worker 领取")
	errDeliveryDeferred       = errors.New("推送任务等待渠道发送窗口")
	errNoDeliveryAvailable    = errors.New("暂无待发送的推送任务")
)

type deliveryError struct {
	message      string
	statusCode   int
	providerCode string
	retryable    bool
	retryAfter   time.Duration
	rateLimited  bool
}

func (e *deliveryError) Error() string { return e.message }

type outboundMessage struct {
	Title           string
	Body            string
	Severity        string
	AppCode         string
	IntegrationID   uint
	IntegrationName string
	EventID         uint
	EventType       string
	ReceivedAt      time.Time
	DetailURL       string
	PublicBaseURL   string
	Presentation    *webhook.Presentation
	Rendered        *renderedNotification
}

func (s *NotificationService) TestChannel(ownerID, id uint) (*NotificationChannelResponse, error) {
	var channel model.NotificationChannel
	if err := database.GetDB().Where("id = ? AND owner_id = ?", id, ownerID).First(&channel).Error; err != nil {
		return nil, ErrNotificationChannelNotFound
	}
	message := outboundMessage{Title: "Seshat 测试通知", Body: "推送渠道配置成功，可以正常接收通知。", Severity: "info"}
	allowPrivate, proxyURL, policyErr := notificationNetworkPolicyForChannel(&channel)
	err := policyErr
	if err == nil {
		err = sendChannel(context.Background(), &channel, message, allowPrivate, proxyURL)
	}
	now := time.Now()
	updates := map[string]interface{}{"last_test_at": now}
	if err != nil {
		updates["last_test_status"] = "failed"
		updates["last_test_error"] = truncateNotificationError(err.Error())
		logging.Warn("notification", "推送渠道测试失败", logging.Fields{"channel_id": channel.ID, "channel_type": channel.Type, "error": err})
	} else {
		updates["last_test_status"] = "succeeded"
		updates["last_test_error"] = ""
		logging.Info("notification", "推送渠道测试成功", logging.Fields{"channel_id": channel.ID, "channel_type": channel.Type})
	}
	if updateErr := database.GetDB().Model(&channel).Updates(updates).Error; updateErr != nil {
		logging.Error("notification", "保存推送渠道测试结果失败", logging.Fields{"channel_id": channel.ID, "error": updateErr})
	}
	if err != nil {
		return nil, err
	}
	channel.LastTestAt = &now
	channel.LastTestStatus = "succeeded"
	channel.LastTestError = ""
	response, responseErr := channelResponse(database.GetDB(), &channel)
	return &response, responseErr
}

type NotificationWorker struct {
	done     chan struct{}
	wg       sync.WaitGroup
	webhooks *WebhookService
}

func NewNotificationWorker() *NotificationWorker {
	return &NotificationWorker{done: make(chan struct{}), webhooks: NewWebhookService()}
}

func (w *NotificationWorker) Start() {
	// A process may stop after claiming a delivery but before updating its final
	// state. Return those orphaned jobs to the queue on startup.
	now := time.Now()
	result := database.GetDB().Model(&model.NotificationDelivery{}).Where("status = ? AND updated_at < ?", "sending", now.Add(-staleDeliveryThreshold)).Updates(map[string]interface{}{"status": "retrying", "next_attempt_at": now})
	if result.Error != nil {
		logging.Error("notification", "恢复中断的推送任务失败", logging.Fields{"error": result.Error})
	} else if result.RowsAffected > 0 {
		logging.Info("notification", "已恢复中断的推送任务", logging.Fields{"count": result.RowsAffected})
	}
	w.wg.Add(1)
	go w.loop()
}

func (w *NotificationWorker) Close() {
	if w == nil {
		return
	}
	close(w.done)
	w.wg.Wait()
}

func (w *NotificationWorker) loop() {
	defer w.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			for i := 0; i < 10 && w.processOne(); i++ {
			}
		case <-w.done:
			return
		}
	}
}

func (w *NotificationWorker) processOne() bool {
	db := database.GetDB()
	var delivery model.NotificationDelivery
	deferred := false
	err := db.Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("status IN ? AND (next_attempt_at IS NULL OR next_attempt_at <= ?)", []string{"pending", "retrying"}, now).Order("id ASC").Limit(1).Find(&delivery)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errNoDeliveryAvailable
		}
		var channel model.NotificationChannel
		if err := tx.First(&channel, delivery.ChannelID).Error; err == nil {
			if adapter, ok := defaultNotificationChannelRegistry.Get(channel.Type); ok {
				if limitedAdapter, ok := adapter.(notificationRateLimitedAdapter); ok {
					policy, policyErr := limitedAdapter.RateLimitPolicy(&channel)
					if policyErr == nil {
						allowed, next, limitErr := reserveNotificationSendSlot(tx, policy, now)
						if limitErr != nil {
							return limitErr
						}
						if !allowed {
							updates := map[string]interface{}{"next_attempt_at": next, "defer_reason": "rate_limited"}
							if err := tx.Model(&model.NotificationDelivery{}).Where("id = ? AND status IN ?", delivery.ID, []string{"pending", "retrying"}).Updates(updates).Error; err != nil {
								return err
							}
							delivery.NextAttemptAt = &next
							delivery.DeferReason = "rate_limited"
							deferred = true
							return nil
						}
					}
				}
			}
		}
		if deferred {
			return nil
		}
		result = tx.Model(&model.NotificationDelivery{}).Where("id = ? AND status IN ?", delivery.ID, []string{"pending", "retrying"}).Updates(map[string]interface{}{"status": "sending", "attempt_count": gorm.Expr("attempt_count + 1"), "defer_reason": ""})
		if result.Error == nil && result.RowsAffected == 0 {
			return errDeliveryAlreadyClaimed
		}
		return result.Error
	})
	if deferred && err == nil {
		err = errDeliveryDeferred
	}
	if errors.Is(err, errDeliveryAlreadyClaimed) {
		return true
	}
	if errors.Is(err, errDeliveryDeferred) {
		logging.Info("notification", "推送等待渠道发送窗口", logging.Fields{"delivery_id": delivery.ID, "channel_id": delivery.ChannelID, "channel_type": delivery.ChannelType, "next_attempt_at": delivery.NextAttemptAt})
		return true
	}
	if errors.Is(err, errNoDeliveryAvailable) {
		return false
	}
	if err != nil {
		logging.Error("notification", "领取推送任务失败", logging.Fields{"error": err})
		return false
	}
	delivery.AttemptCount++
	w.send(&delivery)
	return true
}

func (w *NotificationWorker) send(delivery *model.NotificationDelivery) {
	db := database.GetDB()
	var event model.WebhookEvent
	var integration model.AppIntegration
	var channel model.NotificationChannel
	if err := db.First(&event, delivery.EventID).Error; err != nil {
		w.fail(delivery, 0, errors.New("消息不存在"), false)
		return
	}
	if err := db.First(&integration, delivery.IntegrationID).Error; err != nil {
		w.fail(delivery, 0, errors.New("接入实例不存在"), false)
		return
	}
	if err := db.First(&channel, delivery.ChannelID).Error; err != nil {
		w.fail(delivery, 0, errors.New("推送渠道不存在"), false)
		return
	}
	if !channel.Enabled {
		w.fail(delivery, 0, errors.New("推送渠道已停用"), false)
		return
	}
	if delivery.ContentVersion == 0 {
		if err := w.webhooks.materializeEvent(context.Background(), &event, &integration); err != nil {
			w.fail(delivery, 0, errors.New("生成推送展示内容失败"), true)
			return
		}
	}
	message := outboundMessage{Title: event.Title, Body: event.Summary, Severity: event.Severity, AppCode: event.AppCode, IntegrationID: integration.ID, IntegrationName: integration.Name, EventID: event.ID, EventType: event.DisplayEventType, ReceivedAt: event.ReceivedAt}
	if message.Title == "" {
		message.Title = integration.Name + " · " + event.DisplayEventType
	}
	if message.Body == "" {
		message.Body = "收到一条新的 Webhook 消息"
	}
	publicBaseURL := NewSettingService().PublicBaseURL()
	if publicBaseURL != "" {
		if err := ensurePublicEventToken(&event); err != nil {
			w.fail(delivery, 0, errors.New("生成消息详情链接失败"), true)
			return
		}
		message.DetailURL = publicBaseURL + "/public/events/" + url.PathEscape(event.PublicToken)
		message.PublicBaseURL = publicBaseURL
	}
	if len(event.Presentation) > 0 {
		var presentation webhook.Presentation
		if json.Unmarshal(event.Presentation, &presentation) == nil {
			message.Presentation = &presentation
		}
	}
	allowPrivate, proxyURL, policyErr := notificationNetworkPolicyForChannel(&channel)
	err := policyErr
	if err == nil {
		adapter, ok := defaultNotificationChannelRegistry.Get(channel.Type)
		if !ok {
			err = &deliveryError{message: "不支持的推送渠道类型"}
		} else {
			message, err = messageForDelivery(adapter, &channel, delivery, message)
			if err == nil {
				err = adapter.Send(context.Background(), &channel, message, notificationSendOptions{allowPrivate: allowPrivate, proxyURL: proxyURL})
			}
		}
	}
	if err == nil {
		now := time.Now()
		if updateErr := db.Model(delivery).Updates(map[string]interface{}{"status": "succeeded", "sent_at": now, "next_attempt_at": nil, "last_error": "", "last_status_code": 0, "provider_error_code": "", "defer_reason": ""}).Error; updateErr != nil {
			logging.Error("notification", "保存推送成功状态失败", logging.Fields{"delivery_id": delivery.ID, "error": updateErr})
			return
		}
		logging.Info("notification", "推送发送成功", deliveryFields(delivery))
		return
	}
	var sendErr *deliveryError
	if errors.As(err, &sendErr) {
		if sendErr.rateLimited {
			if limitedAdapter, ok := defaultNotificationChannelRegistry.adapters[channel.Type].(notificationRateLimitedAdapter); ok {
				if policy, policyErr := limitedAdapter.RateLimitPolicy(&channel); policyErr == nil {
					if cooldownErr := applyNotificationRateLimitCooldown(db, policy, time.Now().Add(sendErr.retryAfter)); cooldownErr != nil {
						logging.Error("notification", "保存渠道限流冷却状态失败", logging.Fields{"channel_id": channel.ID, "channel_type": channel.Type, "error": cooldownErr})
					}
				}
			}
		}
		w.failDelivery(delivery, sendErr)
	} else {
		w.fail(delivery, 0, err, true)
	}
}

func (w *NotificationWorker) fail(delivery *model.NotificationDelivery, statusCode int, err error, retryable bool) {
	w.failDelivery(delivery, &deliveryError{message: err.Error(), statusCode: statusCode, retryable: retryable})
}

func (w *NotificationWorker) failDelivery(delivery *model.NotificationDelivery, sendErr *deliveryError) {
	fields := deliveryFields(delivery)
	fields["error"] = sendErr
	fields["attempt"] = delivery.AttemptCount
	if sendErr.providerCode != "" {
		fields["provider_error_code"] = sendErr.providerCode
	}
	updates := map[string]interface{}{"last_error": truncateNotificationError(sendErr.Error()), "last_status_code": sendErr.statusCode, "provider_error_code": sendErr.providerCode, "defer_reason": ""}
	if sendErr.retryable && delivery.AttemptCount <= maxNotificationRetries {
		delay := sendErr.retryAfter
		if delay <= 0 {
			delay = notificationRetryDelay(delivery.AttemptCount)
		}
		next := time.Now().Add(delay)
		updates["status"] = "retrying"
		updates["next_attempt_at"] = next
		if sendErr.rateLimited {
			updates["defer_reason"] = "provider_rate_limited"
			fields["next_attempt_at"] = next
			logging.Warn("notification", "推送渠道触发频率限制", fields)
		} else {
			logging.Warn("notification", "推送发送失败，等待重试", fields)
		}
	} else {
		updates["status"] = "failed"
		updates["next_attempt_at"] = nil
		logging.Error("notification", "推送发送最终失败", fields)
	}
	if updateErr := database.GetDB().Model(delivery).Updates(updates).Error; updateErr != nil {
		logging.Error("notification", "保存推送失败状态失败", logging.Fields{"delivery_id": delivery.ID, "error": updateErr})
	}
}

func notificationRetryDelay(attempt int) time.Duration {
	delays := []time.Duration{10 * time.Second, 30 * time.Second, time.Minute, 2 * time.Minute, 5 * time.Minute, 10 * time.Minute}
	if attempt < 1 {
		return delays[0]
	}
	if attempt > len(delays) {
		return delays[len(delays)-1]
	}
	return delays[attempt-1]
}

func reserveNotificationSendSlot(tx *gorm.DB, policy *notificationRateLimitPolicy, now time.Time) (bool, time.Time, error) {
	state := model.NotificationRateLimit{ScopeType: policy.ScopeType, ScopeKey: policy.ScopeKey}
	if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "scope_type"}, {Name: "scope_key"}}, DoNothing: true}).Create(&state).Error; err != nil {
		return false, time.Time{}, err
	}
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("scope_type = ? AND scope_key = ?", policy.ScopeType, policy.ScopeKey).First(&state).Error; err != nil {
		return false, time.Time{}, err
	}
	next := now
	if state.NextAllowedAt != nil && state.NextAllowedAt.After(next) {
		next = *state.NextAllowedAt
	}
	if state.CooldownUntil != nil && state.CooldownUntil.After(next) {
		next = *state.CooldownUntil
	}
	if next.After(now) {
		return false, next, nil
	}
	nextAllowed := now.Add(policy.MinInterval)
	if err := tx.Model(&state).Updates(map[string]interface{}{"next_allowed_at": nextAllowed}).Error; err != nil {
		return false, time.Time{}, err
	}
	return true, now, nil
}

func applyNotificationRateLimitCooldown(db *gorm.DB, policy *notificationRateLimitPolicy, until time.Time) error {
	return db.Transaction(func(tx *gorm.DB) error {
		state := model.NotificationRateLimit{ScopeType: policy.ScopeType, ScopeKey: policy.ScopeKey}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "scope_type"}, {Name: "scope_key"}}, DoNothing: true}).Create(&state).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("scope_type = ? AND scope_key = ?", policy.ScopeType, policy.ScopeKey).First(&state).Error; err != nil {
			return err
		}
		if state.CooldownUntil != nil && state.CooldownUntil.After(until) {
			until = *state.CooldownUntil
		}
		return tx.Model(&state).Updates(map[string]interface{}{"cooldown_until": until, "next_allowed_at": until}).Error
	})
}

func reserveNotificationSendSlotStandalone(db *gorm.DB, policy *notificationRateLimitPolicy, now time.Time) (allowed bool, next time.Time, err error) {
	err = db.Transaction(func(tx *gorm.DB) error {
		allowed, next, err = reserveNotificationSendSlot(tx, policy, now)
		return err
	})
	return
}

func deliveryFields(delivery *model.NotificationDelivery) logging.Fields {
	return logging.Fields{"delivery_id": delivery.ID, "event_id": delivery.EventID, "integration_id": delivery.IntegrationID, "channel_id": delivery.ChannelID, "channel_type": delivery.ChannelType}
}

func sendChannel(ctx context.Context, channel *model.NotificationChannel, message outboundMessage, allowPrivate bool, proxyURL string) error {
	adapter, ok := defaultNotificationChannelRegistry.Get(channel.Type)
	if !ok {
		return &deliveryError{message: "不支持的推送渠道类型"}
	}
	var err error
	message, err = prepareNotificationMessage(adapter, channel, message)
	if err != nil {
		return err
	}
	if limitedAdapter, ok := adapter.(notificationRateLimitedAdapter); ok {
		policy, policyErr := limitedAdapter.RateLimitPolicy(channel)
		if policyErr != nil {
			return policyErr
		}
		allowed, next, reserveErr := reserveNotificationSendSlotStandalone(database.GetDB(), policy, time.Now())
		if reserveErr != nil {
			return &deliveryError{message: "检查渠道发送窗口失败", retryable: true}
		}
		if !allowed {
			message := policy.WaitMessage
			if message == "" {
				message = "推送渠道正在限速，请稍后重试"
			}
			return &deliveryError{message: message, retryable: true, retryAfter: time.Until(next), rateLimited: true}
		}
	}
	err = adapter.Send(ctx, channel, message, notificationSendOptions{allowPrivate: allowPrivate, proxyURL: proxyURL})
	var sendErr *deliveryError
	if errors.As(err, &sendErr) && sendErr.rateLimited {
		if limitedAdapter, ok := adapter.(notificationRateLimitedAdapter); ok {
			if policy, policyErr := limitedAdapter.RateLimitPolicy(channel); policyErr == nil {
				_ = applyNotificationRateLimitCooldown(database.GetDB(), policy, time.Now().Add(sendErr.retryAfter))
			}
		}
	}
	return err
}

func messageForDelivery(adapter notificationChannelAdapter, channel *model.NotificationChannel, delivery *model.NotificationDelivery, message outboundMessage) (outboundMessage, error) {
	if delivery.ContentVersion > 0 && delivery.ContentFormat != "" {
		message.Rendered = &renderedNotification{
			Format:  notificationFormat(delivery.ContentFormat),
			Profile: delivery.ContentProfile,
			Version: delivery.ContentVersion,
			Title:   delivery.ContentTitle,
			Body:    delivery.ContentBody,
		}
		return prepareNotificationMessage(adapter, channel, message)
	}
	prepared, err := prepareNotificationMessage(adapter, channel, message)
	if err != nil || prepared.Rendered == nil {
		if err != nil {
			return message, err
		}
		return message, errNotificationSnapshotUnavailable
	}
	rendered := prepared.Rendered
	updates := map[string]interface{}{
		"content_format":  string(rendered.Format),
		"content_profile": rendered.Profile,
		"content_version": rendered.Version,
		"content_title":   rendered.Title,
		"content_body":    rendered.Body,
	}
	if err := database.GetDB().Model(delivery).Updates(updates).Error; err != nil {
		return message, &deliveryError{message: "保存推送内容快照失败", retryable: true}
	}
	delivery.ContentFormat = string(rendered.Format)
	delivery.ContentProfile = rendered.Profile
	delivery.ContentVersion = rendered.Version
	delivery.ContentTitle = rendered.Title
	delivery.ContentBody = rendered.Body
	return prepared, nil
}

func sendNotificationHTTPRequest(ctx context.Context, spec *notificationRequestSpec, options notificationSendOptions, validateResponse func(int, []byte) error) error {
	endpointAllowsPrivate := options.allowPrivate && !spec.requirePublicHost
	if err := validateOutboundURL(spec.endpoint, endpointAllowsPrivate); err != nil {
		return &deliveryError{message: err.Error()}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, spec.endpoint, bytes.NewReader(spec.body))
	if err != nil {
		return &deliveryError{message: "无法创建推送请求"}
	}
	for key, value := range spec.headers {
		request.Header.Set(key, value)
	}
	client, err := notificationHTTPClient(spec.requirePublicHost, options.allowPrivate, options.proxyURL)
	if err != nil {
		return &deliveryError{message: "系统 HTTP 代理配置无效"}
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 3 {
			return errors.New("推送重定向次数过多")
		}
		if spec.lockedHost != "" && !strings.EqualFold(req.URL.Hostname(), spec.lockedHost) {
			return errors.New("推送渠道禁止重定向到其他主机")
		}
		return validateOutboundURL(req.URL.String(), endpointAllowsPrivate)
	}
	response, err := client.Do(request)
	if err != nil {
		// Some channel credentials are embedded in the request URL. Do not persist
		// the HTTP client's raw error because it may include that URL.
		return &deliveryError{message: "请求推送渠道失败", retryable: true}
	}
	defer response.Body.Close()
	responseBody, _ := io.ReadAll(io.LimitReader(response.Body, notificationResponseLimit))
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return validateResponse(response.StatusCode, responseBody)
	}
	retryable := response.StatusCode == http.StatusRequestTimeout || response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= 500
	messageText := fmt.Sprintf("推送渠道返回 HTTP %d", response.StatusCode)
	return &deliveryError{message: messageText, statusCode: response.StatusCode, retryable: retryable}
}

func notificationHTTPClient(requirePublicHost bool, allowPrivate bool, proxyURL string) (*http.Client, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	if proxyURL != "" {
		parsedProxy, err := url.Parse(proxyURL)
		if err != nil || parsedProxy.Hostname() == "" || (parsedProxy.Scheme != "http" && parsedProxy.Scheme != "https") {
			return nil, ErrInvalidHTTPProxyURL
		}
		transport.Proxy = http.ProxyURL(parsedProxy)
	}
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, errors.New("推送目标地址无效")
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil || len(ips) == 0 {
			return nil, errors.New("无法解析推送目标地址")
		}
		dialAllowsPrivate := allowPrivate && !requirePublicHost
		if proxyURL != "" {
			dialAllowsPrivate = true
		}
		for _, ip := range ips {
			if err := validateOutboundIP(ip, dialAllowsPrivate); err != nil {
				return nil, err
			}
		}
		dialer := &net.Dialer{Timeout: 10 * time.Second}
		return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
	}
	client.Transport = transport
	return client, nil
}

func validateOutboundURL(rawURL string, allowPrivate bool) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Hostname() == "" {
		return errors.New("推送目标 URL 无效")
	}
	if parsed.Scheme != "https" && !(allowPrivate && parsed.Scheme == "http") {
		if allowPrivate {
			return errors.New("推送目标必须使用 HTTP 或 HTTPS")
		}
		return errors.New("推送目标必须使用 HTTPS")
	}
	ips, err := net.LookupIP(parsed.Hostname())
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

func validateOutboundIP(ip net.IP, allowPrivate bool) error {
	metadataIPv4 := net.ParseIP("100.100.100.200")
	metadataIPv6 := net.ParseIP("fd00:ec2::254")
	if ip.IsUnspecified() || ip.IsMulticast() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.Equal(metadataIPv4) || ip.Equal(metadataIPv6) {
		return errors.New("推送目标地址不允许访问")
	}
	if !allowPrivate && (ip.IsPrivate() || ip.IsLoopback()) {
		return errors.New("默认禁止访问私有网络推送目标")
	}
	return nil
}

func notificationNetworkPolicyForChannel(channel *model.NotificationChannel) (bool, string, error) {
	const allowPrivate = true
	config := map[string]interface{}{}
	_ = json.Unmarshal(channel.Config, &config)
	useProxy, _ := config["use_proxy"].(bool)
	if !useProxy {
		return allowPrivate, "", nil
	}
	proxyURL := NewSettingService().HTTPProxyURL()
	if proxyURL == "" {
		return allowPrivate, "", &deliveryError{message: "渠道已启用代理，但系统尚未配置 HTTP 代理"}
	}
	return allowPrivate, proxyURL, nil
}
func truncateNotificationError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 1000 {
		return value[:1000]
	}
	return value
}
