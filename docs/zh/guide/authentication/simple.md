---
title: 简单模式身份验证
sourceHash: deea96688436
---

# <Icon icon="mdi:account-cog-outline" inline /> 简单模式身份验证

Dozzle 自带的用户管理。用户保存在由 Dozzle 管理的 `users.yml` 文件里，登录页面也由 Dozzle 自己提供。把 `--auth-provider` 设为 `simple` 即可启用。

密码只是证明你是其中某个用户的一种方式，[使用 GitHub 或 OIDC 登录](/zh/guide/authentication/oauth)是另一种，两者读取的都是同一个 `users.yml`。

> [!TIP]
> 请用内置的 [`generate` 命令](/zh/guide/authentication#生成-users-yml)来创建 `users.yml`，而不是手写 bcrypt 哈希。

把 `--auth-provider` 设为 `simple` 即可启用多用户验证。在这种模式下，Dozzle 会尝试从 `/data/` 读取用户文件，如果 `users.yml` 和 `users.yaml` 都存在，优先使用 `users.yml`。如果只存在其中一个，就使用那一个。日志会显示实际读取的是哪个文件（例如 `Reading users.yml file`）。

## 文件路径示例：

- `/data/users.yml`
- `/data/users.yaml`

文件内容大致如下：

```yaml
users:
  # "admin" 是用户名
  admin:
    email: me@email.net
    name: Admin
    # 用 docker run -it --rm amir20/dozzle generate admin --password password --email me@email.net --name "Admin" 生成
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter:
    roles:
```

Dozzle 用 `email` 通过 [Gravatar](https://gravatar.com/) 生成头像，这一项是可选的。密码使用 `bcrypt` 哈希，可以用 `docker run amir20/dozzle generate` 生成。

你需要把这个文件挂载进去，Dozzle 才能找到它。示例如下：

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/dozzle/data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_AUTH_PROVIDER: simple
```

```yaml [users.yml]
users:
  admin:
    email: me@email.net
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
```

:::

或者使用 Docker secrets：

```yaml
services:
  dozzle:
    image: amir20/dozzle:latest
    environment:
      - DOZZLE_AUTH_PROVIDER=simple
    secrets:
      - source: users
        target: /data/users.yml
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - dozzle:/data
secrets:
  users:
    file: users.yml
volumes:
  dozzle:
```

## 延长验证 Cookie 的有效期

默认情况下，Dozzle 使用会话 cookie，浏览器关闭后即失效。你可以把 `--auth-ttl` 设为一个时长来延长 cookie 的有效期。示例如下：

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle --auth-provider simple --auth-ttl 48h
```

```yaml [docker-compose.yml]
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/dozzle/data:/data
    ports:
      - 8080:8080
    environment:
      DOZZLE_AUTH_PROVIDER: simple
      DOZZLE_AUTH_TTL: 48h
```

:::

注意这里只支持时长格式，只能使用 `s`、`m`、`h`，分别表示秒、分钟和小时。

## 为用户设置过滤器

Dozzle 支持为用户设置过滤器。过滤器用于限制用户可见的容器，在 `users.yml` 文件中配置。示例如下：

```yaml
users:
  admin:
    email:
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter:

  guest:
    email:
    name: Guest
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    filter: "label=com.example.app"
```

在这个例子中，`admin` 用户没有过滤器，因此能看到所有容器。`guest` 用户只能看到带有 `com.example.app` 标签的容器。这样就能把访问限制在特定容器上。

> [!NOTE]
> 过滤器也可以用 `--filter` 参数[全局设置](/zh/guide/filters)。该参数对所有用户生效。如果某个用户设置了自己的过滤器，则会覆盖全局过滤器。

## 为用户设置角色

Dozzle 支持给用户分配角色。角色决定用户可以对容器执行哪些操作，在 users.yml 文件中配置。

```yaml
users:
  admin:
    email:
    name: Admin
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    roles:

  guest:
    email:
    name: Guest
    password: $2a$11$9ho4vY2LdJ/WBopFcsAS0uORC0x2vuFHQgT/yBqZyzclhHsoaIkzK
    roles: shell
```

在这个例子中，`admin` 用户没有指定角色，因此拥有所有容器操作的完整权限。`guest` 用户拥有 shell 角色，只能在容器中打开终端。用角色可以很方便地控制和限制用户在 Dozzle 里能做什么。

Dozzle 支持以下角色：

| 角色            | 同时接受               | 授予的权限                                                         |
| --------------- | ---------------------- | ------------------------------------------------------------------ |
| `shell`         | `dozzle_shell`         | 连接容器并打开 exec 会话。实例还需要开启 `--enable-shell`。        |
| `actions`       | `dozzle_actions`       | 启动、停止和重启容器。实例还需要开启 `--enable-actions`。          |
| `download`      | `dozzle_download`      | 把容器日志下载为文件。                                             |
| `notifications` | `dozzle_notifications` | 创建和编辑通知规则与通知目标。                                     |
| `cloud`         | `dozzle_cloud`         | 关联、解除关联并配置 Dozzle Cloud。                                |
| `all`           | `dozzle_all`           | 以上所有角色。`roles` 为空时的默认值。                             |
| `none`          | `dozzle_none`          | 没有任何角色。日志仍可查看，受用户过滤器约束。会覆盖其他所有设置。 |

多个角色之间用逗号或竖线分隔（`shell,actions` 或 `shell|actions`），也可以写成 JSON 数组（`["shell", "actions"]`）。名称不区分大小写。带 `dozzle_` 前缀的别名是为了让身份提供方的用户组名称在前置代理模式下可以原样传入。

> [!WARNING]
> 通知规则是实例级别的。规则按表达式匹配容器，不受用户过滤器约束，因此拥有 `notifications` 角色的用户可以为其过滤器本应隐藏的容器创建规则，并把这些日志行发送到自己控制的目标。只把这个角色授予你信任其访问实例上所有容器的用户。

> [!WARNING]
> Dozzle Cloud 同样是实例级别的。关联操作会保存一个 API key，把警报派发、日志推送和工具执行统统指向某一个云账号，而且云端工具使用的是实例过滤器，而不是执行关联操作那位用户的过滤器。拥有 `cloud` 角色的用户可以把实例关联到自己的云账号，从而看到所有容器，也可以解除已有的关联。只把这个角色授予你信任其访问实例上所有容器的用户。

任何角色都可以加上 `^` 前缀表示排除。排除规则最后生效，所以顺序无关紧要：

```yaml
roles: all,^shell # 除 shell 外的所有角色
```

`none` 是唯一不能被取反的角色。`^none` 会被忽略，而列表中任何位置出现 `none` 都会让其他所有角色失效。
