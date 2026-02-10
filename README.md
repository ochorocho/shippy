# Shippy

A minimal, opinionated deployment tool for TYPO3 projects, inspired by Deployer and Capistrano.

![Shippy](images/logo.svg)

## Features

- **Zero-downtime deployments** with atomic releases
- **Release management** - keeps last N releases with easy rollback
- **Shared files/directories** - persistent data between releases
- **Template variables** from composer.json
- **Pure Go implementation** - single binary, no dependencies
- **.gitignore support** - respects your gitignore patterns
- **SSH-based** deployment with key authentication
- **Colored output** - clear, beautiful deployment progress
- **TYPO3 optimized** - sensible defaults for TYPO3 projects

## Installation

### From Source

```bash
git clone https://github.com/yourusername/shippy.git
cd shippy
go build -o shippy
sudo mv shippy /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/yourusername/shippy@latest
```

## Quick Start

1. **Initialize configuration** in your TYPO3 project root:

```bash
shippy init
```

This will create a `.shippy.yaml` file with sensible TYPO3 defaults and read your project name from `composer.json`.

2. **Edit configuration** with your server details:

```bash
vim .shippy.yaml
```

Update at minimum:
- `hostname` - your server's domain or IP
- `remote_user` - SSH username
- `ssh_key` - path to your SSH private key

3. **Validate configuration**:

```bash
shippy config validate
```

4. **Deploy to production**:

```bash
shippy deploy production
```

## Configuration

### Basic Structure

```yaml
hosts:
  <hostname>:
    # SSH connection
    hostname: <server domain or IP>
    port: <SSH port, default: 22>
    remote_user: <SSH username>
    ssh_key: <path to SSH private key>
    ssh_options: <map of SSH options, see below>

    # Deployment
    deploy_path: <absolute path on server>
    rsync_src: <local source directory>
    keep_releases: <number of releases to keep, default: 5>

    # File management
    shared: <list of shared paths>
    exclude: <additional exclude patterns>
    include: <patterns to include despite .gitignore>

commands:
  - name: <command description>
    run: <command to execute>
```

### Exclude and Include Patterns

**Exclude Patterns:**

Shippy automatically respects `.gitignore` patterns **including nested `.gitignore` files in subdirectories**. You can add additional exclude patterns in your configuration:

```yaml
hosts:
  production:
    hostname: example.com
    remote_user: deploy
    deploy_path: /var/www/myproject
    rsync_src: ./

    # Additional exclude patterns (beyond .gitignore)
    exclude:
      - "*.log"              # Exclude all .log files
      - ".env.example"       # Exclude specific file
      - "tests/"             # Exclude entire directory
      - "*.md"               # Exclude all markdown files
      - ".ddev/"             # Exclude DDEV configuration
      - "node_modules/"      # Exclude node modules (if not in .gitignore)
```

**Include Patterns:**

Use `include` to explicitly include files that are excluded by `.gitignore`:

```yaml
hosts:
  production:
    hostname: example.com
    remote_user: deploy
    deploy_path: /var/www/myproject
    rsync_src: ./

    # Force include files despite .gitignore
    include:
      - "public/.htaccess"   # Include .htaccess files
      - "vendor/"            # Include vendor directory (if gitignored)
      - ".env.production"    # Include specific environment file
```

**Pattern Syntax:**

- Patterns use gitignore-style syntax
- `*` matches any characters except `/`
- `**` matches any characters including `/`
- Trailing `/` means directory only
- No leading `/` means pattern matches at any depth
- Leading `/` means pattern matches from project root

**Examples:**

```yaml
exclude:
  - "*.log"                    # All .log files at any depth
  - "/build/"                  # build/ directory at root only
  - "temp/"                    # temp/ directory at any depth
  - "**/*.test.js"             # All .test.js files anywhere
  - ".DS_Store"                # macOS metadata files
  - "Thumbs.db"                # Windows metadata files
```

### SSH Authentication

**SSH Key Detection:**

The `ssh_key` field is optional. If not specified, Shippy will automatically try to find your SSH key in these locations (in order):
1. `~/.ssh/id_ed25519`
2. `~/.ssh/id_rsa`
3. `~/.ssh/id_ecdsa`

**Explicit SSH Key:**

```yaml
hosts:
  production:
    hostname: example.com
    remote_user: deploy
    ssh_key: ~/.ssh/id_ed25519  # Optional: specify SSH private key
```

**Important:** Always specify the **private key** (e.g., `id_ed25519`), not the public key (e.g., `id_ed25519.pub`).

### SSH Options

You can configure SSH connection behavior using the `ssh_options` field. These options correspond to SSH configuration options (see `man ssh_config`):

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

#### ConnectTimeout

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

#### ServerAliveInterval and ServerAliveCountMax

Keep SSH connections alive during long-running operations (deployments, database migrations, etc.) by sending periodic keepalive messages.

- **ServerAliveInterval**: Interval between keepalive messages. Supports same formats as ConnectTimeout.
- **ServerAliveCountMax**: Number of keepalive messages to send without response before disconnecting (default: 3)

Examples:
```yaml
ssh_options:
  ServerAliveInterval: "60"     # Send keepalive every 60 seconds
  ServerAliveCountMax: "3"      # Disconnect after 3 failed attempts

  # Or with duration format:
  ServerAliveInterval: "1m"     # Send keepalive every minute
```

**Use case:** For long-running deployments or commands, set ServerAliveInterval to prevent SSH timeouts:
```yaml
ssh_options:
  ServerAliveInterval: "30"     # Keepalive every 30 seconds
  ServerAliveCountMax: "5"      # Allow up to 5 failed attempts (2.5 min grace)
```

#### Compression

Enable SSH compression to reduce bandwidth usage. Particularly useful for large file transfers over slow connections.

- **Values**: `"yes"`, `"true"`, `"no"`, `"false"`

Example:
```yaml
ssh_options:
  Compression: "yes"    # Enable compression
```

**Note:** Go's SSH library handles compression negotiation with the server. If the server doesn't support compression, it will be automatically disabled.

#### StrictHostKeyChecking

Controls host key verification:

- `"yes"` - Strict checking, reject unknown hosts (most secure)
- `"accept-new"` - Accept new hosts, verify known hosts (recommended default)
- `"no"` - Disable all verification (insecure, not recommended for production)

Example:
```yaml
ssh_options:
  StrictHostKeyChecking: "accept-new"   # Accept first connection, verify thereafter
  UserKnownHostsFile: "~/.ssh/known_hosts"
```

#### UserKnownHostsFile

Path to the known_hosts file for host key verification. Supports tilde expansion (`~`).

Default: `~/.ssh/known_hosts`

Example:
```yaml
ssh_options:
  UserKnownHostsFile: "~/.ssh/my_known_hosts"
```

#### Complete Example

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

**Note:** The `port` field is a top-level configuration option for convenience. For other SSH options, use the `ssh_options` map.

### Template Variables

Use `{{key.path}}` syntax to reference values from composer.json:

```yaml
hosts:
  production:
    deploy_path: /var/www/{{name}}  # Uses composer.json "name" field
```

Access nested values:

```yaml
deploy_path: /var/www/{{extra.typo3/cms.web-dir}}
```

### Shared Files/Directories

Files and directories in the `shared:` list are symlinked from the `shared/` directory to each release:

```yaml
shared:
  - .env                    # Shared file
  - var/log/                # Shared directory (note trailing slash)
  - public/fileadmin/
  - public/uploads/
```

### Directory Structure

Shippy creates the following structure on the server (following Deployer/Capistrano conventions):

```
/var/www/myproject/
├── current -> releases/20240109120000    # Symlink to latest release
├── releases/
│   ├── 20240109120000/                   # Current release
│   ├── 20240109110000/                   # Previous release
│   └── 20240109100000/                   # Older release
└── shared/
    ├── .env                              # Shared files
    ├── var/
    │   ├── log/
    │   └── session/
    └── public/
        ├── fileadmin/
        └── uploads/
```

## Commands

### Initialize

Create a new configuration file with TYPO3 defaults:

```bash
shippy init
```

Options:
- `--force` or `-f` - Overwrite existing configuration file

This command:
- Checks for `composer.json` in current directory
- Reads project name from composer.json
- Generates `.shippy.yaml` with sensible TYPO3 defaults
- Protects against accidental overwrites (use `--force` to override)

### Deploy

Deploy to a target host:

```bash
shippy deploy <hostname>
```

Example:

```bash
shippy deploy staging
shippy deploy production
```

### Rollback

Rollback to a previous release:

```bash
shippy rollback <hostname>
```

Options:
- `--list` or `-l` - List available releases and exit
- `--release` or `-r` - Switch to a specific release by name
- `--offset` or `-n` - Relative offset from current release (negative = older, positive = newer)

Examples:

```bash
shippy rollback production              # Interactive release selection
shippy rollback production -l           # List available releases
shippy rollback production -n -1        # One version back
shippy rollback production -n +1        # One version forward (e.g., after accidental rollback)
shippy rollback production -n -2        # Two versions back
shippy rollback production -r 20260109120000  # Specific release by name
```

When run without flags, shows an interactive list of available releases with deployment date/time, git commit hash, and git tag. The current release is marked and cannot be selected.

### Validate Configuration

Check if your configuration is valid:

```bash
shippy config validate
```

This command:
- Validates YAML syntax
- Checks required fields
- Tests composer.json template variables
- Shows processed configuration

## Deployment Process

When you run `shippy deploy <host>`, the following steps occur:

1. **Scan files** - Walks source directory, respects .gitignore and exclude patterns
2. **Connect to server** - Establishes SSH connection
3. **Create release** - Creates new timestamped release directory (e.g., `releases/20260109203841`)
4. **Sync files** - Transfers files to the new release directory
5. **Create symlinks** - Links shared files/directories from `shared/` to the release
6. **Execute commands** - Runs commands **in the new release directory** (e.g., cache flush, migrations)
7. **Activate release** - Atomically updates `current` symlink to new release (site goes live)
8. **Cleanup** - Removes old releases, keeps last N

**Important:** Commands execute in the new release directory **before** it goes live. This ensures all preparation (cache warming, migrations, etc.) completes successfully before the atomic switchover. The site only becomes live when the `current` symlink is updated in step 7.

## Example Configuration

### Minimal TYPO3 Configuration

```yaml
hosts:
  production:
    hostname: www.example.com
    remote_user: deploy
    deploy_path: /var/www/{{name}}
    rsync_src: ./
    ssh_key: ~/.ssh/id_rsa
    shared:
      - .env
      - var/log/
      - var/session/
      - public/fileadmin/
      - public/uploads/

commands:
  - name: Clear TYPO3 cache
    run: ./vendor/bin/typo3 cache:flush

  - name: Run extension setup
    run: ./vendor/bin/typo3 extension:setup
```

### Advanced Configuration

```yaml
hosts:
  staging:
    hostname: staging.example.com
    remote_user: deploy
    deploy_path: /var/www/{{name}}/staging
    rsync_src: ./
    ssh_key: ~/.ssh/id_rsa
    keep_releases: 3

    # Additional excludes beyond .gitignore
    exclude:
      - .git/
      - node_modules/
      - .env.local
      - Tests/

    # Force include despite .gitignore
    include:
      - public/.htaccess

    # Shared paths
    shared:
      - .env
      - var/log/
      - var/session/
      - public/fileadmin/
      - public/uploads/

  production:
    hostname: www.example.com
    remote_user: deploy
    deploy_path: /var/www/{{name}}/production
    rsync_src: ./
    ssh_key: ~/.ssh/id_rsa_production
    keep_releases: 10
    shared:
      - .env
      - var/log/
      - var/session/
      - public/fileadmin/
      - public/uploads/

commands:
  - name: Clear TYPO3 cache
    run: ./vendor/bin/typo3 cache:flush

  - name: Run extension setup
    run: ./vendor/bin/typo3 extension:setup

  - name: Database migrations
    run: ./vendor/bin/typo3 upgrade:run

  - name: Warmup caches
    run: ./vendor/bin/typo3 cache:warmup
```

## Default Excludes

Shippy automatically excludes these patterns (in addition to .gitignore):

- `.git/`
- `.gitignore`
- `.shippy.yaml`
- `.shippy.yaml.example`
- `node_modules/`
- `.env.local`
- `.env.*.local`
- `var/cache/`
- `var/log/`
- `var/transient/`
- `.DS_Store`
- `Thumbs.db`

## Requirements

- Go 1.20 or higher (for building)
- SSH access to target server
- SSH key authentication configured

## Project Structure

```
shippy/
├── cmd/
│   ├── root.go          # Root CLI command
│   ├── config.go        # Config validation command
│   └── deploy.go        # Deploy command
├── internal/
│   ├── config/
│   │   ├── config.go    # Configuration parser
│   │   └── template.go  # Template variable processor
│   ├── composer/
│   │   └── parser.go    # Composer.json parser
│   ├── rsync/
│   │   ├── sync.go      # File scanner with gitignore
│   │   └── transfer.go  # File transfer over SSH
│   ├── ssh/
│   │   ├── client.go    # SSH client
│   │   └── executor.go  # Command executor
│   └── deploy/
│       ├── deployer.go  # Main deployment orchestrator
│       └── release.go   # Release management
├── main.go
├── go.mod
└── README.md
```

## Testing

go 1.24: `go build -o shippy `

Test instance

```bash
cd tests/
docker compose up
````

```bash
cd tests/typo3/
composer install
```

Deploy

```bash
../../shippy deploy production
```

Test SSH Connection

```bash
ssh -i tests/ssh_keys/shippy_key root@127.0.0.1 -p 2424
```

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
