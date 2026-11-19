package room

import (
	"bomb-srpg/engine"
	"fmt"
)

const maxCPUReplanAttempts = 5

type CPUStatus struct {
	Phase   engine.CPUTurnPhase
	Plan    []engine.GameEvent
	Resolve []engine.GameEvent
}

func (r *Room) launchCPUTurnIfDueLocked() {
	match := r.match
	if !match.GameCfg.VSCpu || match.WinnerTeamID != 0 || match.WorkingState.ActiveTeam != engine.Team2 ||
		match.CPU.Phase == engine.TurnPhasePlanning {
		return
	}
	match.CPU.Phase = engine.TurnPhasePlanning
	match.CPU.PlanGameEvents = nil
	match.CPU.ResolveTurnGameEvents = nil
	r.deps.Spawn(func() { r.runCPUTurn(match) })
}

func (r *Room) runCPUTurn(match *engine.Match) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.match != match {
		r.log.Info("CPU turn abandoned: match replaced or deleted")
		return
	}
	defer func() {
		match.CPU.Phase = engine.TurnPhaseReady
		r.lastActivity = r.deps.Now()
	}()

	if match.WinnerTeamID != 0 {
		r.log.Info("CPU turn abandoned: match already ended")
		return
	}
	if match.WorkingState.ActiveTeam != engine.Team2 {
		r.log.Info("CPU turn abandoned: not the CPU's turn")
		return
	}

	// A recover only catches panics in its own goroutine, so this one needs its own. Forfeit
	// the turn from a clean sandbox; a panic from the forfeit itself stays fatal.
	defer func() {
		if rec := recover(); rec != nil {
			r.log.Error("CPU turn panicked, forfeiting", "panic", rec)
			match.ResetTurn()
			match.CPU.PlanGameEvents, match.CPU.ResolveTurnGameEvents = match.ResolveTurn()
		}
	}()

	var err error
	for attempt := range maxCPUReplanAttempts {
		cmds, decideErr := r.deps.Decide(match.WorkingState)
		if decideErr != nil {
			r.log.Warn("CPU decide reported errors", "attempt", attempt, "error", decideErr)
		}
		if err = applyPlan(match, cmds); err == nil {
			break
		}
		r.log.Warn("CPU plan rejected, replanning", "attempt", attempt, "error", err)
	}
	if err != nil {
		r.log.Error("CPU replan attempts exhausted, committing partial turn", "attempts", maxCPUReplanAttempts, "error", err)
	}

	match.CPU.PlanGameEvents, match.CPU.ResolveTurnGameEvents = match.ResolveTurn()
}

func applyPlan(match *engine.Match, plan []engine.TurnCommand) error {
	for _, cmd := range plan {
		if _, err := match.ApplyTurnCommand(cmd); err != nil {
			return fmt.Errorf("command %+v rejected: %w", cmd, err)
		}
	}
	return nil
}

// ConsumeCPU returns the CPU turn phase and events, and resets a Ready phase to Idle.
// It launches the CPU turn first if one is due.
func (r *Room) ConsumeCPU(token string) (CPUStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.match == nil {
		r.log.Warn("match not found")
		return CPUStatus{}, fmt.Errorf("%w: roomID=%s", ErrMatchNotFound, r.id)
	}

	if err := r.validateAnyMatchPlayerToken(token); err != nil {
		return CPUStatus{}, err
	}

	r.launchCPUTurnIfDueLocked()

	cpuState := &r.match.CPU
	status := CPUStatus{Phase: cpuState.Phase, Plan: cpuState.PlanGameEvents, Resolve: cpuState.ResolveTurnGameEvents}
	if status.Phase == engine.TurnPhaseReady {
		cpuState.PlanGameEvents = nil
		cpuState.ResolveTurnGameEvents = nil
		cpuState.Phase = engine.TurnPhaseIdle
	}

	return status, nil
}
