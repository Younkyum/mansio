package ws

type ControlMessage struct {
	Type    string `json:"type"`
	Cols    uint16 `json:"cols,omitempty"`
	Rows    uint16 `json:"rows,omitempty"`
	Shell   string `json:"shell,omitempty"`
	Message string `json:"message,omitempty"`
	Title   string `json:"title,omitempty"`
	// Recreated is set on `attached` frames when the backend had to spawn a
	// fresh tmux session because the prior one was gone. The frontend uses
	// this to surface a "session was lost, started fresh" banner instead of
	// silently dropping the user into an empty shell.
	Recreated bool `json:"recreated,omitempty"`
}
