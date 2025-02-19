在当今的互联网应用中，支付业务无疑是核心功能之一。本专题将带你使用 Go 语言中的 Gin 框架，以微信支付为例，从零开始构建一个完整的支付业务API。本专题大致内容有：

- Web 常用框架
- 常用 API 调试工具
- Gin 框架使用
	- 路由
	- 请求/响应格式
	- 中间件
	- basic auth, 跨域
- 日志
- 异常捕获
- 项目结构推荐
- 单元测试
- 使用 Docker 部署
- API文档生成

无论你是刚接触 Gin 框架，还是希望进一步提升后端开发技能，本文都将为你提供实用的指导和最佳实践，帮助你在支付业务API的开发中游刃有余。

## 开发环境启动项目
```shell
./start_dev.sh
```

## 测试

```shell
go test -v ./...
go test -v ./jobs/api
```

## 生产环境部署项目
```shell
# 先使用./docker-build.sh生成镜像
./docker-build.sh

# 然后使用./docker-deploy.sh远程拉取并启动容器
./docker-deploy.sh
```
