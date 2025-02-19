package api

import (
	"encoding/json"
	"flag"
	"go-gin-payment/cmd/cmd_lib"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApiAuthFailed(t *testing.T) {
	router := RunAPI()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/ping", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, 401, w.Code)
}

func TestApiWechatNativePay(t *testing.T) {
	e := flag.String("e", "development", "production | development")
	flag.Parse()

	cleaner := cmd_lib.SetupLog(*e)
	defer cleaner()
	router := RunAPI()

	w := httptest.NewRecorder()
	data := make(map[string]interface{})
	data["store_id"] = "21"
	data["payment_account_id"] = "3"
	data["trans_no"] = "2ss1e32q0ok87pfxui2"
	data["app_id"] = "wx12345678972ca148"
	data["total_price"] = 29800
	data["desp"] = "蛋人网年度订阅"

	d, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", "/wechat/native_pay", strings.NewReader(string(d)))
	req.Header.Add("X_GGP_KEY", "xxx")
	router.ServeHTTP(w, req)
	assert.Equal(t, 200, w.Code)
}
