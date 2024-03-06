package utils

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// GetLanguagesFromPath return the available languages info form specified path.
func GetLanguagesFromPath(dir string) (languages *Languages, err error) {
	fileSystem := os.DirFS(dir)

	return getLanguages(fileSystem)
}

// GetLanguagesFromPath return the available languages info form specified path.
func GetLanguagesFromEmbedFS(fs embed.FS) (languages *Languages, err error) {
	return getLanguages(fs)
}

//nolint:gocyclo
func getLanguages(dir fs.FS) (languages *Languages, err error) {
	//nolint:prealloc
	var locales []string

	languages = &Languages{
		Defaults: DefaultsLanguages{
			Namespace: localeNamespaceDefault,
		},
	}

	var defaultTag language.Tag

	if defaultTag, err = language.Parse(localeDefault); err != nil {
		return nil, fmt.Errorf("failed to parse default language: %w", err)
	}

	caser := cases.Title(defaultTag)

	languages.Defaults.Language = Language{
		Display: caser.String(display.Self.Name(defaultTag)),
		Locale:  localeDefault,
	}

	if err = fs.WalkDir(dir, ".", func(path string, info fs.DirEntry, errWalk error) (err error) {
		if errWalk != nil {
			return errWalk
		}
		if info.IsDir() {
			return nil
		}

		nameLower := strings.ToLower(info.Name())
		ext := filepath.Ext(nameLower)

		ns := strings.Replace(nameLower, ext, "", 1)

		if ext != extJSON {
			return nil
		}

		if !IsStringInSlice(ns, languages.Namespaces) {
			languages.Namespaces = append(languages.Namespaces, ns)
		}

		fdir, _ := filepath.Split(path)

		locale := filepath.Base(fdir)

		if IsStringInSlice(locale, locales) {
			for i, l := range languages.Languages {
				if l.Locale == locale {
					if IsStringInSlice(ns, languages.Languages[i].Namespaces) {
						break
					}

					languages.Languages[i].Namespaces = append(languages.Languages[i].Namespaces, ns)
					break
				}
			}

			return nil
		}

		var tag language.Tag

		if tag, err = language.Parse(locale); err != nil {
			return fmt.Errorf("failed to parse language '%s': %w", locale, err)
		}

		caser := cases.Title(tag)

		l := Language{
			Display:    caser.String(display.Self.Name(tag)),
			Locale:     locale,
			Namespaces: []string{ns},
			Fallbacks:  []string{languages.Defaults.Language.Locale},
			Tag:        tag,
		}

		languages.Languages = append(languages.Languages, l)

		locales = append(locales, l.Locale)

		return nil
	}); err != nil {
		return nil, err
	}

	var langs []Language //nolint:prealloc

	// adding locale fallbacks.
	for i, lang := range languages.Languages {
		p := lang.Tag.Parent()

		if p.String() == "und" || strings.Contains(p.String(), "-") {
			continue
		}

		if p.String() != lang.Locale {
			lang.Fallbacks = append([]string{p.String()}, lang.Fallbacks...)
			lang.Parent = p.String()
		}

		languages.Languages[i] = lang

		if IsStringInSlice(p.String(), locales) {
			continue
		}

		caser := cases.Title(lang.Tag)
		l := Language{
			Display:    caser.String(display.Self.Name(p)),
			Locale:     p.String(),
			Namespaces: lang.Namespaces,
			Fallbacks:  []string{languages.Defaults.Language.Locale},
			Tag:        p,
		}

		langs = append(langs, l)

		locales = append(locales, l.Locale)
	}

	languages.Languages = append(languages.Languages, langs...)

	sort.Slice(languages.Languages, func(i, j int) bool {
		return languages.Languages[i].Locale == localeDefault || languages.Languages[i].Locale < languages.Languages[j].Locale
	})

	return languages, nil
}