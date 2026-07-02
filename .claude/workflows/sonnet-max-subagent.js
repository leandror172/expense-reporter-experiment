export const meta = {
  name: 'sonnet-max-subagent',
  description: 'Run ONE Sonnet 5 subagent at max (or xhigh) reasoning effort on a given task. The only way to set per-agent effort — the Agent tool cannot.',
  whenToUse: 'When you want a single high-effort Sonnet subagent and need the effort knob the Agent tool lacks. Pass the full task prompt via args (string) or args.prompt.',
  phases: [{ title: 'Run', detail: 'single max-effort Sonnet agent', model: 'sonnet' }],
}

// ---------------------------------------------------------------------------
// Single-agent harness. Exists solely to expose `effort` + `model` on one
// subagent, which the Agent tool does not support. Sequential, no fan-out.
//
// Invoke (after the user has opted into running a workflow):
//   Workflow({ name: 'sonnet-max-subagent', args: '<full task prompt>' })
//   Workflow({ name: 'sonnet-max-subagent', args: { prompt: '<task>', effort: 'xhigh' } })
//
// args forms:
//   "<string>"                              → prompt, effort defaults to 'max'
//   { prompt, effort?, agentType?, label? } → effort ∈ low|medium|high|xhigh|max
//                                             (anything else falls back to 'max')
// Returns: { ok, effort, result } or { ok: false, error }.
// ---------------------------------------------------------------------------

phase('Run')

const ALLOWED_EFFORT = ['low', 'medium', 'high', 'xhigh', 'max']

const task = typeof args === 'string' ? args : (args && args.prompt)
if (!task || !String(task).trim()) {
  log('No task provided. Pass the prompt via args (a string) or args.prompt.')
  return { ok: false, error: 'no-task-prompt' }
}

const requested = (args && typeof args === 'object' && args.effort) || 'max'
const effort = ALLOWED_EFFORT.includes(requested) ? requested : 'max'
if (requested !== effort) {
  log(`Unknown effort "${requested}" — falling back to "max".`)
}

const opts = { model: 'sonnet', effort, label: (args && args.label) || `sonnet-${effort}` }
if (args && typeof args === 'object' && args.agentType) {
  opts.agentType = args.agentType
}

log(`Running one Sonnet 5 subagent at effort=${effort}.`)
const result = await agent(task, opts)

return { ok: result != null, effort, result }
