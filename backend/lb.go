package backend

type LB interface {
	//Get(string, []Proxy) Proxy
}

type RandomLB struct {
}
