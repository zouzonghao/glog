package constants

// Settings keys used in database, forms, and templates.
const (
	SettingPassword        = "password"
	SettingSiteLogo        = "site_logo"
	SettingSiteTitle       = "site_title"
	SettingSiteDescription = "site_description"
	SettingOpenAIBaseURL   = "openai_base_url"
	SettingOpenAIToken     = "openai_token"
	SettingOpenAIModel     = "openai_model"

	// Remote backup settings
	SettingGithubRepo           = "github_repo"
	SettingGithubBranch         = "github_branch"
	SettingGithubToken          = "github_token"
	SettingGithubInterval       = "github_interval" // in hours
	SettingWebdavURL            = "webdav_url"
	SettingWebdavUser           = "webdav_user"
	SettingWebdavPassword       = "webdav_password"
	SettingWebdavInterval       = "webdav_interval" // in hours
	SettingGithubLastBackupHash = "github_last_backup_hash"
	SettingWebdavLastBackupHash = "webdav_last_backup_hash"
)

// Session keys
const (
	SessionKeyAuthenticated = "authenticated"
	SessionKeySuccessFlash  = "success"
)

// Context keys
const (
	ContextKeySettings   = "settings"
	ContextKeyIsLoggedIn = "IsLoggedIn"
)
