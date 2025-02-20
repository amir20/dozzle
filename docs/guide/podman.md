---
title: Podman
---

# Podman

Dozzle supports Podman. However, there are some issues with Podman that might prevent Dozzle from working properly. One of the main issues is that Podman doesn't create an engine-id like Docker.

## I am seeing host not found error in the logs. How do I fix it?

This should be mainly a Podman only error: Using Podman doesn't create an engine-id like Docker.
If you are using Docker, check if the `engine-id` file exists with correct permissions in `/var/lib/docker` and has the UUID inside.

It might be necessary to clean up your existing Dozzle deployment under Podman, stop the container and remove the associated data (container/volumes). After you create the engine-id, you can redeploy the Dozzle container and your logs should now show up.

## Create UUID

Options for generating UUIDs:

### uuidgen

:warning: Adjust folder/file permissions if necessary. There isn't any critical info but depending on your existing setup you might want to take additional steps.

1. Install uuidgen
2. Create the folders: `mkdir -p /var/lib/docker`
3. Using uuidgen generate a UUID: `uuidgen > /var/lib/docker/engine-id`
4. Verify with `cat /var/lib/docker/engine-id`

### Ansible

:warning: Depending on your setup you might have to make adjustments for file/folder permissions. The following task snippets would run as the become_user/remote_user of the playbook running these tasks.

If you wish to adjust the user, you have to set individual become/become_user parameters for these tasks.

```yaml
- name: Create /var/lib/docker
  ansible.builtin.file:
    path: /var/lib/docker
    state: directory
    mode: "755"

- name: Create engine-id and derive UUID from hostname
  ansible.builtin.lineinfile:
    path: /var/lib/docker/engine-id
    line: "{{ hostname | to_uuid }}"
    create: true
    mode: "0644"
    insertafter: "EOF"
```
