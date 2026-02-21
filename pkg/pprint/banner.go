// Package pprint: Orbit ASCII banner.
package pprint

import "fmt"

// banner is the ASCII art shown at startup and in help output.
const banner = `
   ██████╗ ██████╗ ██████╗ ██╗████████╗
  ██╔═══██╗██╔══██╗██╔══██╗██║╚══██╔══╝
  ██║   ██║██████╔╝██████╔╝██║   ██║
  ██║   ██║██╔══██╗██╔══██╗██║   ██║
  ╚██████╔╝██║  ██║██████╔╝██║   ██║
   ╚═════╝ ╚═╝  ╚═╝╚═════╝ ╚═╝   ╚═╝
`

// PrintBanner prints the Orbit banner with version and tagline.
func PrintBanner(version, buildDate string) {
	// Gradient-style coloring using Lipgloss
	line1 := StylePrimary.Render("  ██████╗ ██████╗ ██████╗ ██╗████████╗")
	line2 := StylePrimary.Render(" ██╔═══██╗██╔══██╗██╔══██╗██║╚══██╔══╝")
	line3 := StyleAccent.Render(" ██║   ██║██████╔╝██████╔╝██║   ██║")
	line4 := StyleAccent.Render(" ██║   ██║██╔══██╗██╔══██╗██║   ██║")
	line5 := StyleText.Render(" ╚██████╔╝██║  ██║██████╔╝██║   ██║")
	line6 := StyleMuted.Render("  ╚═════╝ ╚═╝  ╚═╝╚═════╝ ╚═╝   ╚═╝")

	fmt.Println()
	fmt.Println(line1)
	fmt.Println(line2)
	fmt.Println(line3)
	fmt.Println(line4)
	fmt.Println(line5)
	fmt.Println(line6)
	fmt.Println()

	tagline := StyleMuted.Render("  Container orchestration for self-hosted infrastructure")
	versionStr := StyleAccent.Render("  " + version)
	if buildDate != "" {
		versionStr += StyleMuted.Render("  built " + buildDate)
	}

	fmt.Println(tagline)
	fmt.Println(versionStr)
	fmt.Println()
}

// PrintBannerSmall prints a compact single-line brand prefix.
func PrintBannerSmall() {
	fmt.Print(StylePrimary.Render("◉ ORBIT") + " ")
}

// _ suppress unused import
var _ = banner
