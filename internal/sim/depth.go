// v1.1: depth lanes (G51) — the water has near and far lanes and every
// small fish drifts slowly between them. The renderer draws the school in
// lane order, so fish sometimes cross the big bodies in front and sometimes
// slip behind them (the 3D read); the big fish hold the mid lane. This file
// also owns the small Fish helpers that outgrew fish.go's line ceiling.
package sim

import (
	"github.com/Rollingzzzzz/Fish-tank/internal/contract"
)

// driftDepth eases a fish between the near and far lanes. Small fish sweep
// the whole range so they cross the big bodies in front (near) or slip
// behind them (far); the scalare pair-pod drifts in the FAR lane — the big
// brothers cruise behind the school, rarely stepping forward (G59).
func (f *Fish) driftDepth(w *World) {
	if f.Sp.Role == contract.RoleTitan || f.Sp.Role == contract.RoleShark {
		f.z = 0.30 + 0.12*sin(w.time*0.05*0.4+f.zPhase)
		return
	}
	if f.Sp.Role != contract.RoleNormal {
		return
	}
	f.z = 0.5 + contract.DepthSwing*sin(w.time*f.zSpeed+f.zPhase)
}

// Z reports the depth lane (the renderer sorts by it, far → near).
func (f *Fish) Z() float64 { return f.z }

// avoidBigBodies steers a small fish around the titans and the sharks —
// nobody crosses through a big body; they flow around the silhouette.
func (f *Fish) avoidBigBodies(w *World, maxSp float64, addForce func(contract.Vec2, float64)) {
	for _, o := range w.fishes {
		if o == f || (o.Sp.Role != contract.RoleTitan && o.Sp.Role != contract.RoleShark) {
			continue
		}
		d := sub(f.Pos, o.Pos)
		l := hyp2(d)
		r := o.bodyLen * contract.AvoidBigRadius
		if l >= r || l < 1 {
			continue
		}
		urgency := 1 + 2*(1-l/r)
		addForce(mulS(d, maxSp*urgency/l), 1.3)
	}
}

// depthBandSteer pulls a fish toward its preferred water band; the shark
// also respects the upper-80% preference (v1.1: the sand line is not for
// the big residents).
func (f *Fish) depthBandSteer(w *World, night, maxSp float64, addForce func(contract.Vec2, float64)) {
	band := contract.Clamp(f.Sp.Behavior.Depth, 0, 1)
	if f.Sp.Behavior.NightActive {
		band = clampF(band-0.25*night, 0, 1)
	}
	prefY := w.H * (0.18 + 0.62*band)
	addForce(v2(0, (prefY-f.Pos.Y)*0.25), 0.25)
	if f.Sp.Role == contract.RoleShark && f.Pos.Y > w.H*contract.TitanUpperBand {
		addForce(v2(0, -maxSp), 1.0)
	}
}

// formationSteer holds a pod member in its staggered slot behind the
// leader (G59) — the procession reads as one loose body, never a stack.
func (f *Fish) formationSteer(w *World, maxSp float64, addForce func(contract.Vec2, float64)) {
	lead := w.titanGiant()
	if lead == nil || lead == f {
		return
	}
	tx := lead.Pos.X - f.slotBack*float64(f.cruise)
	ty := lead.Pos.Y + f.slotY
	d := sub(v2(tx, ty), f.Pos)
	if l := hyp2(d); l > 30 {
		addForce(mulS(d, maxSp*0.55/l), 0.8)
	}
}

// burstMul scales the capture burst with body size: a bigger giant whips
// its tail over more distance in the one lunge (G52 growth-weight rule).
func (f *Fish) burstMul() float64 {
	return contract.TitanLungeMul *
		(contract.TitanBurstFloor + contract.TitanBurstBoost*f.sizeMul)
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

// maybeAttach lets a high-Attachment species clamp onto the glass (N5) —
// moved here from fish.go for the line ceiling; the caller passes dt.
func (f *Fish) maybeAttach(w *World, dt float64) {
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
}
