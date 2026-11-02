package ui

import (
	. "go.hasen.dev/shirei"
	. "go.hasen.dev/shirei/widgets"
)

// Panel draws a raised charcoal surface.
func Panel(attrs AttrSet, body func()) ContainerId {
	return Container(AttrsWith(attrs,
		UseSurface(SurfacePanel), Corners(10), BorderWidth(1),
		BorderColorVec(Alpha(ColBorder, .7)),
	), body)
}

// Rail draws the darkest chrome used by side rails and the keyboard bed.
func Rail(attrs AttrSet, body func()) ContainerId {
	return Container(AttrsWith(attrs,
		BackgroundVec(ColRail), Corners(10),
	), body)
}

// Divider is a one-pixel separator that fills its cross axis.
func Divider() {
	Element(Attrs(Grow(0), ExpandAcross(), FixWidth(1), BackgroundVec(Alpha(ColBorder, .6))))
}

// HDivider is a horizontal separator.
func HDivider() {
	Element(Attrs(ExpandAcross(), FixHeight(1), BackgroundVec(Alpha(ColBorder, .6))))
}

// ExpandAcross expands on the cross axis without growing along the main axis.
func ExpandAcross() AttrsFn {
	return func(a *AttrSet) { a.ExpandAcross = true }
}

// SectionTitle is a small uppercase label used above groups of controls.
func SectionTitle(text string) {
	Label(text, FontSize(10), FontWeight(WeightBold), TextColorVec(ColTextDim))
}

// Caption is dim helper text.
func Caption(text string) {
	Label(text, FontSize(11), TextColorVec(ColTextMute))
}

// BodyText is normal secondary text.
func BodyText(text string) {
	Label(text, FontSize(12), TextColorVec(ColTextDim))
}

// TitleText is a panel heading.
func TitleText(text string) {
	Label(text, FontSize(16), FontWeight(WeightBold), TextColorVec(ColText))
}

// IconButton is an icon-only button with an accessible label.
func IconButton(icon IconGlyph, tip string, enabled bool) bool {
	NextAccessLabel(tip)
	NextButtonType(ButtonDefault)
	return CtrlButton(icon, "", enabled)
}

// PrimaryIconButton is an accented icon-only button.
func PrimaryIconButton(icon IconGlyph, tip string, enabled bool) bool {
	NextAccessLabel(tip)
	NextButtonType(ButtonPrimary)
	return CtrlButton(icon, "", enabled)
}

// Pill is a compact rounded value readout.
func Pill(attrs AttrSet, body func()) ContainerId {
	return Container(AttrsWith(attrs,
		Row, CrossMid, Pad2(4, 9), Corners(7),
		BackgroundVec(ColPanel2), BorderWidth(1),
		BorderColorVec(Alpha(ColBorder, .8)),
	), body)
}

// ValuePill shows a label/value pair.
func ValuePill(caption, value string, accent bool) {
	Pill(Attrs(Gap(6)), func() {
		Label(caption, FontSize(10), TextColorVec(ColTextMute))
		c := ColText
		if accent {
			c = ColAmber
		}
		Label(value, FontSize(12), FontWeight(WeightBold), TextColorVec(c))
	})
}

// CompactSegmented is a segmented control sized for the narrow setup panel.
func CompactSegmented[T comparable](target *T, body func()) bool {
	return SegmentedControlExt(target, SegmentedControlAttrs{
		CellPadH: 7, FrameCorners: 6, MinCellWidth: 28,
	}, body)
}

// FlexSpacer pushes siblings apart along the main axis.
func FlexSpacer() {
	Element(Attrs(Grow(1), MinSize(0, 0)))
}

// Badge is a tiny coloured status dot.
func Badge(c Vec4, size float32) {
	Element(Attrs(FixSize(size, size), Corners(size/2), BackgroundVec(c)))
}

// TrackColor parses a stored hex colour, falling back to amber.
func TrackColor(hex string) Vec4 {
	if hex == "" {
		return ColAmber
	}
	return Col(hex)
}

// PanelHeader is a row with a title and an optional trailing element.
func PanelHeader(title string, trailing func()) {
	Container(Attrs(Row, CrossMid, ExpandAcross(), Gap(8)), func() {
		Label(title, FontSize(13), FontWeight(WeightBold), TextColorVec(ColText))
		FlexSpacer()
		if trailing != nil {
			trailing()
		}
	})
}
