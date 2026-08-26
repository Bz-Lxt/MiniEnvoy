package proxy

func ShouldPause(used, high int) bool { return used >= high }

func ShouldResume(used, low int) bool { return used <= low }
