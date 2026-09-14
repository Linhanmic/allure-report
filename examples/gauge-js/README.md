# Gauge JS 示例项目

用于验证 `allure-report` 插件的最小 Gauge 项目，不依赖浏览器。

覆盖场景：

- 通过 / 失败 / 未实现（跳过）
- Concept 嵌套步骤
- 数据表驱动
- 标签（severity / owner / epic / issue）
- 自定义消息与失败截图

## 运行

先安装插件：

```bash
# 在仓库根目录
go run build/make.go --install
# 或
gauge install allure-report --file deploy/allure-report-0.1.0-linux.x86_64.zip
```

再执行规格：

```bash
cd examples/gauge-js
gauge run specs
```

报告输出：

- `reports/allure-results/`
- `reports/allure-report/index.html`（需本机有 Allure 3 或 Node.js / npx）
