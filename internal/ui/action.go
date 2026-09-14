// G5.3: user actions requested by the menu (tabs) and consumed by the game
// loop (G6.1). Kept in their own file so the queue contract stays easy to
// find; Menu.Actions() drains them each frame.
package ui

// ActionKind enumerates the actions the game consumes (G6.1).
type ActionKind uint8

const (
	ActionResearch      ActionKind = iota + 1 // run the species agent now
	ActionRunAgent                            // ID = "species" | "water" | "pattern"
	ActionRunAll                              // run all three agents
	ActionExportPack                          // Path "" -> game default pack.zip
	ActionImportPack                          // Path "" -> game default pack.zip
	ActionResetTank                           // wipe world + saves (game confirms)
	ActionToggleSound                         // Flag = new soundOn value
	ActionImportKey                           // import GLM key from ZCode dir
	ActionSelectSpecies                       // ID = species id (catalog click)
	ActionApplyPlant                          // ID = plant id (plant click)
)

// Action is one queued user request for the game loop to execute.
type Action struct {
	Kind ActionKind
	ID   string // agent id / species id / plant id
	Path string // pack path ("" = default beside the exe)
	Flag bool   // toggle target value
}
