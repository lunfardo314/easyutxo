package funengine

type libraryFun map[string]*funDef

var theLibrary = libraryFun{
	"concat": {sym: "concat", returnType: "S", varParams: true, paramTypes: []string{"S"}},
	"slice":  {sym: "slice", returnType: "S", varParams: false, paramTypes: []string{"S", "B", "B"}},
}
