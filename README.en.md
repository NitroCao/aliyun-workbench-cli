# aliyun-workbench-cli

[中文](README.md)

**Disclaimer: this is not an official Alibaba Cloud project!**

Use `aliyun-workbench-cli` to connect to ECS instances from your local terminal through Alibaba Cloud ECS Workbench.

## Installation

```bash
go install github.com/nitrocao/aliyun-workbench-cli/cmd/aliyun-workbench@latest

aliyun-workbench --help

aliyun-workbench --version
```

## Authentication

Log in to your Alibaba Cloud account in a browser, then find the cookie named **login_aliyunid_ticket** and set its value as the **LOGIN_ALIYUNID_TICKET** environment variable:

```bash
export LOGIN_ALIYUNID_TICKET='<login_aliyunid_ticket>'
```

**Note: the cookie may contain special characters such as `$`, so the value must be wrapped in single quotes.**

## Usage

**It is strongly recommended to use terminal multiplexers such as tmux on the ECS instance. When the cookie expires, the connection will be closed automatically, and foreground commands will be terminated directly.**

List ECS instances in a region:

```bash
aliyun-workbench list --region cn-beijing
```

Example output:

```text
INSTANCE_ID  NAME          STATUS   PRIVATE_IP  PUBLIC_IP  OS
i-...        example-ecs   Running  10.0.0.1    1.2.3.4    Linux
```

Log in to an ECS instance:

```bash
aliyun-workbench login --region cn-beijing --instance-id i-...
```

Specify the remote OS username:

```bash
aliyun-workbench login --region cn-beijing --instance-id i-... --username root
```

Enable verbose logs for troubleshooting:

```bash
aliyun-workbench --debug login --region cn-beijing --instance-id i-...
```

## Feature Status

- [x] List ECS instances by region.
- [x] Open an interactive SSH terminal through ECS Workbench by region and instance ID.
- [ ] File upload.
- [ ] File download.
