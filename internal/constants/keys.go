package constants

const (
	// Context Keys
	ContextKeyIsLoggedIn = "isLoggedIn"
	ContextKeySettings   = "settings"

	// Session Keys
	SessionKeyAuthenticated = "authenticated"
	SessionKeySuccessFlash  = "success_flash"

	// Setting Keys
	SettingPassword             = "password"
	SettingOpenAIBaseURL        = "openai_base_url"
	SettingOpenAIToken          = "openai_token"
	SettingOpenAIModel          = "openai_model"
	SettingGithubRepo           = "github_repo"
	SettingGithubBranch         = "github_branch"
	SettingGithubToken          = "github_token"
	SettingGithubBackupCron     = "github_backup_cron"
	SettingGithubLastBackupHash = "github_last_backup_hash"
	SettingWebdavURL            = "webdav_url"
	SettingWebdavUser           = "webdav_user"
	SettingWebdavPassword       = "webdav_password"
	SettingWebdavBackupCron     = "webdav_backup_cron"
	SettingWebdavLastBackupHash = "webdav_last_backup_hash"

	// DEPRECATED: These are for backward compatibility with old setting keys.
	// They are now replaced by SettingGithubBackupCron and SettingWebdavBackupCron.
	SettingGithubInterval = "github_interval"
	SettingWebdavInterval = "webdav_interval"
)
