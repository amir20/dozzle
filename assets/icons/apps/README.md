# App Icons

Logos shown next to containers whose image Dozzle recognizes. Every file here is vendored from [homarr-labs/dashboard-icons](https://github.com/homarr-labs/dashboard-icons), and the filename without its extension is the slug. `assets/utils/appIcons.ts` maps an image reference to a slug.

The repo ships an `add-app-icon` skill in `.claude/skills/`. To add icons with an AI coding agent, paste this prompt and fill in the images:

```
Use the add-app-icon skill in .claude/skills/add-app-icon/SKILL.md to add app icons
for these container images: <image1>, <image2>
```
