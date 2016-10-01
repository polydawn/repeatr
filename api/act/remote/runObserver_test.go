package remote

import (
	"io"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/ugorji/go/codec"
	"go.polydawn.net/meep"

	"go.polydawn.net/repeatr/api/act"
	"go.polydawn.net/repeatr/api/def"
)

func Test(t *testing.T) {
	Convey("Given an Event slice", t, func() {
		evtsFixture := []*def.Event{
			{RunID: "aa", Seq: 1, Log: &def.LogItem{Msg: "a log"}},
			{RunID: "aa", Seq: 2, Journal: "user's stdout\n"},
			{RunID: "aa", Seq: 3, Log: &def.LogItem{Msg: "process done"}},
			{RunID: "aa", Seq: 4, RunRecord: &def.RunRecord{}},
		}
		r, w := io.Pipe()
		encoder := codec.NewEncoder(w, &codec.JsonHandle{})
		client := &RunObserverClient{
			Remote: r,
			Codec:  &codec.JsonHandle{},
		}

		Convey("When some events are readable", func() {
			go func() {
				encoder.Encode(evtsFixture[0])
				w.Write([]byte{'\n'})
				encoder.Encode(evtsFixture[1])
				w.Write([]byte{'\n'})
				w.Close() // to make FollowEvents halt on EOF
			}()

			Convey("individual steps should work", func() {
				So(client.readOne().Seq, ShouldEqual, 1)
				So(client.readOne().Seq, ShouldEqual, 2)
				So(client.readOne().Seq, ShouldEqual, 0)
			})

			Convey("FollowEvents should yield those events", func() {
				ch := make(chan *def.Event, 2)
				client.FollowEvents("aa", ch, 0)
				So((<-ch).Seq, ShouldEqual, 1)
				So((<-ch).Seq, ShouldEqual, 2)
			})
		})

		Convey("When some events are readable, then followed with trash", func() {
			go func() {
				encoder.Encode(evtsFixture[0])
				w.Write([]byte{'\n'})
				w.Write([]byte("panic: of some kind!"))
				w.Close()
			}()

			Convey("FollowEvents should yield some events, then panic", func() {
				ch := make(chan *def.Event, 2)
				err := meep.RecoverPanics(func() {
					client.FollowEvents("aa", ch, 0)
				})
				So((<-ch).Seq, ShouldEqual, 1)
				So(err, ShouldNotBeNil)
				So(err, ShouldHaveSameTypeAs, &act.ErrRemotePanic{})
			})
		})

	})
}
