// G2.1: fish entity — spine chain, steering forces, feeding and rest.
package sim

import (
	"math/rand"

	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// Fish is one living fish.
type Fish struct {
	ID      string
	Sp      *contract.Species
	Seed    int64
	rng     *rand.Rand
	Pos     contract.Vec2
	Vel     contract.Vec2
	Spine   []contract.Vec2
	phase   float64
	wanderA float64

	AgeDays float64
	Stage   string
	Satiety float64 // 1 = full belly, decays over ~40 s
	Energy  float64 // 1 = rested, drains with speed
	Resting bool

	Pal contract.Palette // current palette (repainted by the Pattern agent)
	Pat contract.Pattern

	Dying      bool
	Fade       float64 // 1 → 0 while dying
	DieReason  string
	SparedOnce bool
	ElderP     float64 // F7: 0..1 elder fade progress (drives render desat)

	seekBonus float64 // F3: >1 while chasing food — velocity + force allowance

	// F4 ambient behavior states (driven by world tickBehaviors)
	chaseID string
	chaseT  float64
	zoomT   float64
	zoomAng float64
	zoomC   contract.Vec2

	// N5 glass-attach state (pleco hugging the glass)
	attachT  float64
	attachPo contract.Vec2

	// F15 cave-lounge state (driven by world tickLounge)
	loungeT    float64       // remaining dwell seconds inside the cave
	loungeC    contract.Vec2 // center of the claimed cave zone
	loungeNext float64       // countdown to the next lounge pick

	// v0.3.8 roaming: the current tour waypoint and its refresh countdown
	tourC contract.Vec2
	tourT float64

	// F25 door-to-door transit state (driven by world tickTransit)
	transiting bool
	inHole     contract.Hole // mouth we are swimming into
	outHole    contract.Hole // mouth we will emerge from
	transitPh  int           // 0 approach, 1 inside the crag, 2 leaving
	transitT   float64
	transitDur float64
	Hide01     float64 // binary (v0.3.8): 0 visible .. 1 swallowed by a door

	// courtship (managed by the world)
	CourtID    string        // partner fish id, "" when free
	CourtT     float64       // remaining seconds
	CourtTime  float64       // elapsed seconds
	CourtC     contract.Vec2 // circle center
	CourtAng   float64
	EatFlash   float64 // >0 → draw a sparkle bite
	bites      int     // N8: lifetime bites (flake/treat/mite — scramble evidence)
	fleeImp    contract.Vec2
	scareT     float64 // v0.3.8: shelter-seek seconds left after a scare
	bodyLen    float64
	segLen     float64
	curNight   float64 // night factor cache for the spine pass
	prevStage  string
	restTarget *contract.Vec2
}

// targetLen is the desired spine length for the current stage (px).
func (f *Fish) targetLen() float64 {
	return 64 * contract.Clamp(f.Sp.Size, 0.5, 1.6) * StageScale(f.Stage)
}

// maxSpeed applies behavior, stage and night multipliers.
func (f *Fish) maxSpeed(night float64) float64 {
	nMul := 1.0
	if f.Sp.Behavior.NightActive {
		nMul = 0.82 + 0.36*night
	} else {
		nMul = 1.18 - 0.36*night
	}
	stage := StageSpeed(f.Stage)
	if f.Sp.Role == contract.RoleChosen {
		// F23: the eternal one is structurally fastest — adult pace at every
		// age (a restored elder keeps her prime) and a live multiplier that
		// keeps her above even a night-active normal at her worst hour.
		stage = 1
		return contract.BaseSpeed * contract.Clamp(f.Sp.Behavior.Speed, 0.4, 1.8) *
			stage * nMul * contract.ChosenSpeedMul
	}
	return contract.BaseSpeed * contract.Clamp(f.Sp.Behavior.Speed, 0.4, 1.8) *
		stage * nMul * (0.55 + 0.45*f.Fade)
}

// advance integrates steering, spine, feeding and rest for dt seconds.
// ctx carries world lookups (kept as params to keep Fish free of World).
func (f *Fish) advance(dt, night float64, w *World) {
	if f.Dying {
		f.Fade = maxF(0, f.Fade-dt/contract.DeathFadeSec)
		f.Pos.X += f.Vel.X * dt * 0.3
		f.Pos.Y += (f.Vel.Y*dt*0.3 + 8*dt) // slowly sink while fading
		f.followSpine(dt)
		return
	}
	// N5: glass-attach — the pleco clamps onto the wall, sucking in place
	if f.attachT > 0 {
		f.attachT -= dt
		f.Vel = v2(0, 0)
		// slow sucking pulses against the glass
		pulse := sin(w.time*2.2+float64(f.Seed%5)) * 1.2
		f.Pos = v2(f.attachPo.X+pulse, f.attachPo.Y)
		f.Energy = minF(1, f.Energy+dt*0.12) // resting counts double
		f.phase += dt * 1.4
		f.followSpine(dt)
		if f.attachT <= 0 {
			// lets go and glides off to find a new spot
			push := 1.0
			if f.Pos.X < w.W/2 {
				push = -1.0
			}
			f.Vel = v2(push*f.maxSpeed(night)*0.5, 0)
		}
		return
	}
	// F25: inside the crag — the position is scripted along the transit path;
	// the spine keeps following so the tail stays alive on the way out
	if f.transiting && f.transitPh == 1 {
		f.phase += dt * 2
		f.followSpine(dt)
		return
	}
	maxSp := f.maxSpeed(night)
	f.curNight = night
	speed01 := clampF(hyp2(f.Vel)/maxSp, 0, 1)

	// energy economy
	if f.Resting {
		f.Energy = minF(1, f.Energy+dt*0.16)
	} else if f.loungeT > 0 {
		f.Energy = minF(1, f.Energy+dt*0.3) // F15: the cave truly restores
	} else {
		// v0.3.8: gentler drain (~40% less) — the school stays lively instead
		// of constantly filing into the caves to recharge
		f.Energy = clampF(f.Energy-dt*(0.006+0.012*speed01), 0, 1)
	}
	f.Satiety = clampF(f.Satiety-dt/40, 0, 1)

	acc := f.steer(dt, maxSp, night, w)

	// integrate (seekBonus lets a food rush briefly exceed cruise speed)
	f.Vel.X += acc.X * dt
	f.Vel.Y += acc.Y * dt
	sp := hyp2(f.Vel)
	if cap := maxSp * maxF(1, f.seekBonus); sp > cap {
		f.Vel = mulS(f.Vel, cap/sp)
	}
	// flee impulses decay
	f.Vel.X += f.fleeImp.X * dt
	f.Vel.Y += f.fleeImp.Y * dt
	f.fleeImp = mulS(f.fleeImp, maxF(0, 1-1.2*dt))
	f.scareT = maxF(0, f.scareT-dt) // v0.3.8: shelter-seek window ticks down

	f.Pos.X += f.Vel.X * dt
	f.Pos.Y += f.Vel.Y * dt

	// impenetrable tank bounds — a fish can never leave the water
	if f.Pos.X < 8 {
		f.Pos.X = 8
		if f.Vel.X < 0 {
			f.Vel.X = 0
		}
	}
	if f.Pos.X > w.W-8 {
		f.Pos.X = w.W - 8
		if f.Vel.X > 0 {
			f.Vel.X = 0
		}
	}
	if f.Pos.Y < 8 {
		f.Pos.Y = 8
		if f.Vel.Y < 0 {
			f.Vel.Y = 0
		}
	}
	if f.Pos.Y > w.H-8 {
		f.Pos.Y = w.H - 8
		if f.Vel.Y > 0 {
			f.Vel.Y = 0
		}
	}

	// N3: nothing alive but the Chosen may enter the aura
	w.enforceZones(&f.Pos, &f.Vel, f.Sp.Role == contract.RoleChosen)

	// N5: occasional glass attach for high-Attachment species
	if f.Sp.Behavior.Attachment > 0.5 && f.attachT <= 0 && !f.Resting &&
		f.loungeT <= 0 && f.CourtID == "" && f.chaseT <= 0 && f.zoomT <= 0 &&
		f.rng.Float64() < 0.05*f.Sp.Behavior.Attachment*dt {
		x := 14.0
		if f.Pos.X >= w.W/2 {
			x = w.W - 14
		}
		f.attachPo = v2(x, clampF(f.Pos.Y, 70, w.H-90))
		f.attachT = 30 + f.rng.Float64()*60
		f.Vel = v2(0, 0)
		if f.rng.Float64() < 0.4 {
			w.logf("behavior", "the "+f.Sp.Name+" clamps onto the glass")
		}
	}

	// growth toward the stage target length
	f.bodyLen += (f.targetLen() - f.bodyLen) * minF(1, dt*0.5)
	f.segLen = f.bodyLen / (contract.SpineSegments - 1)
	// F7: elders beat their tail slower as the fade advances; a lounging
	// tail barely sways (F15)
	phaseMul := 1.0
	if f.loungeT > 0 {
		phaseMul = 0.3
	}
	f.phase += dt * (3.2 + 7.5*speed01) * (1 - 0.25*f.ElderP) * phaseMul
	f.followSpine(dt)
}

// eat applies a successful bite.
func (f *Fish) eat() {
	f.Satiety = 1
	f.EatFlash = 0.6
	f.Energy = minF(1, f.Energy+0.08)
	f.bites++
}

// Attached reports whether the fish is currently suctioned to the glass
// (F18: the renderer gives attached suckers their flat mouth-on look).
func (f *Fish) Attached() bool { return f.attachT > 0 && !f.Dying }

// AttachWall returns the wall the sucker clings to: -1 left edge, +1 right
// edge (0 when not attached).
func (f *Fish) AttachWall() int {
	if f.attachT <= 0 || f.Dying {
		return 0
	}
	if f.attachPo.X < 30 {
		return -1
	}
	return 1
}

// flee applies a skittish impulse away from (x, y).
func (f *Fish) flee(x, y, strength float64) {
	d := sub(f.Pos, v2(x, y))
	l := maxF(hyp2(d), 1)
	f.fleeImp.X += d.X / l * strength
	f.fleeImp.Y += d.Y / l * strength
	f.scareT = contract.ScareShelterSec // v0.3.8: keep seeking cover after the impulse fades
	f.Resting = false
	f.restTarget = nil
	f.attachT = 0    // a startled pleco lets go of the glass (N5)
	f.loungeT = 0    // fear shatters the cave calm (F15)
	f.abortTransit() // F25: a scared fish abandons the approach
}
