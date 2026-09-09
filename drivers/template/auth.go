package template

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"litepan/internal/domain"
	"litepan/internal/driver"
	"litepan/internal/httpx"
)

const oauthRefreshPath = "/api/oauth/refresh"

func (d *Driver) RefreshAuth(ctx context.Context, _ driver.RefreshCaller) (driver.RefreshOutcome, error) {
	_, err := d.doRefresh(ctx)
	if err != nil {
		return classifyRefreshError(err), err
	}
	return driver.RefreshSuccess, nil
}

func classifyRefreshError(err error) driver.RefreshOutcome {
	// 0.0.17：对齐统一分类契约——骨架驱动的特定致命提示之外，
	// 一律交由 ClassifyOAuthRefreshError（AuthRefreshError/IsAuthExpiredError）判定，
	// 保证与守卫体系（refreshInline/状态机冷却分级）语义一致。
	if ae, ok := domain.AsAppError(err); ok {
		msg := strings.ToLower(ae.Message)
		if strings.Contains(msg, "不能都为空") || strings.Contains(msg, "缺少 refresh_token") {
			return driver.RefreshFatal
		}
	}
	return driver.ClassifyOAuthRefreshError(err)
}

func (d *Driver) oauthServer() string {
	if s := strings.TrimSpace(d.oauthBase); s != "" {
		return strings.TrimRight(s, "/")
	}
	return domain.DefaultOAuthServerURL
}

func (d *Driver) doRefresh(ctx context.Context) (string, error) {
	return d.RefreshToken(ctx, d.currentToken, d.exchangeToken, driver.ClassifyOAuthRefreshError)
}

func (d *Driver) exchangeToken(ctx context.Context) (string, error) {
	d.mu.Lock()
	refresh := strings.TrimSpace(d.refresh)
	d.mu.Unlock()
	if refresh == "" {
		return "", domain.Errorf(domain.CodeAuthExpired, "缺少 refresh_token，无法刷新访问令牌")
	}

	var env struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
		} `json:"data"`
	}
	body := map[string]string{
		"driver_type":   config.OAuthName,
		"refresh_token": refresh,
	}
	if err := d.postOAuthJSON(ctx, d.oauthServer()+oauthRefreshPath, body, &env); err != nil {
		return "", err
	}
	if !env.Success || env.Data.AccessToken == "" {
		msg := env.Message
		if msg == "" {
			msg = "刷新访问令牌失败"
		}
		return "", httpx.OAuthProxyResponseError(msg)
	}

	d.mu.Lock()
	d.token = env.Data.AccessToken
	if env.Data.RefreshToken != "" {
		d.refresh = env.Data.RefreshToken
	}
	token := d.token
	refreshTok := d.refresh
	d.mu.Unlock()

	if d.persist != nil {
		_ = d.persist(ctx, domain.AuthCredentials{
			AccessToken:  token,
			RefreshToken: refreshTok,
		})
	}
	return token, nil
}

func (d *Driver) postOAuthJSON(ctx context.Context, url string, body, out any) error {
	if err := d.waitOperationDelay(ctx); err != nil {
		return err
	}
	resp, data, err := httpx.DoJSON(ctx, d.client, http.MethodPost, url, nil, body, map[string]string{
		"User-Agent": httpx.DefaultUserAgent,
	}, httpx.DefaultReadLimit)
	if err != nil {
		return domain.Wrap(domain.CodeDriverError, err)
	}
	if resp.StatusCode != http.StatusOK {
		// 0.0.17：对齐统一代理契约——401=认证失效、429=限流、403=权限，其余驱动错误。
		// 守卫体系（refreshInline/状态机）按该语义分类冷却与恢复。
		return httpx.OAuthProxyHTTPError(resp.StatusCode, string(data))
	}
	if err := json.Unmarshal(data, out); err != nil {
		return domain.Wrap(domain.CodeDriverError, err)
	}
	return nil
}
