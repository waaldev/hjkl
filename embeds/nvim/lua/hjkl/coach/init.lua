local M = {}
local why = require("hjkl.coach.why")

local default_events = vim.fn.expand("~/.local/share/hjkl/coach-events.jsonl")

local cfg = {
  events_path = default_events,
  state_path = nil, -- default: coach-state.json next to events_path
  quiet = false,
  hint_throttle_ms = 8000,
  flush_ms = 60000,
  -- A pattern's hint is shown this many times in total, then it fades and
  -- only the weekly review mentions it.
  hint_limit = 5,
  hjkl_cmd = "hjkl",
  -- :HjklWhy settings (see why.lua). Suggestions themselves are switched
  -- on in hjkl's config: [ai] suggest = true.
  why = {},
}

local last_hint = 0
-- The most recent anti-pattern, for :HjklDrill and :HjklSnooze.
local last_skill, last_pattern = nil, nil

-- Persistent state: how often each hint was shown, snoozed patterns, and
-- commands already seen in real work.
local state = { hints = {}, snoozed = {}, seen = {}, since = nil }

local function state_path()
  return cfg.state_path or (vim.fn.fnamemodify(cfg.events_path, ":h") .. "/coach-state.json")
end

local function load_state()
  local f = io.open(state_path(), "r")
  if not f then
    return
  end
  local ok, data = pcall(vim.json.decode, f:read("*a"))
  f:close()
  if ok and type(data) == "table" then
    state.hints = data.hints or {}
    state.snoozed = data.snoozed or {}
    state.seen = data.seen or {}
    state.since = data.since
  end
end

local function save_state()
  vim.fn.mkdir(vim.fn.fnamemodify(state_path(), ":h"), "p")
  local f = io.open(state_path(), "w")
  if not f then
    return
  end
  f:write(vim.json.encode(state))
  f:close()
end

local function now_ms()
  if vim.uv and vim.uv.now then
    return vim.uv.now()
  end
  return vim.loop.now()
end

local function append_events(evs)
  if #evs == 0 then
    return
  end
  local dir = vim.fn.fnamemodify(cfg.events_path, ":h")
  vim.fn.mkdir(dir, "p")
  local f = io.open(cfg.events_path, "a")
  if not f then
    return
  end
  local ts = os.date("!%Y-%m-%dT%H:%M:%SZ")
  for _, ev in ipairs(evs) do
    ev.ts = ts
    if not ev.count or ev.count == 0 then
      ev.count = 1
    end
    f:write(vim.json.encode(ev) .. "\n")
  end
  f:close()
end

local function notify(msg)
  local t = now_ms()
  if t - last_hint < (cfg.hint_throttle_ms or 8000) then
    return false
  end
  last_hint = t
  vim.notify("hjkl: " .. msg, vim.log.levels.INFO, { title = "hjkl coach" })
  return true
end

-- hint escalates, then fades: the first showings teach, the last one
-- points at a drill, and after hint_limit the coach stays quiet about it.
local function hint(pattern, skill, msg)
  last_skill, last_pattern = skill, pattern
  if cfg.quiet then
    return
  end
  local until_ts = state.snoozed[pattern]
  if until_ts and os.time() < until_ts then
    return
  end
  local shown = state.hints[pattern] or 0
  if shown >= cfg.hint_limit then
    return
  end
  if shown == cfg.hint_limit - 1 then
    msg = msg .. "  - last reminder: :HjklDrill to practice it, :HjklSnooze to mute"
  end
  if notify(msg) then
    state.hints[pattern] = shown + 1
  end
end

local function emit(pattern, skill, count, keys)
  append_events({ { pattern = pattern, skill = skill, keys = keys, count = count, filetype = vim.bo.filetype or "" } })
end

-- Anti-pattern: the same key repeated. The hint shows when the run reaches
-- its threshold; one event with the final length is written when it ends.
local runs = {
  j = { n = 5, pattern = "repeat-j", skill = "hjkl", msg = "try a count (5j) or /search instead of jjjjj" },
  k = { n = 5, pattern = "repeat-k", skill = "hjkl", msg = "try a count (5k) or ?search instead of kkkkk" },
  h = { n = 5, pattern = "repeat-h", skill = "line-ends", msg = "try 0 ^ b instead of hhhhh" },
  l = { n = 5, pattern = "repeat-l", skill = "line-ends", msg = "try $ e w or f{char} instead of lllll" },
  x = { n = 4, pattern = "repeat-x", skill = "operators", msg = "try dw / diw instead of xxxx" },
  w = { n = 5, pattern = "repeat-w", skill = "search", msg = "try / or f{char} instead of wwwww" },
  ["<arrow>"] = { n = 3, pattern = "arrow-keys", skill = "hjkl", msg = "hjkl beats arrow keys - hands stay on the home row" },
}

local arrows = { ["<Left>"] = true, ["<Right>"] = true, ["<Up>"] = true, ["<Down>"] = true }

-- Anti-pattern: a short sequence. Matched against the tail of `recent`.
local sequences = {
  { keys = "viwd", pattern = "visual-iw-delete", skill = "text-objects", msg = "diw repeats with . - viwd does not" },
  { keys = "viwy", pattern = "visual-iw-yank", skill = "text-objects", msg = "yiw is the operator form" },
  { keys = "Vjjj", pattern = "visual-line-walk", skill = "operators", msg = "2dd or dj beats Vjjjd, and . can replay it" },
}

-- Keys whose next key is an argument, not a command (f{char}, "{reg}, ...).
local arg_keys = { f = true, t = true, F = true, T = true, r = true, m = true, ["'"] = true, ["`"] = true, ['"'] = true, q = true, ["@"] = true }

-- Single-key commands worth counting when you use them in real work.
local used_single = { ["."] = true, [";"] = true, [","] = true, n = true, N = true, ["*"] = true, ["#"] = true, ["%"] = true, ["@"] = true }

local run_key, run_len = nil, 0
local recent = ""
local op_buf = nil -- operator + motion being typed, e.g. "ci" then "ciw"
local skip_arg = false
local used = {} -- keys -> count, flushed periodically

local function flush_run()
  local r = run_key and runs[run_key]
  if r and run_len >= r.n then
    emit(r.pattern, r.skill, run_len, run_key == "<arrow>" and "<arrow>" or string.rep(run_key, run_len))
  end
  run_key, run_len = nil, 0
end

local function feed_run(tok)
  if arrows[tok] then
    tok = "<arrow>"
  end
  if tok ~= run_key then
    flush_run()
    run_key, run_len = tok, 0
  end
  run_len = run_len + 1
  local r = runs[tok]
  if r and run_len == r.n then
    hint(r.pattern, r.skill, r.msg)
  end
end

local function feed_sequence(tok)
  recent = (recent .. tok):sub(-12)
  for _, s in ipairs(sequences) do
    if recent:sub(-#s.keys) == s.keys then
      emit(s.pattern, s.skill, 1, s.keys)
      hint(s.pattern, s.skill, s.msg)
      recent = ""
      return
    end
  end
end

-- The first day only records what you already use. After that, the first
-- real use of a command gets a word of encouragement: it means a drill made
-- it into your daily work.
local baseline_secs = 24 * 60 * 60

local function count_used(keys)
  used[keys] = (used[keys] or 0) + 1
  if state.seen[keys] then
    return
  end
  state.seen[keys] = true
  if not cfg.quiet and state.since and os.time() - state.since > baseline_secs then
    notify("first real " .. keys .. " - nice, that one made it out of the dojo")
  end
end

local function finish_op()
  if op_buf and #op_buf > 1 then
    count_used(op_buf)
  end
  op_buf = nil
end

function M.flush()
  flush_run()
  finish_op()
  save_state()
  local evs = {}
  for keys, n in pairs(used) do
    table.insert(evs, { pattern = "used", keys = keys, count = n })
  end
  used = {}
  append_events(evs)
end

local function keytrans(typed)
  local ok, s = pcall(vim.fn.keytrans, typed)
  if ok and s and s ~= "" then
    return s
  end
  return typed
end

-- on_key runs before the key is handled, so `mode` is the mode the key was
-- typed in: "n", "no" (operator pending), "v"/"V"/"^V" (Visual), "i", ...
function M._on_key(trans, mode)
  local m = mode:sub(1, 1)
  -- A key outside operator-pending mode means the last operator finished.
  if op_buf and mode:sub(1, 2) ~= "no" then
    finish_op()
  end
  if m == "i" and arrows[trans] then
    emit("insert-arrows", "modes", 1, "i+arrow")
    hint("insert-arrows", "modes", "Esc to Normal, move, then i/a. <C-o> for one Normal command")
    return
  end

  if mode:sub(1, 2) == "no" then
    -- Motion or text object for a pending operator.
    if op_buf and trans == "<Esc>" then
      op_buf = nil
    elseif op_buf then
      op_buf = op_buf .. trans
    end
    feed_sequence(trans)
    return
  end

  if m ~= "n" and m ~= "v" and m ~= "V" and m ~= "\22" then
    return
  end

  feed_run(trans)
  feed_sequence(trans)

  if m ~= "n" then
    return
  end
  if skip_arg then
    skip_arg = false
    return
  end
  if trans == "d" or trans == "c" or trans == "y" then
    op_buf = trans
  elseif used_single[trans] then
    count_used(trans)
  end
  if arg_keys[trans] then
    skip_arg = true
  end
end

function M.setup(opts)
  cfg = vim.tbl_deep_extend("force", cfg, opts or {})
  load_state()
  if not state.since then
    state.since = os.time()
    save_state()
  end

  -- :HjklDrill [skill] opens a short drill for the last habit the coach
  -- flagged (or the skill you name), so a hint turns into practice.
  vim.api.nvim_create_user_command("HjklDrill", function(o)
    local skill = o.args ~= "" and o.args or last_skill
    local cmd = { cfg.hjkl_cmd, "drill" }
    if skill then
      table.insert(cmd, skill)
    end
    vim.cmd("tabnew")
    vim.fn.termopen(cmd, {
      on_exit = function()
        vim.schedule(function()
          pcall(vim.cmd, "tabclose")
        end)
      end,
    })
    vim.cmd("startinsert")
  end, { nargs = "?", desc = "hjkl: drill the last flagged habit" })

  -- :HjklSnooze [pattern] mutes a hint for a week (default: the last one).
  vim.api.nvim_create_user_command("HjklSnooze", function(o)
    local pattern = o.args ~= "" and o.args or last_pattern
    if not pattern then
      vim.notify("hjkl: nothing to snooze yet", vim.log.levels.INFO)
      return
    end
    state.snoozed[pattern] = os.time() + 7 * 24 * 60 * 60
    save_state()
    vim.notify("hjkl: " .. pattern .. " snoozed for a week", vim.log.levels.INFO)
  end, { nargs = "?", desc = "hjkl: mute a coach hint for a week" })

  why.setup(vim.tbl_extend("force", { hjkl_cmd = cfg.hjkl_cmd }, cfg.why), hint)

  vim.on_key(function(_, typed)
    if typed == nil or typed == "" then
      return
    end
    local trans = keytrans(typed)
    why.on_key(trans)
    M._on_key(trans, vim.api.nvim_get_mode().mode)
  end)
  local timer = (vim.uv or vim.loop).new_timer()
  timer:start(cfg.flush_ms, cfg.flush_ms, vim.schedule_wrap(M.flush))
  vim.api.nvim_create_autocmd("VimLeavePre", {
    group = vim.api.nvim_create_augroup("hjkl_coach", { clear = true }),
    callback = M.flush,
  })
end

return M
