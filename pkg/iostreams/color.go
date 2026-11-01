package iostreams

import (
	"github.com/fatih/color"
)

// ColorScheme provides terminal color styling methods
type ColorScheme struct {
	enabled bool
}

// Bold formats text in bold
func (c *ColorScheme) Bold(text string) string {
	if !c.enabled {
		return text
	}
	return color.New(color.Bold).Sprint(text)
}

// Success formats text in green for success messages
func (c *ColorScheme) Success(text string) string {
	if !c.enabled {
		return text
	}
	return color.New(color.FgGreen).Sprint(text)
}

// Error formats text in red for error messages
func (c *ColorScheme) Error(text string) string {
	if !c.enabled {
		return text
	}
	return color.New(color.FgRed).Sprint(text)
}

// Warning formats text in yellow for warnings
func (c *ColorScheme) Warning(text string) string {
	if !c.enabled {
		return text
	}
	return color.New(color.FgYellow).Sprint(text)
}

// Gray formats text in faint/gray
func (c *ColorScheme) Gray(text string) string {
	if !c.enabled {
		return text
	}
	return color.New(color.FgHiBlack).Sprint(text)
}
