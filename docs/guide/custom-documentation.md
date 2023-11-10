---
title: Customizing Help Content
---

# Documentation and Content

::: tip Note
This feature was introduced in version `v5.1.x`
:::
Dozzle can be used by many users from different levels. Some teams what a central place to explain how to use Dozzle and what each of their containers do. Dozzle supports markdown content by reading `/data/content/*.md` files. Multiple files are supported. Each file will create a link at the top of Dozzle. Here is an example of what `/data/content/help.md` might look like:

```yml
---
title: Help
---
# This is help

Hello, from Dozzle.

Tables are also supported!

| foo | bar |
| --- | --- |
| baz | bim |
```

`title` is used to show a link on Dozzle's home screen to `/content/help`. Dozzle will need `/data` mounted. Here is an example:

::: code-group

```sh [cli]
$ docker run -v /var/run/docker.sock:/var/run/docker.sock -v /path/to/dozzle/data:/data -p 8080:8080 amir20/dozzle
```

```yaml [docker-compose.yml]
version: "3"
services:
  dozzle:
    image: amir20/dozzle:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /path/to/dozzle/data:/data
    ports:
      - 8080:8080
```

```yml [help.md]
---
title: Help
---
# This is help

Hello, from Dozzle.
```

:::
