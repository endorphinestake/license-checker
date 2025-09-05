package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sync"
)

//go:embed locales/*.json
var embeddedLocales embed.FS

var locales map[string]map[string]string
var once sync.Once

func load() {
	once.Do(func() {
		locales = make(map[string]map[string]string)
		// read embedded locale files from the compiled binary
		entries, err := fs.ReadDir(embeddedLocales, "locales")
		if err != nil {
			fmt.Printf("i18n: failed to read embedded locales: %v\n", err)
			return
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if len(name) < 6 || name[len(name)-5:] != ".json" {
				continue
			}
			b, err := embeddedLocales.ReadFile("locales/" + name)
			if err != nil {
				fmt.Printf("i18n: failed to read embedded file %s: %v\n", name, err)
				continue
			}
			var m map[string]string
			if err := json.Unmarshal(b, &m); err != nil {
				fmt.Printf("i18n: failed to parse embedded %s: %v\n", name, err)
				continue
			}
			lang := name[:len(name)-5]
			locales[lang] = m
		}
	})
}

// T returns translation for key in locale (fallback to en)
func T(locale, key string) string {
	load()
	if locale != "" {
		if m, ok := locales[locale]; ok {
			if v, ok := m[key]; ok {
				return v
			}
		}
	}
	if m, ok := locales["en"]; ok {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return key
}

// DetectLocale normalizes a provided locale string to 2-letter code
func DetectLocale(raw string) string {
	if raw == "" {
		return ""
	}
	if len(raw) >= 2 {
		return raw[:2]
	}
	return raw
}
