package examples

import (
	"regexp"
)

func paveAnsicolors(raw []string) (clean []string) {
	clean = make([]string, len(raw))
	matcher := regexp.MustCompile("\x1b" + `\[[0-9;]+m`)
	for i := range raw {
		clean[i] = matcher.ReplaceAllString(raw[i], "")
	}
	return
}

func paveLogtimes(raw []string) (clean []string) {
	clean = make([]string, len(raw))
	matcher := regexp.MustCompile(`\[[0-9]{2}-[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2}\]`)
	for i := range raw {
		clean[i] = matcher.ReplaceAllString(raw[i], "[MM-DD hh:mm:ss]")
	}
	return
}

func paveRunrecords(raw []string) (clean []string) {
	clean = make([]string, len(raw))
	matcher := regexp.MustCompile(`"guid": "[a-zA-Z0-9]{8}-[a-zA-Z0-9]{8}-[a-zA-Z0-9]{8}"`)
	for i := range raw {
		clean[i] = matcher.ReplaceAllString(raw[i], `"guid": "xxxxxxxx-xxxxxxxx-xxxxxxxx"`)
	}
	matcher = regexp.MustCompile(`"time": [0-9]+`)
	for i := range clean {
		clean[i] = matcher.ReplaceAllString(clean[i], `"time": "22222222222"`)
	}
	matcher = regexp.MustCompile(`"hostname": ".*"`)
	for i := range clean {
		clean[i] = matcher.ReplaceAllString(clean[i], `"hostname": "znn.xxxxx.yyy"`)
	}
	return
}
