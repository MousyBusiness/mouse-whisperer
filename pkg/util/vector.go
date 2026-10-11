package util

import (
	"image"
	"math"
)

type Vector struct {
	X float64
	Y float64
}

func (v Vector) Point() image.Point {
	return image.Point{
		X: int(v.X),
		Y: int(v.Y),
	}
}

func (v Vector) Norm() float64 { return math.Sqrt(v.Dot(v)) }

// Norm2 returns the square of the norm.
func (v Vector) Norm2() float64 { return v.Dot(v) }

// Normalize returns a unit vector in the same direction as v.
func (v Vector) Normalize() Vector {
	n2 := v.Norm2()
	if n2 == 0 {
		return Vector{0, 0}
	}
	return v.Mul(1 / math.Sqrt(n2))
}

// Mul returns the standard scalar product of v and m.
func (v Vector) Mul(m float64) Vector { return Vector{m * v.X, m * v.Y} }

func (v Vector) Sub(ov Vector) Vector { return Vector{v.X - ov.X, v.Y - ov.Y} }

func (v Vector) Add(ov Vector) Vector { return Vector{v.X + ov.X, v.Y + ov.Y} }

// Dot returns the standard dot product of v and ov.
func (v Vector) Dot(ov Vector) float64 { return v.X*ov.X + v.Y*ov.Y }

func (v Vector) Distance(ov Vector) float64 { return v.Sub(ov).Norm() }
