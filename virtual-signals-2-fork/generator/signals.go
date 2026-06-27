package main

import (
	"image/color"

	"github.com/lucasb-eyer/go-colorful"
)

type Background int

const (
	Transparent Background = iota
	Dark
	Colored
	BlackFull
)

type ForeGround int

const (
	PerGroup ForeGround = iota
	White
	Color
)

// Config holds all settings for one generation run.
type Config struct {
	ModName            string
	Description        string
	BackGroundType     Background
	BackGroundScaleMax bool
	Shadow             bool
	ThumbnailPath      string
	ForeGroundType     ForeGround
}

const ModVersion = "2.0.0"
const DescriptionPath = "descriptions/description.md"

// allConfigs lists every mod variant to generate, in order.
var allConfigs = []Config{
	{
		ModName:            "virtual-signals-2-fork",
		Description:        "Larger Virtual Signals 2 fork with transparent background, black contour, extra signals, and reorderable groups.",
		BackGroundType:     Transparent,
		BackGroundScaleMax: false,
		Shadow:             true,
		ThumbnailPath:      "thumbnails/thumbnail.png",
		ForeGroundType:     Color,
	},
	{
		ModName:     "virtual-signals-2-fork-white",
		Description: "White variant of Virtual Signals 2 Fork with transparent background, extra signals, and reorderable groups.",

		BackGroundType:     Transparent,
		BackGroundScaleMax: false,
		Shadow:             true,
		ThumbnailPath:      "thumbnails/thumbnail-white.png",
		ForeGroundType:     White,
	},
	{
		ModName:            "virtual-signals-2-fork-color",
		Description:        "Color variant of Virtual Signals 2 Fork with tinted backgrounds, extra signals, and reorderable groups.",
		BackGroundType:     Colored,
		BackGroundScaleMax: true,
		Shadow:             false,
		ThumbnailPath:      "thumbnails/thumbnail-color.png",
		ForeGroundType:     PerGroup,
	},
	//{
	//	ModName:            "virtual-signals-2-fork-foreground-color",
	//	Description:        "Larger Virtual Signals 2 fork with transparent background, black contour, extra signals, and reorderable groups.",
	//	BackGroundType:     Transparent,
	//	BackGroundScaleMax: false,
	//	Shadow:             true,
	//	ThumbnailPath:      "thumbnails/thumbnail.png",
	//	ForeGroundType:     PerGroup,
	//},
	//{
	//	ModName:            "virtual-signals-2-fork-dark",
	//	Description:        "Dark variant of Virtual Signals 2 Fork with an enlarged dark background, extra signals, and reorderable groups.",
	//	BackGroundType:     Dark,
	//	BackGroundScaleMax: true,
	//	Shadow:             false,
	//	ThumbnailPath:      "thumbnails/thumbnail.png",
	//	ForeGroundType:     PerGroup,
	//},
	//{
	//	ModName:            "virtual-signals-2-fork-dark-def-size",
	//	Description:        "Dark variant of Virtual Signals 2 Fork with a standard-size dark background, extra signals, and reorderable groups.",
	//	BackGroundType:     Dark,
	//	BackGroundScaleMax: false,
	//	Shadow:             false,
	//	ThumbnailPath:      "thumbnails/thumbnail.png",
	//	ForeGroundType:     PerGroup,
	//},
	//{
	//	ModName:            "virtual-signals-2-fork-black-full",
	//	Description:        "Black variant of Virtual Signals 2 Fork with a full black background, extra signals, and reorderable groups.",
	//	BackGroundType:     BlackFull,
	//	BackGroundScaleMax: false,
	//	Shadow:             false,
	//	ThumbnailPath:      "thumbnails/thumbnail.png",
	//	ForeGroundType:     PerGroup,
	//},

}

func (cfg Config) backgroundFileForGroup(light bool) string {

	switch cfg.BackGroundType {
	case Dark:
		return "background/background_default"
	case BlackFull:
		return "background/background_black_full"
	case Colored:
		if light {
			return "background/background_white"
		} else {
			return "background/background_default"
		}
	default: // Transparent
		return "background/background_alpha"
	}
}

func (cfg Config) dirMod() string   { return "../" + cfg.ModName + "/" }
func (cfg Config) dirProto() string { return cfg.dirMod() + "prototypes/" }

const (
	FontIconsPath = "fonts/Awesome_7_Free_Solid_900.otf"
	FontLabelPath = "fonts/Titillium_Web_Bold.ttf"
	FontMathPath  = "fonts/Noto_Sans_Math_Regular.ttf"
)

const (
	groupNumbers     = "virtual-signal-number"
	groupLetters     = "virtual-signal-letter"
	groupPunctuation = "virtual-signal-punctuation"
	groupMath2       = "virtual-signal-math"

	groupColors = "virtual-signal-color"

	groupCommon    = "vs2-cmn"
	groupCommon2   = "vs2-cmn2"
	groupCommon3   = "vs2-cmn3"
	groupMath      = "vs2-math"
	groupGreek     = "vs2-greek"
	groupIcons     = "vs2-fa"
	groupRegisters = "vs2-registers"

	groupShapes        = "shapes"
	groupVirtualSignal = "virtual-signal"
	groupArrows        = "arrows"
)

// HSV creates a fully opaque color.RGBA from Hue (0-360), Saturation (0-1), and Value (0-1)
func HSV(h, s, v float64) color.RGBA {
	return HSVA(h, s, v, 1.0)
}

// HSVA creates a translucent color.RGBA from HSV + Alpha (0-1)
func HSVA(h, s, v, a float64) color.RGBA {
	c := colorful.Hsv(h, s, v)
	return color.RGBA{
		R: uint8(c.R * 255),
		G: uint8(c.G * 255),
		B: uint8(c.B * 255),
		A: uint8(a * 255),
	}
}

type group struct {
	order           string
	name            string
	backgroundFile  string
	ForegroundColor color.RGBA
	BackgroundColor color.RGBA
	glyphType       glyphType
	fineTune        []fineTune
	vanilla         bool
}

type signal struct {
	group    groupID
	name     string
	localeEn string
	localeRu string
	content  string
	fineTune []fineTune
	vanilla  bool
	order    string
	hidden   bool
}

func makeGroups(cfg Config) map[groupID]group {
	// Dark/Light distinguishing for Colored BackGroundType. Other types resolved to the same file
	bgDark := cfg.backgroundFileForGroup(false)
	bgLight := cfg.backgroundFileForGroup(true)

	return map[groupID]group{

		groupNumbers: {
			order:           "b",
			name:            groupNumbers,
			backgroundFile:  bgDark,
			ForegroundColor: HSV(0, 0, 1),
			BackgroundColor: HSV(0, 0, 0),
			glyphType:       glyphTypeLabel,
			vanilla:         true,
			fineTune: []fineTune{
				{textScale: 1.35},
			},
		},

		groupLetters: {
			order:           "c",
			name:            groupLetters,
			backgroundFile:  bgDark,
			ForegroundColor: HSV(0, 0, 1),
			BackgroundColor: HSV(0, 0, 0),
			glyphType:       glyphTypeLabel,
			vanilla:         true,
			fineTune: []fineTune{
				{textScale: 1.35},
			},
		},

		groupPunctuation: {
			order:           "cb",
			name:            groupPunctuation,
			backgroundFile:  bgDark,
			ForegroundColor: HSV(0, 0, 1),
			BackgroundColor: HSV(0, 0, 0),
			glyphType:       glyphTypeLabel,
			vanilla:         true,
			fineTune: []fineTune{
				{textScale: 1.35},
			},
		},

		groupMath2: {
			order:           "cd",
			name:            groupMath2,
			backgroundFile:  bgDark,
			ForegroundColor: HSV(0, 0, 1),
			BackgroundColor: HSV(0, 0, 0),
			glyphType:       glyphTypeLabel,
			vanilla:         true,
			fineTune: []fineTune{
				{textScale: 1.35},
			},
		},

		groupColors: {
			order:          "d",
			name:           groupColors,
			backgroundFile: "background/background_alpha",
			glyphType:      glyphTypeIcon,
			vanilla:        true,
		},

		groupCommon: {
			order:           "d_vs2[10]",
			name:            groupCommon,
			backgroundFile:  bgLight,
			ForegroundColor: HSV(0, 0.3, 1),
			BackgroundColor: HSV(0, 0.5, 1),
			glyphType:       glyphTypeLabel,
			fineTune: []fineTune{
				{textScale: 1.35},
			},
		},
		groupCommon2: {
			order:           "d_vs2[20]",
			name:            groupCommon2,
			backgroundFile:  bgLight,
			ForegroundColor: HSV(43, 0.3, 1),
			BackgroundColor: HSV(43, 0.5, 1),
			glyphType:       glyphTypeLabel,
			fineTune: []fineTune{
				{textScale: 1.35},
			},
		},
		groupCommon3: {
			order:           "d_vs2[30]",
			name:            groupCommon3,
			backgroundFile:  bgLight,
			ForegroundColor: HSV(78, 0.3, 1),
			BackgroundColor: HSV(78, 0.5, 1),
			glyphType:       glyphTypeLabel,
			fineTune: []fineTune{
				{textScale: 1.35},
			},
		},
		groupMath: {
			order:           "d_vs2[40]",
			name:            groupMath,
			backgroundFile:  bgLight,
			ForegroundColor: HSV(100, 0.3, 1),
			BackgroundColor: HSV(100, 0.5, 1),
			glyphType:       glyphTypeMath,
			fineTune: []fineTune{
				{textScale: 1.35},
			},
		},
		groupGreek: {
			order:           "d_vs2[50]",
			name:            groupGreek,
			backgroundFile:  bgLight,
			ForegroundColor: HSV(130, 0.3, 1),
			BackgroundColor: HSV(130, 0.5, 1),
			glyphType:       glyphTypeMath,
			fineTune: []fineTune{
				{textScale: 1.35},
			},
		},
		groupIcons: {
			order:           "d_vs2[60]",
			name:            groupIcons,
			backgroundFile:  bgLight,
			ForegroundColor: HSV(285, 0.3, 1),
			BackgroundColor: HSV(285, 0.5, 1),
			glyphType:       glyphTypeIcon,
			fineTune: []fineTune{
				{textScale: 1.35},
			},
		},
		groupRegisters: {
			order:           "d_vs2[70]",
			name:            groupRegisters,
			backgroundFile:  bgLight,
			ForegroundColor: HSV(325, 0.3, 1),
			BackgroundColor: HSV(325, 0.5, 1),
			glyphType:       glyphTypeLabel,
			fineTune: []fineTune{
				{textScale: 1.35},
			},
		},

		groupVirtualSignal: {
			order:   "e",
			name:    groupVirtualSignal,
			vanilla: true,
		},

		groupShapes: {
			order:          "f",
			name:           groupShapes,
			backgroundFile: "background/sig_alpha",
			glyphType:      glyphTypeIcon,
			vanilla:        true,
			fineTune: []fineTune{
				{textScale: 1.35},
			},
		},

		groupArrows: {
			order:   "g",
			name:    groupArrows,
			vanilla: true,
		},
	}
}
