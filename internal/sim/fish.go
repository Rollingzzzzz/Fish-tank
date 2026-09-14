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
	fadeFast   bool    // v1.1: swallowed prey dissolves almost at once

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

	// v1.1 titan pod state (only ever set on Role == titan fish)
	sizeMul float64       // body scale relative to the species seed (giant = 1)
	lungeT  float64       // remaining hunger-lunge seconds
	lungeCD float64       // seconds until the next lunge is allowed
	lungePt contract.Vec2 // point the current lunge dives toward

	// F25 door-to-door transit state (driven by world tickTransit)
	transiting bool
	inHole     contract.Hole // mouth we are swimming into
	outHole    contract.Hole // mouth we will emerge from
	transitPh  int           // 0 approach, 1 inside the crag, 2 leaving
	transitT   float64
	transitDur float64
	Hide01     float64 // binary (v0.3.8): 0 visible .. 1 swallowed by a door

	// courtship (managed by the world)
	CourtID   string        // partner fish id, "" when free
	CourtT    float64       // remaining seconds
	CourtTime float64       // elapsed seconds
	CourtC    contract.Vec2 // circle center
	CourtAng  float64
	EatFlash  float64 // >0 → draw a sparkle bite
	bites     int     // N8: lifetime bites (flake/treat/mite — scramble evidence)
	fleeImp   contract.Vec2
	scareT    float64       // v0.3.8: shelter-seek seconds left after a scare
	scarePt   contract.Vec2 // v1.1: where the startle came from (pod bolts away)

	// v1.1 depth lanes + pod turns
	zPhase     float64 // where the fish sits in the near/far drift cycle
	zSpeed     float64 // rad/s of that drift
	z          float64 // current lane 0 = far .. 1 = near (0.5 = the big fish)
	turnT      float64 // titan: seconds left in the current 180° curl
	cruise     float64 // titan: +1 sweeping right, -1 sweeping left
	turning    float64 // titan: seconds left of the curl's relaxed bend
	turnH0     float64 // titan: heading at the start of the convoy U-turn
	turnS      float64 // titan: arc sweep sign (+1 up-curl, -1 down-curl)
	altT       float64 // titan: seconds until the next personal altitude draw
	altY       float64 // titan: personal cruise altitude as a fraction of H
	portalPh   int     // chosen G73: 0 idle, 1 opening, 2 entering, 3 exiting
	portalT    float64 // chosen G73: phase clock (s)
	portalCD   float64 // chosen G73: idle cooldown before the next pass
	exhaleT    float64 // shark G76: gill-breath timer between micro-bubble puffs
	lastDa     float64 // G80: last heading correction sign — flip-flops betray noise
	pitchT     float64 // G87: titan level-off hysteresis window
	noiseT     float64 // G80: seconds of incoherent heading signal remaining
	headingA   float64 // titan: sweep heading (0 = right, π = left)
	slotBack   float64 // v1.1: formation distance behind the leader (px)
	slotY      float64 // v1.1: formation vertical offset from the leader (px)
	bodyLen    float64
	segLen     float64
	speed01S   float64 // G90: low-passed speed01 — the visible wave eases, never rescales in a frame
	curNight   float64 // night factor cache for the spine pass
	prevStage  string
	restTarget *contract.Vec2
}

// targetLen is the desired spine length for the current stage (px).
func (f *Fish) targetLen() float64 {
	if f.Sp.Role == contract.RoleTitan {
		// v1.1: giants own their scale class — Size 6..9 (≈384..576 px spine),
		// escorts scaled below it via sizeMul, always adult-sized.
		return 64 * contract.Clamp(f.Sp.Size, contract.TitanSizeMin, contract.TitanSizeMax) * f.sizeMul
	}
	if f.Sp.Role == contract.RoleShark {
		// v1.1: the hunter is big but sleek — Size 2.5..4, adult forever.
		return 64 * contract.Clamp(f.Sp.Size, contract.SharkSizeMin, contract.SharkSizeMax)
	}
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
	if f.Sp.Role == contract.RoleTitan {
		// v1.1: the ponderous cruise — no stage or fade multipliers, and the
		// daily pace HEAVIES as a member grows (G52 growth-weight rule).
		return contract.BaseSpeed * contract.Clamp(f.Sp.Behavior.Speed,
			contract.TitanSpeedMin, contract.TitanSpeedMax) * nMul *
			(1.12 - contract.TitanCruiseDamp*f.sizeMul)
	}
	if f.Sp.Role == contract.RoleShark {
		// v1.1: the hunter patrols at full pace, yet never outswims her —
		// the same-hour supremacy test pins that (F23 holds for the shark).
		return contract.BaseSpeed * contract.Clamp(f.Sp.Behavior.Speed,
			contract.SharkSpeedMin, contract.SharkSpeedMax) * nMul
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
		f.Fade = maxF(0, f.Fade-dt/f.deathFadeRate())
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
	f.driftDepth(w)
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
	// hunger: a giant belly drains ~3× faster — hunger fires the lunge (G41)
	f.Satiety = clampF(f.Satiety-dt/f.satietyDecay(), 0, 1)

	acc := f.steer(dt, maxSp, night, w)

	// G81: the body owns its throttle. addForce sums RAW (desired−vel)
	// deltas with no ceiling, so stacked behaviors (a separation spike, a
	// courtship tangent, a wall push) could change speed by 100+ px/s in a
	// single frame — the tank read like collision-response physics. Every
	// desire now shares the one physical thrust budget (a strike buys extra
	// authority through seekBonus; the titan mix carries its own
	// bonus-weighted clamp inside steerTitan).
	if f.Sp.Role != contract.RoleTitan {
		thrust := contract.MaxForce * maxF(1, f.seekBonus*0.6)
		if m := hyp2(acc); m > thrust {
			acc = mulS(acc, thrust/m)
		}
	}

	// integrate (seekBonus lets a food rush briefly exceed cruise speed)
	f.Vel.X += acc.X * dt
	f.Vel.Y += acc.Y * dt
	sp := hyp2(f.Vel)
	if cap := maxSp * maxF(1, f.seekBonus); sp > cap {
		f.Vel = mulS(f.Vel, glideCap(sp, cap)) // G88: ceiling drops glide, never snap
	}
	// flee impulses decay
	f.Vel.X += f.fleeImp.X * dt
	f.Vel.Y += f.fleeImp.Y * dt
	f.fleeImp = mulS(f.fleeImp, maxF(0, 1-1.2*dt))
	f.scareT = maxF(0, f.scareT-dt) // v0.3.8: shelter-seek window ticks down

	// v1.1 G58/G59: a curling scalare carves the turn forward, never stalls
	if f.turning > 0 && f.Sp.Role == contract.RoleTitan {
		f.carveCurlTurn(maxSp)
	}

	// G62: the hammerhead rides its body axis — bounded turn, cruise floor
	if f.Sp.Role == contract.RoleShark {
		f.constrainForward(dt, maxSp, w)
	}
	if f.Sp.Role != contract.RoleTitan && f.Sp.Role != contract.RoleShark {
		f.capTurn(dt) // G63: school fish ARC, never flip — the body rides nose-first
	}

	f.Pos.X += f.Vel.X * dt
	f.Pos.Y += f.Vel.Y * dt

	f.applyFrameBounds(w, dt)

	// N3/G52: nothing alive but the Chosen may enter the aura — and for a
	// big body "enter" means ANY spine segment, head to tail
	w.enforceFishZones(f, dt)

	// N5: occasional glass attach for high-Attachment species
	f.maybeAttach(w, dt)

	// growth toward the stage target length
	f.bodyLen += (f.targetLen() - f.bodyLen) * minF(1, dt*0.5)
	f.segLen = f.bodyLen / (contract.SpineSegments - 1)
	// F7: elders beat their tail slower as the fade advances; a lounging
	// tail barely sways (F15)
	phaseMul := 1.0
	if f.loungeT > 0 {
		phaseMul = 0.3
	}
	// v1.1: big bodies beat their tails slower — the heavy, real read
	// (64 px reference fish; a 576 px giant beats at ~a quarter the rate)
	// G90: the beat rides the EASED pace (speed01S) — raw speed01 would
	// rescale the beat in a single frame on every burst grant or spend.
	f.phase += dt * (3.2 + 7.5*f.speed01S) * (1 - 0.25*f.ElderP) * phaseMul *
		64 / (64 + contract.BeatBodyDamp*f.bodyLen)
	f.followSpine(dt)
	w.dragSpineOut(f)
	f.clampBodyInFrame(w) // G66: the whole drawn body stays in the view
}
