# allure-report

Gauge 测试框架的 **Allure 3** 报告插件。执行规格后会把 Suite 结果转换成 Allure 结果文件（`*-result.json`），并调用本机 Allure 3 CLI 生成多种类型的 HTML 报告。

## 功能

- 作为 Gauge Execution 插件接入，监听 `SuiteExecutionResult`
- 将 Specification / Scenario / Step / Concept / 数据表场景映射为 Allure 用例与步骤
- 支持 Gauge screenshot 插件截图、自定义消息、Hook、标签、失败堆栈
- 写出兼容 Allure 2/3 的 `allure-results`，截图复制进结果目录避免丢失
- 参考 html-report 支持覆盖 / 历史两种报告模式，覆盖前自动归档
- 支持生成多种类型的报告：Awesome、Classic、Allure2、Dashboard、CSV、Log
- 支持历史记录功能，包括趋势图表和不稳定测试检测
- 默认生成单文件 HTML（`index.html`），便于本地双击打开

## 安装

从源码编译并安装到 Gauge 插件目录：

```bash
go run build/make.go --install
```

或先打包 zip，再离线安装：

```bash
make distro
gauge install allure-report --file deploy/allure-report-0.4.0-<os>.<arch>.zip
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

## 多报告配置

支持生成多种类型的 Allure 3 报告，通过 `allure_reports` 配置项指定启用的报告类型。

### 可用报告类型

| 类型 | 说明 | 默认启用 |
| --- | --- | --- |
| awesome | 现代 UI（推荐） | 是 |
| classic | 经典样式 | 否 |
| allure2 | Allure 2 样式 | 否 |
| dashboard | 仪表板视图 | 否 |
| csv | CSV 导出 | 否 |
| log | 控制台日志 | 否 |

### 多报告配置示例

```properties
# 启用多个报告（逗号分隔）
allure_reports = awesome,classic,dashboard

# 各报告类型开关
allure_awesome_enabled = true
allure_classic_enabled = true
allure_dashboard_enabled = true
allure_allure2_enabled = false
allure_csv_enabled = false

# 各报告类型名称
allure_awesome_name = Gauge Awesome Report
allure_classic_name = Gauge Classic Report
allure_dashboard_name = Gauge Dashboard

# Awesome 插件特有配置
allure_awesome_group_by = epic,feature,story
```

### 多报告目录结构

```
reports/allure-report/
├── awesome/           # Awesome 报告
│   └── index.html
├── classic/           # Classic 报告
│   └── index.html
└── dashboard/         # Dashboard 报告
    └── index.html
```

## 历史记录配置

通过 `allure_history_path` 配置启用历史记录功能，支持趋势图表和不稳定测试检测。

```properties
# 历史记录配置
allure_history_path = ./history.jsonl
allure_history_append = true
allure_history_limit = 0
```

| 属性 | 说明 | 默认值 |
| --- | --- | --- |
| `allure_history_path` | 历史记录文件路径（JSONL 格式） | 空（不启用） |
| `allure_history_append` | 是否追加历史记录 | `true` |
| `allure_history_limit` | 历史记录限制数量（0 表示不限制） | `0` |

### 历史记录功能

启用历史记录后，报告将包含：

- **趋势图表** - 显示测试结果随时间的变化趋势
- **不稳定测试检测** - 识别在多次运行中状态变化的测试
- **历史比较** - 与之前的运行结果进行比较
- **状态转换** - 跟踪测试状态的变化（如从 passed 变为 failed）

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