package clock

import "time"

// Beijing is GMT+8, the project-wide civil timezone.
var Beijing = time.FixedZone("CST", 8*3600)

func Now() time.Time {
	return time.Now().In(Beijing)
}

func Format(t time.Time) string {
	return t.In(Beijing).Format("2006-01-02 15:04:05")
}

func FormatRFC3339(t time.Time) string {
	return t.In(Beijing).Format(time.RFC3339)
}
