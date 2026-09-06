package cloud

import "testing"

// Cloud cannot infer what kind of process is connecting — a swarm replica and a
// standalone server both scope themselves to their own daemon and look
// identical on the wire — so Dozzle reports it. Every build before this one
// reported nothing, and those connections must keep working.
func TestSetDeployment(t *testing.T) {
	c := &Client{}
	if c.mode != "" || c.swarmClusterID != "" {
		t.Fatal("a client that never calls SetDeployment must report nothing")
	}

	c.SetDeployment("swarm", "cluster-abc")
	if c.mode != "swarm" || c.swarmClusterID != "cluster-abc" {
		t.Fatalf("got mode=%q cluster=%q", c.mode, c.swarmClusterID)
	}

	// Outside a swarm there is no cluster to report, but the mode still stands.
	c.SetDeployment("server", "")
	if c.mode != "server" || c.swarmClusterID != "" {
		t.Fatalf("got mode=%q cluster=%q", c.mode, c.swarmClusterID)
	}
}
