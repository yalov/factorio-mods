package main

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"

	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
)

const directory = "./"

type (
	glyphType int
	groupID   string
)

const (
	glyphTypeMath glyphType = iota
	glyphTypeIcon
	glyphTypeLabel
)

type fineTune struct {
	offsetX      float64 // offset in % of glyph height
	offsetY      float64 // offset in % of glyph height
	textScale    float64 // 1 = no scale
	useLabelFont bool
	useIconFont  bool
}

const (
	iconWidth         = 64
	iconHeight        = 64
	previewCellsWidth = 10
	previewCellsSize  = 64
)

var loadedFonts = make(map[string]font.Face)
var loadedImages = make(map[string]image.Image)
var groupSignals = map[groupID][]signal{}

func main() {
	loadSignalsFromCSV()

	for _, cfg := range allConfigs {
		fmt.Printf("Generating %s...\n", cfg.ModName)
		loadedImages = make(map[string]image.Image) // reset image cache per config
		runConfig(cfg)
	}
}

func runConfig(cfg Config) {
	groups := makeGroups(cfg)

	// create folder
	os.MkdirAll(cfg.dirMod()+"graphics/signal", 0777)

	previewHeightCells := 0
	for groupID := range groups {
		iconcounter := 0
		for _, sig := range groupSignals[groupID] {
			if !sig.hidden {
				if iconcounter%previewCellsWidth == 0 {
					previewHeightCells++
				}
				iconcounter++
			}
		}
	}

	previewCanvas := gg.NewContext(previewCellsWidth*previewCellsSize, previewHeightCells*previewCellsSize)

	centerX, centerY := iconWidth/2, iconHeight/2
	previewSellX := 0
	previewSellY := 0

	for _, groupID := range groupIDs(groups) {
		group := groups[groupID]

		for _, sig := range groupSignals[groupID] {

			var dc *gg.Context

			background := loadBackground(cfg, group.backgroundFile)
			if cfg.BackGroundType == Colored && group.vanilla == false {
				background = colorizeImage(background, group.BackgroundColor)
			}
			dc = gg.NewContextForImage(background)

			if foreground_image := TryLoadAndResizeImage("signals/"+sig.name, 64); foreground_image != nil {
				if cfg.Shadow {
					blackImage := colorizeImage(foreground_image, HSV(0, 0, 0))
					edgeSize := 2
					// 8-way shift for a solid border
					offsets := [][2]int{
						{-1, -1}, {0, -1}, {1, -1},
						{-1, 0}, {1, 0},
						{-1, 1}, {0, 1}, {1, 1},
					}

					for _, offset := range offsets {
						dx := offset[0] * edgeSize
						dy := offset[1] * edgeSize
						dc.DrawImageAnchored(blackImage, centerX+dx, centerY+dy, 0.5, 0.5)
					}
				}

				if cfg.ForeGroundType == PerGroup {
					if cfg.BackGroundType == Colored {
						if group.vanilla {
							// do not change color for vanilla groups for color background
							//image = colorizeImage(image, HSV(0, 0, 1))
						} else {
							// make foreground a little transparent for non-vanilla groups for color background
							foreground_image = colorizeImage(foreground_image, HSVA(0, 0, 0, 0.7))
						}
					} else {
						foreground_image = colorizeImage(foreground_image, group.ForegroundColor)
					}
				}

				// 3. Draw the main image on top
				dc.DrawImageAnchored(foreground_image, centerX, centerY, 0.5, 0.5)

			} else {

				offsetX, offsetY, fontHeight := float64(0), float64(0), float64(1)
				fontHeightBig := float64(iconWidth) / 2
				fontHeightMd := float64(iconWidth) / 2.75

				var fontName string

				switch group.glyphType {
				case glyphTypeMath:
					offsetY -= float64(iconHeight / 10)
					fontName = FontMathPath
					fontHeight = fontHeightBig
				case glyphTypeIcon:
					offsetY -= float64(iconHeight / 10)
					fontName = FontIconsPath
					fontHeight = fontHeightBig

				default:
					offsetY -= float64(iconHeight / 20)
					fontName = FontLabelPath
					fontHeight = fontHeightMd
				}

				for _, ft := range sig.fineTune {
					settings := ft
					if settings.textScale != 0 {
						fontHeight *= settings.textScale
					}
					offsetX += (settings.offsetX * 0.01) * fontHeight
					offsetY += (settings.offsetY * 0.01) * fontHeight

					if settings.useLabelFont {
						fontName = FontLabelPath
					}
					if settings.useIconFont {
						fontName = FontIconsPath
					}
				}

				for _, ft := range group.fineTune {
					settings := ft
					if settings.textScale != 0 {
						fontHeight *= settings.textScale
					}
					offsetX += (settings.offsetX * 0.01) * fontHeight
					offsetY += (settings.offsetY * 0.01) * fontHeight

					if settings.useLabelFont {
						fontName = FontLabelPath
					}
					if settings.useIconFont {
						fontName = FontIconsPath
					}
				}

				if cfg.Shadow {
					shadowSize := float64(2)
					shadowScale := 1.03

					textFont := loadFont(fontName, fontHeight*shadowScale)
					dc.SetFontFace(textFont)
					dc.SetColor(HSV(0, 0, 0))
					dc.DrawStringAnchored(sig.content, float64(centerX)+offsetX-shadowSize, float64(centerY)+offsetY-shadowSize, 0.5, 0.5)
					dc.DrawStringAnchored(sig.content, float64(centerX)+offsetX+shadowSize, float64(centerY)+offsetY-shadowSize, 0.5, 0.5)
					dc.DrawStringAnchored(sig.content, float64(centerX)+offsetX+shadowSize, float64(centerY)+offsetY+shadowSize, 0.5, 0.5)
					dc.DrawStringAnchored(sig.content, float64(centerX)+offsetX-shadowSize, float64(centerY)+offsetY+shadowSize, 0.5, 0.5)
				}

				// text
				textFont := loadFont(fontName, fontHeight)
				dc.SetFontFace(textFont)

				if cfg.ForeGroundType == PerGroup {
					if cfg.BackGroundType == Colored {
						if group.vanilla {
							dc.SetColor(HSV(0, 0, 1))
						} else {
							dc.SetColor(HSVA(0, 0, 0, 0.7))
						}
					} else {
						dc.SetColor(group.ForegroundColor)
					}
				} else if cfg.ForeGroundType == White || cfg.ForeGroundType == Color && group.name == groupIcons {
					dc.SetColor(HSV(0, 0, 1))
				} else {
					dc.SetColor(group.ForegroundColor)
				}

				dc.DrawStringAnchored(sig.content, float64(centerX)+offsetX, float64(centerY)+offsetY, 0.5, 0.5)
			}

			// save
			mip := ApplyMipmaps(dc, 4)
			err := mip.SavePNG(cfg.dirMod() + "graphics/signal/" + group.name + "_" + sig.name + ".png")
			if err != nil {
				panic(fmt.Errorf("failed to save png: %w", err))
			}

			// add to preview
			if !sig.hidden {
				finalIcon := dc.Image()

				// 1. Compute grid coordinates
				posX := previewSellX * previewCellsSize
				posY := previewSellY * previewCellsSize

				// 2. Define the exact sub-rectangle on the preview atlas canvas
				dstRect := image.Rect(posX, posY, posX+previewCellsSize, posY+previewCellsSize)

				// 3. Draw and resize the icon directly into the sheet
				draw.BiLinear.Scale(
					previewCanvas.Image().(draw.Image),
					dstRect,
					finalIcon,
					finalIcon.Bounds(),
					draw.Over,
					nil,
				)

				// 4. Draw grid border rectangle over the image
				previewCanvas.SetColor(HSV(0, 0, 0.38))
				previewCanvas.DrawRectangle(
					float64(posX),
					float64(posY),
					float64(previewCellsSize),
					float64(previewCellsSize),
				)
				previewCanvas.Stroke()

				// 5. Advance grid cells
				previewSellX++
				if previewSellX >= previewCellsWidth {
					previewSellX = 0
					previewSellY++
				}
			}
		}

		if previewSellX != 0 {
			previewSellX = 0
			previewSellY++
		}
	}

	// export preview
	err := previewCanvas.SavePNG("preview_" + cfg.ModName + ".png")
	if err != nil {
		panic(fmt.Errorf("failed to save png: %w", err))
	}

	// gen lua files
	createLuaSubGroups(cfg, groups)
	createLuaSignals(cfg, groups)
	createLocales(cfg, groups)
	createLuaVanilla(cfg, groups)
	createLuaSetting(cfg, groups)
	createLuaData(cfg)
	createInfoJSON(cfg)
	createGroup_vs2(cfg)
	copyDescription(cfg)
	copyThumbnail(cfg)
}

func invertImageColors(img image.Image) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			r, g, b, a := originalColor.RGBA()

			// Normalize to 8-bit values, invert, and create new color
			newColor := color.RGBA{
				R: uint8(255 - r/257), // Convert 16-bit to 8-bit and invert
				G: uint8(255 - g/257),
				B: uint8(255 - b/257),
				A: uint8(a / 257), // Preserve alpha
			}
			newImg.Set(x, y, newColor)
		}
	}

	return newImg
}

// This tells the compiler/linter that the function is being referenced
var _ = invertImageColors

func colorizeImage(img image.Image, rgba color.RGBA) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// Keep source shades by multiplying source and target channels.
			src := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			out := color.NRGBA{
				R: uint8((uint16(src.R) * uint16(rgba.R)) / 255),
				G: uint8((uint16(src.G) * uint16(rgba.G)) / 255),
				B: uint8((uint16(src.B) * uint16(rgba.B)) / 255),
				A: uint8((uint16(src.A) * uint16(rgba.A)) / 255),
			}
			newImg.Set(x, y, out)
		}
	}

	return newImg
}

func loadSignalsFromCSV() {
	data, err := os.ReadFile("in/signals.csv")
	if err != nil {
		panic(fmt.Errorf("failed to load csv: %w", err))
	}

	lines := strings.Split(string(data), "\n")
	for lineNumber, line := range lines {
		// skip header
		if lineNumber == 0 {
			continue
		}

		// skip separators
		if strings.HasPrefix(line, "--") ||
			strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, "//") ||
			strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, ",")
		for partInd := range parts {
			parts[partInd] = strings.TrimSpace(parts[partInd])
		}

		dataGroupID := groupID(parts[0])
		dataSignalID := parts[1]
		dataContent := parts[2]

		dataS := parts[3]
		dataX := parts[4]
		dataY := parts[5]

		dataFineTuneUseIconFont := parts[6]
		dataFineTuneUseLabelFont := parts[7]
		dataVanilla := parts[8]
		dataHidden := parts[9]
		dataOrder := parts[10]
		dataLocaleEn := parts[11]
		dataLocaleRu := parts[12]

		if strings.HasPrefix(dataContent, "\\u") {
			iconRune, err := strconv.ParseInt(dataContent[2:], 16, 32)
			if err != nil {
				panic(fmt.Errorf("failed to parse unicode '%s' icon at line %d: %w", dataContent, lineNumber, err))
			}

			dataContent = string(rune(iconRune))
		}

		if _, exist := groupSignals[dataGroupID]; !exist {
			groupSignals[dataGroupID] = make([]signal, 0, 64)
		}

		sig := signal{
			group:    dataGroupID,
			name:     dataSignalID,
			localeEn: dataLocaleEn,
			localeRu: dataLocaleRu,
			content:  dataContent,
			fineTune: make([]fineTune, 0, 8),
			vanilla:  dataVanilla == "Y",
			order:    dataOrder,
			hidden:   dataHidden == "Y",
		}

		if x, err := strconv.ParseFloat(dataX, 64); err == nil {
			sig.fineTune = append(sig.fineTune, fineTune{offsetX: x})
		}

		if y, err := strconv.ParseFloat(dataY, 64); err == nil {
			sig.fineTune = append(sig.fineTune, fineTune{offsetY: y})
		}

		if s, err := strconv.ParseFloat(dataS, 64); err == nil {
			sig.fineTune = append(sig.fineTune, fineTune{textScale: s})
		}

		if dataFineTuneUseIconFont == "Y" {
			sig.fineTune = append(sig.fineTune, fineTune{useIconFont: true})
		}
		if dataFineTuneUseLabelFont == "Y" {
			sig.fineTune = append(sig.fineTune, fineTune{useLabelFont: true})
		}

		groupSignals[dataGroupID] = append(groupSignals[dataGroupID], sig)
	}
}

// ApplyMipmaps creates a horizontal mipmap strip for any number of levels (e.g., 2 or 4)
func ApplyMipmaps(icon *gg.Context, levels int) *gg.Context {
	w, h := icon.Width(), icon.Height()
	iconImage := icon.Image()

	// 1. Dynamically calculate the total strip width based on requested levels
	// Base level (0) is full width. Each subsequent level is half of the previous.
	totalWidth := w
	currentW := w
	for i := 1; i < levels; i++ {
		currentW /= 2
		totalWidth += currentW
	}

	// 2. Initialize the single wide canvas atlas
	mip := gg.NewContext(totalWidth, h)

	// 3. Populate each mipmap layer directly into its allocated coordinate slot
	currentX := 0
	levelW, levelH := w, h

	for i := 0; i < levels; i++ {
		if i == 0 {
			// Level 0: Full size direct draw optimization
			mip.DrawImage(iconImage, 0, 0)
		} else {
			// Higher levels: Directly scale into the specific target sub-rectangle
			dstRect := image.Rect(currentX, 0, currentX+levelW, levelH)
			draw.BiLinear.Scale(mip.Image().(draw.Image), dstRect, iconImage, iconImage.Bounds(), draw.Over, nil)
		}

		// Shift X coordinate forward and half the dimensions for the next iteration step
		currentX += levelW
		levelW /= 2
		levelH /= 2
	}

	return mip
}

func groupIDs(groups map[groupID]group) []groupID {
	result := make([]groupID, 0, len(groups))

	for g := range groups {
		result = append(result, g)
	}

	sort.SliceStable(result, func(i, j int) bool {
		return groups[result[i]].order <= groups[result[j]].order
	})

	return result
}

func loadBackground(cfg Config, name string) image.Image {
	if f, ok := loadedImages[name]; ok {
		return f
	}

	img, err := gg.LoadPNG("in/" + fmt.Sprintf("%s.png", name))
	if err != nil {
		panic(fmt.Errorf("could not load png image: %v", err))
	}

	// optional scale
	if cfg.BackGroundScaleMax {

		// Dimensions for cropping
		cropSize := 56
		originalSize := 64
		offset := (originalSize - cropSize) / 2

		// Create a new gg.Context
		dc := gg.NewContext(cropSize, cropSize)

		// Draw the cropped area (central 56x56)
		dc.DrawImageAnchored(img, -offset, -offset, 0, 0)

		// Create a new context for resizing to 64x64
		outputDC := gg.NewContext(originalSize, originalSize)

		// Scale and draw the cropped image
		outputDC.Scale(float64(originalSize)/float64(cropSize), float64(originalSize)/float64(cropSize))
		outputDC.DrawImage(dc.Image(), 0, 0)
		// Get the final image
		scaledImage := outputDC.Image()
		loadedImages[name] = scaledImage
		return scaledImage

	} else {
		loadedImages[name] = img
		return img
	}
}

func TryLoadAndResizeImage(name string, size int) image.Image {
	img, err := gg.LoadPNG(fmt.Sprintf("in/%s.png", name))
	if err != nil {
		return nil
	}

	bounds := img.Bounds()

	// Scale only if dimensions do not match the target
	if bounds.Dx() != size || bounds.Dy() != size {
		// Initialize a new blank square canvas for the destination
		dst := image.NewRGBA(image.Rect(0, 0, size, size))

		// draw.CatmullRom provides beautiful, crisp anti-aliasing (replaces resize.Lanczos3).
		// Note: Use draw.NearestNeighbor instead if you are working with exact pixel art!
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, draw.Over, nil)
		return dst
	}

	return img
}

func loadFont(name string, lineHeight float64) font.Face {
	id := fmt.Sprintf("font_%s_%.2f", name, lineHeight)
	if f, ok := loadedFonts[id]; ok {
		return f
	}

	fontPath := fmt.Sprintf("in/%s", name)

	fontData, err := os.ReadFile(fontPath)
	if err != nil {
		panic(fmt.Errorf("could not load font: %w", err))
	}

	var loaded font.Face

	// Keep old rendering behavior for TTF fonts (label/math), but still support OTF icons.
	if parsedTTF, ttfErr := truetype.Parse(fontData); ttfErr == nil {
		loaded = truetype.NewFace(parsedTTF, &truetype.Options{
			Size: lineHeight,
		})
	} else {
		parsedFont, parseErr := opentype.Parse(fontData)
		if parseErr != nil {
			panic(fmt.Errorf("could not parse font %s: %w", name, parseErr))
		}

		loaded, err = opentype.NewFace(parsedFont, &opentype.FaceOptions{
			Size:    lineHeight,
			DPI:     72,
			Hinting: font.HintingFull,
		})
		if err != nil {
			panic(fmt.Errorf("could not create font face for %s: %w", name, err))
		}
	}

	loadedFonts[id] = loaded
	return loaded
}

func createLuaSubGroups(cfg Config, groups map[groupID]group) {
	var buf bytes.Buffer

	buf.WriteString("-- generated by Go script\n\n")

	// header
	buf.WriteString("data:extend({")

	// content
	for _, groupID := range groupIDs(groups) {

		group := groups[groupID]
		if !group.vanilla {

			buf.WriteString(fmt.Sprintf(`
			{
			type = "item-subgroup",
			name = "%s",
			group = "signals_group_vs2",
			order = settings.startup["%s"].value,
			},`,
				group.name,
				group.name,
			))
		}
	}

	// footer
	buf.WriteString("\n})")

	// write
	os.MkdirAll(cfg.dirProto(), 0777)
	err := os.WriteFile(cfg.dirProto()+"subgroups.lua", buf.Bytes(), 0666)
	if err != nil {
		panic(fmt.Errorf("could not create lua groups: %w", err))
	}
}

func createLuaSignals(cfg Config, groups map[groupID]group) {
	var buf bytes.Buffer

	buf.WriteString("-- generated by Go script\n\n")

	// header
	buf.WriteString("data:extend({")

	// content
	for _, groupID := range groupIDs(groups) {
		group := groups[groupID]
		for ind, sig := range groupSignals[groupID] {
			if !sig.vanilla {

				iconPath := fmt.Sprintf(`__%s__/graphics/signal/%s_%s.png`, cfg.ModName, group.name, sig.name)

				var order string
				if sig.order != "" {
					order = sig.order
				} else {
					order = fmt.Sprintf("%s_s[%03d]", group.order, ind)
				}

				buf.WriteString(fmt.Sprintf(`
					{
					type = "virtual-signal",
					name = "signal-vs2-%s",
					icon = "%s",
					subgroup = "%s",
					order = "%s",
					hidden = %t
					},`,
					sig.name,
					iconPath,
					group.name,
					order,
					sig.hidden,
				))
			}
		}
	}

	// footer
	buf.WriteString("\n})")

	// write
	os.MkdirAll(cfg.dirProto(), 0777)
	err := os.WriteFile(cfg.dirProto()+"signals.lua", buf.Bytes(), 0666)
	if err != nil {
		panic(fmt.Errorf("could not create lua signals: %w", err))
	}
}

func createLuaVanilla(cfg Config, groups map[groupID]group) {
	var buf bytes.Buffer

	buf.WriteString("-- generated by Go script\n\n")

	for _, groupID := range groupIDs(groups) {
		group := groups[groupID]
		if group.vanilla {
			buf.WriteString(fmt.Sprintf(
				`data.raw["item-subgroup"]["%s"].order = settings.startup["%s"].value`,
				group.name, group.name))

			buf.WriteString("\n")
		}
	}

	for _, groupID := range groupIDs(groups) {
		group := groups[groupID]

		for _, sig := range groupSignals[groupID] {

			if sig.vanilla {
				buf.WriteString(fmt.Sprintf(
					`data.raw["virtual-signal"]["%s"].icon = "__%s__/graphics/signal/%s_%s.png"`,
					sig.name, cfg.ModName, group.name, sig.name))

				buf.WriteString("\n")
			}
		}
	}

	// write
	os.MkdirAll(cfg.dirProto(), 0777)
	outputFile := filepath.Join(cfg.dirProto(), "vanilla.lua")
	err := os.WriteFile(outputFile, buf.Bytes(), 0644)
	if err != nil {
		panic(fmt.Errorf("could not create vanilla.lua: %w", err))
	}

}

func createLocales(cfg Config, groups map[groupID]group) {
	locales := []string{"en", "ru"}

	for _, localeID := range locales {
		var buf bytes.Buffer

		// header
		buf.WriteString("[virtual-signal-name]\n")

		// content
		for _, groupID := range groupIDs(groups) {
			for _, sig := range groupSignals[groupID] {
				if !sig.vanilla {
					value := sig.localeEn
					if localeID == "ru" {
						value = sig.localeRu
					}

					if value == "" {
						panic("unexpected locale")
					}

					if value == "-" {
						value = sig.localeEn
					}

					buf.WriteString(fmt.Sprintf(
						"signal-vs2-%s=%s\n",
						sig.name,
						value,
					))
				}
			}
		}

		buf.WriteString("\n\n[mod-setting-name]\n")

		for _, groupID := range groupIDs(groups) {
			group := groups[groupID]
			buf.WriteString(fmt.Sprintf(
				"%s=Order of [font=default-bold][color=blue]%s[/color][/font] group\n",
				group.name,
				group.name,
			))
		}

		buf.WriteString("\n\n[mod-setting-description]\n")

		for _, groupID := range groupIDs(groups) {
			group := groups[groupID]
			if group.vanilla {
				buf.WriteString(fmt.Sprintf(
					"%s=vanilla group\n",
					group.name,
				))
			}
		}

		// write
		localeDir := filepath.Join(cfg.dirMod(), "locale", localeID)
		os.MkdirAll(localeDir, 0755)
		outputFile := filepath.Join(localeDir, "auto_gen_signals.cfg")
		os.WriteFile(outputFile, buf.Bytes(), 0644)

	}
}

func createLuaSetting(cfg Config, groups map[groupID]group) {
	var buf bytes.Buffer

	buf.WriteString("-- generated by Go script\n\n")

	// header
	buf.WriteString("data:extend({")

	// content
	for _, groupID := range groupIDs(groups) {
		group := groups[groupID]

		buf.WriteString(fmt.Sprintf(`
				{
                    type = "string-setting",
                    name = "%s",
                    order = "%s",
					setting_type = "startup",
                    auto_trim = true,
					default_value = "%s",
                },`,
			group.name,
			group.order,
			group.order,
		))
	}

	// footer
	buf.WriteString("\n})")

	// write
	err := os.WriteFile(cfg.dirMod()+"settings.lua", buf.Bytes(), 0666)
	if err != nil {
		panic(fmt.Errorf("could not create settings.lua: %w", err))
	}
}

func createLuaData(cfg Config) {
	var buf bytes.Buffer

	buf.WriteString("-- generated by Go script\n\n")

	// header
	buf.WriteString(
		`require("prototypes.groups")
require("prototypes.subgroups")
require("prototypes.signals")
require("prototypes.vanilla")
`)

	outputFile := filepath.Join(cfg.dirMod(), "data.lua")

	err := os.WriteFile(outputFile, buf.Bytes(), 0644)
	if err != nil {
		panic(fmt.Errorf("could not create data.lua: %w", err))
	}
}

func copyDescription(cfg Config) {
	source := filepath.Join("in", DescriptionPath)
	target := filepath.Join(cfg.dirMod(), "description.md")

	input, _ := os.Open(source)
	output, _ := os.Create(target)

	defer input.Close()
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		panic(fmt.Errorf("could not copy description: %w", err))
	}
}

func copyThumbnail(cfg Config) {
	source := filepath.Join("in", cfg.ThumbnailPath)
	target := filepath.Join(cfg.dirMod(), "thumbnail.png")

	input, _ := os.Open(source)
	output, _ := os.Create(target)

	defer input.Close()
	defer output.Close()

	if _, err := io.Copy(output, input); err != nil {
		panic(fmt.Errorf("could not copy thumbnail: %w", err))
	}
}

func titleFromModName(modName string) string {
	parts := strings.Split(modName, "-")
	for i, part := range parts {
		if part == "" {
			continue
		}

		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}

	return strings.Join(parts, " ")
}

func createInfoJSON(cfg Config) {
	var buf bytes.Buffer

	buf.WriteString("{\n")
	buf.WriteString(fmt.Sprintf("    \"name\": \"%s\",\n", cfg.ModName))
	buf.WriteString(fmt.Sprintf("    \"version\": \"%s\",\n", ModVersion))
	buf.WriteString(fmt.Sprintf("    \"title\": %q,\n", titleFromModName(cfg.ModName)))
	buf.WriteString("    \"author\": \"flart,fe3dback\",\n")
	buf.WriteString("    \"contact\": \"\",\n")
	buf.WriteString("    \"homepage\": \"\",\n")
	buf.WriteString("    \"dependencies\": [\n")
	buf.WriteString("        \"base\",\n")
	buf.WriteString("        \"! virtual-signals2\"\n")
	buf.WriteString("    ],\n")
	buf.WriteString("    \"factorio_version\": \"2.1\",\n")
	buf.WriteString(fmt.Sprintf("    \"description\": %q\n", cfg.Description))
	buf.WriteString("}\n")

	outputFile := filepath.Join(cfg.dirMod(), "info.json")
	err := os.WriteFile(outputFile, buf.Bytes(), 0644)
	if err != nil {
		panic(fmt.Errorf("could not create info.json: %w", err))
	}
}

func createGroup_vs2(cfg Config) {
	group_icon := TryLoadAndResizeImage("item-group/signals_group_vs2_womm", 128)
	context := gg.NewContextForImage(group_icon)
	group_mm := ApplyMipmaps(context, 2)
	os.MkdirAll(cfg.dirMod()+"graphics/item-group", 0777)
	group_mm.SavePNG(cfg.dirMod() + "graphics/item-group/signals_group_vs2.png")

	var buf bytes.Buffer

	buf.WriteString("-- generated by Go script\n\n")

	// header
	buf.WriteString("data:extend({")

	// content
	buf.WriteString(fmt.Sprintf(`
	{
		type = "item-group",
		name = "signals_group_vs2",
		order = "g1",
		icon = "__%s__/graphics/item-group/signals_group_vs2.png",
		icon_size = 128,
	},`,
		cfg.ModName,
	))

	// footer
	buf.WriteString("\n})")

	// write
	os.MkdirAll(cfg.dirProto(), 0777)

	outputFile := filepath.Join(cfg.dirProto(), "groups.lua")
	err := os.WriteFile(outputFile, buf.Bytes(), 0644)
	if err != nil {
		panic(fmt.Errorf("could not create lua groups: %w", err))
	}

}
