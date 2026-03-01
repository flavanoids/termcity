package tilemap

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
)

// upperHalfBlock is U+2580 "▀" — upper half block character.
const upperHalfBlock = "▀"

// RenderTile decodes a PNG tile and returns terminal rows as ANSI strings.
// Each terminal row encodes 2 pixel rows using the half-block technique.
// Returns 128 strings (for a 256px tall tile).
func RenderTile(pngData []byte) ([]string, error) {
	img, _, err := image.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, fmt.Errorf("decoding tile PNG: %w", err)
	}

	bounds := img.Bounds()
	w := bounds.Max.X - bounds.Min.X
	h := bounds.Max.Y - bounds.Min.Y

	// Ensure even height.
	if h%2 != 0 {
		h--
	}

	rows := make([]string, h/2)
	for row := 0; row < h/2; row++ {
		line := make([]byte, 0, w*30)
		for col := 0; col < w; col++ {
			topPx := img.At(bounds.Min.X+col, bounds.Min.Y+row*2)
			botPx := img.At(bounds.Min.X+col, bounds.Min.Y+row*2+1)
			tr, tg, tb, _ := topPx.RGBA()
			br, bg, bb, _ := botPx.RGBA()
			// RGBA returns 16-bit values; shift to 8-bit.
			line = appendHalfBlock(line, uint8(tr>>8), uint8(tg>>8), uint8(tb>>8),
				uint8(br>>8), uint8(bg>>8), uint8(bb>>8))
		}
		// Reset attributes at end of row.
		line = append(line, "\x1b[0m"...)
		rows[row] = string(line)
	}
	return rows, nil
}

// appendHalfBlock appends an ANSI-colored half-block character to buf.
// fg = top pixel (foreground), bg = bottom pixel (background).
func appendHalfBlock(buf []byte, fr, fg, fb, br, bg, bb uint8) []byte {
	// Set foreground (top pixel) and background (bottom pixel) in 24-bit color.
	buf = append(buf, fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm%s",
		fr, fg, fb, br, bg, bb, upperHalfBlock)...)
	return buf
}

// RGBToANSI256 converts RGB to the nearest xterm-256 color index (fallback).
func RGBToANSI256(r, g, b uint8) int {
	// Grayscale ramp: indices 232-255 (24 shades).
	if r == g && g == b {
		if r < 8 {
			return 16
		}
		if r > 248 {
			return 231
		}
		return int((float64(r)-8)/247*24) + 232
	}
	// 6x6x6 color cube: indices 16-231.
	ri := int(float64(r) / 255 * 5)
	gi := int(float64(g) / 255 * 5)
	bi := int(float64(b) / 255 * 5)
	return 16 + 36*ri + 6*gi + bi
}

// BlendColor blends two colors (alpha 0=fg, 1=bg).
func BlendColor(fg, bg color.RGBA, alpha float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(fg.R)*(1-alpha) + float64(bg.R)*alpha),
		G: uint8(float64(fg.G)*(1-alpha) + float64(bg.G)*alpha),
		B: uint8(float64(fg.B)*(1-alpha) + float64(bg.B)*alpha),
		A: 255,
	}
}

// ParseHexColor parses a "#RRGGBB" hex color string.
func ParseHexColor(hex string) color.RGBA {
	if len(hex) == 0 {
		return color.RGBA{128, 128, 128, 255}
	}
	if hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return color.RGBA{128, 128, 128, 255}
	}
	var r, g, b uint8
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return color.RGBA{r, g, b, 255}
}

// ColoredCell returns an ANSI string for a single colored cell (for incident markers).
func ColoredCell(ch rune, fgHex string, bgR, bgG, bgB uint8) string {
	c := ParseHexColor(fgHex)
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm%c\x1b[0m",
		c.R, c.G, c.B, bgR, bgG, bgB, ch)
}
