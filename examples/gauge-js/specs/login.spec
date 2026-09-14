# 用户认证
tags: epic:账户, feature:登录

This specification exercises Allure mapping for tags, messages, failures and data tables.

## 管理员可以登录
tags: severity:critical, owner:qa, smoke
* 打开首页
* 输入用户名 "admin"
* 输入密码 "secret"
* 登录成功

## 错误密码被拒绝
tags: severity:normal, issue:BUG-12
* 打开首页
* 输入用户名 "admin"
* 输入密码 "wrong"
* 登录失败并提示 "密码错误"

## 数据驱动登录
   |user  |role  |
   |------|------|
   |alice |admin |
   |bob   |user  |
* 使用用户 <user> 和角色 <role> 登录
