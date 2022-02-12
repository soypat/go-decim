package decim

import "math"

type vec struct {
	x, y, z float64
}

// add returns element wise addition p + q
func add(p, q vec) vec {
	return vec{x: p.x + q.x, y: p.y + q.y, z: p.z + q.z}
}

// sub returns element wise subtraction p - q
func sub(p, q vec) vec {
	return vec{x: p.x - q.x, y: p.y - q.y, z: p.z - q.z}
}

func dot(p, q vec) float64 {
	return p.x*q.x + p.y*q.y + p.z*q.z
}

// Norm returns the Euclidean norm of p
//  |p| = sqrt(p_x^2 + p_y^2 + p_z^2).
func norm(p vec) float64 {
	return math.Hypot(p.x, math.Hypot(p.y, p.z))
}

// vcos returns the cosine of the opening angle between p and q.
func vcos(p, q vec) float64 {
	return dot(p, q) / (norm(p) * norm(q))
}
