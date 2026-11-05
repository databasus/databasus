package config

import (
	"context"
	"flag"
	"log/slog"
)

const (
	sourceNotConfigured = "not configured"
	sourceCommandLine   = "command line args"
)

// flagBinding ties one CLI flag to its Config field for the whole lifecycle —
// registration, applying an explicitly-set value, source attribution, and
// logging. Collecting these per flag keeps adding a flag a one-entry change
// instead of edits across four parallel blocks that silently drift apart.
type flagBinding struct {
	name  string
	usage string

	// register declares the flag on fs and returns an applier that copies an
	// explicitly-set value onto c and records the command-line source. A
	// zero/empty flag is treated as "not provided" — it never overrides nor
	// unsets a value already loaded from JSON.
	register func(fs *flag.FlagSet) func(c *Config, sources map[string]string)

	// isSet reports whether c already carries a value (loaded from JSON).
	isSet func(c *Config) bool

	// value is the loggable value, masked for secrets.
	value func(c *Config) any
}

func stringBinding(
	name, usage string,
	get func(*Config) string,
	set func(*Config, string),
) flagBinding {
	return flagBinding{
		name:  name,
		usage: usage,
		register: func(fs *flag.FlagSet) func(*Config, map[string]string) {
			value := fs.String(name, "", usage)

			return func(c *Config, sources map[string]string) {
				if *value != "" {
					set(c, *value)
					sources[name] = sourceCommandLine
				}
			}
		},
		isSet: func(c *Config) bool { return get(c) != "" },
		value: func(c *Config) any { return get(c) },
	}
}

func secretBinding(
	name, usage string,
	get func(*Config) string,
	set func(*Config, string),
) flagBinding {
	binding := stringBinding(name, usage, get, set)
	binding.value = func(c *Config) any { return maskSensitive(get(c)) }

	return binding
}

func intBinding(
	name, usage string,
	get func(*Config) int,
	set func(*Config, int),
) flagBinding {
	return flagBinding{
		name:  name,
		usage: usage,
		register: func(fs *flag.FlagSet) func(*Config, map[string]string) {
			value := fs.Int(name, 0, usage)

			return func(c *Config, sources map[string]string) {
				if *value != 0 {
					set(c, *value)
					sources[name] = sourceCommandLine
				}
			}
		},
		isSet: func(c *Config) bool { return get(c) != 0 },
		value: func(c *Config) any { return get(c) },
	}
}

// boolBinding latches: the flag only ever turns the value on, so a persisted
// true is never flipped back off by its absence on the command line.
func boolBinding(
	name, usage string,
	get func(*Config) bool,
	set func(*Config, bool),
) flagBinding {
	return flagBinding{
		name:  name,
		usage: usage,
		register: func(fs *flag.FlagSet) func(*Config, map[string]string) {
			value := fs.Bool(name, false, usage)

			return func(c *Config, sources map[string]string) {
				if *value {
					set(c, true)
					sources[name] = sourceCommandLine
				}
			}
		},
		isSet: func(c *Config) bool { return get(c) },
		value: func(c *Config) any { return get(c) },
	}
}

func (c *Config) flagBindings() []flagBinding {
	return []flagBinding{
		stringBinding("databasus-host", "Databasus server URL (e.g. https://your-server:4005)",
			func(c *Config) string { return c.DatabasusHost },
			func(c *Config, v string) { c.DatabasusHost = v }),
		stringBinding("agent-id", "Verification agent ID",
			func(c *Config) string { return c.AgentID },
			func(c *Config, v string) { c.AgentID = v }),
		secretBinding("token", "Verification agent token",
			func(c *Config) string { return c.Token },
			func(c *Config, v string) { c.Token = v }),
		intBinding("max-cpu", "Total CPU cores available to the agent",
			func(c *Config) int { return c.MaxCPU },
			func(c *Config, v int) { c.MaxCPU = v }),
		intBinding("max-ram-mb", "Total RAM in MB available to the agent",
			func(c *Config) int { return c.MaxRAMMb },
			func(c *Config, v int) { c.MaxRAMMb = v }),
		intBinding("max-disk-gb", "Total scratch disk in GB available to the agent",
			func(c *Config) int { return c.MaxDiskGb },
			func(c *Config, v int) { c.MaxDiskGb = v }),
		intBinding("max-concurrent-jobs", "Number of verifications to run in parallel",
			func(c *Config) int { return c.MaxConcurrentJobs },
			func(c *Config, v int) { c.MaxConcurrentJobs = v }),
		boolBinding("allow-insecure-http",
			"Permit a plain http:// databasus-host (token and backup data sent unencrypted)",
			func(c *Config) bool { return c.AllowInsecureHTTP },
			func(c *Config, v bool) { c.AllowInsecureHTTP = v }),
		stringBinding("verification-pg-image-repo",
			"Container image repo bundling PostgreSQL extensions for verification "+
				"(default "+DefaultVerificationPgImageRepo+"; set to 'postgres' for the stock image)",
			func(c *Config) string { return c.VerificationPgImageRepo },
			func(c *Config, v string) { c.VerificationPgImageRepo = v }),
	}
}

func sourcesFromConfig(bindings []flagBinding, c *Config) map[string]string {
	sources := make(map[string]string, len(bindings))

	for _, binding := range bindings {
		if binding.isSet(c) {
			sources[binding.name] = configFileName
		} else {
			sources[binding.name] = sourceNotConfigured
		}
	}

	return sources
}

func (c *Config) logConfigSources(ctx context.Context, log *slog.Logger, bindings []flagBinding) {
	for _, binding := range bindings {
		log.InfoContext(
			ctx,
			"config setting",
			slog.String("flag", binding.name),
			slog.Any("value", binding.value(c)),
			slog.String("source", c.flagSources[binding.name]),
		)
	}
}
