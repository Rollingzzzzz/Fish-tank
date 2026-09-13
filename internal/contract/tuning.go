// G0.2: frozen tuning constants — README §4. Use these; no magic numbers elsewhere.
package contract

// SpineSegments is the number of spine points per fish body.
const SpineSegments = 14

// Life stages in tank-days and their speed multipliers.
var (
	StageDays = map[string]float64{"fry": 0, "juvenile": 2, "adult": 5, "elder": 12}
	// StageSpeedMul scales BaseSpeed by life stage.
	StageSpeedMul = map[string]float64{"fry": 0.8, "juvenile": 0.95, "adult": 1.0, "elder": 0.6}
	// StageOrder lists stages oldest→youngest for iteration.
	StageOrder = []string{"fry", "juvenile", "adult", "elder"}
)

// Motion tuning (pixels, seconds).
const (
	BaseSpeed    = 72.0  // px/s × Behavior.Speed × stage/night multipliers
	MaxForce     = 240.0 // px/s²
	SchoolRadius = 110.0
	FoodSense    = 150.0
	EggHatchSec  = 6.0
)

// Care economy.
const (
	CareWatchTick = 10.0 // seconds of mouse activity → +1 care
	CareFeedScore = 2.0  // per flake eaten
)

// CareThresholds trigger a courtship breeding event when crossed.
var CareThresholds = []float64{10, 25, 50, 80}

// Agent schedules (seconds, scaled by Config.AgentFreq) and first-run delays.
var (
	AgentSchedules = map[string]float64{"species": 140, "water": 110, "pattern": 80}
	AgentFirstRun  = map[string]float64{"species": 3, "water": 6, "pattern": 9}
)

// Persistence tiers (G2.5) and lifecycle.
const (
	SaveMinuteSec       = 60.0 // rolling snapshot cadence
	MinFishCount        = 2    // legacy v0.1 floor (death spare)
	MinPopulation       = 4    // FD11: the tank never drops below this
	DeathAfterElderDays = 4.0  // elder + this many days → fade out over 60 s
	SaveSchemaVersion   = 2    // bump on any Save shape change
	DeathFadeSec        = 60.0 // fade duration of a dying elder
	CourtshipSec        = 4.0  // circling duration before eggs spawn
)

// v0.2 ecosystem + behavior tuning (README §10).
const (
	PlantCoverageMax     = 0.35  // fraction of tank width plants+corals may occupy
	MiteCap              = 6     // live water mites at once (N7)
	MiteSpawnMeanSec     = 25.0  // mean seconds between wild mite spawns
	CareDecayPerSec      = 0.02  // care decay above the last tier (recurring breeding)
	CourtshipCooldownSec = 45.0  // min seconds between courtships (care-gated)
	StartleMeanSec       = 50.0  // mean seconds between random startle events
	ZoneRadius           = 140.0 // chosen-aura radius (N3)
	MusicLoopSec         = 64.0  // N6 loop length
	RockCaves            = 2     // N2 cave count
	TreatMax             = 6     // live treats in the tank at once
	FoodSenseFloor       = 190.0 // F3 widened food perception
	ElderLifespanBonus   = 3.0   // max extra elder days at top care (F11)
)

// v0.3 "Wide Glass" tuning (README §11).
const (
	// F13: height-locked logical canvas — width follows the monitor aspect.
	LogicalH    = 720
	LogicalWMin = 1024
	LogicalWMax = 1920

	// F14: live fish always outnumber plants (cap = min(PlantHardCap, alive-1)).
	PlantHardCap = 12

	// F15: cave lounging — idle fish occasionally enjoy the volcanic holes.
	// v0.3.8: the caves are a sanctuary, not a dormitory — rarer picks,
	// shorter stays, one fish per cave and a small share of the school.
	LoungeMeanSec     = 120.0 // mean seconds between lounge picks per fish (was 45)
	LoungeDwellMin    = 5.0   // shortest stay inside a cave (was 8)
	LoungeDwellMax    = 10.0  // longest stay inside a cave (was 18)
	LoungeSchoolShare = 0.20  // max share of the school lounging at once

	// v0.3.8 roaming — idle cruisers sweep the whole tank on soft waypoints
	// instead of milling around the rock towers.
	RoamMeanSec = 7.0 // mean seconds between tour waypoints per fish

	// N10: right-click scare reach and recovery.
	ScareRadius = 190.0
	// v0.3.8: a startled fish keeps seeking cover this long — the impulse
	// alone decayed before it ever reached the cave.
	ScareShelterSec = 8.0

	// N11: the hook-treat pull while a treat is held at the cursor.
	HeldTreatRadius = 260.0 // hungry fish notice the wriggling treat inside this
	HeldTreatRing   = 34.0  // keep-back ring — mouths stay just out of reach
	HeldTreatSpeed  = 1.4   // excitement speed multiplier toward the cursor

	// N9/F23: role-aware supremacy clamps — no normal fish matches the Chosen.
	NormalSizeMax  = 1.15
	NormalFinMax   = 1.35
	NormalTailMax  = 1.30
	NormalSpeedMax = 1.35 // F23: even a night-active normal stays slower than her
	ChosenSpeedMul = 1.45 // F23: her live multipler over the stat (was hardcoded 1.3)

	// N12: music — 75 BPM 4/4, 20 bars, exact 64 s loop (feels pure-imagination).
	MusicBPM = 90.0

	// v0.3.1: the screen IS the tank — object density scales with canvas area
	// relative to the 1280x720 reference aquarium.
	DensityRefW = 1280.0
	DensityRefH = 720.0
	DensityMax  = 4.0 // population multiplier ceiling
	PopCapMax   = 100 // v0.3.7 (F29): the player may run up to 100 live fish
	PlantCapMax = 30  // absolute plant ceiling (still fish-majority gated)
	MiteCapMax  = 16

	// v0.3.2: floor composition — how the tank bottom is shared. The shares
	// are along the floor line; approximate base widths per object feed the
	// budgets (plants ~26 px, corals ~42 px at the 720p reference).
	// v0.3.6 (F25): the volcanic crags grew — rocks take 50%, plants/corals
	// yield 5% each so the open water lane and the center stage stay intact.
	FloorSharePlants = 0.15
	FloorShareRocks  = 0.50 // caves + crags + nest pedestal, enterable
	FloorShareCorals = 0.15 // corals and other decorative things
	PlantBasePx      = 26.0
	CoralBasePx      = 42.0
)

// v0.3.6 (F25): door-to-door transit through the volcanic crags.
// v0.3.8: the hide flip is binary — no ramp constant anymore.
const (
	TransitMinSec    = 1.2   // shortest hidden travel between two mouths
	TransitMaxSec    = 2.5   // longest hidden travel
	TransitExitBoost = 1.6   // exit speed burst (× maxSpeed, decays fast)
	TransitNearPx    = 90.0  // "near a mouth" trigger distance
	TransitChanceSec = 0.10  // per-second chance to transit while near
	HeldScentGrowPx  = 20.0  // F26: held-treat scent radius growth per second
	HeldScentGrowCap = 140.0 // F26: max extra scent radius
)

// v1.1: titan pod (G39–G43) and floor critters (G44–G47). Everything the two
// ambient features need lives here — no config keys, no menu entries.
const (
	// Visit cycle: the pod is absent for the gap, present for the dwell.
	TitanVisitGapMin    = 180.0 // s between visits (absent stretch)
	TitanVisitGapMax    = 360.0
	TitanVisitMin       = 120.0 // shortest roaming dwell
	TitanVisitMax       = 240.0
	TitanPodMin         = 5 // scalare travel as a five (1 leader + 4 escorts)
	TitanPodMax         = 5
	TitanSizeMin        = 6.0 // species size range (core seed only; 64 px × Size)
	TitanSizeMax        = 7.0
	TitanFlatness       = 0.62 // superseded by TitanTallPeak (G58 angelfish read)
	TitanTallPeak       = 0.42 // v1.1 G58: the scalare body is a TALL flat diamond
	TitanFinScale       = 1.6  // dorsal/anal fins tower over the diamond
	TitanSpineBendTurn  = 0.18 // bend allowance while curling a 180° turn
	TitanTurnWindow     = 8.0  // seconds the convoy U takes — a true wide arc
	TitanTurnRadiusFrac = 0.25 // × body length: the arc's radius (G67)
	// TitanEdgeTurn was superseded by the body-scaled curl trigger of G66:
	// the turn now starts at half the body length + 24 px from the glass
	TitanSpeedMin   = 0.25 // behavior speed clamp — the ponderous cruise
	TitanSpeedMax   = 0.60
	TitanLungeMul   = 10.0  // seekBonus while lunging (≥6× cruise burst, G41)
	TitanSpineBend  = 0.085 // rad per segment — ≈63° total arc: the giant sweeps wide and never folds
	SpineBendNormal = 0.50  // rad per segment for the school (G63): the body may
	// arc like a rope around the swim axis but can never coil onto itself
	NormalTurnRate = 5.0 // rad/s the school's motion direction may swing (G63):
	// a rear target or a startle bends the path into an arc — no fish ever
	// flips and glides tail-first
	TitanLungeSec   = 1.2   // burst duration
	TitanLungeCD    = 240.0 // per-fish seconds between lunges (rare shock)
	TitanSatietySec = 12.0  // a giant belly drains this fast (normals: 40 s)

	// Rare predation (G42): giant only, starved long enough, never the Chosen.
	PredationSatiety = 0.15  // starving threshold
	PredationSustain = 20.0  // s the giant must stay starving
	PredationCD      = 300.0 // tank-wide cooldown after a feeding
	PredationPursuit = 6.0   // max chase seconds before breaking off
	// PredationPopFloor: never hunt below this live-fish count (FD11 margin).
	PredationPopFloor = MinPopulation + 2

	// Hammerhead pair (G50): a resident hunter, always on the move, and
	// never faster than the Chosen at any hour (F23 holds for the shark).
	SharkMax      = 2
	SharkSizeMin  = 1.5 // species size range (core seed only; 64 px × Size)
	SharkSizeMax  = 2.0
	SharkSpeedMin = 0.7
	SharkSpeedMax = 1.1 // ≤ her worst hour even at full chase (supremacy test)
	SharkTurnRate = 2.8 // rad/s the body axis may swing — a target behind the
	// back becomes a bounded carve, never a tail-first glide (G62)

	// Depth lanes + pod turns (G51): the 3D read.
	DepthSwing     = 0.35 // fish drift ±this around the mid lane (0 far .. 1 near)
	DepthNearAlpha = 0.88 // far-lane alpha floor (near lane = full)
	DepthFarScale  = 0.90 // far-lane width scale (near lane = 1.10)
	AvoidBigRadius = 0.55 // × body length small fish keep clear of big bodies
	TitanHeadFlat  = 0.25 // wander's vertical share — headings hug the sand line
	TitanUpperBand = 0.78 // the pod favors the upper 80% of the water column
	TitanAltMax    = 0.66 // G69: deepest personal altitude — visiting the nest level, not living there

	// Chosen wormhole pass (G73): rare by contract — a door every few minutes
	PortalGapMin    = 200.0
	PortalGapMax    = 380.0
	TitanCruiseDamp = 0.22 // pod cruise slows as members grow (×(1.12−d·sizeMul))
	TitanBurstFloor = 0.70 // capture burst grows with size (×(floor+boost·sizeMul))
	TitanBurstBoost = 0.30
	BeatBodyDamp    = 0.35  // big bodies beat their tails slower (64px reference)
	FloorLineFrac   = 0.925 // the sand line, as a fraction of tank height
	SandCellPx      = 8.0   // one disturbance cell per this many floor pixels
	SandSettleRate  = 0.25  // per-second return to the flat baseline
	SandDiffuse     = 1.2   // sideways flattening speed of the bed

	// Floor critters (G44–G45): crabs and shrimp walking the tank bottom.
	CreatureCapBase    = 8.0  // × area density, hard-capped below
	CreatureCapMax     = 14   //
	CreatureSpawnMean  = 20.0 // mean seconds between emergences
	CreatureLifeMin    = 90.0 // s of walking before the vulnerable turn
	CreatureLifeMax    = 150.0
	CreatureVulnSec    = 25.0  // final phase: struggling, luring, edible
	CreatureLureRadius = 260.0 // hungry fish notice the struggling critter
	CreatureWalkSpeed  = 14.0  // px/s along the floor
	CreatureBurrowSec  = 2.0   // an uneaten critter sinks back into the floor
)

// GLM output guard.
const MaxOutputTokens = 1600

// Pattern uniqueness gate (C6).
const (
	FingerprintQuantum   = 0.05  // density/size quantization bucket
	HueQuantum           = 10.0  // palette hue quantization bucket (degrees)
	RegistryCap          = 10000 // max fingerprints; oldest dropped
	FingerprintRepairMax = 1     // automatic repair round-trips on duplicate
)

// Agent identity strings (also used as UI labels).
const (
	AgentSpecies = "species"
	AgentWater   = "water"
	AgentPattern = "pattern"
)
