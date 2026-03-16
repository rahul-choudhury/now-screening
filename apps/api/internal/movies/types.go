package movies

type Movie struct {
	Title string `json:"title"`
	Href  string `json:"href"`
}

type Response struct {
	City   string  `json:"city"`
	Movies []Movie `json:"movies"`
	Count  int     `json:"count"`
}
