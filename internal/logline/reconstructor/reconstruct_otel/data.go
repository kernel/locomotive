package reconstruct_otel

var severityTextToNumber = map[string]int{
	"trace": 1,
	"debug": 5,
	"info":  9,
	"warn":  13,
	"error": 17,
	"fatal": 21,
}
