package claude

import (
	"context"
	"testing"
)

func TestParseToolPermission(t *testing.T) {
	tests := []struct {
		name        string
		permission  string
		want        ToolPermission
		expectError bool
	}{
		{
			name:       "Legacy format - simple tool",
			permission: "Bash",
			want: ToolPermission{
				Tool:     "Bash",
				Command:  "",
				Pattern:  "",
				Original: "Bash",
			},
			expectError: false,
		},
		{
			name:       "Legacy format - MCP tool",
			permission: "mcp__filesystem__read_file",
			want: ToolPermission{
				Tool:     "mcp__filesystem__read_file",
				Command:  "",
				Pattern:  "",
				Original: "mcp__filesystem__read_file",
			},
			expectError: false,
		},
		{
			name:       "Enhanced format - tool with command",
			permission: "Bash(git log)",
			want: ToolPermission{
				Tool:     "Bash",
				Command:  "git log",
				Pattern:  "",
				Original: "Bash(git log)",
			},
			expectError: false,
		},
		{
			name:       "Enhanced format - tool with command and pattern",
			permission: "Bash(git log:*)",
			want: ToolPermission{
				Tool:     "Bash",
				Command:  "git log",
				Pattern:  "*",
				Original: "Bash(git log:*)",
			},
			expectError: false,
		},
		{
			name:       "Enhanced format - Write with path pattern",
			permission: "Write(src/**)",
			want: ToolPermission{
				Tool:     "Write",
				Command:  "src/**",
				Pattern:  "",
				Original: "Write(src/**)",
			},
			expectError: false,
		},
		{
			name:       "Enhanced format - complex pattern",
			permission: "Bash(npm install:package.json)",
			want: ToolPermission{
				Tool:     "Bash",
				Command:  "npm install",
				Pattern:  "package.json",
				Original: "Bash(npm install:package.json)",
			},
			expectError: false,
		},
		{
			name:        "Error - empty string",
			permission:  "",
			want:        ToolPermission{},
			expectError: true,
		},
		{
			name:        "Error - malformed parentheses",
			permission:  "Bash(git log",
			want:        ToolPermission{},
			expectError: true,
		},
		{
			name:        "Error - empty tool name",
			permission:  "(git log)",
			want:        ToolPermission{},
			expectError: true,
		},
		{
			name:        "Error - empty command",
			permission:  "Bash()",
			want:        ToolPermission{},
			expectError: true,
		},
		{
			name:        "Error - invalid format",
			permission:  "Bash(git log:pattern:extra)",
			want:        ToolPermission{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseToolPermission(tt.permission)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseToolPermission() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseToolPermission() unexpected error: %v", err)
				return
			}

			if got.Tool != tt.want.Tool {
				t.Errorf("ParseToolPermission() Tool = %v, want %v", got.Tool, tt.want.Tool)
			}
			if got.Command != tt.want.Command {
				t.Errorf("ParseToolPermission() Command = %v, want %v", got.Command, tt.want.Command)
			}
			if got.Pattern != tt.want.Pattern {
				t.Errorf("ParseToolPermission() Pattern = %v, want %v", got.Pattern, tt.want.Pattern)
			}
			if got.Original != tt.want.Original {
				t.Errorf("ParseToolPermission() Original = %v, want %v", got.Original, tt.want.Original)
			}
		})
	}
}

func TestParseToolPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		expectError bool
		wantCount   int
	}{
		{
			name: "Valid mixed permissions",
			permissions: []string{
				"Bash",
				"Bash(git log)",
				"Write(src/**)",
				"mcp__filesystem__read_file",
			},
			expectError: false,
			wantCount:   4,
		},
		{
			name:        "Empty slice",
			permissions: []string{},
			expectError: false,
			wantCount:   0,
		},
		{
			name: "Invalid permission in slice",
			permissions: []string{
				"Bash",
				"Invalid()",
				"Write",
			},
			expectError: true,
			wantCount:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseToolPermissions(tt.permissions)

			if tt.expectError {
				if err == nil {
					t.Errorf("ParseToolPermissions() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseToolPermissions() unexpected error: %v", err)
				return
			}

			if len(got) != tt.wantCount {
				t.Errorf("ParseToolPermissions() count = %v, want %v", len(got), tt.wantCount)
			}
		})
	}
}

func TestToolPermission_Methods(t *testing.T) {
	tests := []struct {
		name       string
		permission ToolPermission
		testCases  []struct {
			method   string
			input    []string
			expected bool
		}
	}{
		{
			name: "Legacy format permission",
			permission: ToolPermission{
				Tool:     "Bash",
				Original: "Bash",
			},
			testCases: []struct {
				method   string
				input    []string
				expected bool
			}{
				{"IsLegacyFormat", nil, true},
				{"HasCommand", nil, false},
				{"HasPattern", nil, false},
				{"MatchesTool", []string{"Bash"}, true},
				{"MatchesTool", []string{"Write"}, false},
				{"MatchesCommand", []string{"git log"}, true},     // No constraint means allow all
				{"MatchesPattern", []string{"src/file.go"}, true}, // No constraint means allow all
			},
		},
		{
			name: "Enhanced format with command and pattern",
			permission: ToolPermission{
				Tool:     "Bash",
				Command:  "git log",
				Pattern:  "src/**",
				Original: "Bash(git log:src/**)",
			},
			testCases: []struct {
				method   string
				input    []string
				expected bool
			}{
				{"IsLegacyFormat", nil, false},
				{"HasCommand", nil, true},
				{"HasPattern", nil, true},
				{"MatchesTool", []string{"Bash"}, true},
				{"MatchesTool", []string{"Write"}, false},
				{"MatchesCommand", []string{"git log"}, true},
				{"MatchesCommand", []string{"git status"}, false},
				{"MatchesPattern", []string{"src/file.go"}, true},
				{"MatchesPattern", []string{"test/file.go"}, false},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, tc := range tt.testCases {
				t.Run(tc.method, func(t *testing.T) {
					var got bool
					switch tc.method {
					case "IsLegacyFormat":
						got = tt.permission.IsLegacyFormat()
					case "HasCommand":
						got = tt.permission.HasCommand()
					case "HasPattern":
						got = tt.permission.HasPattern()
					case "MatchesTool":
						got = tt.permission.MatchesTool(tc.input[0])
					case "MatchesCommand":
						got = tt.permission.MatchesCommand(tc.input[0])
					case "MatchesPattern":
						got = tt.permission.MatchesPattern(tc.input[0])
					default:
						t.Errorf("Unknown test method: %s", tc.method)
						return
					}

					if got != tc.expected {
						t.Errorf("%s() = %v, want %v", tc.method, got, tc.expected)
					}
				})
			}
		})
	}
}

func TestToolPermission_PatternMatching(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		testPath string
		want     bool
	}{
		{"Wildcard all", "*", "any/path", true},
		{"Directory wildcard", "src/**", "src/file.go", true},
		{"Directory wildcard miss", "src/**", "test/file.go", false},
		{"File wildcard", "*.go", "main.go", true},
		{"File wildcard miss", "*.go", "main.js", false},
		{"Exact match", "package.json", "package.json", true},
		{"Exact match miss", "package.json", "package-lock.json", false},
		{"Empty pattern (no constraint)", "", "any/path", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perm := &ToolPermission{Pattern: tt.pattern}
			got := perm.MatchesPattern(tt.testPath)
			if got != tt.want {
				t.Errorf("MatchesPattern(%q) with pattern %q = %v, want %v",
					tt.testPath, tt.pattern, got, tt.want)
			}
		})
	}
}

func TestValidateToolPermissions(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		expectError bool
	}{
		{
			name: "All valid permissions",
			permissions: []string{
				"Bash",
				"Bash(git log)",
				"Write(src/**)",
				"mcp__filesystem__read_file",
			},
			expectError: false,
		},
		{
			name: "Contains invalid permission",
			permissions: []string{
				"Bash",
				"Invalid()",
			},
			expectError: true,
		},
		{
			name:        "Empty slice",
			permissions: []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToolPermissions(tt.permissions)
			if (err != nil) != tt.expectError {
				t.Errorf("ValidateToolPermissions() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// Tests for Permission Callback types and helpers

func TestPermissionResultHelpers(t *testing.T) {
	t.Run("Allow", func(t *testing.T) {
		result := Allow()
		if result.Behavior != PermissionAllow {
			t.Errorf("Allow() returned behavior %v, want %v", result.Behavior, PermissionAllow)
		}
		if result.Message != "" {
			t.Errorf("Allow() returned message %q, want empty", result.Message)
		}
	})

	t.Run("Deny", func(t *testing.T) {
		msg := "Access denied"
		result := Deny(msg)
		if result.Behavior != PermissionDeny {
			t.Errorf("Deny() returned behavior %v, want %v", result.Behavior, PermissionDeny)
		}
		if result.Message != msg {
			t.Errorf("Deny() returned message %q, want %q", result.Message, msg)
		}
	})

	t.Run("Ask", func(t *testing.T) {
		msg := "Allow this operation?"
		result := Ask(msg)
		if result.Behavior != PermissionAsk {
			t.Errorf("Ask() returned behavior %v, want %v", result.Behavior, PermissionAsk)
		}
		if result.Message != msg {
			t.Errorf("Ask() returned message %q, want %q", result.Message, msg)
		}
	})
}

func TestReadOnlyCallback(t *testing.T) {
	ctx := context.Background()
	callback := ReadOnlyCallback()

	tests := []struct {
		name         string
		toolName     string
		wantBehavior PermissionBehavior
	}{
		{"Read is allowed", "Read", PermissionAllow},
		{"Grep is allowed", "Grep", PermissionAllow},
		{"Glob is allowed", "Glob", PermissionAllow},
		{"Write is denied", "Write", PermissionDeny},
		{"Edit is denied", "Edit", PermissionDeny},
		{"Bash is denied", "Bash", PermissionDeny},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := callback(ctx, tt.toolName, ToolInput{})
			if err != nil {
				t.Errorf("ReadOnlyCallback() returned error: %v", err)
				return
			}
			if result.Behavior != tt.wantBehavior {
				t.Errorf("ReadOnlyCallback() behavior = %v, want %v", result.Behavior, tt.wantBehavior)
			}
		})
	}
}

func TestSafeBashCallback(t *testing.T) {
	ctx := context.Background()

	t.Run("with default patterns", func(t *testing.T) {
		callback := SafeBashCallback(nil)

		tests := []struct {
			name         string
			command      string
			wantBehavior PermissionBehavior
		}{
			{"Safe command", "ls -la", PermissionAllow},
			{"Git command", "git status", PermissionAllow},
			{"rm -rf blocked", "rm -rf /", PermissionDeny},
			{"rm -r blocked", "rm -r /tmp/test", PermissionDeny},
			{"> /dev/ blocked", "echo test > /dev/null", PermissionDeny},
			{"Fork bomb blocked", ":(){:|:&};:", PermissionDeny},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := callback(ctx, "Bash", ToolInput{Command: tt.command})
				if err != nil {
					t.Errorf("SafeBashCallback() returned error: %v", err)
					return
				}
				if result.Behavior != tt.wantBehavior {
					t.Errorf("SafeBashCallback() behavior = %v, want %v", result.Behavior, tt.wantBehavior)
				}
			})
		}
	})

	t.Run("allows non-Bash tools", func(t *testing.T) {
		callback := SafeBashCallback(nil)
		result, _ := callback(ctx, "Read", ToolInput{Command: "rm -rf /"})
		if result.Behavior != PermissionAllow {
			t.Errorf("SafeBashCallback() should allow non-Bash tools")
		}
	})

	t.Run("with custom patterns", func(t *testing.T) {
		callback := SafeBashCallback([]string{"custom-blocked"})
		result, _ := callback(ctx, "Bash", ToolInput{Command: "custom-blocked cmd"})
		if result.Behavior != PermissionDeny {
			t.Errorf("SafeBashCallback() should block custom patterns")
		}
	})
}

func TestFilePathCallback(t *testing.T) {
	ctx := context.Background()

	t.Run("with allowed paths", func(t *testing.T) {
		callback := FilePathCallback([]string{"/src/", "/test/"}, nil)

		tests := []struct {
			name         string
			toolName     string
			filePath     string
			wantBehavior PermissionBehavior
		}{
			{"Allowed path", "Write", "/src/main.go", PermissionAllow},
			{"Allowed test path", "Edit", "/test/main_test.go", PermissionAllow},
			{"Denied path", "Write", "/etc/passwd", PermissionDeny},
			{"Non-file tool allowed", "Bash", "/etc/passwd", PermissionAllow},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, err := callback(ctx, tt.toolName, ToolInput{FilePath: tt.filePath})
				if err != nil {
					t.Errorf("FilePathCallback() returned error: %v", err)
					return
				}
				if result.Behavior != tt.wantBehavior {
					t.Errorf("FilePathCallback() behavior = %v, want %v", result.Behavior, tt.wantBehavior)
				}
			})
		}
	})

	t.Run("with denied paths", func(t *testing.T) {
		callback := FilePathCallback(nil, []string{"/etc/", "/root/"})

		tests := []struct {
			name         string
			filePath     string
			wantBehavior PermissionBehavior
		}{
			{"Denied /etc/", "/etc/passwd", PermissionDeny},
			{"Denied /root/", "/root/.bashrc", PermissionDeny},
			{"Allowed other path", "/home/user/file.txt", PermissionAllow},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result, _ := callback(ctx, "Write", ToolInput{FilePath: tt.filePath})
				if result.Behavior != tt.wantBehavior {
					t.Errorf("FilePathCallback() behavior = %v, want %v", result.Behavior, tt.wantBehavior)
				}
			})
		}
	})
}

func TestChainCallbacks(t *testing.T) {
	ctx := context.Background()

	t.Run("all allow", func(t *testing.T) {
		cb1 := func(ctx context.Context, tool string, input ToolInput) (PermissionResult, error) {
			return Allow(), nil
		}
		cb2 := func(ctx context.Context, tool string, input ToolInput) (PermissionResult, error) {
			return Allow(), nil
		}

		chained := ChainCallbacks(cb1, cb2)
		result, err := chained(ctx, "Bash", ToolInput{})

		if err != nil {
			t.Errorf("ChainCallbacks() returned error: %v", err)
		}
		if result.Behavior != PermissionAllow {
			t.Errorf("ChainCallbacks() behavior = %v, want %v", result.Behavior, PermissionAllow)
		}
	})

	t.Run("first denies", func(t *testing.T) {
		cb1 := func(ctx context.Context, tool string, input ToolInput) (PermissionResult, error) {
			return Deny("denied by cb1"), nil
		}
		cb2 := func(ctx context.Context, tool string, input ToolInput) (PermissionResult, error) {
			return Allow(), nil
		}

		chained := ChainCallbacks(cb1, cb2)
		result, _ := chained(ctx, "Bash", ToolInput{})

		if result.Behavior != PermissionDeny {
			t.Errorf("ChainCallbacks() behavior = %v, want %v", result.Behavior, PermissionDeny)
		}
		if result.Message != "denied by cb1" {
			t.Errorf("ChainCallbacks() message = %q, want %q", result.Message, "denied by cb1")
		}
	})

	t.Run("second asks", func(t *testing.T) {
		cb1 := func(ctx context.Context, tool string, input ToolInput) (PermissionResult, error) {
			return Allow(), nil
		}
		cb2 := func(ctx context.Context, tool string, input ToolInput) (PermissionResult, error) {
			return Ask("confirm?"), nil
		}

		chained := ChainCallbacks(cb1, cb2)
		result, _ := chained(ctx, "Bash", ToolInput{})

		if result.Behavior != PermissionAsk {
			t.Errorf("ChainCallbacks() behavior = %v, want %v", result.Behavior, PermissionAsk)
		}
	})

	t.Run("handles nil callbacks", func(t *testing.T) {
		cb1 := func(ctx context.Context, tool string, input ToolInput) (PermissionResult, error) {
			return Allow(), nil
		}

		chained := ChainCallbacks(nil, cb1, nil)
		result, err := chained(ctx, "Bash", ToolInput{})

		if err != nil {
			t.Errorf("ChainCallbacks() returned error: %v", err)
		}
		if result.Behavior != PermissionAllow {
			t.Errorf("ChainCallbacks() behavior = %v, want %v", result.Behavior, PermissionAllow)
		}
	})
}
