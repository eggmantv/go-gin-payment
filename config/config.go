package config

const (
	APIPort = ":5011"
)

var Env string
var WebURL string
var SelfAPIURL string

const WebAPISecret = "xxx"

func Parse(e string) {
	Env = e

	if IsPrd() {
		WebURL = "https://eggman.tv"
		SelfAPIURL = "https://xx.eggman.com"
	} else {
		WebURL = "http://localhost:5010"
		// 这个地址服务器端配置了转发到ssh tunnel，再转发到本地，用于开发测试
		// nginx -> ssh tunnel -> local dev
		SelfAPIURL = "https://xx.eggman.com"
	}
}

func IsPrd() bool {
	return Env == "production"
}
