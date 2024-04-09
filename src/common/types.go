package common

type Pair[T1 any, T2 any] struct {
	First  T1
	Second T2
}

type Entry struct {
	Name      string
	Address   string
	Phone     string
	Longitude float64
	Latitude  float64
}
