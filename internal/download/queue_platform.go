package download

import (
	"strings"

	"gamarr/internal/platform"
)

func applyQueuePlatform(searchSlug, platf, platSlug string, isPC bool) (string, string, bool) {
	p, s, pc, _ := platform.ApplySearchPlatformFilter(searchSlug, platf, platSlug, isPC)
	return p, s, pc
}

func searchPlatformJobFields(searchSlug string) map[string]interface{} {
	searchSlug = strings.TrimSpace(strings.ToLower(searchSlug))
	if searchSlug == "" || searchSlug == "all" {
		return nil
	}
	return map[string]interface{}{"search_platform_slug": searchSlug}
}

func mergeJobFields(job map[string]interface{}, extra map[string]interface{}) {
	for k, v := range extra {
		job[k] = v
	}
}
