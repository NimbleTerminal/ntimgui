// Copyright (c) 2026 Neomantra Corp.

package ntimgui

import "image"

// Point represents a 2D coordinate in logical terminal cell units (Column X, Row Y).
type Point struct {
	X, Y int
}

// Pt is shorthand for Point{X: x, Y: y}.
func Pt(x, y int) Point {
	return Point{X: x, Y: y}
}

// Add returns the vector sum p + q.
func (p Point) Add(q Point) Point {
	return Point{X: p.X + q.X, Y: p.Y + q.Y}
}

// Sub returns the vector difference p - q.
func (p Point) Sub(q Point) Point {
	return Point{X: p.X - q.X, Y: p.Y - q.Y}
}

// ImagePoint converts Point to an image.Point.
func (p Point) ImagePoint() image.Point {
	return image.Point{X: p.X, Y: p.Y}
}

// Rect represents an axis-aligned rectangle in logical terminal cell units.
type Rect struct {
	X, Y, W, H int
}

// NewRect constructs a Rect with top-left (x, y) and dimensions (w, h).
func NewRect(x, y, w, h int) Rect {
	return Rect{X: x, Y: y, W: w, H: h}
}

// Empty reports whether the rectangle has zero or negative area.
func (r Rect) Empty() bool {
	return r.W <= 0 || r.H <= 0
}

// Right returns the exclusive right edge X + W.
func (r Rect) Right() int {
	return r.X + r.W
}

// Bottom returns the exclusive bottom edge Y + H.
func (r Rect) Bottom() int {
	return r.Y + r.H
}

// Min returns the top-left corner of r.
func (r Rect) Min() Point {
	return Point{X: r.X, Y: r.Y}
}

// Max returns the exclusive bottom-right corner of r.
func (r Rect) Max() Point {
	return Point{X: r.X + r.W, Y: r.Y + r.H}
}

// Center returns the center point of r.
func (r Rect) Center() Point {
	return Point{X: r.X + r.W/2, Y: r.Y + r.H/2}
}

// Contains reports whether cell coordinate (x, y) is inside r.
func (r Rect) Contains(x, y int) bool {
	return !r.Empty() && x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// ContainsPoint reports whether p is inside r.
func (r Rect) ContainsPoint(p Point) bool {
	return r.Contains(p.X, p.Y)
}

// Inset returns r shrunk by (dx, dy) on each side.
func (r Rect) Inset(dx, dy int) Rect {
	w := max(0, r.W-2*dx)
	h := max(0, r.H-2*dy)
	return Rect{X: r.X + dx, Y: r.Y + dy, W: w, H: h}
}

// Intersect returns the overlapping region of r and other.
func (r Rect) Intersect(other Rect) Rect {
	if r.Empty() || other.Empty() {
		return Rect{}
	}
	x0 := max(r.X, other.X)
	y0 := max(r.Y, other.Y)
	x1 := min(r.Right(), other.Right())
	y1 := min(r.Bottom(), other.Bottom())
	if x1 <= x0 || y1 <= y0 {
		return Rect{}
	}
	return Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

// ImageRect converts r into a standard library image.Rectangle.
func (r Rect) ImageRect() image.Rectangle {
	if r.Empty() {
		return image.Rectangle{}
	}
	return image.Rect(r.X, r.Y, r.X+r.W, r.Y+r.H)
}

// Scaled returns an image.Rectangle with coordinates multiplied by (scaleX, scaleY).
func (r Rect) Scaled(scaleX, scaleY int) image.Rectangle {
	if r.Empty() || scaleX <= 0 || scaleY <= 0 {
		return image.Rectangle{}
	}
	return image.Rect(
		r.X*scaleX,
		r.Y*scaleY,
		(r.X+r.W)*scaleX,
		(r.Y+r.H)*scaleY,
	)
}

// Align specifies horizontal text alignment inside a bounding rectangle.
type Align uint8

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)
