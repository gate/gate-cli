package intelfacade

// freezeToolBaseline copies each entry in src into dst using deepCloneSchemaMap so the
// frozen table is isolated from later mutations of the static literals (CR-209).
func freezeToolBaseline(dst map[string]map[string]interface{}, src map[string]map[string]interface{}) {
	for k, v := range src {
		dst[k] = deepCloneSchemaMap(v)
	}
}
