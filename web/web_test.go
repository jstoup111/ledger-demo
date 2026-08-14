package web

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestEmbeddedStylesheetActivatesBalanceErrorAndTableRules(t *testing.T) {
	stylesheet, err := FS.ReadFile("style.css")
	if err != nil {
		t.Fatalf("read embedded style.css: %v", err)
	}

	rawCSS := string(stylesheet)
	activeCSS := regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(rawCSS, "")
	rootFontSize := regexp.MustCompile(`(?ms)^\s*(?:html|:root)\s*\{[^}]*font-size:\s*20px;[^}]*\}`)
	if !rootFontSize.MatchString(activeCSS) {
		t.Error("active stylesheet must declare font-size: 20px on html or :root")
	}

	for name, rule := range map[string]*regexp.Regexp{
		"balance":                     regexp.MustCompile(`(?s)\.balance\s*\{[^}]*font-size:\s*4rem;[^}]*font-weight:\s*700;[^}]*\}`),
		"account navigation spacing":  regexp.MustCompile(`(?s)nav\s+a\s*\{[^}]*margin-right:\s*[^;]+;[^}]*\}`),
		"selected account body scale": regexp.MustCompile(`(?s)\.selected-account\s*\{[^}]*font-size:\s*1rem;[^}]*font-weight:\s*(?:400|normal);[^}]*\}`),
		"error":                       regexp.MustCompile(`(?s)\.error\s*\{[^}]*background:\s*#fdecea;[^}]*border-left:\s*6px\s+solid\s+#b3261e;[^}]*\}`),
		"table":                       regexp.MustCompile(`(?s)table\s*\{[^}]*border-collapse:\s*collapse;[^}]*\}`),
	} {
		if !rule.MatchString(activeCSS) {
			t.Errorf("active stylesheet is missing the required %s rule", name)
		}
	}

	for _, forbidden := range []string{"@media", "prefers-color-scheme", "@keyframes", "@font-face"} {
		if strings.Contains(rawCSS, forbidden) {
			t.Errorf("stylesheet must not contain %q", forbidden)
		}
	}
}

func TestEmbeddedStylesheetActivatesProjectorLegibleDownloadButtonRule(t *testing.T) {
	stylesheet, err := FS.ReadFile("style.css")
	if err != nil {
		t.Fatalf("read embedded style.css: %v", err)
	}

	activeCSS := regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(string(stylesheet), "")
	downloadRules := regexp.MustCompile(`(?s)\.download\s*\{([^}]*)\}`).FindAllStringSubmatch(activeCSS, -1)
	buttonLike := false
	for _, rule := range downloadRules {
		declarations := rule[1]
		if regexp.MustCompile(`display:\s*inline-block\s*;`).MatchString(declarations) &&
			regexp.MustCompile(`padding:\s*[^;]+;`).MatchString(declarations) &&
			regexp.MustCompile(`background(?:-color)?:\s*[^;]+;`).MatchString(declarations) &&
			regexp.MustCompile(`(?:^|;)\s*color:\s*[^;]+;`).MatchString(declarations) &&
			regexp.MustCompile(`font-weight:\s*(?:700|bold)\s*;`).MatchString(declarations) {
			buttonLike = true
			break
		}
	}
	if !buttonLike {
		t.Errorf("active stylesheet has %d .download rules, want one with inline-block display, padding, background, foreground color, and bold text", len(downloadRules))
	}

	for _, forbidden := range []string{"@media", "prefers-color-scheme", "@keyframes", "@font-face"} {
		if strings.Contains(string(stylesheet), forbidden) {
			t.Errorf("stylesheet must not contain %q", forbidden)
		}
	}
}

func TestOfflineAssetsAndDependencies(t *testing.T) {
	stylesheet, err := FS.ReadFile("style.css")
	if err != nil {
		t.Fatalf("read embedded style.css: %v", err)
	}

	activeCSS := strings.ToLower(string(stylesheet))
	for _, forbidden := range []string{"@import", "@font-face"} {
		if strings.Contains(activeCSS, forbidden) {
			t.Errorf("stylesheet must not contain %q", forbidden)
		}
	}

	goMod, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	directRequirements := regexp.MustCompile(`(?m)^require\s+([^\s]+)\s+[^\s]+\s*$`).FindAllStringSubmatch(string(goMod), -1)
	if got, want := len(directRequirements), 1; got != want {
		t.Fatalf("direct non-standard-library requirements = %d, want %d", got, want)
	}
	if got, want := directRequirements[0][1], "modernc.org/sqlite"; got != want {
		t.Errorf("direct non-standard-library requirement = %q, want %q", got, want)
	}
}
