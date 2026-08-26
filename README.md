# 29-media-watermark-fingerprinting

纯Go数字水印与媒体指纹分析服务。它提供媒体分片续传、容器元数据解析、纯Go模拟音视频解码、分段感知指纹、鲁棒候选匹配、签名水印嵌入/检测、证据时间线和可取消异步任务。

## 快速启动

```bash
go mod tidy
gofmt -w .
go run ./cmd/server -config configs/config.yaml
```

HTTP监听`8090`，gRPC监听`9090`。除`/healthz`、`/readyz`、`/metrics`外，请求需要`X-API-Key: development-key`。

```bash
curl -i http://localhost:8090/healthz
curl -i http://localhost:8090/readyz
curl -X POST -H 'X-API-Key: development-key' -H 'X-Asset-Name: sample.wav' --data-binary @sample.wav http://localhost:8090/v1/assets
```

## 主要流程

1. `POST /v1/assets`上传完整媒体；`POST /v1/uploads`配合`Upload-Offset`和`Upload-Final`用于分片续传。
2. `POST /v1/assets/{id}/fingerprints`，JSON体为`{"kind":"audio"}`或`{"kind":"video"}`。
3. `POST /v1/assets/{id}/watermarks/embed`嵌入带HMAC签名的水印；`POST /v1/watermarks/detect`验证并区分`absent`、`corrupt`、`unsupported`。
4. `POST /v1/jobs`创建分析任务，`POST /v1/jobs/{id}/cancel`取消排队或运行中的任务。失败达到3次进入`dead`状态。
5. `POST /v1/nodes/current/drain`排空节点；新任务拒绝，运行任务完成后节点转为`stopped`。

## 工程说明

存储、解码器、指纹算法、水印算法、密钥、队列、时钟和节点均通过接口或构造函数注入。默认使用内存适配器以便无外部依赖启动；`migrations/001_init.sql`提供PostgreSQL表结构，`api/openapi.yaml`和`api/media.proto`是HTTP/gRPC契约。配置支持简化YAML和`MWF_*`环境变量覆盖。HTTP包含请求ID、认证、恢复、超时、限流、大小限制、统一错误体和结构化日志。

## 验证

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
```

非测试Go源码目标不少于2300行；测试、迁移、配置和构建产物不计入统计。
