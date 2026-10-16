---
title: SSH Connections
description: SSH key detection, ssh-agent, custom ports, timeouts, keepalives, compression, multiplexing and host key verification.
---

# SSH Connections

Shippy talks to your server over SSH using key authentication. There is no agent or daemon to install
on the remote side.

## SSH Authentication

### SSH Key Detection

The `ssh_key` field is optional. If not specified, Shippy will automatically try to find your SSH key
in these locations (in order):

1. `~/.ssh/id_ed25519`
2. `~/.ssh/id_rsa`
3. `~/.ssh/id_ecdsa`

### Explicit SSH Key

```yaml
hosts:
  production:
    hostname: example.com
    remote_user: deploy
    ssh_key: ~/.ssh/id_ed25519  # Optional: specify SSH private key
```

::: warning
Always specify the **private key** (e.g. `id_ed25519`), not the public key (e.g. `id_ed25519.pub`).
:::

### SSH Agent

If `ssh-agent` is running (`SSH_AUTH_SOCK` is set) or, on Windows, Pageant is running, Shippy also
offers every key the agent holds — in addition to, not instead of, `ssh_key`. The server tries every
offered key during authentication, so both sources are tried automatically; nothing needs to be
configured to enable this.

This matters most for passphrase-protected keys: Shippy cannot decrypt a passphrase-protected private
key file itself, but if the same key is already loaded in your agent (`ssh-add`), it's used from
there instead, and `ssh_key` can point at that same encrypted file without issue. With an agent
running, `ssh_key` also becomes fully optional — no default key needs to exist on disk at all.

```yaml
hosts:
  production:
    hostname: example.com
    remote_user: deploy
    # No ssh_key needed - authenticates entirely via ssh-agent.
    # Run `ssh-add ~/.ssh/id_ed25519` beforehand so the agent holds the key.
```

## SSH Options

You can configure SSH connection behavior using the `ssh_options` field. These options correspond to
SSH configuration options (see `man ssh_config`):

```yaml
hosts:
  production:
    hostname: example.com
    port: 2222  # Custom SSH port (default: 22)
    remote_user: deploy
    # ssh_key is optional - will auto-detect from ~/.ssh/

    # Advanced SSH options
    ssh_options:
      ConnectTimeout: "30"           # Connection timeout (default: 30 seconds)
      ServerAliveInterval: "60"      # Send keepalive every 60 seconds
      ServerAliveCountMax: "3"       # Disconnect after 3 failed keepalives
      Compression: "yes"             # Enable SSH compression
      StrictHostKeyChecking: "accept-new"  # Host key verification mode
      UserKnownHostsFile: "~/.ssh/known_hosts"  # Known hosts file path
```

### ConnectTimeout

Specifies the timeout for establishing an SSH connection. Supports multiple formats:

- **Integer** (seconds): `ConnectTimeout: "30"` or `ConnectTimeout: 30`
- **Duration string**: `ConnectTimeout: "30s"`, `ConnectTimeout: "5m"`, `ConnectTimeout: "1h"`

Default: `30` seconds

Examples:

```yaml
ssh_options:
  ConnectTimeout: "10"      # 10 seconds
  ConnectTimeout: "30s"     # 30 seconds
  ConnectTimeout: "2m"      # 2 minutes
```

### ServerAliveInterval and ServerAliveCountMax

Keep SSH connections alive during long-running operations (deployments, database migrations, etc.) by
sending periodic keepalive messages.

- **ServerAliveInterval**: Interval between keepalive messages. Supports same formats as
  ConnectTimeout.
- **ServerAliveCountMax**: Number of keepalive messages to send without response before disconnecting
  (default: 3)

Examples:

```yaml
ssh_options:
  ServerAliveInterval: "60"     # Send keepalive every 60 seconds
  ServerAliveCountMax: "3"      # Disconnect after 3 failed attempts

  # Or with duration format:
  ServerAliveInterval: "1m"     # Send keepalive every minute
```

**Use case:** For long-running deployments or commands, set ServerAliveInterval to prevent SSH
timeouts:

```yaml
ssh_options:
  ServerAliveInterval: "30"     # Keepalive every 30 seconds
  ServerAliveCountMax: "5"      # Allow up to 5 failed attempts (2.5 min grace)
```

### Compression

Enable SSH compression to reduce bandwidth usage. Particularly useful for large file transfers over
slow connections.

- **Values**: `"yes"`, `"true"`, `"no"`, `"false"`

Example:

```yaml
ssh_options:
  Compression: "yes"    # Enable compression
```

::: info
Go's SSH library handles compression negotiation with the server. If the server doesn't support
compression, it will be automatically disabled.
:::

### StrictHostKeyChecking

Controls host key verification:

- `"yes"` — Strict checking, reject unknown hosts (most secure)
- `"accept-new"` — Accept new hosts, verify known hosts (recommended default)
- `"no"` — Disable all verification (insecure, not recommended for production)

Example:

```yaml
ssh_options:
  StrictHostKeyChecking: "accept-new"   # Accept first connection, verify thereafter
  UserKnownHostsFile: "~/.ssh/known_hosts"
```

### UserKnownHostsFile

Path to the known_hosts file for host key verification. Supports tilde expansion (`~`).

Default: `~/.ssh/known_hosts`

Example:

```yaml
ssh_options:
  UserKnownHostsFile: "~/.ssh/my_known_hosts"
```

## Complete Example

```yaml
hosts:
  production:
    hostname: example.com
    port: 22
    remote_user: deploy
    ssh_key: ~/.ssh/id_ed25519
    deploy_path: /var/www/myproject

    ssh_options:
      # Connection and timeout settings
      ConnectTimeout: "30"              # 30 second connection timeout
      ServerAliveInterval: "60"         # Keepalive every 60 seconds
      ServerAliveCountMax: "3"          # Disconnect after 3 failures

      # Performance
      Compression: "yes"                # Enable compression

      # Security
      StrictHostKeyChecking: "accept-new"
      UserKnownHostsFile: "~/.ssh/known_hosts"
```

::: info
The `port` field is a top-level configuration option for convenience. For other SSH options, use the
`ssh_options` map.
:::

## SSH Multiplexing

By default, Shippy opens a fresh SSH connection for each remote operation. Set
`ssh_multiplexing: true` to reuse a single shared connection (SSH `ControlMaster`) for all operations
against a host, which noticeably reduces overhead on high-latency links or deployments that run many
commands:

```yaml
hosts:
  production:
    hostname: example.com
    remote_user: deploy
    ssh_multiplexing: true   # Reuse one connection for all operations (default: false)
```

## Testing a connection

Before the first deployment it is worth confirming the key works on its own:

```bash
ssh -i ~/.ssh/id_ed25519 deploy@example.com
```

If that succeeds, `shippy config validate` and `shippy deploy` will use the same credentials.
