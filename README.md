# aliyun-workbench-cli

[English](README.en.md)

**免责声明：非阿里云官方项目！**


使用 `aliyun-workbench-cli` 可以在本地终端中通过阿里云 ECS Workbench 连接 ECS 实例。

## 安装

从源码构建：

```bash
go build -o $HOME/.local/bin/aliyun-workbench ./cmd/aliyun-workbench

export PATH=$HOME/.local/bin:$PATH

aliyun-workbench --help

aliyun-workbench --version
```

## 认证

在浏览器中登录账号，然后从 cookie 中找到名为 **login_aliyunid_ticket** 的 cookie，使用该 cookie 值设置环境变量 **LOGIN_ALIYUNID_TICKET**：

```bash
export LOGIN_ALIYUNID_TICKET='<login_aliyunid_ticket>'
```

**注意：因 cookie 中可能包含特殊字符如 `$`，因此必须使用单引号包裹 cookie 值。**

## 使用

**强烈建议在 ECS 上使用 tmux 等终端复用工具，因为 cookie 到期后连接会自动断开，正在运行的前台命令会被直接终止。**

列出指定 region 下的 ECS 实例：

```bash
aliyun-workbench list --region cn-beijing
```

输出示例：

```text
INSTANCE_ID  NAME          STATUS   PRIVATE_IP  PUBLIC_IP  OS
i-...        example-ecs   Running  10.0.0.1    1.2.3.4    Linux
```

登录指定 ECS 实例：

```bash
aliyun-workbench login --region cn-beijing --instance-id i-...
```

指定远端系统用户名：

```bash
aliyun-workbench login --region cn-beijing --instance-id i-... --username root
```

开启详细日志用于排查问题：

```bash
aliyun-workbench --debug login --region cn-beijing --instance-id i-...
```


## 功能状态

- [x] 按 region 列出 ECS 实例。
- [x] 指定 region 和实例 ID，通过 ECS Workbench 打开交互式 SSH 终端。
- [ ] 文件上传。
- [ ] 文件下载。
