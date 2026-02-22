package services

import "strings"

const (
	CoverStatusNone       = "none"
	CoverStatusGenerating = "generating"
	CoverStatusReady      = "ready"
	CoverStatusFailed     = "failed"
	CoverStatusCancelled  = "cancelled"

	legacyCoverGeneratingPrefix = "glog:cover:generating:"
)

func resolveCoverStatusFromCover(cover string) string {
	if strings.TrimSpace(cover) == "" {
		return CoverStatusNone
	}
	return CoverStatusReady
}

func normalizeCoverStatus(status, cover string) string {
	status = strings.TrimSpace(status)
	if status == "" {
		return resolveCoverStatusFromCover(cover)
	}

	switch status {
	case CoverStatusNone, CoverStatusGenerating, CoverStatusReady, CoverStatusFailed, CoverStatusCancelled:
		return status
	default:
		return resolveCoverStatusFromCover(cover)
	}
}
