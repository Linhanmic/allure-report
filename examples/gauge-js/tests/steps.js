/* globals gauge, step, beforeSuite */
"use strict";

const assert = require("assert");
const fs = require("fs");
const path = require("path");
const { Buffer } = require("buffer");

const PNG_1X1 = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
  "base64"
);

beforeSuite(function () {
  gauge.message("suite setup");
});

gauge.customScreenshotWriter = function () {
  const dir = process.env.gauge_screenshots_dir;
  fs.mkdirSync(dir, { recursive: true });
  const fileName = `screenshot-${Date.now()}.png`;
  fs.writeFileSync(path.join(dir, fileName), PNG_1X1);
  return fileName;
};

step("打开首页", function () {
  gauge.message("页面加载完成");
});

step("输入用户名 <user>", function (user) {
  gauge.message("user=" + user);
});

step("输入密码 <password>", function (password) {
  gauge.dataStore.scenarioStore.put("password", password);
});

step("登录成功", function () {
  assert.strictEqual(gauge.dataStore.scenarioStore.get("password"), "secret");
});

step("登录失败并提示 <msg>", function (msg) {
  const fileName = gauge.customScreenshotWriter();
  gauge.message("截图: " + fileName);
  throw new Error("expected status 401, got 200: " + msg);
});

step("使用用户 <user> 和角色 <role> 登录", function (user, role) {
  gauge.message(user + " as " + role);
  assert.ok(user.length > 0);
  assert.ok(role.length > 0);
});

step("选择支付方式 <method>", function (method) {
  gauge.message("pay with " + method);
});

step("确认订单", function () {
  assert.ok(true);
});
