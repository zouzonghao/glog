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
