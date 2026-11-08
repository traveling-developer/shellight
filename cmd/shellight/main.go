// Command shellight is the Shellight menu bar app for NVIDIA OpenShell.
package main

import "fmt"

// buildID is set at link time to tell builds apart.
var buildID = "dev"

func main() {
	fmt.Println("shellight", buildID)
}
