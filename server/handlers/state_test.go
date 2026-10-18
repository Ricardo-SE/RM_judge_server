package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"rm-judge-server/store"
)

func setupServerTestRouter() (*gin.Engine, *store.MemoryStateStore) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	s := store.NewMemoryStateStore()
	h := NewStateHandler(s)
	h.RegisterRoutes(r)
	return r, s
}

// TestPingHandlers 验证健康检查接口
func TestPingHandlers(t *testing.T) {
	r, _ := setupServerTestRouter()

	for _, path := range []string{"/api/ping", "/api/v1/ping"} {
		req, _ := http.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("接口 %s 预期返回 200，实际返回: %d", path, w.Code)
		}
	}
}

// TestState_ETag_And_OCC 验证 RFC 7232 标准流程与并发冲突防线
func TestState_ETag_And_OCC(t *testing.T) {
	r, _ := setupServerTestRouter()

	// 1. 初始 GET 查询：验证 ETag 标头为 "1"
	reqGet, _ := http.NewRequest(http.MethodGet, "/api/state", nil)
	wGet := httptest.NewRecorder()
	r.ServeHTTP(wGet, reqGet)

	if wGet.Code != http.StatusOK {
		t.Fatalf("预期 GET 成功，实际返回: %d", wGet.Code)
	}
	etag := wGet.Header().Get("ETag")
	if etag != "\"1\"" {
		t.Fatalf("预期初始 ETag 为 \"1\"，实际为: %s", etag)
	}

	// 2. 主裁判带 If-Match: "1" 提交更新比分
	referee1Payload := `{"tournamentName":"2026机甲杯","redScore":10,"blueScore":8}`
	reqPut1, _ := http.NewRequest(http.MethodPut, "/api/state", bytes.NewBufferString(referee1Payload))
	reqPut1.Header.Set("Content-Type", "application/json")
	reqPut1.Header.Set("If-Match", "\"1\"")

	wPut1 := httptest.NewRecorder()
	r.ServeHTTP(wPut1, reqPut1)

	if wPut1.Code != http.StatusOK {
		t.Fatalf("主裁判首次提交预期 200，实际返回: %d, body: %s", wPut1.Code, wPut1.Body.String())
	}
	newEtag := wPut1.Header().Get("ETag")
	if newEtag != "\"2\"" {
		t.Fatalf("主裁判更新后预期 ETag 递增为 \"2\"，实际为: %s", newEtag)
	}

	// 3. 副裁判手慢，仍然拿着旧暗号 If-Match: "1" 试图提交
	referee2Payload := `{"tournamentName":"2026机甲杯","redScore":8,"blueScore":10}`
	reqPut2, _ := http.NewRequest(http.MethodPut, "/api/state", bytes.NewBufferString(referee2Payload))
	reqPut2.Header.Set("Content-Type", "application/json")
	reqPut2.Header.Set("If-Match", "\"1\"") // 旧暗号！

	wPut2 := httptest.NewRecorder()
	r.ServeHTTP(wPut2, reqPut2)

	// 断言：副裁判必须当场被 412 Precondition Failed 拦截！杜绝静默覆盖！
	if wPut2.Code != http.StatusPreconditionFailed {
		t.Fatalf("【并发事故】副裁判携带过期版本，预期被 412 拦截，实际返回: %d, body: %s", wPut2.Code, wPut2.Body.String())
	}
	t.Logf("【防线验证成功】副裁判被安全拦截: 412 Precondition Failed, 判罚数据零丢失！")

	// 4. 副裁判重新拉取最新数据，取得新暗号 "2" 后再次提交
	reqPut3, _ := http.NewRequest(http.MethodPut, "/api/state", bytes.NewBufferString(referee2Payload))
	reqPut3.Header.Set("Content-Type", "application/json")
	reqPut3.Header.Set("If-Match", "\"2\"") // 新暗号

	wPut3 := httptest.NewRecorder()
	r.ServeHTTP(wPut3, reqPut3)

	if wPut3.Code != http.StatusOK {
		t.Fatalf("副裁判基于最新版本提交预期 200，实际返回: %d", wPut3.Code)
	}
	finalEtag := wPut3.Header().Get("ETag")
	if finalEtag != "\"3\"" {
		t.Fatalf("最终版本号预期为 \"3\"，实际为: %s", finalEtag)
	}
}
