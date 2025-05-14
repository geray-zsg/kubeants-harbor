# 功能说明：
## 获取所有项目：

请求：GET /api/projects

返回：项目列表

## 获取项目仓库：

请求：GET /api/projects/{project}/repositories

参数：project需要URL编码（如包含/需编码为%2F）

返回：仓库列表

## 获取制品信息：

请求：GET /api/projects/{project}/repositories/{repository}/artifacts

参数：project和repository都需要URL编码

返回：包含标签信息的制品列表

## 复制镜像：

请求：POST /api/copy-image

请求体示例：
```
{
  "src_project": "source-project",
  "src_repo": "source-repo",
  "src_tag": "latest",
  "dest_project": "dest-project",
  "dest_repo": "dest-repo",
  "dest_tag": "v1.0"
}
```
# 使用说明：
安装依赖：

bash
go get -u github.com/gin-gonic/gin
运行服务：

bash
go run main.go
测试接口（示例）：

bash
# 获取所有项目
curl http://localhost:8080/api/projects

# 获取项目仓库
curl http://localhost:8080/api/projects/library/repositories

# 获取制品信息
curl http://localhost:8080/api/projects/library/repositories/nginx/artifacts

# 复制镜像
curl -X POST http://localhost:8080/api/copy-image \
  -H "Content-Type: application/json" \
  -d '{
    "src_project": "library",
    "src_repo": "nginx",
    "src_tag": "latest",
    "dest_project": "test",
    "dest_repo": "nginx-copy",
    "dest_tag": "v1.0"
  }'
  
 # 注意事项：
确保Harbor版本为v2.x

处理包含特殊字符的项目/仓库名称时需正确编码

生产环境应配置有效证书并移除InsecureSkipVerify

复制操作需要目标项目存在且具有相应权限

分页查询最大页大小可根据实际情况调整



# 代码推送
```bash
git init
git add *
git commit -m "镜像制品复制"
git branch -M main
git tag v0.0.1
git tag v0.0.1
git remote add origin git@github.com:geray-zsg/kubeants-harbor.git
git push -u origin main v0.0.1
```
