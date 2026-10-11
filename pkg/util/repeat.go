package util

func Repeat(f func(), n int) {
	for i := 0; i < n; i++ {
		f()
	}
}
