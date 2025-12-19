package generator

import "testing"

// Бенчмарк дефолтового генератора коротких URL.
func BenchmarkDefaultGenerator(b *testing.B) {
	length := 8

	b.Run("default generator benchmark", func(b *testing.B) {
		benchGen := NewDefaultGenerator()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := benchGen.MakeShortURL(length)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
