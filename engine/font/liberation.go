package font

import "fmt"

var liberationMap = map[string]string{
	"Helvetica":            "LiberationSans-Regular",
	"Helvetica-Bold":       "LiberationSans-Bold",
	"Helvetica-Oblique":    "LiberationSans-Italic",
	"Helvetica-BoldOblique": "LiberationSans-BoldItalic",
	"Times-Roman":          "LiberationSerif-Regular",
	"Times-Bold":           "LiberationSerif-Bold",
	"Times-Italic":         "LiberationSerif-Italic",
	"Times-BoldItalic":     "LiberationSerif-BoldItalic",
	"Courier":              "LiberationMono-Regular",
	"Courier-Bold":         "LiberationMono-Bold",
	"Courier-Oblique":      "LiberationMono-Italic",
	"Courier-BoldOblique":  "LiberationMono-BoldItalic",
}

func LiberationFontFor(standardName string) (string, bool) {
	name, ok := liberationMap[standardName]
	return name, ok
}

func LiberationPaths() map[string]string {
	base := "/usr/share/fonts/truetype/liberation"
	return map[string]string{
		"LiberationSans-Regular":    base + "/LiberationSans-Regular.ttf",
		"LiberationSans-Bold":       base + "/LiberationSans-Bold.ttf",
		"LiberationSans-Italic":     base + "/LiberationSans-Italic.ttf",
		"LiberationSans-BoldItalic": base + "/LiberationSans-BoldItalic.ttf",
		"LiberationSerif-Regular":   base + "/LiberationSerif-Regular.ttf",
		"LiberationSerif-Bold":      base + "/LiberationSerif-Bold.ttf",
		"LiberationSerif-Italic":    base + "/LiberationSerif-Italic.ttf",
		"LiberationSerif-BoldItalic": base + "/LiberationSerif-BoldItalic.ttf",
		"LiberationMono-Regular":    base + "/LiberationMono-Regular.ttf",
		"LiberationMono-Bold":       base + "/LiberationMono-Bold.ttf",
		"LiberationMono-Italic":     base + "/LiberationMono-Italic.ttf",
		"LiberationMono-BoldItalic": base + "/LiberationMono-BoldItalic.ttf",
	}
}

type Registry struct {
	fonts map[string]*Font
}

func NewRegistry() *Registry {
	return &Registry{
		fonts: make(map[string]*Font),
	}
}

func (r *Registry) Register(name string, font *Font) {
	r.fonts[name] = font
}

func (r *Registry) Get(name string) *Font {
	return r.fonts[name]
}

func (r *Registry) RegisterStandardFont(name string, basePath string) (*Font, error) {
	liberationName, ok := LiberationFontFor(name)
	if !ok {
		return nil, fmt.Errorf("font: no Liberation mapping for standard font %q", name)
	}

	paths := LiberationPaths()
	libPath, ok := paths[liberationName]
	if !ok || libPath == "" {
		return nil, fmt.Errorf("font: no path for Liberation font %q", liberationName)
	}

	font, err := LoadFromPath(libPath)
	if err != nil {
		return nil, fmt.Errorf("font: loading Liberation font %q from %s: %w", liberationName, libPath, err)
	}

	r.Register(liberationName, font)
	return font, nil
}

func (r *Registry) UsedNames() []string {
	names := make([]string, 0, len(r.fonts))
	for name := range r.fonts {
		names = append(names, name)
	}
	return names
}
