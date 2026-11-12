// The workspace menu's Open Repository…, whose dialog is internal/opener's: the message that asks kittui to open it,
// and its glyph.
package shell

// OpenMsg asks kittui to open the Open Repository… dialog over the screen shown.
type OpenMsg struct{}

// Opening is Open Repository…'s glyph, and the dialog's line that opens what is typed, two cells wide, as every glyph
// in a menu is.
const Opening = "📂"
