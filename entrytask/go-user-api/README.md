## 功能接口
- POST   /api/v1/users      创建用户
- GET    /api/v1/users      查询所有用户
- GET    /api/v1/users/:id  查询单个用户
- PUT    /api/v1/users/:id  更新用户
- DELETE /api/v1/users/:id  删除用户

## 环境
- Go 1.26.1
- MySQL 8.0+
- 已创建数据库：user_db


## 快速运行
### 1. 配置环境变量
修改根目录 .env：
DB_USER=root
DB_PASSWORD=keyao001124
DB_HOST=127.0.0.1
DB_PORT=3306
DB_NAME=user_db
SERVER_PORT=8080

### 2. 安装依赖
go mod tidy

### 3. 启动服务
go run cmd/api/main.go

服务启动地址：http://localhost:8080

### 4. 单元测试
单元测试文件：go-user-api/main_test.go

### 5.运行单元测试
go test -v

## API接口文档
方法	     接口	       描述	         状态码
POST	/users	       创建用户	     201 Created
GET	    /users	       列出所有用户	 200 OK
GET	    /users/{id}	   获取单个用户	 200 OK / 404 Not Found
PUT	    /users/{id}	   更新用户信息	 200 OK / 404 Not Found
DELETE	/users/{id}	   删除用户	     204 No Content

## 接口示例
### 创建用户
curl -X POST http://localhost:8080/users \
-H "Content-Type: application/json" \
-d '{"username":"keyao","email":"keyao@example.com","password":"123456"}'
### 列出所有用户
curl http://localhost:8080/users
### 获取单个用户
curl http://localhost:8080/users/1
### 更新用户
curl -X PUT http://localhost:8080/users/1 \
-H "Content-Type: application/json" \
-d '{"username":"keyao_new","email":"keyao_new@example.com","password":"654321"}'
### 删除用户
curl -X DELETE http://localhost:8080/users/1