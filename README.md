#官网 WWWW.3Whello.com


# Go POC SDK

基于 YAML 配置的 Web 安全测试 SDK，支持 Cookie 处理、表达式评估和规则链式执行。

## 功能特性

- ✅ 解析 YAML POC 配置文件
- ✅ HTTP 请求执行（GET、POST 等）
- ✅ Cookie 提取和使用
- ✅ Rust 风格表达式评估
- ✅ 规则链式执行
- ✅ 重试机制
- ✅ 超时控制

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 编译项目

```bash
go build -o gopoc.exe ./cmd/gopoc
```

### 3. 使用示例

#### 基本用法

```bash
# 单个目标 + 单个 POC
./gopoc.exe -t https://www.example.com -poc test.yaml

# 批量目标（从文件）+ 单个 POC
./gopoc.exe -f targets.txt -poc test.yaml -v

# 单个目标 + 多个 POC（目录）
./gopoc.exe -t https://www.example.com -pocs ./pocs/

# 批量目标 + 多个 POC（逗号分隔的文件列表）
./gopoc.exe -f targets.txt -pocs poc1.yaml,poc2.yaml -v

# 批量目标 + 多个 POC（目录）
./gopoc.exe -f targets.txt -pocs ./pocs/ -v

# 保存结果到文件
./gopoc.exe -t https://www.example.com -poc test.yaml -o result.txt -v
```

#### 命令行参数说明

- `-t <URL>`: 指定单个目标 URL
- `-f <file>`: 指定目标 URL 列表文件（每行一个 URL，支持 `#` 注释）
- `-poc <file>`: 指定单个 POC 文件
- `-pocs <paths>`: 指定多个 POC 文件或目录
  - 逗号分隔的多个文件：`poc1.yaml,poc2.yaml`
  - 目录路径：会自动扫描目录下所有 `.yaml` 和 `.yml` 文件
- `-o` <file>: 保存结果到文件（包含详细扫描结果和统计信息）
- `-v`: 详细输出模式

#### 目标文件格式（-f）

```
# 这是注释
https://www.example.com
http://192.168.1.1:8080
https://target.com
# 空行会被忽略
```

#### 保存结果到文件

结果文件格式简洁，每行一个结果：

```
[成功] https://www.example.com | robots.txt | test.yaml
[失败] https://target.com | poc_name | poc.yaml
```

格式：`[状态] 目标URL | POC名称 | POC文件路径`

```bash
./gopoc.exe -t https://www.example.com -poc test.yaml -o result.txt
```

#### 开发时运行

```bash
go run ./cmd/gopoc -t https://www.example.com -poc test.yaml -v
```

### 4. 代码示例

```go
package main

import (
    "fmt"
    "github.com/gopoc/sdk"
)

func main() {
    // 加载配置
    config, err := sdk.LoadConfig("test.yaml")
    if err != nil {
        panic(err)
    }

    // 创建执行引擎
    engine := sdk.NewEngine(config, "http://target.com")

    // 执行 POC
    success, err := engine.Execute()
    if err != nil {
        panic(err)
    }

    if success {
        fmt.Println("POC 执行成功")
    } else {
        fmt.Println("POC 执行失败")
    }
}
```

## 配置文件格式

### 基本结构

```yaml
name: "POC 名称"
author: "作者"
category: "类别"
cve_id: "CVE-2024-0001"
level: "高危"
source: "来源"
rules:
  r0:
    method: "POST"
    path: "/login"
    timeout: 10
    retry_count: 3
    headers:
      user-agent: "Mozilla/5.0..."
    body:
      - "username=admin&password=admin"
    extract_cookie: "response.headers.get('Set-Cookie')"
    expression: "response.status==200 && response.body.contains('admin')"
  
  r1:
    method: "GET"
    path: "/admin"
    use_cookie: "session_id=abc123"
    cookie_expression: "cookie.contains('session_id')"
    expression: "response.status==200"

expression: "r0() && r1()"
```

### 字段说明

#### 规则字段

- `method`: HTTP 方法（GET、POST、PUT、DELETE 等）
- `path`: 请求路径
- `timeout`: 超时时间（秒）
- `retry_count`: 重试次数
- `headers`: HTTP 请求头
- `body`: 请求体（字符串数组）
- `extract_cookie`: Cookie 提取表达式
- `use_cookie`: 使用的 Cookie 字符串或 `response.extracted_cookie`
- `cookie_expression`: Cookie 验证表达式
- `expression`: 响应验证表达式

#### 表达式语法

##### 基本比较
```
response.status==200
response.status!=404
response.status>=200
response.status<500
```

##### 字符串包含
```
response.body.contains('admin')
response.body.contains("success")
```

##### Cookie 验证
```
cookie.contains('session_id')
```

##### 头部提取
```
response.headers.get('Set-Cookie')
```

##### 逻辑运算
```
response.status==200 && response.body.contains('admin')
response.status==200 || response.status==302
```

## API 文档

### LoadConfig

加载 YAML 配置文件。

```go
func LoadConfig(filePath string) (*POCConfig, error)
```

### NewEngine

创建执行引擎。

```go
func NewEngine(config *POCConfig, baseURL string) *Engine
```

### Execute

执行整个 POC。

```go
func (e *Engine) Execute() (bool, error)
```

### NewHTTPClient

创建 HTTP 客户端。

```go
func NewHTTPClient(baseURL string) *HTTPClient
```

### ExecuteRequest

执行 HTTP 请求。

```go
func (c *HTTPClient) ExecuteRequest(opts RequestOptions) (*Response, error)
```

## 示例输出

```
==================================================
✓ POC 执行成功: 插件可配置Cookie处理演示
```

或

```
==================================================
✗ POC 执行失败: 插件可配置Cookie处理演示

规则执行详情:
  ✓ r0: true
  ✗ r1: false
  ✓ r2: true
```

## 依赖

- `gopkg.in/yaml.v3` - YAML 解析
- `github.com/google/uuid` - UUID 生成（如需要）

## 开发计划

- [ ] 支持更多表达式函数（matches、extract、count 等）
- [ ] 支持正则表达式提取
- [ ] 支持变量存储和引用
- [ ] 支持 JSON 请求体
- [ ] 支持代理配置
- [ ] 添加单元测试

## 许可证

MIT

