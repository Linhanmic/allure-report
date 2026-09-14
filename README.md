# allure-report

Gauge 测试框架的 **Allure 3** 报告插件。执行规格后会把 Suite 结果转换成 Allure 结果文件（`*-result.json`），并调用本机 Allure 3 CLI 生成单文件 Awesome HTML 报告。

## 功能

- 作为 Gauge Execution 插件接入，监听 `SuiteExecutionResult`
- 将 Specification / Scenario / Step / Concept / 数据表场景映射为 Allure 用例与步骤
- 支持 Gauge screenshot 插件截图、自定义消息、Hook、标签、失败堆栈
- 写出兼容 Allure 2/3 的 `allure-results`，截图复制进结果目录避免丢失
- 参考 html-report 支持覆盖 / 历史两种报告模式，覆盖前自动归档
- 默认生成单文件 HTML（`index.html`），便于本地双击打开

## 安装

从源码编译并安装到 Gauge 插件目录：

```bash
go run build/make.go --install
```

或先打包 zip，再离线安装：

```bash
make distro
gauge install allure-report --file deploy/allure-report-0.2.0-<os>.<arch>.zip
```

在 Gauge 项目的 `manifest.json` 中加入插件：

```json
{
  "Language": "js",
  "Plugins": ["screenshot", "allure-report"]
}
```

生成 HTML 需要本机安装 **Allure 3 CLI**（`allure`）或 **Node.js**（`npx allure@3`）。

## 使用

```bash
gauge run specs
```

### 报告目录（与 html-report 一致）

| `overwrite_reports` | 结果目录 | HTML 目录 |
| --- | --- | --- |
| `true`（默认） | `reports/allure-results/` | `reports/allure-report/` |
| `false` | `reports/allure-report/{时间戳}/allure-results/` | `reports/allure-report/{时间戳}/allure-report/` |

覆盖模式（`overwrite_reports=true`）在写入新报告前，会将当前 `allure-results` 与 `allure-report` **整体归档**到：

```
reports/archive/{时间戳}/allure-results/
reports/archive/{时间戳}/allure-report/
```

归档目录包含已复制的截图附件，避免 Gauge 临时截图目录被清理后报告丢失图片。

### 手动生成 HTML

若仅保留 `allure-results`，可稍后执行：

```bash
allure awesome reports/allure-results \
  --output reports/allure-report \
  --single-file \
  --report-language zh
```

## 配置

在 `env/default/default.properties` 中设置：

```properties
gauge_reports_dir = reports
overwrite_reports = true
screenshot_on_failure = true

allure_report_generate = true
allure_report_single_file = true
allure_report_language = zh
allure_report_theme = auto
allure_report_name = Gauge Allure Report
```

| 属性 | 说明 | 默认值 |
| --- | --- | --- |
| `gauge_reports_dir` | 报告根目录 | `reports` |
| `overwrite_reports` | `true` 覆盖并归档旧报告；`false` 按时间戳保留历史 | `true` |
| `allure_report_generate` | 是否生成 Allure 3 HTML | `true` |
| `allure_report_single_file` | 生成单文件 HTML | `true` |
| `allure_report_language` | 报告语言 | `zh` |
| `allure_report_theme` | 报告主题：`light` / `dark` / `auto` | `auto` |
| `allure_report_name` | 报告标题 | `Gauge Allure Report` |
| `allure_results_dir` | 自定义结果目录（跳过内置目录规则） | 空 |
| `allure_report_dir` | 自定义 HTML 目录 | 空 |
| `allure_archive_max_count` | 保留的归档数量（0 表示不限制） | `0` |

## 截图

插件从 Gauge screenshot 插件写入的 `gauge_screenshots_dir` 读取截图文件，复制到 `allure-results` 作为附件，并在报告中以原始文件名展示。

建议在项目中启用：

```properties
screenshot_on_failure = true
```

并在 `manifest.json` 安装 `screenshot` 插件。

## 开发

```bash
go test ./...
make build
make example
```

要求 Go 1.22+。生成 HTML 需要本机 Allure 3 或 Node.js。
