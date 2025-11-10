package util

import "time"

func UserUpdateIfNotEmpty(u, v string) string {
	if v != "" {
		return v
	}
	return u
}

func UserUpdateIfNotNil[T any](u, v *T) *T {
	if v != nil {
		return v
	}
	return u
}

func FormatTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format("2006-01-02T15:04:05.999Z")
}