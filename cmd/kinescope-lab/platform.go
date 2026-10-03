package main

// platform is what the lab needs of where it runs: a place to keep its
// state and a way to hand text to the user.
type platform interface {
	// State is the state the lab started with.
	State() string
	// SetState keeps the lab's state: in a browser, in the page's address,
	// so the link shares the setup.
	SetState(state string)
	// Export hands text to the user: the clipboard in a browser, standard
	// output on a desktop.
	Export(label string, text string)
	// Link is how a state is shared: a link in a browser, a command line on
	// a desktop.
	Link(state string) string
}
