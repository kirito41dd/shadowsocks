package ss

import "testing"

func Test_GetPublicIP(t *testing.T) {
	str := GetPublicIP()
	t.Logf("Public ip:(%s)", str)
	t.Log("ipstr len = ", len(str))
}
