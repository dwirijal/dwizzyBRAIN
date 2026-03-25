package irag

import "strings"

func publicProviderCode(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case string(ProviderNexure):
		return "n"
	case string(ProviderRyzumi):
		return "r"
	case string(ProviderKanata):
		return "k"
	case string(ProviderYTDLP):
		return "y"
	case string(ProviderChocomilk):
		return "c"
	default:
		return ""
	}
}

func publicProviderCodes(names []string) []string {
	codes := make([]string, 0, len(names))
	for _, name := range names {
		if code := publicProviderCode(name); code != "" {
			codes = append(codes, code)
		}
	}
	return codes
}
