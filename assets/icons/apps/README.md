# App Icons

Logos shown next to containers whose image Dozzle recognizes. Most files here are vendored from [homarr-labs/dashboard-icons](https://github.com/homarr-labs/dashboard-icons), and the filename without its extension is the slug. `assets/utils/appIcons.ts` maps an image reference to a slug.

`magpie.webp` comes from [Magpie's official icon](https://github.com/yetone/magpie/blob/64aaf421ac39fb2a2dab8c9df1f7729c8e39b4e8/site/public/img/icon-256.png), resized to 64px and encoded as WebP (quality 80, method 6). Magpie is [MIT licensed](https://github.com/yetone/magpie/blob/64aaf421ac39fb2a2dab8c9df1f7729c8e39b4e8/LICENSE).

The repo ships an `add-app-icon` skill in `.claude/skills/`. To add icons with an AI coding agent, paste this prompt and fill in the images:

```
Use the add-app-icon skill in .claude/skills/add-app-icon/SKILL.md to add app icons
for these container images: <image1>, <image2>
```
