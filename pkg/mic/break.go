package mic

// breakDecision is what the mic loop must do when the silence break fires
// and the utterance ends. Pure data so the break-path exits (HANDOVER R2)
// are testable without audio hardware.
type breakDecision struct {
	// discardNoise: the session carried <45ms of audio — reset without
	// finalizing the WAV or delivering the session anywhere
	discardNoise bool
	// streamFinal: notify the streaming worker the session ended
	streamFinal bool
	// streamAborted: the Final is an abort (noise, or a prompted request
	// took the recording); only meaningful when streamFinal is set
	streamAborted bool
	// deliverLegacy: finalize and deliver the session on the legacy channel
	deliverLegacy bool
	// clearRequest: the pending prompted request was consumed
	clearRequest bool
}

// decideBreak maps the session's shape at break time to the exit the loop
// takes. Exactly one consumer ever owns a session: a noise break discards
// it, a clean streamed Final replaces legacy delivery, and a prompted
// takeover aborts the stream so only the legacy prompted path acts.
func decideBreak(streamed, prompted, noise bool) breakDecision {
	switch {
	case noise:
		// ir survives a noise false-start on purpose: the next real start
		// re-enters the start block with the same request (startSignaledFor)
		return breakDecision{discardNoise: true, streamFinal: streamed, streamAborted: streamed}
	case streamed && !prompted:
		// Streaming owns triggered sessions: the Final partial replaces the
		// legacy transcribe-then-evaluate path
		return breakDecision{streamFinal: true}
	case streamed && prompted:
		// A prompted request arrived mid-utterance and takes the recording;
		// cancel the streamed session, then the untouched legacy prompted path
		return breakDecision{streamFinal: true, streamAborted: true, deliverLegacy: true, clearRequest: true}
	default:
		return breakDecision{deliverLegacy: true, clearRequest: true}
	}
}
