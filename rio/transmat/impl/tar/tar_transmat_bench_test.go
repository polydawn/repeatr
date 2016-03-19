package tar

import (
	"os"
	"strconv"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"polydawn.net/repeatr/lib/testutil"
	"polydawn.net/repeatr/rio/tests"
)

func BenchmarkTarScan(b *testing.B) {
	Convey("Bench", b, testutil.WithTmpdir(func() {
		for i := 0; i < b.N; i++ {
			Convey(strconv.Itoa(i), func() {
				tests.CheckScanWithoutMutation(Kind, New)
			})
		}
	}))
}

func BenchmarkTarBounce(b *testing.B) {
	Convey("Bench", b, testutil.WithTmpdir(func() {
		os.Mkdir("bounce", 0755) // make dir for the warehouse
		for i := 0; i < b.N; i++ {
			Convey(strconv.Itoa(i), func() {
				tests.CheckRoundTrip(Kind, New, "file+ca://bounce", "content-addressible")
			})
		}
	}))
}
