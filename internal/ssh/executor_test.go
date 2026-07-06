package ssh

import "testing"

func TestBuildRemoteCommand(t *testing.T) {
	tests := []struct {
		name    string
		cmd     Command
		workDir string
		want    string
	}{
		{
			name:    "no context, no workdir runs verbatim",
			cmd:     Command{Run: "./vendor/bin/typo3 cache:flush"},
			workDir: "",
			want:    "./vendor/bin/typo3 cache:flush",
		},
		{
			name:    "no context cds into workdir first",
			cmd:     Command{Run: "composer install"},
			workDir: "/var/www/app/releases/20260706120000",
			want:    "cd '/var/www/app/releases/20260706120000' && composer install",
		},
		{
			name:    "context wraps the whole cd-and-run in sh -c",
			cmd:     Command{Run: "composer install", Context: "docker exec php85"},
			workDir: "/var/www/app/releases/20260706120000",
			want:    "docker exec php85 sh -c 'cd '\\''/var/www/app/releases/20260706120000'\\'' && composer install'",
		},
		{
			name:    "context without workdir still wraps in sh -c",
			cmd:     Command{Run: "php -v", Context: "docker exec php85"},
			workDir: "",
			want:    "docker exec php85 sh -c 'php -v'",
		},
		{
			name:    "single quotes in the inner command are escaped for sh -c",
			cmd:     Command{Run: "echo 'hi'", Context: "docker exec php85"},
			workDir: "",
			want:    `docker exec php85 sh -c 'echo '\''hi'\'''`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildRemoteCommand(tt.cmd, tt.workDir)
			if got != tt.want {
				t.Errorf("buildRemoteCommand()\n  got:  %s\n  want: %s", got, tt.want)
			}
		})
	}
}
