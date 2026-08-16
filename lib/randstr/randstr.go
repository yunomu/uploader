package randstr

import (
	"math/rand"
	"strings"
)

type Generator interface {
	Generate() (string, error)
}

type MathRand struct {
	rand   *rand.Rand
	length int

	runes []rune
	n     int
}

var _ Generator = (*MathRand)(nil)

func NewMathRand(
	rand *rand.Rand,
	length int,
) *MathRand {
	charset := `ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_`

	return &MathRand{
		rand:   rand,
		length: length,

		runes: []rune(charset),
		n:     len(charset),
	}
}

func (g *MathRand) Generate() (string, error) {
	var buf strings.Builder
	for i := 0; i < g.length; i++ {
		if _, err := buf.WriteRune(g.runes[g.rand.Intn(g.n)]); err != nil {
			return "", err
		}
	}

	return buf.String(), nil
}
