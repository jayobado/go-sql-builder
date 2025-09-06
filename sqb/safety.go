package sqb

type SafetyProfile struct {
	RequireWhereUpdate bool
	RequireWhereDelete bool
	RequireFromSelect  bool
	RequireLimitSelect bool
}

// Conservative writes, lenient reads — adjust in app init if you like.
var DefaultSafety = SafetyProfile{
	RequireWhereUpdate: true, // ← enabled by default
	RequireWhereDelete: true, // ← enabled by default
	RequireFromSelect:  false,
	RequireLimitSelect: false,
}

type AuditMeta struct {
	Op     string
	Table  string
	Reason string
	SQL    string
	Args   []any
	DryRun bool
}