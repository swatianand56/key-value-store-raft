package client

import "fmt"

func main() {
	var oldValue string
	fmt.Printf("calling put function ---- %d \n", kv739_put("a", "523", &oldValue))
	fmt.Printf("old value for xkey a is === %s ", oldValue)

	fmt.Printf("getting key value a from the server --- %d ", kv739_get("a", oldValue))
	fmt.Printf("value for key a from server is === %s ", oldValue)
}
