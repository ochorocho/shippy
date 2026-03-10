<?php
/**
 * Shippy Database Credential Extractor
 *
 * Reads database credentials from PHP application configuration files.
 * Supports: .env files (Laravel/Symfony/TYPO3), TYPO3 settings.php
 *
 * Usage: php credentials.php <shared_path> <current_path> [source]
 * Output: JSON with driver, host, port, name, user, password
 */

$sharedPath  = $argv[1] ?? '';
$currentPath = $argv[2] ?? '';
$source      = $argv[3] ?? 'auto';

$result = [];

/**
 * Parse a .env file into an associative array
 */
function parseEnvFile(string $path): array {
    if (!file_exists($path)) {
        return [];
    }
    $env = [];
    $lines = file($path, FILE_IGNORE_NEW_LINES | FILE_SKIP_EMPTY_LINES);
    foreach ($lines as $line) {
        $line = trim($line);
        if ($line === '' || $line[0] === '#') {
            continue;
        }
        $pos = strpos($line, '=');
        if ($pos === false) {
            continue;
        }
        $key = trim(substr($line, 0, $pos));
        $value = trim(substr($line, $pos + 1));
        // Remove surrounding quotes
        if (strlen($value) >= 2 && ($value[0] === '"' || $value[0] === "'")) {
            $value = substr($value, 1, -1);
        }
        $env[$key] = $value;
    }
    return $env;
}

/**
 * Extract credentials from standard .env keys (Laravel/Symfony)
 */
function extractFromDotenv(array $env): array {
    $driver = $env['DB_CONNECTION'] ?? $env['DATABASE_DRIVER'] ?? '';
    $host = $env['DB_HOST'] ?? $env['DATABASE_HOST'] ?? '';
    $name = $env['DB_DATABASE'] ?? $env['DATABASE_NAME'] ?? '';

    if ($driver === '' && $host === '' && $name === '') {
        return [];
    }

    return [
        'driver'   => $driver ?: 'mysql',
        'host'     => $host ?: '127.0.0.1',
        'port'     => (int)($env['DB_PORT'] ?? $env['DATABASE_PORT'] ?? 3306),
        'name'     => $name,
        'user'     => $env['DB_USERNAME'] ?? $env['DATABASE_USER'] ?? '',
        'password' => $env['DB_PASSWORD'] ?? $env['DATABASE_PASSWORD'] ?? '',
    ];
}

/**
 * Extract credentials from TYPO3 .env keys
 */
function extractFromTypo3Env(array $env): array {
    $prefix = 'TYPO3_CONF_VARS__DB__Connections__Default__';
    $driver = $env[$prefix . 'driver'] ?? '';
    $host = $env[$prefix . 'host'] ?? '';
    $name = $env[$prefix . 'dbname'] ?? '';

    if ($driver === '' && $host === '' && $name === '') {
        return [];
    }

    return [
        'driver'   => $driver ?: 'mysqli',
        'host'     => $host ?: '127.0.0.1',
        'port'     => (int)($env[$prefix . 'port'] ?? 3306),
        'name'     => $name,
        'user'     => $env[$prefix . 'user'] ?? '',
        'password' => $env[$prefix . 'password'] ?? '',
    ];
}

/**
 * Extract credentials from TYPO3 settings.php or LocalConfiguration.php
 */
function extractFromTypo3Settings(string $sharedPath, string $currentPath): array {
    // Try TYPO3 v12+ settings.php locations
    $candidates = [
        $sharedPath . '/config/system/settings.php',
        $currentPath . '/config/system/settings.php',
        $sharedPath . '/typo3conf/LocalConfiguration.php',
        $currentPath . '/typo3conf/LocalConfiguration.php',
    ];

    foreach ($candidates as $path) {
        if (!file_exists($path)) {
            continue;
        }

        $settings = @include $path;
        if (!is_array($settings)) {
            continue;
        }

        $db = $settings['DB']['Connections']['Default'] ?? [];
        if (empty($db)) {
            continue;
        }

        return [
            'driver'   => $db['driver'] ?? 'mysqli',
            'host'     => $db['host'] ?? '127.0.0.1',
            'port'     => (int)($db['port'] ?? 3306),
            'name'     => $db['dbname'] ?? '',
            'user'     => $db['user'] ?? '',
            'password' => $db['password'] ?? '',
        ];
    }

    return [];
}

// --- Main extraction logic ---

// Find .env files
$envPaths = [
    $sharedPath . '/.env',
    $currentPath . '/.env',
];

$env = [];
foreach ($envPaths as $envPath) {
    $env = parseEnvFile($envPath);
    if (!empty($env)) {
        break;
    }
}

switch ($source) {
    case 'dotenv':
        $result = extractFromDotenv($env);
        break;

    case 'typo3':
        // Try TYPO3 .env keys first, then settings.php
        $result = extractFromTypo3Env($env);
        if (empty($result) || empty($result['name'])) {
            $result = extractFromTypo3Settings($sharedPath, $currentPath);
        }
        break;

    case 'auto':
    default:
        // Try all strategies in order

        // 1. Standard .env keys (Laravel/Symfony)
        $result = extractFromDotenv($env);

        // 2. TYPO3 .env keys
        if (empty($result) || empty($result['name'])) {
            $result = extractFromTypo3Env($env);
        }

        // 3. TYPO3 settings.php / LocalConfiguration.php
        if (empty($result) || empty($result['name'])) {
            $result = extractFromTypo3Settings($sharedPath, $currentPath);
        }
        break;
}

if (empty($result) || empty($result['name'])) {
    fwrite(STDERR, "Error: Could not extract database credentials from any known configuration source.\n");
    fwrite(STDERR, "Searched in:\n");
    foreach ($envPaths as $p) {
        fwrite(STDERR, "  - $p\n");
    }
    fwrite(STDERR, "  - $sharedPath/config/system/settings.php\n");
    fwrite(STDERR, "  - $currentPath/config/system/settings.php\n");
    fwrite(STDERR, "  - $sharedPath/typo3conf/LocalConfiguration.php\n");
    fwrite(STDERR, "  - $currentPath/typo3conf/LocalConfiguration.php\n");
    exit(1);
}

echo json_encode($result);
