package dao

func getEsHighLights(field map[string][]string) map[string][]string {
	highlights := make(map[string][]string)
	if field != nil {
		highlights = field
	}
	return highlights
}
