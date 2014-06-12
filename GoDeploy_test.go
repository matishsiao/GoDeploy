package main

import (
	"testing"
)


func TestScript(t *testing.T) {
	 if testing.Short() {
        t.Skip("skipping test in short mode.")
    }
    
    //go sendScript(cmd,true)
}
