---
title: 默认配置文件
sourceHash: 1ef0edd24fb4
---

# 默认配置文件

Dozzle 会把每个用户的界面偏好（主题、语言、置顶的容器、折叠的分组、可见的 JSON 键等）持久化到磁盘上的 `/data/<username>/profile.json`。当[身份验证](/zh/guide/authentication)被禁用时，或者对于任何尚未登录并自定义设置的用户，Dozzle 会回退到一个名为 `__default__` 的特殊配置文件。

你可以通过创建 `/data/__default__/profile.json` 文件来预先配置一份配置文件。匿名访客以及任何还没有保存过配置的新用户，在首次访问时都会加载这些设置。

## 文件位置

```
/data/__default__/profile.json
```

如果该文件不存在，Dozzle 会使用内置的默认值启动。只有当你想覆盖这些默认值时才需要创建它。

## 示例

```json
{
  "settings": {
    "showTimestamp": true,
    "showStd": false,
    "showAllContainers": false,
    "softWrap": true,
    "collapseNav": false,
    "smallerScrollbars": false,
    "search": false,
    "compact": false,
    "menuWidth": 15,
    "size": "medium",
    "lightTheme": "auto",
    "hourStyle": "auto",
    "dateLocale": "auto",
    "locale": "en",
    "groupContainers": "at-least-2",
    "automaticRedirect": "delayed"
  },
  "pinned": [],
  "visibleKeys": [],
  "collapsedGroups": []
}
```

所有字段都是可选的，只需要写上你想覆盖的那些。

## 可用设置

| 字段                | 类型    | 说明                                                      |
| ------------------- | ------- | --------------------------------------------------------- |
| `showTimestamp`     | boolean | 在每行日志旁显示时间戳                                    |
| `showStd`           | boolean | 显示 stdout/stderr 流标识                                 |
| `showAllContainers` | boolean | 在侧边栏中包含已停止的容器                                |
| `softWrap`          | boolean | 折行显示过长的日志，而不是横向滚动                        |
| `collapseNav`       | boolean | 启动时折叠侧边栏                                          |
| `smallerScrollbars` | boolean | 使用更细的滚动条                                          |
| `search`            | boolean | 默认启用行内搜索                                          |
| `compact`           | boolean | 使用紧凑的日志行间距                                      |
| `menuWidth`         | number  | 侧边栏宽度占窗口的百分比，上限为 `50`。                   |
| `size`              | string  | 字体大小：`small`、`medium`、`large`                      |
| `lightTheme`        | string  | 主题偏好：`auto`、`light`、`dark`                         |
| `hourStyle`         | string  | 时间格式：`auto`、`12`、`24`                              |
| `dateLocale`        | string  | 日期/时间格式：`auto`、`en-US`、`en-GB`、`de-DE`、`en-CA` |
| `locale`            | string  | 界面语言（例如 `en`、`fr`、`de`）                         |
| `groupContainers`   | string  | 侧边栏分组方式：`always`、`at-least-2`、`never`           |
| `automaticRedirect` | string  | 跳转到新容器的方式：`instant`、`delayed`、`none`          |

这些集合之外的值不会被接受，所以 `groupContainers: "stack"` 或者写成 `fr-FR` 的 `dateLocale` 不会产生你期望的效果。

顶层字段 `pinned`、`visibleKeys` 和 `collapsedGroups` 接受数组，可以为首次访问的用户预先置顶容器或预先折叠分组。Dozzle 还会在顶层写入 `releaseSeen`、`dismissedImageUpdates` 和 `dismissedLinkHint`，用来记住用户已经关闭过哪些提示。预置 `dismissedLinkHint: true` 会对所有人隐藏首次运行时的链接提示。

## 工作原理

- 页面加载时，Dozzle 会读取已登录用户的 `/data/<username>/profile.json`；如果没有用户通过身份验证，则读取 `/data/__default__/profile.json`。
- 当用户在界面中修改设置时，新值会保存到他自己的用户名下（在关闭身份验证时则写回 `__default__`）。
- 因此在未启用身份验证的部署中，`__default__` 配置文件既是**新访客的模板**，也是**匿名用户的实时配置文件**。

::: tip
如果你只想预置默认值，同时仍允许匿名用户在运行时自行修改，可以以只读方式挂载该文件。这样 Dozzle 无法保存修改，但界面仍能正常工作。
:::
