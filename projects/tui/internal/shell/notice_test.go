// A notice of what was done fades after its while; one of why not stays; a fade for an earlier notice leaves a later.
package shell

import "testing"

func TestANoticeOfWhatWasDoneFadesAndOneOfWhyNotStays(t *testing.T) {
	var none Notice
	n := none.Did("saved")
	if n.Text != "saved" || !n.Done || n.Fading(none, 0) == nil {
		t.Fatalf("did: %+v", n)
	}
	if n.Fading(n, 0) != nil {
		t.Error("a notice put up before fades again")
	}
	if msg := n.Fading(none, 0)(); msg != n.FadeMsg() {
		t.Errorf("the fade: %+v", msg)
	}
	if n.Faded(n.FadeMsg()).Text != "" {
		t.Error("the notice outlived its fade")
	}
	// a fade for an earlier notice leaves a later one
	earlier := n.FadeMsg()
	if later := n.Did("saved again"); later.Faded(earlier).Text != "saved again" {
		t.Error("an earlier fade took a later notice")
	}
	// a notice of why not, put in the place of one of what was done, stays through that one's fade
	why := n
	why.Text, why.Done = "could not save", false
	if why.Faded(n.FadeMsg()).Text != "could not save" {
		t.Error("a notice of why not faded")
	}
}
