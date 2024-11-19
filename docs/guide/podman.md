---
title: Podman
---

## I am seeing host not found error in the logs. How do I fix it?

This should be mainly a Podman only error: Using Podman doesn't create an engine-id like Docker.

It might be neccessary to clean up your existing dozzle deployment under Podman, stop the container and remove the associated data (container/volumes). After you created the engine-id you can redeploy the Dozzle container and your logs should now show up.

## Create UUID

Options for generating UUIDs

### uuidgen

1. Install uuidgen
2. Create the folders:  ```mkdir -p /var/lib/docker```
3. Using uuidgen generate an UUID: ```uuidgen > /var/lib/docker/engine-id```
4. Verify with ```cat /var/lib/docker/engine-id```

### Ansible

:warning: Depending on your setup you might have to take adjustments for file/folder permissions. The following task snippets would run as the become_user of the playbook running these tasks.

If you wish to adjust the user have to set individual become/become_user parameters for the task.

```
- name: Create /var/lib/docker
  ansible.builtin.file:
    path: /var/lib/docker
    state: directory
    mode: '755'

- name: Create engine-id and derive UUID from hostname
  ansible.builtin.lineinfile:
    path: /var/lib/docker/engine-id
    line: "{{ hostname | to_uuid }}"
    create: true
    mode: "0644"
    insertafter: "EOF"
```
