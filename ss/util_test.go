package ss

import "testing"

func Test_GetPublicIP(t *testing.T) {
	str := GetPublicIP()
	t.Log("Public ip:", str)
}
