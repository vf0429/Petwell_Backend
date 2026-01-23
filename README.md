# PetWell Backend

PetWell App 的后端服务仓库。目前这是一个轻量级的 Python 服务，主要用于为 PetWell iOS 客户端提供静态数据（如疫苗信息），未来将扩展为包含 AI 代理和业务逻辑的完整后端。

## 📁 目录结构

```
Backend/
├── server.py        # 简单的 HTTP 服务器入口
├── vaccines.json    # 疫苗静态数据源
└── README.md        # 说明文档
```

## 🚀 快速开始

### 1. 运行环境
- Python 3.x

### 2. 启动服务
在终端中进入 `Backend` 目录并运行：

```bash
python3 server.py
```

服务默认运行在 `http://localhost:8000`。

### 3. API 接口说明

#### 获取疫苗列表
- **Endpoint**: `/vaccines`
- **Method**: `GET`
- **Response**: JSON 数组，包含疫苗名称、描述、接种时间等信息。

**测试命令**:
```bash
curl http://localhost:8000/vaccines
```

## 📱 前端适配指南

为了让 iOS 前端 (`PetWell`) 能够连接到此后端，请确保以下配置正确：

### 1. 疫苗服务 (VaccineService)
前端文件: `Services/VaccineService.swift`

确保 `urlString` 指向正确的后端地址：
```swift
private let urlString = "http://localhost:8000/vaccines" 
// 如果在真机调试，请将 localhost 替换为电脑的局域网 IP
```

### 2. AI 服务 (AIService) - *待开发*
前端文件: `Services/AIService.swift`

目前前端代码中包含 AI 问诊功能的配置。为了安全起见（隐藏 API Key），我们需要在后端实现 AI 代理接口。

**维护计划**:
- [ ] 在后端实现 `/ai/triage` 或类似接口。
- [ ] 后端持有 OpenAI/DeepSeek API Key。
- [ ] 接收前端的 prompt，转发给 LLM 提供商，并将结果返回给前端。
- [ ] 更新 `AIService.swift` 中的 `Endpoint` 配置为 `.backend(url: ...)`。

### 3. 保险服务 (InsuranceService) - *待迁移*
前端文件: `Services/InsuranceService.swift`

目前保险数据存储在客户端的 SQLite (`insurance_list_test.sql`) 中。未来可以考虑将保险产品数据迁移至后端数据库，通过 API 下发，以便动态更新保险产品信息。

## 🛠 开发与维护

1. **修改数据**: 直接编辑 `vaccines.json` 即可更新疫苗列表，无需重启服务（每次请求都会重新读取文件）。
2. **扩展功能**: 建议未来引入 Flask 或 FastAPI 框架来替代当前的 `http.server`，以支持更复杂的路由和数据库连接。
