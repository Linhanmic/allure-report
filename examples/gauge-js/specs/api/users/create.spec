# 用户管理
tags: epic:账户, feature:用户管理

## 创建新用户
tags: severity:critical, owner:qa, smoke
* 打开用户管理页面
* 点击创建用户按钮
* 输入用户名 "testuser"
* 输入邮箱 "test@example.com"
* 点击保存按钮
* 验证用户创建成功

## 用户名已存在
tags: severity:normal, issue:BUG-45
* 打开用户管理页面
* 点击创建用户按钮
* 输入用户名 "admin"
* 输入邮箱 "admin@example.com"
* 点击保存按钮
* 验证错误提示 "用户名已存在"