package util

type serviceConstants struct {
	Redis string
	Nats  string
}

var Services = serviceConstants{
	Redis: "redis",
	Nats:  "nats",
}

type rules struct {
	Denylist             string
	Velocity             string
	HorizontalBruteForce string
	ImpossibleTravel     string
}

var Rules = rules{
	Denylist:             "denylist",
	Velocity:             "velocity",
	HorizontalBruteForce: "horizontalBruteForce",
	ImpossibleTravel:     "impossibleTravel",
}

type strategies struct {
	Override string
	Average  string
}

var Strategies = strategies{
	Override: "override",
	Average:  "average",
}
